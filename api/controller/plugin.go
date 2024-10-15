// Copyright (C) 2014-2018 Wutong Co., Ltd.
// WUTONG, Application Management Platform

// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version. For any non-GPL usage of Wutong
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
	"log"
	"net/http"

	"github.com/wutong-paas/wutong/api/handler/share"

	"github.com/go-chi/chi"
	"github.com/wutong-paas/wutong/api/handler"
	"github.com/wutong-paas/wutong/util"

	api_model "github.com/wutong-paas/wutong/api/model"
	ctxutil "github.com/wutong-paas/wutong/api/util/ctx"
	httputil "github.com/wutong-paas/wutong/util/http"
)

// PluginSysAction plugin sys action
func (t *TenantEnvStruct) SysPluginAction(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "PUT":
		t.UpdateSysPlugin(w, r)
	case "DELETE":
		t.DeleteSysPlugin(w, r)
	case "POST":
		t.CreateSysPlugin(w, r)
	}
}

// PluginAction plugin action
func (t *TenantEnvStruct) PluginAction(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "PUT":
		t.UpdatePlugin(w, r)
	case "DELETE":
		t.DeletePlugin(w, r)
	case "POST":
		t.CreatePlugin(w, r)
	case "GET":
		t.GetPlugins(w, r)
	}
}

// CreatePlugin add plugin
func (t *TenantEnvStruct) CreateSysPlugin(w http.ResponseWriter, r *http.Request) {
	// swagger:operation POST /v2/sys-plugin v2 createSysPlugin 创建系统插件
	//
	// 创建系统插件
	//
	// create sys plugin
	//
	// ---
	// consumes:
	// - application/json
	// - application/x-protobuf
	//
	// produces:
	// - application/json
	// - application/xml
	//
	// responses:
	//   default:
	//     schema:
	//       "$ref": "#/responses/commandResponse"
	//     description: 统一返回格式
	var cps api_model.CreateSysPluginStruct
	if ok := httputil.ValidatorRequestStructAndErrorResponse(r, w, &cps.Body, nil); !ok {
		return
	}
	if err := handler.GetPluginManager().CreateSysPluginAct(&cps); err != nil {
		err.Handle(r, w)
		return
	}
	httputil.ReturnSuccess(r, w, nil)
}

// CreatePlugin add plugin
func (t *TenantEnvStruct) CreatePlugin(w http.ResponseWriter, r *http.Request) {
	// swagger:operation POST /v2/tenants/{tenant_name}/envs/{tenant_env_name}/plugin v2 createPlugin
	//
	// 创建插件
	//
	// create plugin
	//
	// ---
	// consumes:
	// - application/json
	// - application/x-protobuf
	//
	// produces:
	// - application/json
	// - application/xml
	//
	// responses:
	//   default:
	//     schema:
	//       "$ref": "#/responses/commandResponse"
	//     description: 统一返回格式
	tenantEnvID := r.Context().Value(ctxutil.ContextKey("tenant_env_id")).(string)
	tenantEnvName := r.Context().Value(ctxutil.ContextKey("tenant_env_name")).(string)
	var cps api_model.CreatePluginStruct
	if ok := httputil.ValidatorRequestStructAndErrorResponse(r, w, &cps.Body, nil); !ok {
		return
	}
	cps.Body.TenantEnvID = tenantEnvID
	cps.TenantEnvName = tenantEnvName
	if err := handler.GetPluginManager().CreatePluginAct(&cps); err != nil {
		err.Handle(r, w)
		return
	}
	httputil.ReturnSuccess(r, w, nil)
}

// UpdateSysPlugin UpdateSysPlugin
func (t *TenantEnvStruct) UpdateSysPlugin(w http.ResponseWriter, r *http.Request) {
	// swagger:operation PUT /v2/sys-plugin v2 updateSysPlugin
	//
	// 插件更新 全量更新，但pluginID和所在租户不提供修改
	//
	// update plugin
	//
	// ---
	// consumes:
	// - application/json
	// - application/x-protobuf
	//
	// produces:
	// - application/json
	// - application/xml
	//
	// responses:
	//   default:
	//     schema:
	//       "$ref": "#/responses/commandResponse"
	//     description: 统一返回格式

	pluginID := r.Context().Value(ctxutil.ContextKey("plugin_id")).(string)
	var ups api_model.UpdateSysPluginStruct
	if ok := httputil.ValidatorRequestStructAndErrorResponse(r, w, &ups.Body, nil); !ok {
		return
	}
	if err := handler.GetPluginManager().UpdateSysPluginAct(pluginID, &ups); err != nil {
		err.Handle(r, w)
		return
	}
	httputil.ReturnSuccess(r, w, nil)
}

