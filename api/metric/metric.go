// WUTONG, Application Management Platform
// Copyright (C) 2020-2020 Wutong Co., Ltd.

// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version. For any non-GPL usage of Wutong,
// one or multiple Commercial Licenses authorized by Wutong Co., Ltd.
// must be obtained first.

// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.

// You should have received a copy of the GNU General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package metric

import (
	"context"
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/wutong-paas/wutong/api/handler"
)

// Metric name parts.
const (
	// Namespace for all metrics.
	namespace = "wt_api"
	// Subsystem(s).
	exporter = "exporter"
)

// NewExporter new exporter
func NewExporter() *Exporter {
	return &Exporter{
		apiRequest: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: exporter,
			Name:      "api_request",
			Help:      "wutong cluster api request metric",
		}, []string{"code", "path"}),
		tenantEnvLimit: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: exporter,
			Name:      "tenant_env_memory_limit",
			Help:      "wutong tenant env memory limit",
		}, []string{"tenant_env_id", "namespace"}),
		clusterMemoryTotal: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: exporter,
			Name:      "cluster_memory_total",
			Help:      "wutong cluster memory total",
		}),
		clusterCPUTotal: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: exporter,
			Name:      "cluster_cpu_total",
			Help:      "wutong cluster cpu total",
		}),
	}
}

// Exporter exporter
type Exporter struct {
	apiRequest         *prometheus.CounterVec
	tenantEnvLimit     *prometheus.GaugeVec
	clusterCPUTotal    prometheus.Gauge
	clusterMemoryTotal prometheus.Gauge
}

// RequestInc request inc
func (e *Exporter) RequestInc(code int, path string) {
	e.apiRequest.WithLabelValues(fmt.Sprintf("%d", code), path).Inc()
}

// Describe implements prometheus.Collector.
func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	metricCh := make(chan prometheus.Metric)
	doneCh := make(chan struct{})

	go func() {
		for m := range metricCh {
			ch <- m.Desc()
		}
		close(doneCh)
	}()

	e.Collect(metricCh)
	close(metricCh)
	<-doneCh
}

// Collect implements prometheus.Collector.
func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	e.apiRequest.Collect(ch)
	// tenant env limit value
	tenantEnvs, _ := handler.GetTenantEnvManager().GetAllTenantEnvs("")
	for _, t := range tenantEnvs {
		e.tenantEnvLimit.WithLabelValues(t.UUID, t.UUID).Set(float64(t.LimitMemory))
	}
	// cluster memory
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()
	resource := handler.GetTenantEnvManager().GetClusterResource(ctx)
	if resource != nil {
		e.clusterMemoryTotal.Set(float64(resource.AllMemory))
		e.clusterCPUTotal.Set(float64(resource.AllCPU))
	}
	e.tenantEnvLimit.Collect(ch)
	e.clusterMemoryTotal.Collect(ch)
	e.clusterCPUTotal.Collect(ch)
}
