package discovery

import (
	"context"
	"sync"

	"github.com/go-kit/kit/log"
	"github.com/prometheus/prometheus/discovery/targetgroup"
	"github.com/sapcc/atlas/pkg/adapter"
	"github.com/sapcc/atlas/pkg/config"
)

type Discovery interface {
	Up() bool
	Targets() map[string]int
	GetName() string
	Lock()
	Unlock()
	Run(ctx context.Context, ch chan<- []*targetgroup.Group)
	GetOutputFile() string
	GetAdapter() adapter.Adapter
}

type DiscoveryFactory func(config interface{}, ctx context.Context, opts config.Options, l log.Logger) (Discovery, error)

type Status struct {
	sync.Mutex
	Up      bool
	Targets map[string]int
}
