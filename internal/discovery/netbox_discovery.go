package discovery

import (
	"context"
	"github.com/sapcc/ipmi_sd/pkg/netbox"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/prometheus/prometheus/discovery/targetgroup"
)

type NetboxDiscovery struct {
	netbox          *netbox.Netbox
	region          string
	refreshInterval int
	logger          log.Logger
	Status          *Status
}

//NewDiscovery creates a new NetboxDiscovery
func NewNetboxDiscovery(n *netbox.Netbox, region string, refreshInterval int, logger log.Logger) *NetboxDiscovery {

	return &NetboxDiscovery{
		netbox:          n,
		refreshInterval: refreshInterval,
		logger:          logger,
		Status:          &Status{Up: false},
	}
}

func (nd *NetboxDiscovery) Run(ctx context.Context, ch chan<- []*targetgroup.Group) {
	for c := time.Tick(time.Duration(nd.refreshInterval) * time.Second); ; {
		tgs, err := nd.getNodes()
		if err == nil {
			nd.Status.Lock()
			nd.Status.Up = true
			nd.Status.Unlock()
			ch <- tgs
		} else {
			nd.Status.Lock()
			nd.Status.Up = false
			nd.Status.Unlock()
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

//TODO:onur
func (nd *NetboxDiscovery) getNodes() ([]*targetgroup.Group, error) {

	var tgroups []*targetgroup.Group

	return tgroups, nil
}
