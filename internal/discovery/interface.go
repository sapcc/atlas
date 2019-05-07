package discovery

import (
	"context"
	"sync"

	"github.com/sapcc/atlas/pkg/writer"

	"github.com/go-kit/kit/log"
	promDiscovery "github.com/prometheus/prometheus/discovery"
	"github.com/prometheus/prometheus/discovery/targetgroup"
	"github.com/sapcc/atlas/pkg/adapter"
	"github.com/sapcc/atlas/pkg/config"
)

type Discovery interface {
	Up() bool
	Lock()
	Unlock()
	Run(ctx context.Context, ch chan<- []*targetgroup.Group)
	GetOutputFile() string
	StartAdapter()
	GetAdapter() adapter.Adapter
	GetManager() *promDiscovery.Manager
}

type DiscoveryFactory func(config interface{}, ctx context.Context, m *promDiscovery.Manager, opts config.Options, w writer.Writer, l log.Logger) (Discovery, error)

type Status struct {
	sync.Mutex
	Up bool
}
