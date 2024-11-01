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
	"strings"

	"github.com/go-chi/chi"
	"github.com/wutong-paas/wutong/api/handler"
	api_model "github.com/wutong-paas/wutong/api/model"
	ctxutil "github.com/wutong-paas/wutong/api/util/ctx"
	httputil "github.com/wutong-paas/wutong/util/http"
)

// Check service check
// swagger:operation POST /v2/tenants/{tenant_name}/envs/{tenant_env_name}/servicecheck v2 serviceCheck
//
// 应用构建源检测，支持docker run, source code
//
// service check
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
func Check(w http.ResponseWriter, r *http.Request) {
	var gt api_model.ServiceCheckStruct
	if ok := httputil.ValidatorRequestStructAndErrorResponse(r, w, &gt.Body, nil); !ok {
		return
	}
	tenantEnvID := r.Context().Value(ctxutil.ContextKey("tenant_env_id")).(string)
	gt.Body.TenantEnvID = tenantEnvID
	result, eventID, err := handler.GetServiceManager().ServiceCheck(&gt)
	if err != nil {
		err.Handle(r, w)
		return
	}
	re := struct {
		CheckUUID string `json:"check_uuid"`
		EventID   string `json:"event_id"`
	}{
		CheckUUID: result,
		EventID:   eventID,
	}
	httputil.ReturnSuccess(r, w, re)
}

// GetServiceCheckInfo get service check info
// swagger:operation GET /v2/tenants/{tenant_name}/envs/{tenant_env_name}/servicecheck/{uuid} v2 getServiceCheckInfo
//
//	获取构建检测信息
//
// get service check info
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
func GetServiceCheckInfo(w http.ResponseWriter, r *http.Request) {
	uuid := strings.TrimSpace(chi.URLParam(r, "uuid"))
	si, err := handler.GetServiceManager().GetServiceCheckInfo(uuid)
	if err != nil {
		err.Handle(r, w)
		return
	}
	httputil.ReturnSuccess(r, w, si)
}