// UpdatePlugin UpdatePlugin
func (t *TenantEnvStruct) UpdatePlugin(w http.ResponseWriter, r *http.Request) {
	// swagger:operation PUT /v2/tenants/{tenant_name}/envs/{tenant_env_name}/plugin/{plugin_id} v2 updatePlugin
	//
	// 插件更新 全量更新，但pluginID和所在租户不提供修改
	//
	// update plugin
	//
	// ---
	// consumes:
	// - application/json
	// - application/x-protobuf
	//
	// produces:
	// - application/json
	// - application/xml
	//
	// responses:
	//   default:
	//     schema:
	//       "$ref": "#/responses/commandResponse"
	//     description: 统一返回格式

	pluginID := r.Context().Value(ctxutil.ContextKey("plugin_id")).(string)
	tenantEnvID := r.Context().Value(ctxutil.ContextKey("tenant_env_id")).(string)
	var ups api_model.UpdatePluginStruct
	if ok := httputil.ValidatorRequestStructAndErrorResponse(r, w, &ups.Body, nil); !ok {
		return
	}
	if err := handler.GetPluginManager().UpdatePluginAct(pluginID, tenantEnvID, &ups); err != nil {
		err.Handle(r, w)
		return
	}
	httputil.ReturnSuccess(r, w, nil)
}

// DeleteSysPlugin DeleteSysPlugin
func (t *TenantEnvStruct) DeleteSysPlugin(w http.ResponseWriter, r *http.Request) {
	// swagger:operation DELETE /v2/sys-plugin v2 deleteSysPlugin
	//
	// 插件系统删除
	//
	// delete sys plugin
	//
	// ---
	// consumes:
	// - application/json
	// - application/x-protobuf
	//
	// produces:
	// - application/json
	// - application/xml
	//
	// responses:
	//   default:
	//     schema:
	//       "$ref": "#/responses/commandResponse"
	//     description: 统一返回格式
	pluginID := r.Context().Value(ctxutil.ContextKey("plugin_id")).(string)
	if err := handler.GetPluginManager().DeleteSysPluginAct(pluginID); err != nil {
		err.Handle(r, w)
		return
	}
	httputil.ReturnSuccess(r, w, nil)
}

// DeletePlugin DeletePlugin
func (t *TenantEnvStruct) DeletePlugin(w http.ResponseWriter, r *http.Request) {
	// swagger:operation DELETE /v2/tenants/{tenant_name}/envs/{tenant_env_name}/plugin/{plugin_id} v2 deletePlugin
	//
	// 插件删除
	//
	// delete plugin
	//
	// ---
	// consumes:
	// - application/json
	// - application/x-protobuf
	//
	// produces:
	// - application/json
	// - application/xml
	//
	// responses:
	//   default:
	//     schema:
	//       "$ref": "#/responses/commandResponse"
	//     description: 统一返回格式
	pluginID := r.Context().Value(ctxutil.ContextKey("plugin_id")).(string)
	tenantEnvID := r.Context().Value(ctxutil.ContextKey("tenant_env_id")).(string)
	if err := handler.GetPluginManager().DeletePluginAct(pluginID, tenantEnvID); err != nil {
		err.Handle(r, w)
		return
	}
	httputil.ReturnSuccess(r, w, nil)
}

// GetPlugins GetPlugins
func (t *TenantEnvStruct) GetPlugins(w http.ResponseWriter, r *http.Request) {
	// swagger:operation GET /v2/tenants/{tenant_name}/envs/{tenant_env_name}/plugin v2 getPlugins
	//
	// 获取当前租户下所有的可用插件
	//
	// get plugins
	//
	// ---
	// consumes:
	// - application/json
	// - application/x-protobuf
	//
	// produces:
	// - application/json
	// - application/xml
	//
	// responses:
	//   default:
	//     schema:
	//       "$ref": "#/responses/commandResponse"
	//     description: 统一返回格式
	tenantEnvID := r.Context().Value(ctxutil.ContextKey("tenant_env_id")).(string)
	plugins, err := handler.GetPluginManager().GetPlugins(tenantEnvID)
	if err != nil {
		err.Handle(r, w)
		return
	}
	httputil.ReturnSuccess(r, w, plugins)
}

