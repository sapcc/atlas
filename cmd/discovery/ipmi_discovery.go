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
	"github.com/namsral/flag"
	"github.com/sapcc/ipmi_sd/internal/discovery"
	"github.com/sapcc/ipmi_sd/pkg/adapter"
	internalClients "github.com/sapcc/ipmi_sd/pkg/clients"
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
	configmapName     string
	provider          *gophercloud.ProviderClient
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

	logger = log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
	logger = log.With(logger, "ts", log.DefaultTimestampUTC)
	if appEnv == "production" {
		logger = level.NewFilter(logger, level.AllowInfo())
	} else {
		logger = level.NewFilter(logger, level.AllowDebug())
	}

	if val, ok := os.LookupEnv("OS_PROM_CONFIGMAP_NAME"); ok {
		configmapName = val
	} else {
		level.Error(log.With(logger, "component", "ipmi_discovery")).Log("err", "no configmap name given")
		os.Exit(2)
	}

	authOptions := gophercloud.AuthOptions{
		IdentityEndpoint: identityEndpoint,
		Username:         username,
		Password:         password,
		DomainName:       domainName,
		AllowReauth:      true,
		Scope: &gophercloud.AuthScope{
			ProjectName: projectName,
			DomainName:  projectDomainName,
		},
	}
	var err error
	provider, err = openstack.AuthenticatedClient(authOptions)
	if err != nil {
		level.Error(log.With(logger, "component", "AuthenticatedClient")).Log("err", err)
		os.Exit(2)
	}
}

func main() {
	v3c, err := openstack.NewIdentityV3(provider, gophercloud.EndpointOpts{
		Region: os.Getenv("OS_REGION_NAME"),
	})
	if err != nil {
		level.Error(log.With(logger, "component", "NewIdentityV3")).Log("err", err)
	}

	ic, err := internalClients.NewIronicClient(provider)
	if err != nil {
		level.Error(log.With(logger, "component", "NewIronicClient")).Log("err", err)
	}

	cc, err := internalClients.NewComputeClient(provider)
	if err != nil {
		level.Error(log.With(logger, "component", "NewComputeClient")).Log("err", err)
	}

	disc, err := discovery.NewDiscovery(ic, cc, v3c, refreshInterval, logger)
	if err != nil {
		level.Error(log.With(logger, "component", "NewDiscovery")).Log("err", err)
	}
	ctx := context.Background()

	sdAdapter, err := adapter.NewAdapter(ctx, outputFile, "ipmiDiscovery", disc, configmapName, "kube-monitoring", logger)
	if err != nil {
		level.Error(log.With(logger, "component", "NewAdapter")).Log("err", err)
	}
	sdAdapter.Run()

	<-ctx.Done()
}
