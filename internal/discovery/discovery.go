package discovery

import (
	"context"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/discovery/targetgroup"
	"github.com/sapcc/ipmi_sd/pkg/clients"
)

type discovery struct {
	novaClient      *gophercloud.ServiceClient
	ironicClient    *clients.IronicClient
	refreshInterval int
	tagSeparator    string
	logger          log.Logger
}

func (d *discovery) parseServiceNodes() ([]*targetgroup.Group, error) {
	ironicNodes, err := d.ironicClient.GetNodes()
	if err != nil {
		level.Error(log.With(d.logger, "component", "ironicClient")).Log("err", err)
		return nil, err
	}

	if len(ironicNodes) == 0 {
		err := level.Error(log.With(d.logger, "component", "discovery")).Log("err", "no ironic nodes found")
		return nil, err
	}

	var tgroups []*targetgroup.Group

	for _, node := range ironicNodes {
		var instance *servers.Server

		tgroup := targetgroup.Group{
			Source:  node.DriverInfo.IpmiAddress,
			Labels:  make(model.LabelSet),
			Targets: make([]model.LabelSet, 0, 1),
		}

		target := model.LabelSet{model.AddressLabel: model.LabelValue(node.DriverInfo.IpmiAddress)}
		labels := model.LabelSet{
			model.LabelName("serial"):       model.LabelValue(node.Properties.SerialNumber),
			model.LabelName("manufacturer"): model.LabelValue(node.Properties.Manufacturer),
			model.LabelName("model"):        model.LabelValue(node.Properties.Model),
		}

		// Check if node is used by a tenant and add instance metadata
		if node.InstanceUuid != "" {
			instance, err = servers.Get(d.novaClient, node.InstanceUuid).Extract()
			if err != nil {
				level.Error(log.With(d.logger, "component", "novaClient")).Log("err", err)
				return nil, err
			}
			labels[model.LabelName("server_id")] = model.LabelValue(node.InstanceUuid)
			labels[model.LabelName("server_name")] = model.LabelValue(instance.Name)
			labels[model.LabelName("project_id")] = model.LabelValue(instance.TenantID)
			labels[model.LabelName("user_id")] = model.LabelValue(instance.UserID)
		}

		tgroup.Labels = labels
		tgroup.Targets = append(tgroup.Targets, target)
		tgroups = append(tgroups, &tgroup)
	}

	return tgroups, nil
}

func NewDiscovery(novaClient *gophercloud.ServiceClient, ironicClient *clients.IronicClient, refreshInterval int, logger log.Logger) (*discovery, error) {
	cd := &discovery{
		novaClient:      novaClient,
		ironicClient:    ironicClient,
		refreshInterval: refreshInterval,
		logger:          logger,
	}
	return cd, nil
}

func (d *discovery) Run(ctx context.Context, ch chan<- []*targetgroup.Group) {
	for c := time.Tick(time.Duration(d.refreshInterval) * time.Second); ; {
		tgs, err := d.parseServiceNodes()
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
