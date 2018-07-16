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

package main

import (
	"context"
	"flag"
	"fmt"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/tokens"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/discovery/targetgroup"
	"github.com/prometheus/prometheus/documentation/examples/custom-sd/adapter"
)

var (
	outputFile = flag.String("output.file", "impi_sd.json", "Output file for file_sd compatible file.")
	logger     log.Logger
)

type discovery struct {
	ironicClient     ironicClient
	address          string
	refreshInterval  int
	clientDatacenter string
	tagSeparator     string
	logger           log.Logger
}

func (d *discovery) parseServiceNodes() ([]*targetgroup.Group, error) {
	nodes, err := d.ironicClient.GetNodes()
	if err != nil {
		logger.Log(err.Error())
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
			model.LabelName("job"):          "impi",
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

// Note: create a config struct for your custom SD type here.
type sdConfig struct {
	IronicClient    *ironicClient
	RefreshInterval int
}

func newDiscovery(conf sdConfig) (*discovery, error) {
	cd := &discovery{
		ironicClient:    *conf.IronicClient,
		refreshInterval: conf.RefreshInterval,
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

func main() {

	authOptions := &tokens.AuthOptions{
		IdentityEndpoint: "https:",
		Username:         "",
		Password:         "",
		DomainName:       "",
		AllowReauth:      true,
		Scope: tokens.Scope{
			ProjectName: "",
			DomainName:  "",
		},
	}

	provider, err := openstack.NewClient("https:")
	if err != nil {
		fmt.Errorf("could not initialize openstack client: %v", err)
	}

	err = openstack.AuthenticateV3(provider, authOptions, gophercloud.EndpointOpts{})
	if err != nil {
		fmt.Errorf("could not authenticat provider client: %v", err)
	}

	ic, err := NewIronicClient(provider)

	ctx := context.Background()

	// NOTE: create an instance of your new SD implementation here.
	cfg := sdConfig{
		RefreshInterval: 30,
		IronicClient:    ic,
	}

	disc, err := newDiscovery(cfg)
	if err != nil {
		fmt.Println("err: ", err)
	}
	sdAdapter := adapter.NewAdapter(ctx, *outputFile, "impiDiscovery", disc, logger)
	sdAdapter.Run()

	<-ctx.Done()
}
