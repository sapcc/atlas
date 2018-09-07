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
	"sort"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/identity/v3/tokens"
	"github.com/sapcc/ipmi_sd/internal/discovery"
	"github.com/sapcc/ipmi_sd/pkg/adapter"
	"github.com/sapcc/ipmi_sd/pkg/clients"
	cli "gopkg.in/urfave/cli.v1"
)

type settings struct {
	debug           bool
	refreshInterval int
	outputFile      string
	isConfigmap     bool
	configmapName   string
}

func main() {

	var logger log.Logger
	logger = log.NewJSONLogger(log.NewSyncWriter(os.Stderr))
	cfg := &settings{}
	authOptions := &tokens.AuthOptions{}
	authOptions.AllowReauth = true

	app := cli.NewApp()
	app.Name = "ipmi_sd"
	app.Usage = "discover OpenStack Ironic nodes for Prometheus, enrich them with metadata labels from Nova and write them to a file or Kubernetes configmap"
	app.Version = "0.0.1"

	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:        "debug",
			Usage:       "Enable more verbose logging",
			Destination: &cfg.debug,
		},
		// TODO check if it could be nicely replaced with a DurationFlag
		cli.IntFlag{
			Name:        "interval",
			EnvVar:      "REFRESH_INTERVAL",
			Value:       600,
			Usage:       "Refresh interval for fetching ironic nodes",
			Destination: &cfg.refreshInterval,
		},
		cli.StringFlag{
			Name:        "filename",
			Value:       "ipmi_targets.json",
			Usage:       "Output file name for the file_sd compatible file.",
			Destination: &cfg.outputFile,
		},
		cli.BoolFlag{
			Name:        "configmap",
			Usage:       "Write file into a Kubernetes configmap",
			Destination: &cfg.isConfigmap,
		},
		cli.StringFlag{
			Name:        "configmap-name",
			EnvVar:      "OS_PROM_CONFIGMAP_NAME",
			Value:       "ipmi-sd",
			Usage:       "Name of the configmap to be created",
			Destination: &cfg.configmapName,
		},
		cli.StringFlag{
			Name:        "auth-url",
			EnvVar:      "OS_AUTH_URL",
			Usage:       "OpenStack identity endpoint URI",
			Destination: &authOptions.IdentityEndpoint,
		},
		cli.StringFlag{
			Name:        "user",
			EnvVar:      "OS_USERNAME",
			Usage:       "OpenStack username",
			Destination: &authOptions.Username,
		},
		cli.StringFlag{
			Name:        "password",
			EnvVar:      "OS_PASSWORD",
			Usage:       "The OpenStack password. Declaration by flag is inherently insecure, because every user can read flags of running programs",
			Destination: &authOptions.Password,
		},
		cli.StringFlag{
			Name:        "user-domain",
			EnvVar:      "OS_USER_DOMAIN_NAME",
			Usage:       "OpenStack user domain name",
			Destination: &authOptions.DomainName,
		},
		cli.StringFlag{
			Name:        "project",
			EnvVar:      "OS_PROJECT_NAME",
			Usage:       "OpenStack project name",
			Destination: &authOptions.Scope.ProjectName,
		},
		cli.StringFlag{
			Name:        "project-domain",
			EnvVar:      "OS_PROJECT_DOMAIN_NAME",
			Usage:       "OpenStack project domain name",
			Destination: &authOptions.Scope.DomainName,
		},
	}

	app.Action = func(c *cli.Context) error {
		if cfg.debug == true {
			logger = level.NewFilter(logger, level.AllowDebug())
		} else {
			logger = level.NewFilter(logger, level.AllowWarn())
		}

		provider, err := openstack.NewClient(authOptions.IdentityEndpoint)
		if err != nil {
			level.Error(log.With(logger, "component", "ipmi_discovery")).Log("err", err)
		}

		err = openstack.AuthenticateV3(provider, authOptions, gophercloud.EndpointOpts{})
		if err != nil {
			level.Error(log.With(logger, "component", "ipmi_discovery")).Log("err", err)
		}

		ic, err := clients.NewIronicClient(provider)
		ctx := context.Background()

		disc, err := discovery.NewDiscovery(ic, cfg.refreshInterval, logger)
		if err != nil {
			level.Error(log.With(logger, "component", "ipmi_discovery")).Log("err", err)
		}

		sdAdapter := adapter.NewAdapter(ctx, cfg.outputFile, "ipmiDiscovery", disc, cfg.configmapName, cfg.isConfigmap, "kube-monitoring", logger)
		sdAdapter.Run()

		<-ctx.Done()
		return nil
	}

	sort.Sort(cli.FlagsByName(app.Flags))

	err := app.Run(os.Args)
	if err != nil {
		level.Error(log.With(logger, "component", "ipmi_discovery")).Log("err", err)
	}

}
