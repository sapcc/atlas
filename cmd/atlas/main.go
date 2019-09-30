/*******************************************************************************
*
* Copyright 2019 SAP SE
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
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/namsral/flag"
	"github.com/sapcc/atlas/internal/discovery"
	"github.com/sapcc/atlas/pkg/config"
	"github.com/sapcc/atlas/pkg/writer"
)

var (
	logger log.Logger
	opts   config.Options
	cfg    config.Config
	w      writer.Writer
	err    error
)

func init() {
	logger = log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
	logger = log.With(logger, "ts", log.DefaultTimestampUTC)

	flag.StringVar(&opts.LogLevel, "LOG_LEVEL", "debug", "To set Log Level")
	flag.StringVar(&opts.Version, "OS_VERSION", "v0.3.0", "IPMI SD Version")
	flag.StringVar(&opts.NameSpace, "K8S_NAMESPACE", "kube-monitoring", "k8s Namespace the service is running in")
	flag.StringVar(&opts.Region, "K8S_REGION", "qa-de-1", "k8s Region the service is running in")
	flag.StringVar(&opts.WriteTo, "WRITE_TO", "file", "k8s Region the service is running in")

	flag.StringVar(&opts.ConfigFilePath, "CONFIG_FILE", "./etc/config/config.yaml", "Path to the config file")
	if val, ok := os.LookupEnv("PROM_CONFIGMAP_NAME"); ok {
		opts.ConfigmapName = val
	} else {
		level.Error(log.With(logger, "component", "atlas")).Log("err", "no configmap name given")
		os.Exit(2)
	}

	switch strings.ToLower(opts.LogLevel) {
	case "info":
		logger = level.NewFilter(logger, level.AllowInfo())
	case "debug":
		logger = level.NewFilter(logger, level.AllowDebug())
	case "warn":
		logger = level.NewFilter(logger, level.AllowWarn())
	case "error":
		logger = level.NewFilter(logger, level.AllowError())
	}

	switch strings.ToLower(opts.WriteTo) {
	case "configmap":
		w, err = writer.NewConfigMap(opts.ConfigmapName, opts.NameSpace, logger)
	case "file":
		w, err = writer.NewFile(opts.ConfigmapName, logger)
	}

	if err != nil {
		level.Error(log.With(logger, "component", "atlas")).Log("err", err)
		os.Exit(2)
	}

	level.Debug(log.With(logger, "component", "atlas")).Log("info", opts.ConfigFilePath)
}

func main() {
	cfg, err := config.GetConfig(opts)
	if err != nil {
		level.Error(log.With(logger, "component", "atlas")).Log("err", err)
		os.Exit(2)
	}

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	wg := &sync.WaitGroup{}

	discovery := discovery.New(ctx, opts, w, logger)

	go discovery.Start(ctx, wg, cfg, opts)

	handleReload(cancel)

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

func handleReload(cancel context.CancelFunc) {
	http.HandleFunc("/-/reload", func(w http.ResponseWriter, req *http.Request) {
		level.Info(log.With(logger, "component", "atlas")).Log("info", "atlas config file changed! Reloading . . . ")
		w.WriteHeader(http.StatusOK)
		cancel()
	})
}
