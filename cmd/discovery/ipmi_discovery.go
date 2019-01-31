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
	"os/signal"
	"syscall"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/namsral/flag"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sapcc/ipmi_sd/internal/discovery"
	"github.com/sapcc/ipmi_sd/pkg/adapter"
	"github.com/sapcc/ipmi_sd/pkg/auth"
	"github.com/sapcc/ipmi_sd/pkg/config"
	"github.com/sapcc/ipmi_sd/pkg/metrics"
	"github.com/sapcc/ipmi_sd/pkg/server"
)

var (
	logger log.Logger
	opts   config.Options
	adpt   *adapter.Adapter
	disc   *discovery.Discovery
)

func init() {
	flag.StringVar(&opts.AppEnv, "APP_ENV", "development", "To set Log Level: development or production")
	flag.StringVar(&opts.OutputFile, "output.file", "ipmi_targets.json", "Output file for file_sd compatible file.")
	flag.IntVar(&opts.RefreshInterval, "REFRESH_INTERVAL", 600, "refreshInterval for fetching ironic nodes")
	flag.StringVar(&opts.IdentityEndpoint, "OS_AUTH_URL", "", "Openstack identity endpoint")
	flag.StringVar(&opts.Username, "OS_USERNAME", "", "Openstack username")
	flag.StringVar(&opts.Password, "OS_PASSWORD", "", "Openstack password")
	flag.StringVar(&opts.DomainName, "OS_USER_DOMAIN_NAME", "", "Openstack domain name")
	flag.StringVar(&opts.ProjectName, "OS_PROJECT_NAME", "", "Openstack project")
	flag.StringVar(&opts.Version, "OS_VERSION", "v0.3.0", "IPMI SD Version")
	flag.StringVar(&opts.ProjectDomainName, "OS_PROJECT_DOMAIN_NAME", "", "Openstack project domain name")
	flag.Parse()

	logger = log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
	logger = log.With(logger, "ts", log.DefaultTimestampUTC)
	if opts.AppEnv == "production" {
		logger = level.NewFilter(logger, level.AllowInfo())
	} else {
		logger = level.NewFilter(logger, level.AllowDebug())
	}

	if val, ok := os.LookupEnv("OS_PROM_CONFIGMAP_NAME"); ok {
		opts.ConfigmapName = val
	} else {
		level.Error(log.With(logger, "component", "ipmi_discovery")).Log("err", "no configmap name given")
		os.Exit(2)
	}
}

func main() {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	client, err := auth.NewProviderClient(opts)
	if err != nil {
		level.Error(log.With(logger, "component", "AuthenticatedClient")).Log("err", err)
		os.Exit(2)
	}

	disc, err = discovery.New(client, opts.RefreshInterval, logger)
	if err != nil {
		level.Error(log.With(logger, "component", "NewDiscovery")).Log("err", err)
		os.Exit(2)
	}

	adpt, err = adapter.New(ctx, opts.OutputFile, "ipmiDiscovery", disc, opts.ConfigmapName, "kube-monitoring", logger)
	if err != nil {
		level.Error(log.With(logger, "component", "NewAdapter")).Log("err", err)
		os.Exit(2)
	}

	prometheus.MustRegister(metrics.NewMetricsCollector(adpt, disc, opts.Version))
	go server.New(adpt, disc, logger).Start()

	adpt.Run()

	defer func() {
		signal.Stop(c)
		cancel()
	}()

	select {
	case <-c:
		cancel()
	case <-ctx.Done():
	}
}
