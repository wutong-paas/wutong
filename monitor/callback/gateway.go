// WUTONG, Application Management Platform
// Copyright (C) 2014-2019 Wutong Co., Ltd.

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
	"time"

	"github.com/prometheus/common/model"
	"github.com/sirupsen/logrus"
	"github.com/wutong-paas/wutong/discover"
	"github.com/wutong-paas/wutong/discover/config"
	"github.com/wutong-paas/wutong/monitor/prometheus"
	"github.com/wutong-paas/wutong/monitor/utils"
)

//GatewayNode node discover
type GatewayNode struct {
	discover.Callback
	Prometheus      *prometheus.Manager
	sortedEndpoints []string

	endpoints []*config.Endpoint
}

//UpdateEndpoints update endpoints
func (e *GatewayNode) UpdateEndpoints(endpoints ...*config.Endpoint) {
	newArr := utils.TrimAndSort(endpoints)

	if utils.ArrCompare(e.sortedEndpoints, newArr) {
		logrus.Debugf("The endpoints is not modify: %s", e.Name())
		return
	}

	e.sortedEndpoints = newArr

	scrape := e.toScrape()
	e.Prometheus.UpdateScrape(scrape)
}

func (e *GatewayNode) Error(err error) {
	logrus.Error(err)
}

//Name name
func (e *GatewayNode) Name() string {
	return "gateway"
}

func (e *GatewayNode) toScrape() *prometheus.ScrapeConfig {
	ts := make([]string, 0, len(e.sortedEndpoints))
	for _, end := range e.sortedEndpoints {
		ts = append(ts, end)
	}

	return &prometheus.ScrapeConfig{
		JobName:        e.Name(),
		ScrapeInterval: model.Duration(30 * time.Second),
		ScrapeTimeout:  model.Duration(30 * time.Second),
		MetricsPath:    "/metrics",
		ServiceDiscoveryConfig: prometheus.ServiceDiscoveryConfig{
			StaticConfigs: []*prometheus.Group{
				{
					Targets: ts,
					Labels:  map[model.LabelName]model.LabelValue{},
				},
			},
		},
	}
}

//AddEndpoint add endpoint
func (e *GatewayNode) AddEndpoint(end *config.Endpoint) {
	e.endpoints = append(e.endpoints, end)
	e.UpdateEndpoints(e.endpoints...)
}
