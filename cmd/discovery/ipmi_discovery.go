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
	"os"
	"strconv"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/tokens"
	"github.com/sapcc/ipmi_sd/internal/discovery"
	"github.com/sapcc/ipmi_sd/pkg/adapter"
	"github.com/sapcc/ipmi_sd/pkg/clients"
)

var (
	outputFile      = flag.String("output.file", "ipmi_targets.json", "Output file for file_sd compatible file.")
	refreshInterval = flag.Int("refreshInterval", 600, "refreshInterval for fetching ironic nodes")
	logger          log.Logger
)

func main() {
	logger = log.NewJSONLogger(log.NewSyncWriter(os.Stderr))
	env := os.Getenv("APP_ENV")
	if env == "production" {
		logger = level.NewFilter(logger, level.AllowWarn())
	} else {
		logger = level.NewFilter(logger, level.AllowDebug())
	}

	authOptions := &tokens.AuthOptions{
		IdentityEndpoint: os.Getenv("OS_AUTH_URL"),
		Username:         os.Getenv("OS_USERNAME"),
		Password:         os.Getenv("OS_PASSWORD"),
		DomainName:       os.Getenv("OS_USER_DOMAIN_NAME"),
		AllowReauth:      true,
		Scope: tokens.Scope{
			ProjectName: os.Getenv("OS_PROJECT_NAME"),
			DomainName:  os.Getenv("OS_PROJECT_DOMAIN_NAME"),
		},
	}

	provider, err := openstack.NewClient(os.Getenv("OS_AUTH_URL"))
	if err != nil {
		level.Error(log.With(logger, "component", "ipmi_discovery")).Log("err", err)
	}

	err = openstack.AuthenticateV3(provider, authOptions, gophercloud.EndpointOpts{})
	if err != nil {
		level.Error(log.With(logger, "component", "ipmi_discovery")).Log("err", err)
	}

	ic, err := clients.NewIronicClient(provider)
	nc, err := openstack.NewComputeV2(provider, gophercloud.EndpointOpts{})

	ctx := context.Background()

	if val, ok := os.LookupEnv("REFRESH_INTERVAL"); ok {
		val, err := strconv.Atoi(val)
		if err != nil {
			level.Error(log.With(logger, "component", "ipmi_discovery")).Log("err", err)
		} else {
			*refreshInterval = val
		}
	}
	var configmapName string
	if val, ok := os.LookupEnv("OS_PROM_CONFIGMAP_NAME"); ok {
		configmapName = val
	} else {
		level.Error(log.With(logger, "component", "ipmi_discovery")).Log("err", "no configmap name given")
	}

	disc, err := discovery.NewDiscovery(nc, ic, *refreshInterval, logger)
	if err != nil {
		level.Error(log.With(logger, "component", "ipmi_discovery")).Log("err", err)
	}
	sdAdapter := adapter.NewAdapter(ctx, *outputFile, "ipmiDiscovery", disc, configmapName, "kube-monitoring", logger)
	sdAdapter.Run()

	<-ctx.Done()
}
