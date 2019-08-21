package discovery

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
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
	NetboxDiscovery struct {
		manager         *promDiscovery.Manager
		adapter         adapter.Adapter
		netbox          *netbox.Netbox
		region          string
		refreshInterval int
		logger          log.Logger
		status          *Status
		outputFile      string
		cfg             netboxConfig
	}

	netboxConfig struct {
		RefreshInterval int            `yaml:"refresh_interval"`
		NetboxHost      string         `yaml:"netbox_host"`
		NetboxAPIToken  string         `yaml:"netbox_api_token"`
		TargetsFileName string         `yaml:"targets_file_name"`
		DCIM            dcim           `yaml:"dcim"`
		Virtualization  virtualization `yaml:"virtualization"`
	}

	configValues struct {
		Region string
	}
)

func init() {
	Register("netbox", NewNetboxDiscovery)
}

//NewNetboxDiscovery creates
func NewNetboxDiscovery(disc interface{}, ctx context.Context, m *promDiscovery.Manager, opts config.Options, w writer.Writer, l log.Logger) (d Discovery, err error) {
	var cfg netboxConfig
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

	return &NetboxDiscovery{
		adapter:         a,
		manager:         m,
		netbox:          nClient,
		region:          opts.Region,
		refreshInterval: cfg.RefreshInterval,
		logger:          l,
		status:          &Status{Up: false, Targets: make(map[string]int)},
		outputFile:      cfg.TargetsFileName,
		cfg:             cfg,
	}, err

}