// PluginDefaultENV PluginDefaultENV
func (t *TenantEnvStruct) PluginDefaultENV(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		t.AddDefatultENV(w, r)
	case "DELETE":
		t.DeleteDefaultENV(w, r)
	case "PUT":
		t.UpdateDefaultENV(w, r)
	}
}

// AddDefatultENV AddDefatultENV
func (t *TenantEnvStruct) AddDefatultENV(w http.ResponseWriter, r *http.Request) {
	pluginID := r.Context().Value(ctxutil.ContextKey("plugin_id")).(string)
	versionID := chi.URLParam(r, "version_id")
	var est api_model.ENVStruct
	if ok := httputil.ValidatorRequestStructAndErrorResponse(r, w, &est.Body, nil); !ok {
		return
	}
	est.VersionID = versionID
	est.PluginID = pluginID
	if err := handler.GetPluginManager().AddDefaultEnv(&est); err != nil {
		err.Handle(r, w)
		return
	}
}

// DeleteDefaultENV DeleteDefaultENV
func (t *TenantEnvStruct) DeleteDefaultENV(w http.ResponseWriter, r *http.Request) {
	pluginID := r.Context().Value(ctxutil.ContextKey("plugin_id")).(string)
	envName := chi.URLParam(r, "env_name")
	versionID := chi.URLParam(r, "version_id")
	if err := handler.GetPluginManager().DeleteDefaultEnv(pluginID, versionID, envName); err != nil {
		err.Handle(r, w)
		return
	}
}

// UpdateDefaultENV UpdateDefaultENV
func (t *TenantEnvStruct) UpdateDefaultENV(w http.ResponseWriter, r *http.Request) {

	pluginID := r.Context().Value(ctxutil.ContextKey("plugin_id")).(string)
	versionID := chi.URLParam(r, "version_id")
	var est api_model.ENVStruct
	if ok := httputil.ValidatorRequestStructAndErrorResponse(r, w, &est.Body, nil); !ok {
		return
	}
	est.PluginID = pluginID
	est.VersionID = versionID
	if err := handler.GetPluginManager().UpdateDefaultEnv(&est); err != nil {
		err.Handle(r, w)
		return
	}
}

// GetPluginDefaultEnvs GetPluginDefaultEnvs
func (t *TenantEnvStruct) GetPluginDefaultEnvs(w http.ResponseWriter, r *http.Request) {
	pluginID := r.Context().Value(ctxutil.ContextKey("plugin_id")).(string)
	versionID := chi.URLParam(r, "version_id")
	envs, err := handler.GetPluginManager().GetDefaultEnv(pluginID, versionID)
	if err != nil {
		err.Handle(r, w)
		return
	}
	httputil.ReturnSuccess(r, w, envs)
}

// PluginBuild PluginBuild
// swagger:operation POST /v2/tenants/{tenant_name}/envs/{tenant_env_name}/plugin/{plugin_id}/build v2 buildPlugin
//
// 构建plugin
//
// build plugin
//
// ---
// consumes:
// - application/json
// - application/x-protobuf
//
// produces:
// - application/json
// - application/xml
//
// responses:
//
//	default:
//	  schema:
//	    "$ref": "#/responses/commandResponse"
//	  description: 统一返回格式
func (t *TenantEnvStruct) PluginBuild(w http.ResponseWriter, r *http.Request) {
	var build api_model.BuildPluginStruct
	ok := httputil.ValidatorRequestStructAndErrorResponse(r, w, &build.Body, nil)
	if !ok {
		return
	}
	tenantEnvName := r.Context().Value(ctxutil.ContextKey("tenant_env_name")).(string)
	tenantEnvID := r.Context().Value(ctxutil.ContextKey("tenant_env_id")).(string)
	pluginID := r.Context().Value(ctxutil.ContextKey("plugin_id")).(string)
	build.TenantEnvName = tenantEnvName
	build.PluginID = pluginID
	build.Body.TenantEnvID = tenantEnvID
	pbv, err := handler.GetPluginManager().BuildPluginManual(&build)
	if err != nil {
		err.Handle(r, w)
		return
	}
	httputil.ReturnSuccess(r, w, pbv)
}

