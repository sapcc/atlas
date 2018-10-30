package discovery

import (
	"context"
	"strconv"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/gophercloud/gophercloud"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/discovery/targetgroup"
	"github.com/sapcc/ipmi_sd/pkg/clients"
	internalClients "github.com/sapcc/ipmi_sd/pkg/clients"
)

type discovery struct {
	providerClient  *gophercloud.ProviderClient
	ironicClient    *clients.IronicClient
	refreshInterval int
	logger          log.Logger
	upGauge         prometheus.Gauge
}

//NewDiscovery creates a new Discovery
func NewDiscovery(p *gophercloud.ProviderClient, refreshInterval int, logger log.Logger, upGauge prometheus.Gauge) (*discovery, error) {
	ic, err := internalClients.NewIronicClient(p)
	if err != nil {
		level.Error(log.With(logger, "component", "NewIronicClient")).Log("err", err)
		return nil, err
	}

	return &discovery{
		providerClient:  p,
		ironicClient:    ic,
		refreshInterval: refreshInterval,
		logger:          logger,
		upGauge:         upGauge,
	}, nil
}

func (d *discovery) Run(ctx context.Context, ch chan<- []*targetgroup.Group) {
	for c := time.Tick(time.Duration(d.refreshInterval) * time.Second); ; {
		tgs, err := d.parseServiceNodes()
		d.setAdditionalLabels(tgs)
		if err == nil {
			d.upGauge.Set(1)
			ch <- tgs
		} else {
			d.upGauge.Set(0)
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
		if node.ProvisionStateEnroll() {
			continue
		}

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

func (d *discovery) setAdditionalLabels(tgroups []*targetgroup.Group) {
	labels, err := NewLabels(d.providerClient, d.logger)
	if err != nil {
		level.Error(log.With(d.logger, "component", "nodeLabels")).Log("err", err)
		return
	}

	serverLabels, err := labels.getComputeLabels(tgroups)
	projectLabels, err := labels.getProjectLabels(serverLabels)

	if err != nil {
		level.Error(log.With(d.logger, "component", "nodeLabels")).Log("err", err)
		return
	}

	for _, group := range tgroups {
		id := string(group.Labels[model.LabelName("server_id")])
		if len(id) == 0 {
			continue
		}
		server := serverLabels[id]
		if server != nil {
			group.Labels[model.LabelName("project_id")] = model.LabelValue(server.TenantID)
			project := projectLabels[server.TenantID]
			if project != nil {
				group.Labels[model.LabelName("domain_id")] = model.LabelValue(project.DomainID)
			}
		}

	}
}
