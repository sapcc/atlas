package discovery

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/common/model"
	"github.com/sapcc/ipmi_sd/pkg/netbox"

	"github.com/go-kit/kit/log"
	"github.com/prometheus/prometheus/discovery/targetgroup"
)

type NetboxDiscovery struct {
	netbox          *netbox.Netbox
	region          string
	refreshInterval int
	logger          log.Logger
	status          *Status
}

//NewDiscovery creates a new NetboxDiscovery
func NewNetboxDiscovery(n *netbox.Netbox, region string, refreshInterval int, logger log.Logger) Discovery {

	return &NetboxDiscovery{
		netbox:          n,
		region:          region,
		refreshInterval: refreshInterval,
		logger:          logger,
		status:          &Status{Up: false},
	}
}

func (nd *NetboxDiscovery) Run(ctx context.Context, ch chan<- []*targetgroup.Group) {
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

func (d *NetboxDiscovery) Up() bool {
	return d.status.Up

}
func (d *NetboxDiscovery) Lock() {
	d.status.Lock()

}
func (d *NetboxDiscovery) Unlock() {
	d.status.Unlock()
}

func (nd *NetboxDiscovery) getNodes() ([]*targetgroup.Group, error) {

	servers, err := nd.netbox.ServersByRegion("cp", nd.region)
	if err != nil {
		level.Error(log.With(nd.logger, "component", "NetboxDiscovery")).Log("err", err)
		return nil, err
	}

	var tgroups []*targetgroup.Group

	for _, server := range servers {

		if *server.Status.Value != 1 {
			level.Warn(log.With(nd.logger, "component", "NetboxDiscovery")).Log("warn", fmt.Sprintf("Status value is not 1 for server: %s. Skipping the server!", server.Name))
			continue
		}

		ip, err := nd.netbox.ManagementIP(server.ID)
		if err != nil {
			level.Warn(log.With(nd.logger, "component", "NetboxDiscovery")).Log("warn", fmt.Sprintf("Error during getting management IP for server %s: %v. Skipping the server!", server.Name, err))
			continue
		}

		nodeName := ""
		if server.PrimaryIP != nil {
			primaryIP, err := nd.netbox.IPAddress(server.PrimaryIP.ID)

			if err != nil {
				level.Warn(log.With(nd.logger, "component", "NetboxDiscovery")).Log("warn", err)
			} else {
				nodeName = primaryIP.Description
			}

		} else {
			level.Warn(log.With(nd.logger, "component", "NetboxDiscovery")).Log("warn", fmt.Sprintf("Primary IP is not set for server: %s", server.Name))
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
