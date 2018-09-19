package discovery

import (
	"context"
	"strconv"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/gophercloud/gophercloud"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/discovery/targetgroup"
	"github.com/sapcc/ipmi_sd/pkg/clients"
)

type discovery struct {
	ironicClient    *clients.IronicClient
	computeClient   *clients.ComputeClient
	refreshInterval int
	tagSeparator    string
	logger          log.Logger
}

func (d *discovery) parseServiceNodes() ([]*targetgroup.Group, error) {
	nodes, err := d.ironicClient.GetNodes()
	if err != nil {
		level.Error(log.With(d.logger, "component", "ironicClient")).Log("err", err)
		return nil, err
	}

	if len(nodes) == 0 {
		level.Info(log.With(d.logger, "component", "discovery")).Log("info", "no ironic nodes found")
	}

	var tgroups []*targetgroup.Group

	for _, node := range nodes {

		tgroup := targetgroup.Group{
			Source:  node.DriverInfo.IpmiAddress,
			Labels:  make(model.LabelSet),
			Targets: make([]model.LabelSet, 0, 1),
		}

		target := model.LabelSet{model.AddressLabel: model.LabelValue(node.DriverInfo.IpmiAddress)}
		labels := model.LabelSet{
			model.LabelName("job"):             "baremetal/ironic",
			model.LabelName("server_name"):     model.LabelValue(node.Name),
			model.LabelName("provision_state"): model.LabelValue(node.ProvisionState),
			model.LabelName("maintenance"):     model.LabelValue(strconv.FormatBool(node.Maintenance)),
			model.LabelName("serial"):          model.LabelValue(node.Properties.SerialNumber),
			model.LabelName("manufacturer"):    model.LabelValue(node.Properties.Manufacturer),
			model.LabelName("model"):           model.LabelValue(node.Properties.Model),
		}

		if len(node.InstanceUuID) > 0 {
			labels[model.LabelName("server_id")] = model.LabelValue(node.InstanceUuID)
		}

		tgroup.Labels = labels
		tgroup.Targets = append(tgroup.Targets, target)
		tgroups = append(tgroups, &tgroup)
	}

	return tgroups, nil
}

func (d *discovery) setServerLabels(tgroups []*targetgroup.Group) {
GroupsLoop:
	for _, group := range tgroups {
		id := string(group.Labels[model.LabelName("server_id")])
		if len(id) == 0 {
			continue
		}
		server, err := d.computeClient.GetServer(id)
		if err != nil {
			switch err.(type) {
			case gophercloud.ErrUnexpectedResponseCode:
				// seems like our user misses the needed role: stop this loop
				level.Info(log.With(d.logger, "component", "compute", "target", group.Source)).Log("info", "user missing role!")
				break GroupsLoop
			default:
				level.Error(log.With(d.logger, "component", "compute", "target", group.Source)).Log("err", err)
			}
		} else {
			group.Labels[model.LabelName("project_id")] = model.LabelValue(server.TenantID)
			group.Labels[model.LabelName("user_id")] = model.LabelValue(server.UserID)
		}
	}
}

func NewDiscovery(ic *clients.IronicClient, cc *clients.ComputeClient, refreshInterval int, logger log.Logger) (*discovery, error) {
	cd := &discovery{
		ironicClient:    ic,
		computeClient:   cc,
		refreshInterval: refreshInterval,
		logger:          logger,
	}
	return cd, nil
}

func (d *discovery) Run(ctx context.Context, ch chan<- []*targetgroup.Group) {
	for c := time.Tick(time.Duration(d.refreshInterval) * time.Second); ; {
		tgs, err := d.parseServiceNodes()
		d.setServerLabels(tgs)
		if err == nil {
			ch <- tgs
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
