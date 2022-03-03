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

package collectors

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/wutong-paas/wutong/gateway/controller/openresty/nginxcmd"
)

//NginxCmdMetric -
type NginxCmdMetric struct {
}

//Describe -
func (n *NginxCmdMetric) Describe(ch chan<- *prometheus.Desc) {
	nginxcmd.PromethesuScrape(ch)
}

//Collect -
func (n *NginxCmdMetric) Collect(ch chan<- prometheus.Metric) {
	nginxcmd.PrometheusCollect(ch)
}
