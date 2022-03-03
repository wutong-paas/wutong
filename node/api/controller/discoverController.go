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

	"github.com/wutong-paas/wutong/api/util"

	"github.com/go-chi/chi"
	"github.com/sirupsen/logrus"
	httputil "github.com/wutong-paas/wutong/util/http"
)

//ServiceDiscover service discover service
func ServiceDiscover(w http.ResponseWriter, r *http.Request) {
	serviceInfo := chi.URLParam(r, "service_name")
	sds, err := discoverService.DiscoverService(serviceInfo)
	if err != nil {
		err.Handle(r, w)
		return
	}
	httputil.ReturnNoFomart(r, w, 200, sds)
}

//ListenerDiscover ListenerDiscover
func ListenerDiscover(w http.ResponseWriter, r *http.Request) {
	tenantService := chi.URLParam(r, "tenant_service")
	serviceNodes := chi.URLParam(r, "service_nodes")
	lds, err := discoverService.DiscoverListeners(tenantService, serviceNodes)
	if err != nil {
		err.Handle(r, w)
		return
	}
	httputil.ReturnNoFomart(r, w, 200, lds)
}

//ClusterDiscover ClusterDiscover
func ClusterDiscover(w http.ResponseWriter, r *http.Request) {
	tenantService := chi.URLParam(r, "tenant_service")
	serviceNodes := chi.URLParam(r, "service_nodes")
	cds, err := discoverService.DiscoverClusters(tenantService, serviceNodes)
	if err != nil {
		err.Handle(r, w)
		return
	}
	httputil.ReturnNoFomart(r, w, 200, cds)
}

//RoutesDiscover RoutesDiscover
//no impl
func RoutesDiscover(w http.ResponseWriter, r *http.Request) {
	namespace := chi.URLParam(r, "tenant_id")
	serviceNodes := chi.URLParam(r, "service_nodes")
	routeConfig := chi.URLParam(r, "route_config")
	logrus.Debugf("route_config is %s, namespace %s, serviceNodes %s", routeConfig, namespace, serviceNodes)
	w.WriteHeader(200)
}

//PluginResourcesConfig discover plugin config
func PluginResourcesConfig(w http.ResponseWriter, r *http.Request) {
	namespace := chi.URLParam(r, "tenant_id")
	serviceAlias := chi.URLParam(r, "service_alias")
	pluginID := chi.URLParam(r, "plugin_id")
	ss, err := discoverService.GetPluginConfigs(namespace, serviceAlias, pluginID)
	if err != nil {
		util.CreateAPIHandleError(500, err).Handle(r, w)
		return
	}
	if ss == nil {
		util.CreateAPIHandleError(404, err).Handle(r, w)
	}
	httputil.ReturnNoFomart(r, w, 200, ss)
}