// SysPluginBuild SysPluginBuild
// swagger:operation POST /v2/sys-plugin/{plugin_id}/build v2 buildSysPlugin
//
// 构建 sys plugin
//
// build sys plugin
//
// ---
// consumes:
// - application/json
// - application/x-protobuf
//
// produces:
// - application/json
// - application/xml
//
// responses:
//
//	default:
//	  schema:
//	    "$ref": "#/responses/commandResponse"
//	  description: 统一返回格式
func (t *TenantEnvStruct) SysPluginBuild(w http.ResponseWriter, r *http.Request) {
	log.Println("debug 000")
	var build api_model.BuildSysPluginStruct
	ok := httputil.ValidatorRequestStructAndErrorResponse(r, w, &build.Body, nil)
	if !ok {
		log.Println("debug 111")
		return
	}
	log.Println("debug 222")
	pluginID := r.Context().Value(ctxutil.ContextKey("plugin_id")).(string)
	build.PluginID = pluginID
	pbv, err := handler.GetPluginManager().BuildSysPluginManual(&build)
	if err != nil {
		log.Println("debug 333")
		err.Handle(r, w)
		return
	}
	httputil.ReturnSuccess(r, w, pbv)
}

// GetAllPluginBuildVersions 获取该插件所有的构建版本
// swagger:operation GET /v2/tenants/{tenant_name}/envs/{tenant_env_name}/plugin/{plugin_id}/build-version v2 allPluginVersions
//
// 获取所有的构建版本信息
//
// all plugin versions
//
// ---
// consumes:
// - application/json
// - application/x-protobuf
//
// produces:
// - application/json
// - application/xml
//
// responses:
//
//	default:
//	  schema:
//	    "$ref": "#/responses/commandResponse"
//	  description: 统一返回格式
func (t *TenantEnvStruct) GetAllPluginBuildVersions(w http.ResponseWriter, r *http.Request) {
	pluginID := r.Context().Value(ctxutil.ContextKey("plugin_id")).(string)
	versions, err := handler.GetPluginManager().GetAllPluginBuildVersions(pluginID)
	if err != nil {
		err.Handle(r, w)
		return
	}
	httputil.ReturnSuccess(r, w, versions)
}

// GetPluginBuildVersion 获取某构建版本信息
// swagger:operation GET /v2/tenants/{tenant_name}/envs/{tenant_env_name}/plugin/{plugin_id}/build-version/{version_id} v2 pluginVersion
//
// 获取某个构建版本信息
//
// plugin version
//
// ---
// consumes:
// - application/json
// - application/x-protobuf
//
// produces:
// - application/json
// - application/xml
//
// responses:
//
//	default:
//	  schema:
//	    "$ref": "#/responses/commandResponse"
//	  description: 统一返回格式
func (t *TenantEnvStruct) GetPluginBuildVersion(w http.ResponseWriter, r *http.Request) {
	pluginID := r.Context().Value(ctxutil.ContextKey("plugin_id")).(string)
	versionID := chi.URLParam(r, "version_id")
	version, err := handler.GetPluginManager().GetPluginBuildVersion(pluginID, versionID)
	if err != nil {
		err.Handle(r, w)
		return
	}
	httputil.ReturnSuccess(r, w, version)
}

// DeletePluginBuildVersion DeletePluginBuildVersion
// swagger:operation DELETE /v2/tenants/{tenant_name}/envs/{tenant_env_name}/plugin/{plugin_id}/build-version/{version_id} v2 deletePluginVersion
//
// 删除某个构建版本信息
//
// delete plugin version
//
// ---
// consumes:
// - application/json
// - application/x-protobuf
//
// produces:
// - application/json
// - application/xml
//
// responses:
//
//	default:
//	  schema:
//	    "$ref": "#/responses/commandResponse"
//	  description: 统一返回格式
func (t *TenantEnvStruct) DeletePluginBuildVersion(w http.ResponseWriter, r *http.Request) {
	pluginID := r.Context().Value(ctxutil.ContextKey("plugin_id")).(string)
	versionID := chi.URLParam(r, "version_id")
	err := handler.GetPluginManager().DeletePluginBuildVersion(pluginID, versionID)
	if err != nil {
		err.Handle(r, w)
		return
	}
	httputil.ReturnSuccess(r, w, nil)
}

