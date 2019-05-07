package discovery

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"strings"
	"sync"

	"github.com/sapcc/atlas/pkg/writer"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	promDiscovery "github.com/prometheus/prometheus/discovery"
	"github.com/sapcc/atlas/pkg/adapter"
	"github.com/sapcc/atlas/pkg/config"
	"gopkg.in/yaml.v2"
)

var discoveryFactories = make(map[string]DiscoveryFactory)

type (
	discovery struct {
		opts   config.Options
		ctx    context.Context
		writer writer.Writer
		log    log.Logger
	}
	configTemplateValues struct {
		Region string
	}
)

func New(ctx context.Context, o config.Options, w writer.Writer, l log.Logger) *discovery {
	discovery := &discovery{
		o,
		ctx,
		w,
		l,
	}

	return discovery
}

func Register(name string, factory DiscoveryFactory) (err error) {
	if factory == nil {
		return fmt.Errorf("Handler factory %s does not exist", name)
	}
	_, registered := discoveryFactories[name]
	if registered {
		//log.Errorf("Handler factory %s already registered. Ignoring.", name)
	}
	discoveryFactories[name] = factory
	return
}

func (d discovery) createDiscovery(name string, config interface{}) (Discovery, error) {

	discoveryFactory, ok := discoveryFactories[name]
	if !ok {
		availableDiscoveries := d.getDiscoveries()
		return nil, fmt.Errorf(fmt.Sprintf("Invalid Handler name. Must be one of: %s", strings.Join(availableDiscoveries, ", ")))
	}

	m := promDiscovery.NewManager(d.ctx, d.log)
	// Run the factory with the configuration.
	return discoveryFactory(config, d.ctx, m, d.opts, d.writer, d.log)
}

func (d discovery) getDiscoveries() []string {
	availableHandlers := make([]string, len(discoveryFactories))
	for k := range discoveryFactories {
		availableHandlers = append(availableHandlers, k)
	}
	return availableHandlers
}

// Start is running all enabled discovery services
func (d discovery) Start(ctx context.Context, wg *sync.WaitGroup, cfg config.Config, opts config.Options) {
	defer wg.Done()
	wg.Add(1)
	adapterList := make([]adapter.Adapter, 0)
	discoveryList := make([]Discovery, 0)

	for name, discovery := range cfg.Discoveries {
		level.Info(log.With(d.log, "component", "discovery")).Log("info", fmt.Sprintf("=============> Loading discovery: %s", name))
		disc, err := d.createDiscovery(name, discovery)
		if err != nil {
			level.Error(log.With(d.log, "component", "discovery")).Log("err", err)
			continue
		}

		go disc.GetManager().Run()
		disc.GetManager().StartCustomProvider(ctx, name, disc)
		go disc.StartAdapter()

		adapterList = append(adapterList, disc.GetAdapter())
		discoveryList = append(discoveryList, disc)
		prometheus.MustRegister(NewMetricsCollector(name+"_sd_up", "Shows if discovery is running", disc.GetAdapter(), disc, opts.Version))

	}
	go NewServer(adapterList, discoveryList, d.log).Start()
}

func UnmarshalHandler(discIn, discOut, values interface{}) error {
	var tpl bytes.Buffer
	h, err := yaml.Marshal(discIn)
	if err != nil {
		return err
	}

	t, err := template.New("config").Parse(string(h))
	err = t.Execute(&tpl, values)

	return yaml.Unmarshal(tpl.Bytes(), discOut)
}
