package discovery

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/common/model"
	"github.com/sapcc/ipmi_sd/pkg/adapter"
	"github.com/sapcc/ipmi_sd/pkg/config"
	"github.com/sapcc/ipmi_sd/pkg/netbox"
	"github.com/sapcc/ipmi_sd/pkg/writer"

	"github.com/go-kit/kit/log"
	promDiscovery "github.com/prometheus/prometheus/discovery"
	"github.com/prometheus/prometheus/discovery/targetgroup"
)

type (
	ControlPlaneDiscovery struct {
		manager         *promDiscovery.Manager
		adapter         adapter.Adapter
		netbox          *netbox.Netbox
		region          string
		refreshInterval int
		logger          log.Logger
		status          *Status
		outputFile      string
	}
	netboxConfig struct {
		RefreshInterval int    `yaml:"refresh_interval"`
		NetboxHost      string `yaml:"netbox_host"`
		NetboxAPIToken  string `yaml:"netbox_api_token"`
		TargetsFileName string `yaml:"targets_file_name"`
	}
)

func init() {
	Register("control_plane", NewControlPlaneDiscovery)
}

//NewControlPlaneDiscovery creates a new ControlPlaneDiscovery
func NewControlPlaneDiscovery(disc interface{}, ctx context.Context, m *promDiscovery.Manager, opts config.Options, w writer.Writer, l log.Logger) (d Discovery, err error) {
	var cfg netboxConfig
	if err := UnmarshalHandler(disc, &cfg); err != nil {
		return nil, err
	}
	nClient, err := netbox.New(cfg.NetboxHost, cfg.NetboxAPIToken)
	if err != nil {
		return nil, err
	}

	if err != nil {
		return d, err
	}

	a := adapter.NewPrometheus(ctx, m, opts.ConfigmapName, w, l)

	return &ControlPlaneDiscovery{
		adapter:         a,
		manager:         m,
		netbox:          nClient,
		region:          opts.Region,
		refreshInterval: cfg.RefreshInterval,
		logger:          l,
		status:          &Status{Up: false},
		outputFile:      cfg.TargetsFileName,
	}, err
}

func (nd *ControlPlaneDiscovery) Run(ctx context.Context, ch chan<- []*targetgroup.Group) {
	for c := time.Tick(time.Duration(nd.refreshInterval) * time.Second); ; {
		tgs, err := nd.getNodes()
		if err == nil {
			nd.status.Lock()
			nd.status.Up = true
			nd.status.Unlock()
			ch <- tgs
		} else {
			nd.status.Lock()
			nd.status.Up = false
			nd.status.Unlock()
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

func (d *ControlPlaneDiscovery) Up() bool {
	return d.status.Up

}
func (d *ControlPlaneDiscovery) Lock() {
	d.status.Lock()

}
func (d *ControlPlaneDiscovery) Unlock() {
	d.status.Unlock()
}

func (d *ControlPlaneDiscovery) GetOutputFile() string {
	return d.outputFile
}

func (d *ControlPlaneDiscovery) StartAdapter() {
	d.adapter.Run()
}

func (d *ControlPlaneDiscovery) GetAdapter() adapter.Adapter {
	return d.adapter
}

func (d *ControlPlaneDiscovery) GetManager() *promDiscovery.Manager {
	return d.manager
}

func (nd *ControlPlaneDiscovery) getNodes() ([]*targetgroup.Group, error) {

	servers, err := nd.netbox.ServersByRegion("cp", nd.region)
	if err != nil {
		level.Error(log.With(nd.logger, "component", "ControlPlaneDiscovery")).Log("err", err)
		return nil, err
	}

	var tgroups []*targetgroup.Group

	for _, server := range servers {

		if *server.Status.Value != 1 {
			level.Warn(log.With(nd.logger, "component", "ControlPlaneDiscovery")).Log("warn", fmt.Sprintf("Status value is not 1 for server: %s. Skipping the server!", server.Name))
			continue
		}

		ip, err := nd.netbox.ManagementIP(server.ID)
		if err != nil {
			level.Warn(log.With(nd.logger, "component", "ControlPlaneDiscovery")).Log("warn", fmt.Sprintf("Error during getting management IP for server %s: %v. Skipping the server!", server.Name, err))
			continue
		}

		nodeName := ""
		if server.PrimaryIP != nil {
			primaryIP, err := nd.netbox.IPAddress(server.PrimaryIP.ID)

			if err != nil {
				level.Warn(log.With(nd.logger, "component", "ControlPlaneDiscovery")).Log("warn", err)
			} else {
				nodeName = primaryIP.Description
			}

		} else {
			level.Warn(log.With(nd.logger, "component", "ControlPlaneDiscovery")).Log("warn", fmt.Sprintf("Primary IP is not set for server: %s", server.Name))
		}

		tgroup := targetgroup.Group{
			Source:  ip,
			Labels:  make(model.LabelSet),
			Targets: make([]model.LabelSet, 0, 1),
		}

		target := model.LabelSet{model.AddressLabel: model.LabelValue(ip)}
		labels := model.LabelSet{
			model.LabelName("job"):             "cp/netbox",
			model.LabelName("server_name"):     model.LabelValue(server.Name),
			model.LabelName("provision_state"): model.LabelValue(*server.Status.Label),
			//model.LabelName("maintenance"):     model.LabelValue(*server.Status.Label),
			model.LabelName("manufacturer"): model.LabelValue(*server.DeviceType.Manufacturer.Name),
			model.LabelName("model"):        model.LabelValue(*server.DeviceType.Model),
			model.LabelName("server_id"):    model.LabelValue(strconv.Itoa(int(server.ID))),
		}

		if server.Serial != "" {
			labels[model.LabelName("serial")] = model.LabelValue(server.Serial)
		}

		if nodeName != "" {
			labels[model.LabelName("node_name")] = model.LabelValue(nodeName)
		}

		tgroup.Labels = labels
		tgroup.Targets = append(tgroup.Targets, target)
		tgroups = append(tgroups, &tgroup)
	}

	return tgroups, nil
}
