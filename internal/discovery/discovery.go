package discovery

import (
	"context"
	"errors"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/discovery/targetgroup"
	"github.com/sapcc/ipmi_sd/pkg/clients"
)

var (
	logger log.Logger
)

type discovery struct {
	ironicClient    *clients.IronicClient
	refreshInterval int
	tagSeparator    string
	logger          log.Logger
}

func (d *discovery) parseServiceNodes() ([]*targetgroup.Group, error) {
	nodes, err := d.ironicClient.GetNodes()
	if err != nil {
		level.Error(log.With(logger, "component", "ironicClient")).Log("err", err)
		return nil, err
	}

	if len(nodes) == 0 {
		err = errors.New("did not find any ironic nodes")
		level.Error(log.With(logger, "component", "discovery")).Log("err", err)
		return nil, err
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
			model.LabelName("job"):          "baremetal/ironic",
			model.LabelName("serial"):       model.LabelValue(node.Properties.SerialNumber),
			model.LabelName("manufacturer"): model.LabelValue(node.Properties.Manufacturer),
			model.LabelName("model"):        model.LabelValue(node.Properties.Model),
		}
		tgroup.Labels = labels
		tgroup.Targets = append(tgroup.Targets, target)
		tgroups = append(tgroups, &tgroup)
	}

	return tgroups, nil
}

func NewDiscovery(client *clients.IronicClient, refreshInterval int, logger log.Logger) (*discovery, error) {
	cd := &discovery{
		ironicClient:    client,
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
