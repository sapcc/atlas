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
	"github.com/sapcc/ipmi_sd/pkg/netbox"
	"github.com/sapcc/ipmi_sd/pkg/server"
)

var (
	logger log.Logger
	opts   config.Options
	adpt   *adapter.Adapter
	disc   discovery.Discovery
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

	flag.BoolVar(&opts.NetboxDiscovery, "NETBOX", false, "Use Netbox for discovery")
	flag.StringVar(&opts.NetboxHost, "NETBOX_HOST", "netbox.global.cloud.sap", "Netbox host address")
	flag.StringVar(&opts.NetboxAPIToken, "NETBOX_API_TOKEN", "", "Netbox API token")
	flag.StringVar(&opts.NetboxOutputFile, "output.file.netbox", "netbox_targets.json", "Output file for file_sd compatible file")
	flag.StringVar(&opts.Region, "OS_REGION", "", "Openstack region")
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

	if opts.NetboxDiscovery {
		if opts.NetboxHost == "" {
			level.Error(log.With(logger, "component", "ipmi_discovery")).Log("err", "no netbox host given")
			os.Exit(2)
		}
		if opts.NetboxAPIToken == "" {
			level.Error(log.With(logger, "component", "ipmi_discovery")).Log("err", "no netbox api token given")
			os.Exit(2)
		}
		if opts.NetboxOutputFile == "" {
			level.Error(log.With(logger, "component", "ipmi_discovery")).Log("err", "no netbox output file given")
			os.Exit(2)
		}
		if opts.Region == "" {
			level.Error(log.With(logger, "component", "ipmi_discovery")).Log("err", "no region given")
			os.Exit(2)
		}
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

	disc, err = discovery.NewIronicDiscovery(client, opts.RefreshInterval, logger)
	if err != nil {
		level.Error(log.With(logger, "component", "NewDiscovery")).Log("err", err)
		os.Exit(2)
	}

	adpt, err = adapter.New(ctx, opts.OutputFile, "ipmiDiscovery", disc, opts.ConfigmapName, "kube-monitoring", logger)
	if err != nil {
		level.Error(log.With(logger, "component", "NewAdapter")).Log("err", err)
		os.Exit(2)
	}

	prometheus.MustRegister(metrics.NewMetricsCollector("ipmi_sd_up", "Shows if service can load ironic nodes", adpt, disc, opts.Version))

	adapterList := []*adapter.Adapter{adpt}
	discoveryList := []discovery.Discovery{disc}

	var nAdapter *adapter.Adapter
	if opts.NetboxDiscovery {
		nClient, err := netbox.New(opts.NetboxHost, opts.NetboxAPIToken)
		if err != nil {
			level.Error(log.With(logger, "component", "NetboxClient")).Log("err", err)
			os.Exit(2)
		}
		nDisc := discovery.NewNetboxDiscovery(nClient, opts.Region, opts.RefreshInterval, logger)

		nAdapter, err = adapter.New(ctx, opts.NetboxOutputFile, "netboxDiscovery", nDisc, opts.ConfigmapName, "kube-monitoring", logger)
		if err != nil {
			level.Error(log.With(logger, "component", "NewAdapter")).Log("err", err)
			os.Exit(2)
		}

		prometheus.MustRegister(metrics.NewMetricsCollector("ipmi_netbox_sd_up", "Shows if service can load netbox nodes", nAdapter, nDisc, opts.Version))
		adapterList = append(adapterList, nAdapter)
		discoveryList = append(discoveryList, nDisc)
	}

	go server.New(adapterList, discoveryList, logger).Start()

	adpt.Run()
	if opts.NetboxDiscovery {
		nAdapter.Run()
	}

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
