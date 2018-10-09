package discovery

import (
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/projects"

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
	serviceClient   *gophercloud.ServiceClient
	refreshInterval int
	logger          log.Logger
}

//NewDiscovery creates a new Discovery
func NewDiscovery(ic *clients.IronicClient, cc *clients.ComputeClient, sc *gophercloud.ServiceClient, refreshInterval int, logger log.Logger) (*discovery, error) {
	cd := &discovery{
		ironicClient:    ic,
		computeClient:   cc,
		serviceClient:   sc,
		refreshInterval: refreshInterval,
		logger:          logger,
	}
	return cd, nil
}

func (d *discovery) Run(ctx context.Context, ch chan<- []*targetgroup.Group) {
	for c := time.Tick(time.Duration(d.refreshInterval) * time.Second); ; {
		tgs, err := d.parseServiceNodes()
		d.setAdditionalLabels(tgs)
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

func (d *discovery) setAdditionalLabels(tgroups []*targetgroup.Group) {
	if d.computeClient.Authorized == false {
		return
	}

	serverLabels := getServerLabels(d.computeClient, tgroups, d.logger)
	projectLabels := getProjectLabels(d.serviceClient, serverLabels, d.logger)

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

func getServerLabels(client *clients.ComputeClient, tgroups []*targetgroup.Group, logger log.Logger) map[string]*servers.Server {
	var wg sync.WaitGroup
	serversCh := make(chan *servers.Server)
	errCh := make(chan error)
	result := make(map[string]*servers.Server)
	for _, group := range tgroups {
		id := string(group.Labels[model.LabelName("server_id")])
		if len(id) == 0 {
			continue
		}
		wg.Add(1)
		go client.GetServer(id, serversCh, errCh, &wg)
	}

	go func() {
		for err := range errCh {
			level.Error(log.With(logger, "component", "compute")).Log("err", err)
		}
	}()

	go func() {
		defer close(serversCh)
		defer close(errCh)
		wg.Wait()
	}()

	for server := range serversCh {
		result[server.ID] = server
	}
	return result
}

func getProjectLabels(client *gophercloud.ServiceClient, s map[string]*servers.Server, logger log.Logger) map[string]*projects.Project {
	var wg sync.WaitGroup
	projectslCh := make(chan map[string]*projects.Project)
	result := make(map[string]*projects.Project)
	errCh := make(chan error)

	_, err := projects.Get(client, "").Extract()
	if err != nil {
		switch err.(type) {
		case gophercloud.ErrDefault403:
			level.Info(log.With(logger, "component", "project")).Log("info", "user missing role!")
			return result
		}
	}

	for _, server := range s {
		wg.Add(1)
		go fetchProjectLabels(client, server.TenantID, projectslCh, errCh, &wg)
	}

	go func() {
		for err := range errCh {
			level.Error(log.With(logger, "component", "project")).Log("error", err)
		}
	}()

	go func() {
		defer close(projectslCh)
		defer close(errCh)
		wg.Wait()
	}()

	for project := range projectslCh {
		for k, v := range project {
			result[k] = v
		}
	}

	return result
}

func fetchProjectLabels(client *gophercloud.ServiceClient, tenantID string, pc chan<- map[string]*projects.Project, ec chan<- error, wg *sync.WaitGroup) {
	p, err := projects.Get(client, tenantID).Extract()
	r := make(map[string]*projects.Project)
	defer wg.Done()
	if err != nil {
		ec <- err
	} else {
		r[tenantID] = p
		pc <- r
	}
}
