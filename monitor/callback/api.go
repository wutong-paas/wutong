// Copyright (C) 2014-2018 Wutong Co., Ltd.
// WUTONG, Application Management Platform

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

package callback

import (
	"os"
	"time"

	"github.com/prometheus/common/model"
	"github.com/sirupsen/logrus"
	"github.com/wutong-paas/wutong/discover"
	"github.com/wutong-paas/wutong/discover/config"
	"github.com/wutong-paas/wutong/monitor/prometheus"
)

//WtAPI wt api metrics
type WtAPI struct {
	discover.Callback
	Prometheus      *prometheus.Manager
	sortedEndpoints []string
}

//UpdateEndpoints update endpoint
func (b *WtAPI) UpdateEndpoints(endpoints ...*config.Endpoint) {
	scrape := b.toScrape()
	b.Prometheus.UpdateScrape(scrape)
}

//Error handle error
func (b *WtAPI) Error(err error) {
	logrus.Error(err)
}

//Name name
func (b *WtAPI) Name() string {
	return "wtapi"
}

func (b *WtAPI) toScrape() *prometheus.ScrapeConfig {
	ts := make([]string, 0, len(b.sortedEndpoints))
	for _, end := range b.sortedEndpoints {
		ts = append(ts, end)
	}
	namespace := os.Getenv("NAMESPACE")

	return &prometheus.ScrapeConfig{
		JobName:        b.Name(),
		ScrapeInterval: model.Duration(time.Minute),
		ScrapeTimeout:  model.Duration(30 * time.Second),
		MetricsPath:    "/metrics",
		HonorLabels:    true,
		ServiceDiscoveryConfig: prometheus.ServiceDiscoveryConfig{
			KubernetesSDConfigs: []*prometheus.SDConfig{
				&prometheus.SDConfig{
					Role: prometheus.RoleEndpoint,
					NamespaceDiscovery: prometheus.NamespaceDiscovery{
						Names: []string{namespace},
					},
					Selectors: []prometheus.SelectorConfig{
						prometheus.SelectorConfig{
							Role:  prometheus.RoleEndpoint,
							Field: "metadata.name=wt-api-api-inner",
						},
					},
				},
			},
		},
	}
}
