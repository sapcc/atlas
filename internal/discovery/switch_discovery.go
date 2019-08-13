package discovery

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/hosting-de-labs/go-netbox/netbox/client/dcim"
	"github.com/hosting-de-labs/go-netbox/netbox/client/virtualization"
	"github.com/hosting-de-labs/go-netbox/netbox/models"
	"github.com/prometheus/common/model"
	promDiscovery "github.com/prometheus/prometheus/discovery"
	"github.com/prometheus/prometheus/discovery/targetgroup"
	"github.com/sapcc/atlas/pkg/adapter"
	"github.com/sapcc/atlas/pkg/config"
	"github.com/sapcc/atlas/pkg/netbox"
	"github.com/sapcc/atlas/pkg/writer"
)

type (
	SwitchDiscovery struct {
		manager         *promDiscovery.Manager
		adapter         adapter.Adapter
		netbox          *netbox.Netbox
		region          string
		refreshInterval int
		logger          log.Logger
		status          *Status
		outputFile      string
		cfg             switchConfig
	}
	Device struct {
		Module string `yaml:"module"`
		Ctx    string `yaml:"ctx"`
	}
	switchConfig struct {
		RefreshInterval int    `yaml:"refresh_interval"`
		NetboxHost      string `yaml:"netbox_host"`
		NetboxAPIToken  string `yaml:"netbox_api_token"`
		TargetsFileName string `yaml:"targets_file_name"`
		DCIM            []DCIM `yaml:"dcim"`
		VM              []VM   `yaml:"vm"`
	}

	DCIM struct {
		dcim.DcimDevicesListParams `yaml:",inline"`
		Device                     `yaml:",inline"`
	}

	VM struct {
		virtualization.VirtualizationVirtualMachinesListParams `yaml:",inline"`
		Device                                                 `yaml:",inline"`
	}

	configValues struct {
		Region string
	}

	group struct {
	}
)

func init() {
	Register("switch", NewSwitchDiscovery)
}

//NewSwitchDiscovery creates
func NewSwitchDiscovery(disc interface{}, ctx context.Context, m *promDiscovery.Manager, opts config.Options, w writer.Writer, l log.Logger) (d Discovery, err error) {
	var cfg switchConfig
	configValues := configValues{Region: opts.Region}
	if err := UnmarshalHandler(disc, &cfg, configValues); err != nil {
		return nil, err
	}

	nClient, err := netbox.New(cfg.NetboxHost, cfg.NetboxAPIToken)
	if err != nil {
		return nil, err
	}

	if err != nil {
		return d, err
	}

	a := adapter.NewPrometheus(ctx, m, cfg.TargetsFileName, w, l)

	return &SwitchDiscovery{
		adapter:         a,
		manager:         m,
		netbox:          nClient,
		region:          opts.Region,
		refreshInterval: cfg.RefreshInterval,
		logger:          l,
		status:          &Status{Up: false},
		outputFile:      cfg.TargetsFileName,
		cfg:             cfg,
	}, err

}

func (sd *SwitchDiscovery) Run(ctx context.Context, ch chan<- []*targetgroup.Group) {
	for c := time.Tick(time.Duration(sd.refreshInterval) * time.Second); ; {
		level.Debug(log.With(sd.logger, "component", "SwitchDiscovery")).Log("debug", "Loading Switches")
		tgs, err := sd.getSwitches()
		if err == nil {
			level.Debug(log.With(sd.logger, "component", "SwitchDiscovery")).Log("debug", "Done Loading Switches")
			sd.status.Lock()
			sd.status.Up = true
			sd.status.Unlock()
			ch <- tgs
		} else {
			level.Error(log.With(sd.logger, "component", "SwitchDiscovery")).Log("err", err)
			sd.status.Lock()
			sd.status.Up = false
			sd.status.Unlock()
			continue
		}
		// Wait for ticker or exit when ctx is closed.
		select {
		case <-c:
			continue
		case <-ctx.Done():
			return
		}
	}
}

func (sd *SwitchDiscovery) getSwitches() (tgroups []*targetgroup.Group, err error) {
	var tg []*targetgroup.Group
	for _, dcim := range sd.cfg.DCIM {
		if dcim.Module == "" {
			return tgroups, fmt.Errorf("Mandatory field Module not set for dcim device")
		}
		tg, err = sd.loadDCIMs(dcim)
		if err != nil {
			return tgroups, err
		}
		tgroups = append(tgroups, tg...)
	}
	for _, vm := range sd.cfg.VM {
		if vm.Module == "" {
			return tgroups, fmt.Errorf("Mandatory field Module not set for vm device")
		}
		tg, err = sd.loadVMs(vm)
		if err != nil {
			return tgroups, err
		}
		tgroups = append(tgroups, tg...)
	}
	return tgroups, err
}

func (sd *SwitchDiscovery) loadDCIMs(d DCIM) (tgroups []*targetgroup.Group, err error) {
	dcims, err := sd.netbox.DevicesByParams(d.DcimDevicesListParams)
	if err != nil {
		return tgroups, level.Error(log.With(sd.logger, "component", "SwitchDiscovery")).Log("error", err)
	}
	for _, dv := range dcims {
		var names []string
		if d.Ctx != "" {
			context := strings.Split(d.Ctx, ",")
			for _, c := range context {
				names = append(names, d.Module+c)

			}
		} else {
			names = append(names, d.Module)
		}
		for _, n := range names {
			tgroup, err := sd.createGroup(n, dv)
			if err != nil {
				level.Error(log.With(sd.logger, "component", "SwitchDiscovery")).Log("error", err)
				continue
			}
			tgroups = append(tgroups, tgroup)
		}
	}

	return tgroups, err
}

