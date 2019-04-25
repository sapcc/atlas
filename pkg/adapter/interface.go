package adapter

import (
	"context"
	"sync"

	"github.com/go-kit/kit/log"
	promDiscovery "github.com/prometheus/prometheus/discovery"
	"github.com/sapcc/ipmi_sd/pkg/writer"
)

type Status struct {
	sync.Mutex
	Up bool
}

type AdapterFactory func(ctx context.Context, m *promDiscovery.Manager, w writer.Writer, logger log.Logger) (Adapter, error)

type Adapter interface {
	GetStatus() *Status
	Run()
}
