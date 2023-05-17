package discovery

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"sync"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/go-openapi/runtime"
	multierror "github.com/hashicorp/go-multierror"
	"github.com/netbox-community/go-netbox/netbox/models"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/discovery/targetgroup"
	"github.com/sapcc/atlas/pkg/adapter"
	"github.com/sapcc/atlas/pkg/config"
	"github.com/sapcc/atlas/pkg/errgroup"
	"github.com/sapcc/atlas/pkg/netbox"
	"github.com/sapcc/atlas/pkg/writer"
	"gopkg.in/yaml.v2"
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
		rateLimiter     *time.Ticker
	}

	netboxConfig struct {
		RefreshInterval int            `yaml:"refresh_interval"`
		NetboxHost      string         `yaml:"netbox_host"`
		NetboxAPIToken  string         `yaml:"netbox_api_token"`
		RateLimiter     time.Duration  `yaml:"rate_limit"`
		TargetsFileName string         `yaml:"targets_file_name"`
		DCIM            dcim           `yaml:"dcim"`
		Virtualization  virtualization `yaml:"virtualization"`
		ConfigmapName   string         `yaml:"configmap_name"`
	}

	configValues struct {
		Region string
	}
)

const netboxDiscovery = "netbox"

func init() {
	Register(netboxDiscovery, NewNetboxDiscovery)
}

