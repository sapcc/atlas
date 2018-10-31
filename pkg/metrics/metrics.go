package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sapcc/ipmi_sd/internal/discovery"
	"github.com/sapcc/ipmi_sd/pkg/adapter"
)

type MetricsCollector struct {
	upGauge   *prometheus.Desc
	adapter   *adapter.Adapter
	discovery *discovery.Discovery
}

func (c *MetricsCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.upGauge
}

func (c *MetricsCollector) Collect(ch chan<- prometheus.Metric) {
	c.discovery.Status.Lock()
	c.adapter.Status.Lock()
	defer func() {
		c.discovery.Status.Unlock()
		c.adapter.Status.Unlock()
	}()
	up := 0
	if c.discovery.Status.Up && c.adapter.Status.Up {
		up = 1
	}
	ch <- prometheus.MustNewConstMetric(
		c.upGauge,
		prometheus.GaugeValue,
		float64(up),
	)
}

func NewMetricsCollector(a *adapter.Adapter, d *discovery.Discovery, v string) *MetricsCollector {
	return &MetricsCollector{
		upGauge:   prometheus.NewDesc("ipmi_sd_up", "Shows if service can load ironic nodes", nil, prometheus.Labels{"version": v}),
		adapter:   a,
		discovery: d,
	}
}
