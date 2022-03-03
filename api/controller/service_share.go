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

	"github.com/go-chi/chi"
	"github.com/wutong-paas/wutong/api/handler"
	api_model "github.com/wutong-paas/wutong/api/model"
	ctxutil "github.com/wutong-paas/wutong/api/util/ctx"
	httputil "github.com/wutong-paas/wutong/util/http"
)

//Share 应用分享
func (t *TenantStruct) Share(w http.ResponseWriter, r *http.Request) {
	//Share ShareService
	// swagger:operation POST /v2/tenants/{tenant_name}/services/{service_alias}/share  v2 shareService
	//
	// 分享应用介质
	//
	// share service
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
	var ccs api_model.ServiceShare
	ok := httputil.ValidatorRequestStructAndErrorResponse(r, w, &ccs.Body, nil)
	if !ok {
		return
	}
	serviceID := r.Context().Value(ctxutil.ContextKey("service_id")).(string)
	ccs.Body.EventID = r.Context().Value(ctxutil.ContextKey("event_id")).(string)
	res, errS := handler.GetShareHandle().Share(serviceID, ccs)
	if errS != nil {
		errS.Handle(r, w)
		return
	}
	httputil.ReturnSuccess(r, w, res)
}

//ShareResult 获取分享结果
func (t *TenantStruct) ShareResult(w http.ResponseWriter, r *http.Request) {
	//ShareResult ShareResult
	// swagger:operation GET /v2/tenants/{tenant_name}/services/{service_alias}/share  v2 get_share_result
	//
	// 获取分享应用介质结果
	//
	// share service
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
	shareID := chi.URLParam(r, "share_id")
	res, errS := handler.GetShareHandle().ShareResult(shareID)
	if errS != nil {
		errS.Handle(r, w)
		return
	}
	httputil.ReturnSuccess(r, w, res)
}
