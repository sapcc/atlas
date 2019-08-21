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

package discovery

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sapcc/atlas/pkg/adapter"
)

type MetricsCollector struct {
	upGauge   *prometheus.Desc
	targets   *prometheus.Desc
	adapter   adapter.Adapter
	discovery Discovery
}

func (c *MetricsCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.upGauge
}

func (c *MetricsCollector) Collect(ch chan<- prometheus.Metric) {
	c.discovery.Lock()
	c.adapter.GetStatus().Lock()
	defer func() {
		c.discovery.Unlock()
		c.adapter.GetStatus().Unlock()
	}()
	up := 0
	if c.discovery.Up() && c.adapter.GetStatus().Up {
		up = 1
	}
	ch <- prometheus.MustNewConstMetric(
		c.upGauge,
		prometheus.GaugeValue,
		float64(up),
	)
	for key, value := range c.discovery.Targets() {
		ch <- prometheus.MustNewConstMetric(
			c.targets,
			prometheus.GaugeValue,
			float64(value),
			key,
		)
	}
}

func NewMetricsCollector(a adapter.Adapter, d Discovery, v string) *MetricsCollector {
	return &MetricsCollector{
		targets: prometheus.NewDesc(
			"atlas_targets",
			"Number of found targets",
			[]string{"module"},
			prometheus.Labels{"version": v}),
		upGauge: prometheus.NewDesc(
			"atlas_sd_up",
			"Shows if discovery is running",
			nil,
			prometheus.Labels{
				"version":   v,
				"discovery": d.GetName(),
			}),
		adapter:   a,
		discovery: d,
	}
}