// NewNetboxDiscovery creates
func NewNetboxDiscovery(disc interface{}, ctx context.Context, opts config.Options, l log.Logger) (d Discovery, err error) {
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

	var w writer.Writer
	if cfg.ConfigmapName != "" {
		w, err = writer.NewConfigMap(cfg.ConfigmapName, opts.NameSpace, l)
	} else {
		w, err = writer.NewFile(cfg.TargetsFileName, l)
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
	defer func() {
		if sd.rateLimiter != nil {
			sd.rateLimiter.Stop()
		}
	}()
	for c := time.Tick(time.Duration(sd.refreshInterval) * time.Second); ; {
		level.Debug(log.With(sd.logger, "component", "NetboxDiscovery")).Log("debug", "Loading Netbox data")
		if sd.cfg.RateLimiter > 0 && sd.rateLimiter == nil {
			sd.rateLimiter = time.NewTicker(sd.cfg.RateLimiter * time.Millisecond)
		}
		tgs, err := sd.loadData()
		if err == nil {
			level.Debug(log.With(sd.logger, "component", "NetboxDiscovery")).Log("debug", "netbox data loaded")
			sd.status.Lock()
			sd.status.Up = true
			sd.status.Unlock()
			if len(tgs) == 0 {
				level.Debug(log.With(sd.logger, "component", "NetboxDiscovery")).Log("debug", "empty netbox device list; no update...")
			} else {
				ch <- tgs
			}
		} else {
			level.Debug(log.With(sd.logger, "component", "NetboxDiscovery")).Log("error", "error loading netbox data "+err.Error())
			merr, ok := err.(*multierror.Error)
			if merr != nil {
				if ok {
					for _, err := range merr.Errors {
						var apiError *runtime.APIError
						if errors.As(err, &apiError) {
							if apiError.Code == 400 {
								level.Error(log.With(sd.logger, "component", "NetboxDiscovery")).Log("error", "invalid netbox query / "+err.Error())
							}
							if apiError.Code > 400 {
								level.Error(log.With(sd.logger, "component", "NetboxDiscovery")).Log("error", "error netbox api", apiError.Error())
							}
						} else {
							level.Error(log.With(sd.logger, "component", "NetboxDiscovery")).Log("error", err)
						}
					}
				}
			}
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

func (sd *NetboxDiscovery) loadData() (tgroups []*targetgroup.Group, err error) {
	groupCh := make(chan []*targetgroup.Group, 0)
	var eg errgroup.Group
	for _, dcim := range sd.cfg.DCIM.Devices {
		func(dcim dcimDevice) {
			eg.Go(func() error {
				return sd.loadDcimDevices(dcim, groupCh)
			})
		}(dcim)
	}
	for _, vm := range sd.cfg.Virtualization.VMs {
		func(vm virtualizationVM) {
			eg.Go(func() error {
				return sd.loadVirtualizationVMs(vm, groupCh)
			})
		}(vm)
	}
	go func() error {
		if err = eg.Wait(); err != nil {
			close(groupCh)
			return err
		}
		close(groupCh)
		return nil
	}()

	for groups := range groupCh {
		tgroups = append(tgroups, groups...)
	}
	return
}

func (sd *NetboxDiscovery) loadDcimDevices(d dcimDevice, groupsCh chan<- []*targetgroup.Group) (err error) {
	var dcims []models.DeviceWithConfigContext
	var wg sync.WaitGroup
	var tgroups []*targetgroup.Group
	groupCh := make(chan *targetgroup.Group, 0)
	dcims, err = sd.netbox.DevicesByParams(d.DcimDevicesListParams)
	if err != nil {
		dout, _ := yaml.Marshal(d.DcimDevicesListParams)
		return fmt.Errorf("Error loading devices / Query=%s: %w", string(dout), err)
	}
	level.Debug(log.With(sd.logger, "component", "NetboxDiscovery")).Log("debug", fmt.Sprintf("found %d dcimDevices", len(dcims)))
	wg.Add(len(dcims))
	for _, dv := range dcims {
		if sd.cfg.RateLimiter > 0 {
			<-sd.rateLimiter.C
		}

		go sd.createGroups(d.CustomLabels, d.MetricsLabel, d.Target, dv, &wg, groupCh)
	}
	go func() {
		wg.Wait()
		close(groupCh)
	}()

	for group := range groupCh {
		tgroups = append(tgroups, group)
	}
	groupsCh <- tgroups
	return
}

func (sd *NetboxDiscovery) loadVirtualizationVMs(d virtualizationVM, groupsCh chan<- []*targetgroup.Group) (err error) {
	var wg sync.WaitGroup
	var tgroups []*targetgroup.Group
	groupCh := make(chan *targetgroup.Group, 0)
	vms, err := sd.netbox.VMsByParams(d.VirtualizationVirtualMachinesListParams)
	if err != nil {
		dout, _ := yaml.Marshal(d)
		return fmt.Errorf("Error loading vms %s: %w", string(dout), err)
	}
	level.Debug(log.With(sd.logger, "component", "NetboxDiscovery")).Log("debug", fmt.Sprintf("found %d virtualizationVM", len(vms)))
	wg.Add(len(vms))
	for _, vm := range vms {
		if sd.cfg.RateLimiter > 0 {
			<-sd.rateLimiter.C
		}
		go sd.createGroups(d.CustomLabels, d.MetricsLabel, d.Target, vm, &wg, groupCh)
	}
	go func() {
		wg.Wait()
		close(groupCh)
	}()

	for group := range groupCh {
		tgroups = append(tgroups, group)
	}
	groupsCh <- tgroups
	return
}

func (sd *NetboxDiscovery) createGroups(c map[string]string, metricsLabel string, t int, d interface{}, wg *sync.WaitGroup, groupsCh chan<- *targetgroup.Group) {
	cLabels := model.LabelSet{}
	var tgroup *targetgroup.Group
	defer wg.Done()
	for k, v := range c {
		cLabels[model.LabelName(k)] = model.LabelValue(v)
	}
	switch dv := d.(type) {
	case models.DeviceWithConfigContext:
		deviceIPs, err := sd.getDeviceIP(t, dv.ID, dv.PrimaryIP)
		id := strconv.Itoa(int(dv.ID))
		if err != nil {
			level.Error(log.With(sd.logger, "component", "NetboxDiscovery")).Log("error", fmt.Errorf("Ignoring device: %s. Error: %s", id, err.Error()))
			return
		}
		if len(deviceIPs) == 0 {
			level.Error(log.With(sd.logger, "component", "NetboxDiscovery")).Log("error", fmt.Errorf("Ignoring device: %s. Error: no device ips", id))
			return
		}
		for _, deviceIP := range deviceIPs {
			tgroup = &targetgroup.Group{
				Source:  strconv.Itoa(rand.Intn(300000000)),
				Labels:  make(model.LabelSet),
				Targets: make([]model.LabelSet, 0, 1),
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
			if dv.Site != nil && dv.Site.Slug != nil {
				labels[model.LabelName("site")] = model.LabelValue(*dv.Site.Slug)
			}
			if dv.Cluster != nil && dv.Cluster.Name != nil {
				labels[model.LabelName("cluster")] = model.LabelValue(*dv.Cluster.Name)
			}

			labels = labels.Merge(cLabels)

			tgroup.Labels = labels
			tgroup.Targets = append(tgroup.Targets, target)
		}

	case models.VirtualMachineWithConfigContext:
		id := strconv.Itoa(int(dv.ID))
		deviceIPs, err := sd.getDeviceIP(t, dv.ID, dv.PrimaryIP)
		if err != nil {
			level.Error(log.With(sd.logger, "component", "NetboxDiscovery")).Log("error", fmt.Errorf("Ignoring vm: %s. Error: %s", id, err.Error()))
			return
		}
		if len(deviceIPs) == 0 {
			level.Error(log.With(sd.logger, "component", "NetboxDiscovery")).Log("error", fmt.Errorf("Ignoring device: %s. Error: no vm ips", id))
			return
		}
		for _, deviceIP := range deviceIPs {
			tgroup = &targetgroup.Group{
				Source:  strconv.Itoa(rand.Intn(300000000)),
				Labels:  make(model.LabelSet),
				Targets: make([]model.LabelSet, 0, 1),
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
		}

	default:
		level.Error(log.With(sd.logger, "component", "NetboxDiscovery")).Log("error", fmt.Errorf("not supported device interface"))
		return
	}
	groupsCh <- tgroup
}

func (sd *NetboxDiscovery) getDeviceIP(t int, id int64, i *models.NestedIPAddress) (ips []string, err error) {
	ips = make([]string, 0)
	switch t {
	case managementIP:
		ips, err = sd.netbox.ManagementIPs(strconv.FormatInt(id, 10))
	case primaryIP:
		var ip string
		ip, err = sd.netbox.GetNestedDeviceIP(i)
		ips = append(ips, ip)
	case loopback10:
		ips, err = sd.netbox.DeviceInterfaceNameIPs("Loopback10", strconv.FormatInt(id, 10))
	default:
		return ips, fmt.Errorf("Error getting ip from device: %d. Error: %s", id, "unknown target in config")
	}
	if err != nil {
		return ips, fmt.Errorf("Error getting ip from device: %d. Error: %s", id, err.Error())
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
