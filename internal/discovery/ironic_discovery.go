/*******************************************************************************
*
* Copyright 2018 SAP SE
*
* Licensed under the Apache License, Version 2.0 (the "License");
* you may not use this file except in compliance with the License.
* You should have received a copy of the License along with this
* program. If not, you may obtain a copy of the License at
*
*     http://www.apache.org/licenses/LICENSE-2.0
*
* Unless required by applicable law or agreed to in writing, software
* distributed under the License is distributed on an "AS IS" BASIS,
* WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
* See the License for the specific language governing permissions and
* limitations under the License.
*
*******************************************************************************/
package discovery

import (
	"context"
	"strconv"
	"time"

	"github.com/sapcc/atlas/pkg/config"
	"github.com/sapcc/atlas/pkg/writer"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/gophercloud/gophercloud"
	"github.com/prometheus/common/model"
	promDiscovery "github.com/prometheus/prometheus/discovery"
	"github.com/prometheus/prometheus/discovery/targetgroup"
	"github.com/sapcc/atlas/pkg/adapter"
	"github.com/sapcc/atlas/pkg/auth"
	"github.com/sapcc/atlas/pkg/clients"
	internalClients "github.com/sapcc/atlas/pkg/clients"
)

type (
	IronicDiscovery struct {
		manager         *promDiscovery.Manager
		adapter         adapter.Adapter
		providerClient  *gophercloud.ProviderClient
		ironicClient    *clients.IronicClient
		refreshInterval int
		logger          log.Logger
		status          *Status
		outputFile      string
	}
	ironicConfig struct {
		RefreshInterval int             `yaml:"refresh_interval"`
		TargetsFileName string          `yaml:"targets_file_name"`
		OpenstackAuth   auth.OSProvider `yaml:"os_auth"`
	}
)

func init() {
	Register("ironic", NewIronicDiscovery)
}

//NewIronicDiscovery creates a new Ironic Discovery
func NewIronicDiscovery(disc interface{}, ctx context.Context, m *promDiscovery.Manager, opts config.Options, w writer.Writer, l log.Logger) (d Discovery, err error) {
	var cfg ironicConfig
	if err := UnmarshalHandler(disc, &cfg, nil); err != nil {
		return d, err
	}

	p, err := auth.NewProviderClient(cfg.OpenstackAuth)
	if err != nil {
		level.Error(log.With(l, "component", "IronicDiscovery")).Log("err", err)
		return d, err
	}
	i, err := internalClients.NewIronicClient(p)
	if err != nil {
		level.Error(log.With(l, "component", "IronicDiscovery")).Log("err", err)
		return d, err
	}

	a := adapter.NewPrometheus(ctx, m, cfg.TargetsFileName, w, l)

	return &IronicDiscovery{
		manager:         m,
		adapter:         a,
		providerClient:  p,
		ironicClient:    i,
		refreshInterval: cfg.RefreshInterval,
		logger:          l,
		status:          &Status{Up: false},
		outputFile:      cfg.TargetsFileName,
	}, nil
}

func (d *IronicDiscovery) Run(ctx context.Context, ch chan<- []*targetgroup.Group) {
	for c := time.Tick(time.Duration(d.refreshInterval) * time.Second); ; {
		tgs, err := d.parseServiceNodes()
		d.setAdditionalLabels(tgs)
		if err == nil {
			d.status.Lock()
			d.status.Up = true
			d.status.Unlock()
			ch <- tgs
		} else {
			d.status.Lock()
			d.status.Up = false
			d.status.Unlock()
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

func (d *IronicDiscovery) StartAdapter() {
	d.adapter.Run()
}

func (d *IronicDiscovery) GetAdapter() adapter.Adapter {
	return d.adapter
}

func (d *IronicDiscovery) Up() bool {
	return d.status.Up

}
func (d *IronicDiscovery) Lock() {
	d.status.Lock()

}
func (d *IronicDiscovery) Unlock() {
	d.status.Unlock()
}

func (d *IronicDiscovery) GetOutputFile() string {
	return d.outputFile
}

func (d *IronicDiscovery) GetManager() *promDiscovery.Manager {
	return d.manager
}

func (d *IronicDiscovery) parseServiceNodes() ([]*targetgroup.Group, error) {
	nodes, err := d.ironicClient.GetNodes()
	if err != nil {
		level.Error(log.With(d.logger, "component", "IronicDiscovery")).Log("err", err)
		return nil, err
	}

	if len(nodes) == 0 {
		level.Info(log.With(d.logger, "component", "IronicDiscovery")).Log("info", "no ironic nodes found")
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

func (d *IronicDiscovery) setAdditionalLabels(tgroups []*targetgroup.Group) {
	labels, err := NewLabels(d.providerClient, d.logger)
	if err != nil {
		level.Error(log.With(d.logger, "component", "IronicDiscovery")).Log("err", err)
		return
	}

	serverLabels, err := labels.getComputeLabels(tgroups)
	if err != nil {
		level.Error(log.With(d.logger, "component", "IronicDiscovery")).Log("err", err)
		return
	}

	projectLabels, err := labels.getProjectLabels(serverLabels)
	if err != nil {
		level.Error(log.With(d.logger, "component", "IronicDiscovery")).Log("err", err)
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
