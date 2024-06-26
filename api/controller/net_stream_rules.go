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
	"net/http"

	"github.com/wutong-paas/wutong/api/handler"
	api_model "github.com/wutong-paas/wutong/api/model"
	ctxutil "github.com/wutong-paas/wutong/api/util/ctx"
	httputil "github.com/wutong-paas/wutong/util/http"
)

// SetDownStreamRule 设置下游规则
// swagger:operation POST /v2/tenants/{tenant_name}/envs/{tenant_env_name}/services/{service_alias}/net-rule/downstream v2 setNetDownStreamRuleStruct
//
// 设置下游网络规则
//
// set NetDownStreamRuleStruct
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
func (t *TenantEnvStruct) SetDownStreamRule(w http.ResponseWriter, r *http.Request) {
	var rs api_model.SetNetDownStreamRuleStruct
	ok := httputil.ValidatorRequestStructAndErrorResponse(r, w, &rs.Body, nil)
	if !ok {
		return
	}
	serviceID := r.Context().Value(ctxutil.ContextKey("service_id")).(string)
	tenantEnvName := r.Context().Value(ctxutil.ContextKey("tenant_env_name")).(string)
	serviceAlias := r.Context().Value(ctxutil.ContextKey("service_alias")).(string)
	tenantEnvID := r.Context().Value(ctxutil.ContextKey("tenant_env_id")).(string)
	rs.TenantEnvName = tenantEnvName
	rs.ServiceAlias = serviceAlias
	rs.Body.Rules.ServiceID = serviceID
	if err := handler.GetRulesManager().CreateDownStreamNetRules(tenantEnvID, &rs); err != nil {
		err.Handle(r, w)
		return
	}
	httputil.ReturnSuccess(r, w, nil)
}

// GetDownStreamRule 获取下游规则
// swagger:operation GET /v2/tenants/{tenant_name}/envs/{tenant_env_name}/services/{service_alias}/net-rule/downstream/{dest_service_alias}/{port} v2 getNetDownStreamRuleStruct
//
// 获取下游网络规则
//
// set NetDownStreamRuleStruct
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
func (t *TenantEnvStruct) GetDownStreamRule(w http.ResponseWriter, r *http.Request) {
	serviceAlias := r.Context().Value(ctxutil.ContextKey("service_alias")).(string)
	destServiceAlias := r.Context().Value(ctxutil.ContextKey("dest_service_alias")).(string)
	tenantEnvID := r.Context().Value(ctxutil.ContextKey("tenant_env_id")).(string)
	port := r.Context().Value(ctxutil.ContextKey("port")).(string)

	nrs, err := handler.GetRulesManager().GetDownStreamNetRule(
		tenantEnvID,
		serviceAlias,
		destServiceAlias,
		port)
	if err != nil {
		err.Handle(r, w)
		return
	}
	httputil.ReturnSuccess(r, w, nrs)
}

// DeleteDownStreamRule 删除下游规则
func (t *TenantEnvStruct) DeleteDownStreamRule(w http.ResponseWriter, r *http.Request) {}

// UpdateDownStreamRule 更新下游规则
// swagger:operation PUT /v2/tenants/{tenant_name}/envs/{tenant_env_name}/services/{service_alias}/net-rule/downstream/{dest_service_alias}/{port} v2 updateNetDownStreamRuleStruct
//
// 更新下游网络规则
//
// update NetDownStreamRuleStruct
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
func (t *TenantEnvStruct) UpdateDownStreamRule(w http.ResponseWriter, r *http.Request) {
	var urs api_model.UpdateNetDownStreamRuleStruct
	ok := httputil.ValidatorRequestStructAndErrorResponse(r, w, &urs.Body, nil)
	if !ok {
		return
	}
	serviceID := r.Context().Value(ctxutil.ContextKey("service_id")).(string)
	tenantEnvName := r.Context().Value(ctxutil.ContextKey("tenant_env_name")).(string)
	serviceAlias := r.Context().Value(ctxutil.ContextKey("service_alias")).(string)
	tenantEnvID := r.Context().Value(ctxutil.ContextKey("tenant_env_id")).(string)
	destServiceAlias := r.Context().Value(ctxutil.ContextKey("dest_service_alias")).(string)
	port := r.Context().Value(ctxutil.ContextKey("tenant_env_id")).(int)

	urs.DestServiceAlias = destServiceAlias
	urs.Port = port
	urs.ServiceAlias = serviceAlias
	urs.TenantEnvName = tenantEnvName
	urs.Body.Rules.ServiceID = serviceID

	if err := handler.GetRulesManager().UpdateDownStreamNetRule(tenantEnvID, &urs); err != nil {
		err.Handle(r, w)
		return
	}
	httputil.ReturnSuccess(r, w, nil)
}
