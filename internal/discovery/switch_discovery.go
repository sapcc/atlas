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
	switchConfig struct {
		RefreshInterval int      `yaml:"refresh_interval"`
		NetboxHost      string   `yaml:"netbox_host"`
		NetboxAPIToken  string   `yaml:"netbox_api_token"`
		TargetsFileName string   `yaml:"targets_file_name"`
		Switches        []device `yaml:"switches"`
	}
	device struct {
		Name         string `yaml:"name"`
		Manufacturer string `yaml:"manufacturer"`
		API          string `yaml:"api"`
	}
	configValues struct {
		Region string
	}
)

func init() {
	Register("switch", NewSwitchDiscovery)
}

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
		tgs, err := sd.getSwitches()
		if err == nil {
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

	for _, device := range sd.cfg.Switches {
		switch device.API {
		case "dcim":
			tg, err = sd.loadSDCIMSwitches(device)
		case "vm":
			tg, err = sd.loadVMSwitches(device)
		}
		if err != nil {
			return tgroups, err
		}
		tgroups = append(tgroups, tg...)
	}
	return tgroups, err
}

func (sd *SwitchDiscovery) loadSDCIMSwitches(d device) (tgroups []*targetgroup.Group, err error) {
	dcims, err := sd.netbox.DevicesByRegion(d.Name, d.Manufacturer, sd.region, "1")
	if err != nil {
		return tgroups, level.Error(log.With(sd.logger, "component", "SwitchDiscovery")).Log("error", err)
	}
	for _, dv := range dcims {
		if dv.PrimaryIP == nil {
			level.Error(log.With(sd.logger, "component", "SwitchDiscovery")).Log("error", fmt.Sprintf("cannot find ip address of switch %d. Error: %s", dv.ID, err))
			continue
		}
		tgroup, err := sd.createGroup(d.Name, *dv.Name, *dv.PrimaryIP.Address, *dv.Status.Label, *dv.DeviceType.Manufacturer.Name, *dv.DeviceType.Model, strconv.Itoa(int(dv.ID)))
		if err != nil {
			level.Error(log.With(sd.logger, "component", "SwitchDiscovery")).Log("error", err)
			continue
		}
		tgroups = append(tgroups, tgroup)
	}

	return tgroups, err
}

func (sd *SwitchDiscovery) loadVMSwitches(d device) (tgroups []*targetgroup.Group, err error) {
	vms, err := sd.netbox.VMsByTag(sd.region, "1", "admin-context")
	if err != nil {
		return tgroups, level.Error(log.With(sd.logger, "component", "SwitchDiscovery")).Log("error", err)
	}
	for _, vm := range vms {
		if vm.PrimaryIP == nil {
			level.Error(log.With(sd.logger, "component", "SwitchDiscovery")).Log("error", fmt.Sprintf("cannot find ip address of switch %d. Error: %s", vm.ID, err))
			continue
		}
		tgroup, err := sd.createGroup(d.Name, *vm.Name, *vm.PrimaryIP.Address, *vm.Status.Label, "", "", strconv.Itoa(int(vm.ID)))
		if err != nil {
			level.Error(log.With(sd.logger, "component", "SwitchDiscovery")).Log("error", err)
			continue
		}
		tgroups = append(tgroups, tgroup)
	}

	return tgroups, err
}

func (sd *SwitchDiscovery) createGroup(name, deviceName, ip, status, manufacturer, modelName, id string) (tgroup *targetgroup.Group, err error) {
	var deviceIP net.IP
	if strings.ToUpper(status) != "ACTIVE" {
		return tgroup, fmt.Errorf("Ignoring device: %s. Status: %s", id, status)
	}
	deviceIP, _, err = net.ParseCIDR(ip)
	if err != nil {
		deviceIP = net.ParseIP(ip)
	}
	tgroup = &targetgroup.Group{
		Source:  deviceIP.String(),
		Labels:  make(model.LabelSet),
		Targets: make([]model.LabelSet, 0, 1),
	}

	target := model.LabelSet{model.AddressLabel: model.LabelValue(deviceIP.String())}
	labels := model.LabelSet{
		model.LabelName("module"):       model.LabelValue(strings.Replace(name, sd.region+"-", "", 1)),
		model.LabelName("server_name"):  model.LabelValue(deviceName),
		model.LabelName("state"):        model.LabelValue(status),
		model.LabelName("manufacturer"): model.LabelValue(manufacturer),
		model.LabelName("model"):        model.LabelValue(modelName),
		model.LabelName("server_id"):    model.LabelValue(id),
	}

	tgroup.Labels = labels
	tgroup.Targets = append(tgroup.Targets, target)
	return tgroup, err
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
