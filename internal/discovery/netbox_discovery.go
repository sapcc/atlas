package discovery

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/hosting-de-labs/go-netbox/netbox/models"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/discovery/targetgroup"
	"github.com/sapcc/atlas/pkg/adapter"
	"github.com/sapcc/atlas/pkg/config"
	"github.com/sapcc/atlas/pkg/netbox"
	"github.com/sapcc/atlas/pkg/writer"
)

type (
	NetboxDiscovery struct {
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

const netboxDiscovery = "netbox"

func init() {
	Register(netboxDiscovery, NewNetboxDiscovery)
}

//NewNetboxDiscovery creates
func NewNetboxDiscovery(disc interface{}, ctx context.Context, opts config.Options, w writer.Writer, l log.Logger) (d Discovery, err error) {
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

	a := adapter.NewPrometheus(ctx, cfg.TargetsFileName, w, l)

	return &NetboxDiscovery{
		adapter:         a,
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
	var wg sync.WaitGroup
	groupCh := make(chan []*targetgroup.Group, 0)
	wg.Add(len(sd.cfg.DCIM.Devices))
	for _, dcim := range sd.cfg.DCIM.Devices {
		go sd.loadDcimDevices(dcim, &wg, groupCh)
	}
	wg.Add(len(sd.cfg.Virtualization.VMs))
	for _, vm := range sd.cfg.Virtualization.VMs {
		go sd.loadVirtualizationVMs(vm, &wg, groupCh)
	}
	go func() {
		wg.Wait()
		close(groupCh)
	}()
	for groups := range groupCh {
		tgroups = append(tgroups, groups...)
	}
	return tgroups, err
}

func (sd *NetboxDiscovery) loadDcimDevices(d dcimDevice, w *sync.WaitGroup, groupsCh chan<- []*targetgroup.Group) {
	var dcims []models.Device
	var wg sync.WaitGroup
	var tgroups []*targetgroup.Group
	defer w.Done()
	groupCh := make(chan *targetgroup.Group, 0)
	dcims, err := sd.netbox.DevicesByParams(d.DcimDevicesListParams)
	if err != nil {
		level.Error(log.With(sd.logger, "component", "NetboxDiscovery")).Log("error", fmt.Errorf("Error loading devices. Error: %s", err.Error()))
		return
	}
	wg.Add(len(dcims))
	for _, dv := range dcims {
		go sd.createGroup(d.CustomLabels, d.MetricsLabel, d.Target, dv, &wg, groupCh)
	}
	go func() {
		wg.Wait()
		close(groupCh)
	}()

	for group := range groupCh {
		tgroups = append(tgroups, group)
	}
	groupsCh <- tgroups
}

func (sd *NetboxDiscovery) loadVirtualizationVMs(d virtualizationVM, w *sync.WaitGroup, groupsCh chan<- []*targetgroup.Group) {
	var wg sync.WaitGroup
	var tgroups []*targetgroup.Group
	groupCh := make(chan *targetgroup.Group, 0)
	defer w.Done()
	vms, err := sd.netbox.VMsByParams(d.VirtualizationVirtualMachinesListParams)
	if err != nil {
		level.Error(log.With(sd.logger, "component", "NetboxDiscovery")).Log("error", fmt.Errorf("Error loading vms. Error: %s", err.Error()))
		return
	}
	wg.Add(len(vms))
	for _, vm := range vms {
		go sd.createGroup(d.CustomLabels, d.MetricsLabel, d.Target, vm, &wg, groupCh)
	}
	go func() {
		wg.Wait()
		close(groupCh)
	}()

	for group := range groupCh {
		tgroups = append(tgroups, group)
	}
	groupsCh <- tgroups
}

func (sd *NetboxDiscovery) createGroup(c map[string]string, metricsLabel string, t int, d interface{}, wg *sync.WaitGroup, groupsCh chan<- *targetgroup.Group) {
	cLabels := model.LabelSet{}
	var tgroup *targetgroup.Group
	defer wg.Done()
	for k, v := range c {
		cLabels[model.LabelName(k)] = model.LabelValue(v)
	}
	switch dv := d.(type) {
	case models.Device:
		deviceIP, err := sd.getDeviceIP(t, dv.ID, dv.PrimaryIP)
		id := strconv.Itoa(int(dv.ID))
		if err != nil {
			level.Error(log.With(sd.logger, "component", "NetboxDiscovery")).Log("error", fmt.Errorf("Ignoring device: %s. Error: %s", id, err.Error()))
			return
		}

		tgroup = &targetgroup.Group{
			Source:  strconv.Itoa(rand.Intn(300000000)),
			Labels:  make(model.LabelSet),
			Targets: make([]model.LabelSet, 0, 1),
		}
		if strings.ToUpper(*dv.Status.Label) != "ACTIVE" {
			level.Error(log.With(sd.logger, "component", "NetboxDiscovery")).Log("error", fmt.Errorf("Ignoring device: %s. Status: %s", id, *dv.Status.Label))
			return
		}
		target := model.LabelSet{model.AddressLabel: model.LabelValue(deviceIP)}
		labels := model.LabelSet{
			model.LabelName("name"):          model.LabelValue(dv.DisplayName),
			model.LabelName("server_name"):   model.LabelValue(*dv.Name),
			model.LabelName("manufacturer"):  model.LabelValue(*dv.DeviceType.Manufacturer.Name),
			model.LabelName("status"):        model.LabelValue(*dv.Status.Label),
			model.LabelName("serial"):        model.LabelValue(dv.Serial),
			model.LabelName("model"):         model.LabelValue(*dv.DeviceType.Model),
			model.LabelName("server_id"):     model.LabelValue(id),
			model.LabelName("role"):          model.LabelValue(*dv.DeviceRole.Slug),
			model.LabelName("metrics_label"): model.LabelValue(metricsLabel),
		}
		labels = labels.Merge(cLabels)

		tgroup.Labels = labels
		tgroup.Targets = append(tgroup.Targets, target)
	case models.VirtualMachine:
		id := strconv.Itoa(int(dv.ID))
		deviceIP, err := sd.getDeviceIP(t, dv.ID, dv.PrimaryIP)
		if err != nil {
			level.Error(log.With(sd.logger, "component", "NetboxDiscovery")).Log("error", fmt.Errorf("Ignoring vm: %s. Error: %s", id, err.Error()))
			return
		}
		tgroup = &targetgroup.Group{
			Source:  strconv.Itoa(rand.Intn(300000000)),
			Labels:  make(model.LabelSet),
			Targets: make([]model.LabelSet, 0, 1),
		}
		if strings.ToUpper(*dv.Status.Label) != "ACTIVE" {
			level.Error(log.With(sd.logger, "component", "NetboxDiscovery")).Log("error", fmt.Errorf("Ignoring vm: %s. Status: %s", id, *dv.Status.Label))
			return
		}
		target := model.LabelSet{model.AddressLabel: model.LabelValue(deviceIP)}
		labels := model.LabelSet{
			model.LabelName("state"):         model.LabelValue(*dv.Status.Label),
			model.LabelName("server_name"):   model.LabelValue(*dv.Name),
			model.LabelName("server_id"):     model.LabelValue(id),
			model.LabelName("role"):          model.LabelValue(*dv.Role.Slug),
			model.LabelName("metrics_label"): model.LabelValue(metricsLabel),
		}
		labels = labels.Merge(cLabels)
		tgroup.Labels = labels
		tgroup.Targets = append(tgroup.Targets, target)
	default:
		level.Error(log.With(sd.logger, "component", "NetboxDiscovery")).Log("error", fmt.Errorf("Not supported device interface"))
		return
	}
	groupsCh <- tgroup
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

func (sd *NetboxDiscovery) setMetrics() {
	labels := make(map[string]int, 0)
	for _, dcim := range sd.cfg.DCIM.Devices {
		if _, ok := labels[dcim.MetricsLabel]; !ok {
			labels[dcim.MetricsLabel] = 0
			setMetricsLabelAndValue(sd.status.Targets, dcim.MetricsLabel, sd.adapter.GetNumberOfTargetsFor(dcim.MetricsLabel))
		}
	}
	for _, vm := range sd.cfg.Virtualization.VMs {
		if _, ok := labels[vm.MetricsLabel]; !ok {
			labels[vm.MetricsLabel] = 0
			setMetricsLabelAndValue(sd.status.Targets, vm.MetricsLabel, sd.adapter.GetNumberOfTargetsFor(vm.MetricsLabel))
		}
	}
}

func (sd *NetboxDiscovery) Up() bool {
	return sd.status.Up
}

func (sd *NetboxDiscovery) Targets() map[string]int {
	sd.status.Targets = make(map[string]int)
	sd.setMetrics()
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

func (sd *NetboxDiscovery) GetName() string {
	return netboxDiscovery
}

func (sd *NetboxDiscovery) GetAdapter() adapter.Adapter {
	return sd.adapter
}
