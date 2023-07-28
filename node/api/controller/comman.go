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

package controller

import (
	"net/http"

	"github.com/wutong-paas/wutong/cmd/node/option"
	"github.com/wutong-paas/wutong/node/core/config"
	"github.com/wutong-paas/wutong/node/core/service"
	"github.com/wutong-paas/wutong/node/kubecache"
	"github.com/wutong-paas/wutong/node/masterserver"
)

var datacenterConfig *config.DataCenterConfig
var prometheusService *service.PrometheusService
var appService *service.AppService
var nodeService *service.NodeService
var discoverService *service.DiscoverAction
var kubecli kubecache.KubeClient

// Init 初始化
func Init(c *option.Conf, ms *masterserver.MasterServer, kube kubecache.KubeClient) {
	if ms != nil {
		prometheusService = service.CreatePrometheusService(c)
		datacenterConfig = config.GetDataCenterConfig()
		nodeService = service.CreateNodeService(c, ms.Cluster, kube)
	}
	appService = service.CreateAppService(c)
	discoverService = service.CreateDiscoverActionManager(c, kube)
	kubecli = kube
}

// Exist 退出
func Exist(i interface{}) {
	if datacenterConfig != nil {
		datacenterConfig.Stop()
	}
}

// Ping Ping
func Ping(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
}
