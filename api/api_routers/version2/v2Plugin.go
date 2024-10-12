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

package version2

import (
	"github.com/wutong-paas/wutong/api/controller"
	"github.com/wutong-paas/wutong/api/middleware"
	dbmodel "github.com/wutong-paas/wutong/db/model"

	"github.com/go-chi/chi"
)

// PluginRouter plugin router
func (v2 *V2) pluginRouter() chi.Router {
	r := chi.NewRouter()
	//初始化应用信息
	r.Use(middleware.InitPlugin)
	//plugin uri
	//update/delete plugin
	r.Put("/", controller.GetManager().PluginAction)
	r.Delete("/", controller.GetManager().PluginAction)
	r.Post("/build", controller.GetManager().PluginBuild)
	//get this plugin all build version
	r.Get("/build-version", controller.GetManager().GetAllPluginBuildVersions)
	r.Get("/build-version/{version_id}", controller.GetManager().GetPluginBuildVersion)
	r.Delete("/build-version/{version_id}", controller.GetManager().DeletePluginBuildVersion)
	return r
}

func (v2 *V2) serviceRelatePluginRouter() chi.Router {
	r := chi.NewRouter()
	//service relate plugin
	// v2/tenantEnv/tenant_env_name/services/service_alias/plugin/xxx
	r.Post("/", middleware.WrapEL(controller.GetManager().PluginSet, dbmodel.TargetTypeService, "create-service-plugin", dbmodel.SyncEventType))
	r.Put("/", middleware.WrapEL(controller.GetManager().PluginSet, dbmodel.TargetTypeService, "update-service-plugin", dbmodel.SyncEventType))
	r.Get("/", controller.GetManager().PluginSet)
	r.Delete("/{plugin_id}", middleware.WrapEL(controller.GetManager().DeletePluginRelation, dbmodel.TargetTypeService, "delete-service-plugin", dbmodel.SyncEventType))
	// app plugin config supdate
	r.Post("/{plugin_id}/setenv", middleware.WrapEL(controller.GetManager().UpdateVersionEnv, dbmodel.TargetTypeService, "update-service-plugin-config", dbmodel.SyncEventType))
	r.Put("/{plugin_id}/upenv", middleware.WrapEL(controller.GetManager().UpdateVersionEnv, dbmodel.TargetTypeService, "update-service-plugin-config", dbmodel.SyncEventType))
	//deprecated
	r.Get("/{plugin_id}/envs", controller.GetManager().GePluginEnvWhichCanBeSet)
	return r
}
