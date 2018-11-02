package discovery

import (
	"errors"
	"os"
	"sync"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/projects"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/discovery/targetgroup"
	"github.com/sapcc/ipmi_sd/pkg/clients"
	internalClients "github.com/sapcc/ipmi_sd/pkg/clients"
)

type labels struct {
	computeClient *clients.ComputeClient
	serviceClient *gophercloud.ServiceClient
	logger        log.Logger
}

func NewLabels(p *gophercloud.ProviderClient, logger log.Logger) (*labels, error) {
	cc, err := internalClients.NewComputeClient(p)
	if err != nil {
		level.Error(log.With(logger, "component", "NewComputeClient")).Log("err", err)
		return nil, err
	}
	sc, err := openstack.NewIdentityV3(p, gophercloud.EndpointOpts{
		Region: os.Getenv("OS_REGION_NAME"),
	})
	if err != nil {
		level.Error(log.With(logger, "component", "NewIdentityV3")).Log("err", err)
		return nil, err
	}

	return &labels{
		computeClient: cc,
		serviceClient: sc,
		logger:        logger,
	}, nil
}

func (l *labels) getComputeLabels(tgroups []*targetgroup.Group) (map[string]*servers.Server, error) {
	var wg sync.WaitGroup
	serversCh := make(chan *servers.Server)
	errCh := make(chan error)
	result := make(map[string]*servers.Server)

	if l.computeClient.Authorized == false {
		return result, errors.New("user missing role for compute labels")
	}

	for _, group := range tgroups {
		id := string(group.Labels[model.LabelName("server_id")])
		if len(id) == 0 {
			continue
		}
		wg.Add(1)
		go l.computeClient.GetServer(id, serversCh, errCh, &wg)
	}

	go func() {
		for err := range errCh {
			level.Error(log.With(l.logger, "component", "compute")).Log("err", err)
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
	return result, nil
}

func (l *labels) getProjectLabels(s map[string]*servers.Server) (map[string]*projects.Project, error) {
	var wg sync.WaitGroup
	projectslCh := make(chan map[string]*projects.Project)
	result := make(map[string]*projects.Project)
	errCh := make(chan error)

	_, err := projects.Get(l.serviceClient, "").Extract()
	if err != nil {
		switch err.(type) {
		case gophercloud.ErrDefault403:
			return result, errors.New("user missing role for porject labels")
		}
	}

	for _, server := range s {
		wg.Add(1)
		go fetchProjectLabels(l.serviceClient, server.TenantID, projectslCh, errCh, &wg)
	}

	go func() {
		for err := range errCh {
			level.Error(log.With(l.logger, "component", "project")).Log("error", err)
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

	return result, nil
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