// PluginSet PluginSet
func (t *TenantEnvStruct) PluginSet(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "PUT":
		t.updatePluginSet(w, r)
	case "POST":
		t.addPluginSet(w, r)
	case "GET":
		t.getPluginSet(w, r)
	}
}

// swagger:operation PUT /v2/tenants/{tenant_name}/envs/{tenant_env_name}/services/{service_alias}/plugin v2 updatePluginSet
//
// 更新插件设定
//
// update plugin setting
//
// ---
// consumes:
// - application/json
// - application/x-protobuf
//
// produces:
// - application/json
// - application/xml
//
// responses:
//
//	default:
//	  schema:
//	    "$ref": "#/responses/commandResponse"
//	  description: 统一返回格式
func (t *TenantEnvStruct) updatePluginSet(w http.ResponseWriter, r *http.Request) {
	var pss api_model.PluginSetStruct
	ok := httputil.ValidatorRequestStructAndErrorResponse(r, w, &pss.Body, nil)
	if !ok {
		return
	}
	serviceID := r.Context().Value(ctxutil.ContextKey("service_id")).(string)
	relation, err := handler.GetServiceManager().UpdateTenantEnvServicePluginRelation(serviceID, &pss)
	if err != nil {
		err.Handle(r, w)
		return
	}
	httputil.ReturnSuccess(r, w, relation)
}

// swagger:operation POST /v2/tenants/{tenant_name}/envs/{tenant_env_name}/services/{service_alias}/plugin v2 addPluginSet
//
// 添加插件设定
//
// add plugin setting
//
// ---
// consumes:
// - application/json
// - application/x-protobuf
//
// produces:
// - application/json
// - application/xml
//
// responses:
//
//	default:
//	  schema:
//	    "$ref": "#/responses/commandResponse"
//	  description: 统一返回格式
func (t *TenantEnvStruct) addPluginSet(w http.ResponseWriter, r *http.Request) {
	var pss api_model.PluginSetStruct
	ok := httputil.ValidatorRequestStructAndErrorResponse(r, w, &pss.Body, nil)
	if !ok {
		return
	}
	serviceID := r.Context().Value(ctxutil.ContextKey("service_id")).(string)
	tenantEnvID := r.Context().Value(ctxutil.ContextKey("tenant_env_id")).(string)
	serviceAlias := r.Context().Value(ctxutil.ContextKey("service_alias")).(string)
	tenantEnvName := r.Context().Value(ctxutil.ContextKey("tenant_env_name")).(string)
	pss.ServiceAlias = serviceAlias
	pss.TenantEnvName = tenantEnvName
	re, err := handler.GetServiceManager().SetTenantEnvServicePluginRelation(tenantEnvID, serviceID, &pss)
	if err != nil {
		err.Handle(r, w)
		return
	}
	httputil.ReturnSuccess(r, w, re)
}

// swagger:operation GET /v2/tenants/{tenant_name}/envs/{tenant_env_name}/services/{service_alias}/plugin v2 getPluginSet
//
// 获取插件设定
//
// get plugin setting
//
// ---
// consumes:
// - application/json
// - application/x-protobuf
//
// produces:
// - application/json
// - application/xml
//
// responses:
//
//	default:
//	  schema:
//	    "$ref": "#/responses/commandResponse"
//	  description: 统一返回格式
func (t *TenantEnvStruct) getPluginSet(w http.ResponseWriter, r *http.Request) {
	serviceID := r.Context().Value(ctxutil.ContextKey("service_id")).(string)
	gps, err := handler.GetServiceManager().GetTenantEnvServicePluginRelation(serviceID)
	if err != nil {
		err.Handle(r, w)
		return
	}
	httputil.ReturnSuccess(r, w, gps)

}

