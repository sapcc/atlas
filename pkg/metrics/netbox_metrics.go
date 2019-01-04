package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sapcc/ipmi_sd/internal/discovery"
	"github.com/sapcc/ipmi_sd/pkg/adapter"
)

type MetricsNetboxCollector struct {
	upGauge   *prometheus.Desc
	adapter   *adapter.Adapter
	discovery *discovery.NetboxDiscovery
}

func (c *MetricsNetboxCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.upGauge
}

func (c *MetricsNetboxCollector) Collect(ch chan<- prometheus.Metric) {
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

func NewMetricsNetboxCollector(a *adapter.Adapter, d *discovery.NetboxDiscovery, v string) *MetricsNetboxCollector {
	return &MetricsNetboxCollector{
		upGauge:   prometheus.NewDesc("netbox_sd_up", "Shows if service can load netbox nodes", nil, prometheus.Labels{"version": v}),
		adapter:   a,
		discovery: d,
	}
}