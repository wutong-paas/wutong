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

package router

import (
	"github.com/go-chi/chi"
	"github.com/wutong-paas/wutong/node/api/controller"
)

// DisconverRoutes envoy discover api
// v1 api will abandoned in 5.2
func DisconverRoutes() chi.Router {
	r := chi.NewRouter()
	r.Get("/ping", controller.Ping)
	r.Mount("/listeners", ListenersRoutes())
	r.Mount("/clusters", ClustersRoutes())
	r.Mount("/registration", RegistrationRoutes())
	r.Mount("/routes", RoutesRouters())
	r.Mount("/resources", SourcesRoutes())
	return r
}

// ListenersRoutes listeners routes lds
// GET /v1/listeners/(string: service_cluster)/(string: service_node)
func ListenersRoutes() chi.Router {
	r := chi.NewRouter()
	r.Get("/ping", controller.Ping)
	r.Get("/{tenant_env_service}/{service_nodes}", controller.ListenerDiscover)
	return r
}

// ClustersRoutes cds
// GET /v1/clusters/(string: service_cluster)/(string: service_node)
func ClustersRoutes() chi.Router {
	r := chi.NewRouter()
	r.Get("/ping", controller.Ping)
	r.Get("/{tenant_env_service}/{service_nodes}", controller.ClusterDiscover)
	return r
}

// RegistrationRoutes sds
// GET /v1/registration/(string: service_name)
func RegistrationRoutes() chi.Router {
	r := chi.NewRouter()
	r.Get("/ping", controller.Ping)
	r.Get("/{service_name}", controller.ServiceDiscover)
	return r
}

// RoutesRouters rds
// GET /v1/routes/(string: route_config_name)/(string: service_cluster)/(string: service_node)
func RoutesRouters() chi.Router {
	r := chi.NewRouter()
	r.Get("/ping", controller.Ping)
	r.Get("/{route_config}/{tenant_env_service}/{service_nodes}", controller.RoutesDiscover)
	return r
}

// SourcesRoutes SourcesRoutes
// GET /v1/resources/(string: tenant_env_id)/(string: service_alias)/(string: plugin_id)
func SourcesRoutes() chi.Router {
	r := chi.NewRouter()
	r.Get("/{tenant_env_id}/{service_alias}/{plugin_id}", controller.PluginResourcesConfig)
	return r
}
