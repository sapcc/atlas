package server

import (
	"fmt"
	"net/http"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sapcc/ipmi_sd/internal/discovery"
	"github.com/sapcc/ipmi_sd/pkg/adapter"
)

type Server struct {
	adapter   *adapter.Adapter
	discovery *discovery.Discovery
	logger    log.Logger
}

func New(a *adapter.Adapter, d *discovery.Discovery, l log.Logger) *Server {
	return &Server{
		adapter:   a,
		discovery: d,
		logger:    l,
	}
}

func (s *Server) Start() {
	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/healthz", s.health)
	if err := http.ListenAndServe(":8080", nil); err != nil {
		panic(err)
	}
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	s.discovery.Status.Lock()
	s.adapter.Status.Lock()
	defer func() {
		s.discovery.Status.Unlock()
		s.adapter.Status.Unlock()
	}()
	if s.discovery.Status.Up && s.adapter.Status.Up {
		level.Debug(log.With(s.logger, "component", "health")).Log("debug", "health probe OK")
		w.WriteHeader(http.StatusOK)
		return
	}
	level.Debug(log.With(s.logger, "component", "health")).Log("debug", fmt.Sprintf("health probe NOK! discovery: %t, adapter: %t", s.discovery.Status.Up, s.adapter.Status.Up))
	w.WriteHeader(http.StatusServiceUnavailable)
}
