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
	"time"

	"github.com/prometheus/common/model"
	"github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
	"github.com/wutong-paas/wutong/discover"
	"github.com/wutong-paas/wutong/discover/config"
	"github.com/wutong-paas/wutong/monitor/prometheus"
	"github.com/wutong-paas/wutong/monitor/utils"
)

//Mq discover
type Mq struct {
	discover.Callback
	Prometheus      *prometheus.Manager
	sortedEndpoints []string
}

//UpdateEndpoints update endpoint
func (m *Mq) UpdateEndpoints(endpoints ...*config.Endpoint) {
	newEndpoints := make([]*config.Endpoint, 0, len(endpoints))
	for _, end := range endpoints {
		newEnd := *end
		newEndpoints = append(newEndpoints, &newEnd)
	}

	for i, end := range endpoints {
		newEndpoints[i].URL = gjson.Get(end.URL, "Addr").String()
	}

	newArr := utils.TrimAndSort(newEndpoints)

	if utils.ArrCompare(m.sortedEndpoints, newArr) {
		logrus.Debugf("The endpoints is not modify: %s", m.Name())
		return
	}

	m.sortedEndpoints = newArr

	scrape := m.toScrape()
	m.Prometheus.UpdateScrape(scrape)
}

func (m *Mq) Error(err error) {
	logrus.Error(err)
}

//Name name
func (m *Mq) Name() string {
	return "mq"
}

func (m *Mq) toScrape() *prometheus.ScrapeConfig {
	ts := make([]string, 0, len(m.sortedEndpoints))
	for _, end := range m.sortedEndpoints {
		ts = append(ts, end)
	}

	return &prometheus.ScrapeConfig{
		JobName:        m.Name(),
		ScrapeInterval: model.Duration(time.Minute),
		ScrapeTimeout:  model.Duration(30 * time.Second),
		MetricsPath:    "/metrics",
		HonorLabels:    true,
		ServiceDiscoveryConfig: prometheus.ServiceDiscoveryConfig{
			StaticConfigs: []*prometheus.Group{
				{
					Targets: ts,
					Labels: map[model.LabelName]model.LabelValue{
						"service_name": model.LabelValue(m.Name()),
						"component":    model.LabelValue(m.Name()),
					},
				},
			},
		},
	}
}
