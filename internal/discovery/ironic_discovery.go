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
	"fmt"
	"strconv"
	"time"

	netbox_dcim "github.com/netbox-community/go-netbox/netbox/client/dcim"
	"github.com/sapcc/atlas/pkg/config"
	"github.com/sapcc/atlas/pkg/errgroup"
	"github.com/sapcc/atlas/pkg/netbox"
	"github.com/sapcc/atlas/pkg/writer"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/gophercloud/gophercloud"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/discovery/targetgroup"
	"github.com/sapcc/atlas/pkg/adapter"
	"github.com/sapcc/atlas/pkg/auth"
	"github.com/sapcc/atlas/pkg/clients"
	internalClients "github.com/sapcc/atlas/pkg/clients"
)

type (
	IronicDiscovery struct {
		adapter          adapter.Adapter
		netbox           *netbox.Netbox
		providerClient   *gophercloud.ProviderClient
		ironicClient     *clients.IronicClient
		refreshInterval  int
		logger           log.Logger
		status           *Status
		outputFile       string
		metricsLabel     string
		mgmtInterfaceIPs *bool
	}
	ironicConfig struct {
		NetboxHost       string          `yaml:"netbox_host"`
		NetboxAPIToken   string          `yaml:"netbox_api_token"`
		MgmtInterfaceIPs *bool           `yaml:"mgmt_interface_ips"`
		RefreshInterval  int             `yaml:"refresh_interval"`
		TargetsFileName  string          `yaml:"targets_file_name"`
		OpenstackAuth    auth.OSProvider `yaml:"os_auth"`
		MetricsLabel     string          `yaml:"metrics_label"`
		ConfigmapName    string          `yaml:"configmap_name"`
	}
)

const ironicDiscovery = "ironic"

func init() {
	Register(ironicDiscovery, NewIronicDiscovery)
}

//NewIronicDiscovery creates a new Ironic Discovery
func NewIronicDiscovery(disc interface{}, ctx context.Context, opts config.Options, l log.Logger) (d Discovery, err error) {
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

	nClient, err := netbox.New(cfg.NetboxHost, cfg.NetboxAPIToken)
	if err != nil {
		return nil, err
	}

	var w writer.Writer
	if cfg.ConfigmapName != "" {
		w, err = writer.NewConfigMap(cfg.ConfigmapName, opts.NameSpace, l)
	} else {
		w, err = writer.NewFile(cfg.TargetsFileName, l)
	}

	a := adapter.NewPrometheus(ctx, cfg.TargetsFileName, w, l)

	return &IronicDiscovery{
		adapter:          a,
		providerClient:   p,
		ironicClient:     i,
		refreshInterval:  cfg.RefreshInterval,
		metricsLabel:     cfg.MetricsLabel,
		logger:           l,
		status:           &Status{Up: false, Targets: make(map[string]int)},
		outputFile:       cfg.TargetsFileName,
		netbox:           nClient,
		mgmtInterfaceIPs: cfg.MgmtInterfaceIPs,
	}, nil
}

