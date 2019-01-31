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

package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sapcc/ipmi_sd/internal/discovery"
	"github.com/sapcc/ipmi_sd/pkg/adapter"
)

type MetricsCollector struct {
	upGauge   *prometheus.Desc
	adapter   *adapter.Adapter
	discovery discovery.Discovery
}

func (c *MetricsCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.upGauge
}

func (c *MetricsCollector) Collect(ch chan<- prometheus.Metric) {
	c.discovery.Lock()
	c.adapter.Status.Lock()
	defer func() {
		c.discovery.Unlock()
		c.adapter.Status.Unlock()
	}()
	up := 0
	if c.discovery.Up() && c.adapter.Status.Up {
		up = 1
	}
	ch <- prometheus.MustNewConstMetric(
		c.upGauge,
		prometheus.GaugeValue,
		float64(up),
	)
}

func NewMetricsCollector(name string, help string, a *adapter.Adapter, d discovery.Discovery, v string) *MetricsCollector {
	return &MetricsCollector{
		upGauge:   prometheus.NewDesc(name, help, nil, prometheus.Labels{"version": v}),
		adapter:   a,
		discovery: d,
	}
}