// DeletePluginRelation DeletePluginRelation
// swagger:operation DELETE /v2/tenants/{tenant_name}/envs/{tenant_env_name}/services/{service_alias}/plugin/{plugin_id} v2 deletePluginRelation
//
// 删除插件依赖
//
// delete plugin relation
//
// ---
// consumes:
// - application/json
// - application/x-protobuf
//
// produces:
// - application/json
// - application/xml
//
// responses:
//
//	default:
//	  schema:
//	    "$ref": "#/responses/commandResponse"
//	  description: 统一返回格式
func (t *TenantEnvStruct) DeletePluginRelation(w http.ResponseWriter, r *http.Request) {
	pluginID := chi.URLParam(r, "plugin_id")
	serviceID := r.Context().Value(ctxutil.ContextKey("service_id")).(string)
	tenantEnvID := r.Context().Value(ctxutil.ContextKey("tenant_env_id")).(string)
	if err := handler.GetServiceManager().TenantEnvServiceDeletePluginRelation(tenantEnvID, serviceID, pluginID); err != nil {
		err.Handle(r, w)
		return
	}
	httputil.ReturnSuccess(r, w, nil)
}

// GePluginEnvWhichCanBeSet GePluginEnvWhichCanBeSet
// swagger:operation GET /v2/tenants/{tenant_name}/envs/{tenant_env_name}/services/{service_alias}/plugin/{plugin_id}/envs v2 getVersionEnvs
//
// 获取可配置的env; 从service plugin对应中取, 若不存在则返回默认可修改的变量
//
// get version env
//
// ---
// consumes:
// - application/json
// - application/x-protobuf
//
// produces:
// - application/json
// - application/xml
//
// responses:
//
//	default:
//	  schema:
//	    "$ref": "#/responses/commandResponse"
//	  description: 统一返回格式
func (t *TenantEnvStruct) GePluginEnvWhichCanBeSet(w http.ResponseWriter, r *http.Request) {
	serviceID := r.Context().Value(ctxutil.ContextKey("service_id")).(string)
	pluginID := chi.URLParam(r, "plugin_id")
	envs, err := handler.GetPluginManager().GetEnvsWhichCanBeSet(serviceID, pluginID)
	if err != nil {
		err.Handle(r, w)
		return
	}
	httputil.ReturnSuccess(r, w, envs)
}

// UpdateVersionEnv UpdateVersionEnv
// swagger:operation PUT /v2/tenants/{tenant_name}/envs/{tenant_env_name}/services/{service_alias}/plugin/{plugin_id}/upenv v2 updateVersionEnv
//
// modify the app plugin config info. it will Thermal effect
//
// update version env
//
// ---
// consumes:
// - application/json
// - application/x-protobuf
//
// produces:
// - application/json
// - application/xml
//
// responses:
//
//	default:
//	  schema:
//	    "$ref": "#/responses/commandResponse"
//	  description: 统一返回格式
func (t *TenantEnvStruct) UpdateVersionEnv(w http.ResponseWriter, r *http.Request) {
	var uve api_model.SetVersionEnv
	ok := httputil.ValidatorRequestStructAndErrorResponse(r, w, &uve.Body, nil)
	if !ok {
		return
	}
	serviceID := r.Context().Value(ctxutil.ContextKey("service_id")).(string)
	serviceAlias := r.Context().Value(ctxutil.ContextKey("service_alias")).(string)
	tenantEnvID := r.Context().Value(ctxutil.ContextKey("tenant_env_id")).(string)
	pluginID := chi.URLParam(r, "plugin_id")
	uve.PluginID = pluginID
	uve.Body.TenantEnvID = tenantEnvID
	uve.ServiceAlias = serviceAlias
	uve.Body.ServiceID = serviceID
	if err := handler.GetServiceManager().UpdateVersionEnv(&uve); err != nil {
		err.Handle(r, w)
		return
	}
	httputil.ReturnSuccess(r, w, nil)
}

// UpdateComponentPluginConfig 更新组件插件配置
// swagger:operation PUT /v2/tenants/{tenant_name}/envs/{tenant_env_name}/services/{service_alias}/plugin/{plugin_id}/config v2 UpdateComponentPluginConfig
//
// modify the app plugin config info. it will Thermal effect
//
// update component plugin config
//
// ---
// consumes:
// - application/json
// - application/x-protobuf
//
// produces:
// - application/json
// - application/xml
//
// responses:
//
//	default:
//	  schema:
//	    "$ref": "#/responses/commandResponse"
//	  description: 统一返回格式
func (t *TenantEnvStruct) UpdateComponentPluginConfig(w http.ResponseWriter, r *http.Request) {
	var req api_model.UpdateComponentPluginConfigRequest
	ok := httputil.ValidatorRequestStructAndErrorResponse(r, w, &req.Body, nil)
	if !ok {
		return
	}
	serviceID := r.Context().Value(ctxutil.ContextKey("service_id")).(string)
	serviceAlias := r.Context().Value(ctxutil.ContextKey("service_alias")).(string)
	tenantEnvID := r.Context().Value(ctxutil.ContextKey("tenant_env_id")).(string)
	req.Body.TenantEnvID = tenantEnvID
	req.ServiceAlias = serviceAlias
	req.Body.ServiceID = serviceID
	if err := handler.GetServiceManager().UpdateComponentPluginConfig(&req); err != nil {
		err.Handle(r, w)
		return
	}
	httputil.ReturnSuccess(r, w, nil)
}

