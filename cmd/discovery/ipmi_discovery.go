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
	"os"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/tokens"
	"github.com/namsral/flag"
	"github.com/sapcc/ipmi_sd/internal/discovery"
	"github.com/sapcc/ipmi_sd/pkg/adapter"
	"github.com/sapcc/ipmi_sd/pkg/clients"
)

var (
	appEnv            string
	outputFile        string
	refreshInterval   int
	identityEndpoint  string
	username          string
	password          string
	domainName        string
	projectName       string
	projectDomainName string
	logger            log.Logger
)

func init() {
	flag.StringVar(&appEnv, "APP_ENV", "development", "To set Log Level: development or production")
	flag.StringVar(&outputFile, "output.file", "ipmi_targets.json", "Output file for file_sd compatible file.")
	flag.IntVar(&refreshInterval, "REFRESH_INTERVAL", 600, "refreshInterval for fetching ironic nodes")
	flag.StringVar(&identityEndpoint, "OS_AUTH_URL", "", "Openstack identity endpoint")
	flag.StringVar(&username, "OS_USERNAME", "", "Openstack username")
	flag.StringVar(&password, "OS_PASSWORD", "", "Openstack password")
	flag.StringVar(&domainName, "OS_USER_DOMAIN_NAME", "", "Openstack domain name")
	flag.StringVar(&projectName, "OS_PROJECT_NAME", "", "Openstack project")
	flag.StringVar(&projectDomainName, "OS_PROJECT_DOMAIN_NAME", "", "Openstack project domain name")
	flag.Parse()
}

func main() {
	logger = log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
	logger = log.With(logger, "ts", log.DefaultTimestampUTC)
	if appEnv == "production" {
		logger = level.NewFilter(logger, level.AllowInfo())
	} else {
		logger = level.NewFilter(logger, level.AllowDebug())
	}

	authOptions := &tokens.AuthOptions{
		IdentityEndpoint: identityEndpoint,
		Username:         username,
		Password:         password,
		DomainName:       domainName,
		AllowReauth:      true,
		Scope: tokens.Scope{
			ProjectName: projectName,
			DomainName:  projectDomainName,
		},
	}

	provider, err := openstack.NewClient(identityEndpoint)
	if err != nil {
		level.Error(log.With(logger, "component", "ipmi_discovery")).Log("err", err)
	}

	err = openstack.AuthenticateV3(provider, authOptions, gophercloud.EndpointOpts{})
	if err != nil {
		level.Error(log.With(logger, "component", "ipmi_discovery")).Log("err", err)
	}

	ic, err := clients.NewIronicClient(provider)
	if err != nil {
		level.Error(log.With(logger, "component", "ipmi_discovery")).Log("err", err)
	}
	cc, err := clients.NewComputeClient(provider)
	if err != nil {
		level.Error(log.With(logger, "component", "compute_client")).Log("err", err)
	}

	var configmapName string
	if configmapName, ok := os.LookupEnv("OS_PROM_CONFIGMAP_NAME"); !ok {
		level.Error(log.With(logger, "component", "ipmi_discovery")).Log("err", "no configmap name given")
		os.Exit(2)
	}

	disc, err := discovery.NewDiscovery(ic, cc, refreshInterval, logger)
	if err != nil {
		level.Error(log.With(logger, "component", "ipmi_discovery")).Log("err", err)
	}
	ctx := context.Background()

	sdAdapter, err := adapter.NewAdapter(ctx, outputFile, "ipmiDiscovery", disc, configmapName, "kube-monitoring", logger)
	if err != nil {
		level.Error(log.With(logger, "component", "ipmi_discovery")).Log("err", err)
	}
	sdAdapter.Run()

	<-ctx.Done()
}