func (sd *NetboxDiscovery) Run(ctx context.Context, ch chan<- []*targetgroup.Group) {
	for c := time.Tick(time.Duration(sd.refreshInterval) * time.Second); ; {
		level.Debug(log.With(sd.logger, "component", "NetboxDiscovery")).Log("debug", "Loading Devices")
		tgs, err := sd.getData()
		if err == nil {
			level.Debug(log.With(sd.logger, "component", "NetboxDiscovery")).Log("debug", "Done Loading Devices")
			sd.status.Lock()
			sd.status.Up = true
			sd.status.Unlock()
			ch <- tgs
		} else {
			level.Error(log.With(sd.logger, "component", "NetboxDiscovery")).Log("error", err)
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

func (sd *NetboxDiscovery) getData() (tgroups []*targetgroup.Group, err error) {
	var tg []*targetgroup.Group
	for _, dcim := range sd.cfg.DCIM.Devices {
		tg, err = sd.loadDcimDevices(dcim)
		if err != nil {
			return tgroups, err
		}
		tgroups = append(tgroups, tg...)
		setMetricsLabelAndValue(sd.status.Targets, dcim.MetricsLabel, len(tg))
	}
	for _, vm := range sd.cfg.Virtualization.VMs {
		tg, err = sd.loadVirtualizationVMs(vm)
		if err != nil {
			return tgroups, err
		}
		tgroups = append(tgroups, tg...)
		setMetricsLabelAndValue(sd.status.Targets, vm.MetricsLabel, len(tg))
	}
	return tgroups, err
}

func (sd *NetboxDiscovery) loadDcimDevices(d dcimDevice) (tgroups []*targetgroup.Group, err error) {
	dcims, err := sd.netbox.DevicesByParams(d.DcimDevicesListParams)
	if err != nil {
		return tgroups, level.Error(log.With(sd.logger, "component", "NetboxDiscovery")).Log("error", err)
	}
	for _, dv := range dcims {
		tgroup, err := sd.createGroup(d.CustomLabels, d.Target, dv)
		if err != nil {
			level.Error(log.With(sd.logger, "component", "NetboxDiscovery")).Log("error", err)
			continue
		}
		tgroups = append(tgroups, tgroup)
	}

	return tgroups, err
}

func (sd *NetboxDiscovery) loadVirtualizationVMs(d virtualizationVM) (tgroups []*targetgroup.Group, err error) {
	vms, err := sd.netbox.VMsByParams(d.VirtualizationVirtualMachinesListParams)
	if err != nil {
		return tgroups, level.Error(log.With(sd.logger, "component", "NetboxDiscovery")).Log("error", err)
	}
	for _, vm := range vms {
		tgroup, err := sd.createGroup(d.CustomLabels, d.Target, vm)
		if err != nil {
			level.Error(log.With(sd.logger, "component", "NetboxDiscovery")).Log("error", err)
			continue
		}
		tgroups = append(tgroups, tgroup)
	}

	return tgroups, err
}

func (sd *NetboxDiscovery) createGroup(c map[string]string, t int, d interface{}) (tgroup *targetgroup.Group, err error) {
	cLabels := model.LabelSet{}
	for k, v := range c {
		cLabels[model.LabelName(k)] = model.LabelValue(v)
	}
	switch dv := d.(type) {
	case models.Device:
		deviceIP, err := sd.getDeviceIP(t, dv.ID, dv.PrimaryIP)
		id := strconv.Itoa(int(dv.ID))
		if err != nil {
			return tgroup, fmt.Errorf("Ignoring device: %s. Error: %s", id, err.Error())
		}

		tgroup = &targetgroup.Group{
			Source:  strconv.Itoa(rand.Intn(300000000)),
			Labels:  make(model.LabelSet),
			Targets: make([]model.LabelSet, 0, 1),
		}
		if strings.ToUpper(*dv.Status.Label) != "ACTIVE" {
			return tgroup, fmt.Errorf("Ignoring device: %s. Status: %s", id, *dv.Status.Label)
		}
		target := model.LabelSet{model.AddressLabel: model.LabelValue(deviceIP)}
		labels := model.LabelSet{
			model.LabelName("name"):         model.LabelValue(dv.DisplayName),
			model.LabelName("server_name"):  model.LabelValue(*dv.Name),
			model.LabelName("manufacturer"): model.LabelValue(*dv.DeviceType.Manufacturer.Name),
			model.LabelName("status"):       model.LabelValue(*dv.Status.Label),
			model.LabelName("model"):        model.LabelValue(*dv.DeviceType.Model),
			model.LabelName("server_id"):    model.LabelValue(id),
			model.LabelName("role"):         model.LabelValue(*dv.DeviceRole.Slug),
		}
		labels = labels.Merge(cLabels)

		tgroup.Labels = labels
		tgroup.Targets = append(tgroup.Targets, target)
	case models.VirtualMachine:
		id := strconv.Itoa(int(dv.ID))
		deviceIP, err := sd.getDeviceIP(t, dv.ID, dv.PrimaryIP)
		if err != nil {
			return tgroup, fmt.Errorf("Ignoring device: %s. Error: %s", id, err.Error())
		}
		tgroup = &targetgroup.Group{
			Source:  strconv.Itoa(rand.Intn(300000000)),
			Labels:  make(model.LabelSet),
			Targets: make([]model.LabelSet, 0, 1),
		}
		if strings.ToUpper(*dv.Status.Label) != "ACTIVE" {
			return tgroup, fmt.Errorf("Ignoring device: %s. Status: %s", id, *dv.Status.Label)
		}
		target := model.LabelSet{model.AddressLabel: model.LabelValue(deviceIP)}
		labels := model.LabelSet{
			model.LabelName("state"):       model.LabelValue(*dv.Status.Label),
			model.LabelName("server_name"): model.LabelValue(*dv.Name),
			model.LabelName("server_id"):   model.LabelValue(id),
			model.LabelName("role"):        model.LabelValue(*dv.Role.Slug),
		}
		labels = labels.Merge(cLabels)
		tgroup.Labels = labels
		tgroup.Targets = append(tgroup.Targets, target)
	default:
		return tgroup, fmt.Errorf("Not supported device interface")
	}

	return tgroup, err
}

func (sd *NetboxDiscovery) getDeviceIP(t int, id int64, i *models.NestedIPAddress) (ip string, err error) {
	switch t {
	case managementIP:
		ip, err = sd.netbox.ManagementIP(id)
	case primaryIP:
		ip, err = sd.netbox.GetNestedDeviceIP(i)
	default:
		return ip, fmt.Errorf("Error getting ip from device: %d. Error: %s", id, "unknown target in config")
	}

	if err != nil {
		return ip, fmt.Errorf("Error getting ip from device: %d. Error: %s", id, err.Error())
	}
	return
}

func (sd *NetboxDiscovery) Up() bool {
	return sd.status.Up
}

func (sd *NetboxDiscovery) Targets() map[string]int {
	return sd.status.Targets
}

func (sd *NetboxDiscovery) Lock() {
	sd.status.Lock()

}
func (sd *NetboxDiscovery) Unlock() {
	sd.status.Unlock()
}

func (sd *NetboxDiscovery) GetOutputFile() string {
	return sd.outputFile
}

func (sd *NetboxDiscovery) StartAdapter() {
	sd.adapter.Run()
}

func (sd *NetboxDiscovery) GetAdapter() adapter.Adapter {
	return sd.adapter
}

func (sd *NetboxDiscovery) GetManager() *promDiscovery.Manager {
	return sd.manager
}