// ToggleComponentPlugin 启用/停用组件插件
// swagger:operation PUT /v2/tenants/{tenant_name}/envs/{tenant_env_name}/services/{service_alias}/plugin/toggle v2 ToggleComponentPlugin
//
// 更新插件设定
//
// toggle component plugin
//
// ---
// consumes:
// - application/json
// - application/x-protobuf
//
// produces:
// - application/json
// - application/xml
//
// responses:
//
//	default:
//	  schema:
//	    "$ref": "#/responses/commandResponse"
//	  description: 统一返回格式
func (t *TenantEnvStruct) ToggleComponentPlugin(w http.ResponseWriter, r *http.Request) {
	var req api_model.ToggleComponentPluginRequest
	ok := httputil.ValidatorRequestStructAndErrorResponse(r, w, &req.Body, nil)
	if !ok {
		return
	}
	req.Body.ServiceID = r.Context().Value(ctxutil.ContextKey("service_id")).(string)
	err := handler.GetServiceManager().ToggleComponentPlugin(&req)
	if err != nil {
		err.Handle(r, w)
		return
	}
	httputil.ReturnSuccess(r, w, nil)
}

// SharePlugin share tenantEnvs plugin
func (t *TenantEnvStruct) SharePlugin(w http.ResponseWriter, r *http.Request) {
	var sp share.PluginShare
	ok := httputil.ValidatorRequestStructAndErrorResponse(r, w, &sp.Body, nil)
	if !ok {
		return
	}
	tenantEnvID := r.Context().Value(ctxutil.ContextKey("tenant_env_id")).(string)
	sp.TenantEnvID = tenantEnvID
	sp.PluginID = chi.URLParam(r, "plugin_id")
	if sp.Body.EventID == "" {
		sp.Body.EventID = util.NewUUID()
	}
	res, errS := handler.GetPluginShareHandle().Share(sp)
	if errS != nil {
		errS.Handle(r, w)
		return
	}
	httputil.ReturnSuccess(r, w, res)
}

// SharePluginResult SharePluginResult
func (t *TenantEnvStruct) SharePluginResult(w http.ResponseWriter, r *http.Request) {
	shareID := chi.URLParam(r, "share_id")
	res, errS := handler.GetPluginShareHandle().ShareResult(shareID)
	if errS != nil {
		errS.Handle(r, w)
		return
	}
	httputil.ReturnSuccess(r, w, res)
}

// BatchInstallPlugins -
func (t *TenantEnvStruct) BatchInstallPlugins(w http.ResponseWriter, r *http.Request) {
	tenantEnvID := r.Context().Value(ctxutil.ContextKey("tenant_env_id")).(string)
	var req api_model.BatchCreatePlugins
	if ok := httputil.ValidatorRequestStructAndErrorResponse(r, w, &req, nil); !ok {
		return
	}
	if err := handler.GetPluginManager().BatchCreatePlugins(tenantEnvID, req.Plugins); err != nil {
		err.Handle(r, w)
		return
	}
	httputil.ReturnSuccess(r, w, nil)
}

// BatchBuildPlugins -
func (t *TenantEnvStruct) BatchBuildPlugins(w http.ResponseWriter, r *http.Request) {
	var builds api_model.BatchBuildPlugins
	ok := httputil.ValidatorRequestStructAndErrorResponse(r, w, &builds, nil)
	if !ok {
		return
	}
	tenantEnvID := r.Context().Value(ctxutil.ContextKey("tenant_env_id")).(string)
	err := handler.GetPluginManager().BatchBuildPlugins(&builds, tenantEnvID)
	if err != nil {
		err.Handle(r, w)
		return
	}
	httputil.ReturnSuccess(r, w, nil)
}
