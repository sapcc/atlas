package discovery

import (
	"fmt"
	"net/http"
	"os"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sapcc/atlas/pkg/adapter"
)

type Server struct {
	adapter   []adapter.Adapter
	discovery []Discovery
	logger    log.Logger
}

func NewServer(a []adapter.Adapter, d []Discovery, l log.Logger) *Server {
	return &Server{
		adapter:   a,
		discovery: d,
		logger:    l,
	}
}

func (s *Server) Start() {
	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/healthz", s.health)
	http.HandleFunc("/-/reload", s.reloadHandler)
	if err := http.ListenAndServe(":8080", nil); err != nil {
		panic(err)
	}
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	for _, d := range s.discovery {
		d.Lock()
	}
	for _, a := range s.adapter {
		a.GetStatus().Lock()
	}
	defer func() {
		for _, d := range s.discovery {
			d.Unlock()
		}
		for _, a := range s.adapter {
			a.GetStatus().Unlock()
		}
	}()

	if s.Up() {
		level.Debug(log.With(s.logger, "component", "health")).Log("debug", "health probe OK")
		w.WriteHeader(http.StatusOK)
		return
	}
	level.Debug(log.With(s.logger, "component", "health")).Log("debug", fmt.Sprintf("health probe NOK!"))
	w.WriteHeader(http.StatusServiceUnavailable)
}

func (s *Server) reloadHandler(w http.ResponseWriter, req *http.Request) {
	level.Info(log.With(s.logger, "component", "reloader")).Log("info", "CONFIG_FILE changed. Restart atlas sd")
	w.WriteHeader(http.StatusOK)
	os.Exit(1)
}

func (s *Server) Up() bool {
	if len(s.discovery) == 0 || len(s.adapter) == 0 {
		return false
	}

	for _, d := range s.discovery {
		if !d.Up() {
			return false
		}
	}
	for _, a := range s.adapter {
		if !a.GetStatus().Up {
			return false
		}
	}
	return true
}