func (sd *SwitchDiscovery) loadVMs(d VM) (tgroups []*targetgroup.Group, err error) {
	vms, err := sd.netbox.VMsByParams(d.VirtualizationVirtualMachinesListParams)
	if err != nil {
		return tgroups, level.Error(log.With(sd.logger, "component", "SwitchDiscovery")).Log("error", err)
	}
	for _, vm := range vms {
		name := d.Module
		if d.Module == "" {
			name = *vm.Platform.Slug
		}
		tgroup, err := sd.createGroup(name, vm)
		if err != nil {
			level.Error(log.With(sd.logger, "component", "SwitchDiscovery")).Log("error", err)
			continue
		}
		tgroups = append(tgroups, tgroup)
	}

	return tgroups, err
}

func (sd *SwitchDiscovery) createGroup(n string, d interface{}) (tgroup *targetgroup.Group, err error) {
	switch dv := d.(type) {
	case models.Device:
		id := strconv.Itoa(int(dv.ID))
		deviceIP, err := getDeviceIP(dv.PrimaryIP)
		if err != nil {
			return tgroup, fmt.Errorf("Ignoring device: %s. Error: %s", id, err.Error())
		}
		tgroup = &targetgroup.Group{
			Source:  deviceIP.String(),
			Labels:  make(model.LabelSet),
			Targets: make([]model.LabelSet, 0, 1),
		}
		if strings.ToUpper(*dv.Status.Label) != "ACTIVE" {
			return tgroup, fmt.Errorf("Ignoring device: %s. Status: %s", id, *dv.Status.Label)
		}
		target := model.LabelSet{model.AddressLabel: model.LabelValue(deviceIP.String())}
		labels := model.LabelSet{
			model.LabelName("module"):       model.LabelValue(strings.Replace(n, sd.region+"-", "", 1)),
			model.LabelName("server_name"):  model.LabelValue(dv.DisplayName),
			model.LabelName("state"):        model.LabelValue(*dv.Status.Label),
			model.LabelName("manufacturer"): model.LabelValue(*dv.DeviceType.Manufacturer.Name),
			model.LabelName("model"):        model.LabelValue(*dv.DeviceType.Model),
			model.LabelName("server_id"):    model.LabelValue(id),
			model.LabelName("role"):         model.LabelValue(*dv.DeviceRole.Slug),
		}

		tgroup.Labels = labels
		tgroup.Targets = append(tgroup.Targets, target)
	case models.VirtualMachine:
		id := strconv.Itoa(int(dv.ID))
		deviceIP, err := getDeviceIP(dv.PrimaryIP)
		if err != nil {
			return tgroup, fmt.Errorf("Ignoring device: %s. Error: %s", id, err.Error())
		}
		tgroup = &targetgroup.Group{
			Source:  deviceIP.String(),
			Labels:  make(model.LabelSet),
			Targets: make([]model.LabelSet, 0, 1),
		}
		if strings.ToUpper(*dv.Status.Label) != "ACTIVE" {
			return tgroup, fmt.Errorf("Ignoring device: %s. Status: %s", id, *dv.Status.Label)
		}
		target := model.LabelSet{model.AddressLabel: model.LabelValue(deviceIP.String())}
		labels := model.LabelSet{
			model.LabelName("module"):    model.LabelValue(strings.Replace(n, sd.region+"-", "", 1)),
			model.LabelName("state"):     model.LabelValue(*dv.Status.Label),
			model.LabelName("server_id"): model.LabelValue(id),
			model.LabelName("role"):      model.LabelValue(*dv.Role.Slug),
		}

		tgroup.Labels = labels
		tgroup.Targets = append(tgroup.Targets, target)
	default:
		return tgroup, fmt.Errorf("Not supported device interface")
	}

	return tgroup, err
}

func getDeviceIP(i *models.NestedIPAddress) (ip net.IP, err error) {
	if i == nil {
		return ip, fmt.Errorf("No IP Address")
	}
	ip, _, err = net.ParseCIDR(*i.Address)
	if err != nil {
		ip = net.ParseIP(*i.Address)
	}
	return
}

func (sd *SwitchDiscovery) addCtx() {

}

func (sd *SwitchDiscovery) Up() bool {
	return sd.status.Up

}
func (sd *SwitchDiscovery) Lock() {
	sd.status.Lock()

}
func (sd *SwitchDiscovery) Unlock() {
	sd.status.Unlock()
}

func (sd *SwitchDiscovery) GetOutputFile() string {
	return sd.outputFile
}

func (sd *SwitchDiscovery) StartAdapter() {
	sd.adapter.Run()
}

func (sd *SwitchDiscovery) GetAdapter() adapter.Adapter {
	return sd.adapter
}

func (sd *SwitchDiscovery) GetManager() *promDiscovery.Manager {
	return sd.manager
}