func (d *IronicDiscovery) Run(ctx context.Context, ch chan<- []*targetgroup.Group) {
	for c := time.Tick(time.Duration(d.refreshInterval) * time.Second); ; {
		tgs, err := d.parseServiceNodes()
		d.setAdditionalLabels(tgs)
		if err == nil {
			level.Debug(log.With(d.logger, "component", "IronicDiscovery")).Log("debug", "Done Loading Nodes")
			d.status.Lock()
			d.status.Up = true
			d.status.Unlock()
			ch <- tgs
		} else {
			level.Error(log.With(d.logger, "component", "IronicDiscovery")).Log("error", err)
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

func (d *IronicDiscovery) GetAdapter() adapter.Adapter {
	return d.adapter
}

func (d *IronicDiscovery) Up() bool {
	return d.status.Up

}
func (d *IronicDiscovery) Targets() map[string]int {
	d.status.Targets = make(map[string]int)
	setMetricsLabelAndValue(d.status.Targets, d.metricsLabel, d.adapter.GetNumberOfTargetsFor(d.metricsLabel))
	return d.status.Targets
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

func (s *IronicDiscovery) GetName() string {
	return ironicDiscovery
}

func (d *IronicDiscovery) parseServiceNodes() (tgroups []*targetgroup.Group, err error) {
	nodes, err := d.ironicClient.GetNodes()
	if err != nil {
		level.Error(log.With(d.logger, "component", "IronicDiscovery")).Log("err", err)
		return
	}

	if len(nodes) == 0 {
		level.Info(log.With(d.logger, "component", "IronicDiscovery")).Log("info", "no ironic nodes found")
	}

	level.Debug(log.With(d.logger, "component", "IronicDiscovery")).Log("debug", fmt.Sprintf("found %d nodes", len(nodes)))

	groupCh := make(chan []*targetgroup.Group, 0)
	var eg errgroup.Group
	for _, node := range nodes {
		func(node internalClients.IronicNode, groupCh chan<- []*targetgroup.Group) {
			eg.Go(func() error {
				var tgs []*targetgroup.Group
				if node.ProvisionStateEnroll() {
					return nil
				}

				if d.mgmtInterfaceIPs == nil || !*d.mgmtInterfaceIPs {
					tgroup, err := d.createNodeGroup(node, node.DriverInfo.IpmiAddress)
					if err != nil {
						return err
					}
					tgs = append(tgs, tgroup)
					groupCh <- tgs
					return nil
				}

				params := netbox_dcim.DcimDevicesListParams{
					Name: &node.Name,
				}
				dev, err := d.netbox.DeviceByParams(params)
				if err != nil {
					return err
				}

				ips, err := d.netbox.ManagementIPs(strconv.FormatInt(dev.ID, 10))
				for _, ip := range ips {
					tgroup, err := d.createNodeGroup(node, ip)
					if err != nil {
						return err
					}
					tgs = append(tgs, tgroup)
				}
				level.Debug(log.With(d.logger, "component", "IronicDiscovery")).Log("debug", fmt.Sprintf("finished node: %s", node.Name))
				groupCh <- tgs
				return nil
			})
		}(node, groupCh)
	}
	go func() error {
		if err = eg.Wait(); err != nil {
			close(groupCh)
			return err
		}
		close(groupCh)
		return nil
	}()
	for groups := range groupCh {
		tgroups = append(tgroups, groups...)
	}
	return tgroups, err
}

func (d *IronicDiscovery) createNodeGroup(node internalClients.IronicNode, ipAddress string) (tgroup *targetgroup.Group, err error) {
	tgroup = &targetgroup.Group{
		Source:  ipAddress,
		Labels:  make(model.LabelSet),
		Targets: make([]model.LabelSet, 0, 1),
	}
	target := model.LabelSet{model.AddressLabel: model.LabelValue(ipAddress)}
	labels := model.LabelSet{
		model.LabelName("server_name"):     model.LabelValue(node.Name),
		model.LabelName("provision_state"): model.LabelValue(node.ProvisionState),
		model.LabelName("maintenance"):     model.LabelValue(strconv.FormatBool(node.Maintenance)),
		model.LabelName("serial"):          model.LabelValue(node.Properties.SerialNumber),
		model.LabelName("manufacturer"):    model.LabelValue(node.Properties.Manufacturer),
		model.LabelName("model"):           model.LabelValue(node.Properties.Model),
		model.LabelName("metrics_label"):   model.LabelValue(d.metricsLabel),
	}

	if len(node.InstanceUuID) > 0 {
		labels[model.LabelName("server_id")] = model.LabelValue(node.InstanceUuID)
	}

	tgroup.Labels = labels
	tgroup.Targets = append(tgroup.Targets, target)
	return
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
