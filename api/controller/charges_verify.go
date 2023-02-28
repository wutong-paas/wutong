// WUTONG, Application Management Platform
// Copyright (C) 2014-2017 Wutong Co., Ltd.

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
	"os"
	"strconv"

	"github.com/wutong-paas/wutong/api/handler/cloud"

	"github.com/wutong-paas/wutong/db"
	"github.com/wutong-paas/wutong/db/model"

	ctxutil "github.com/wutong-paas/wutong/api/util/ctx"
	httputil "github.com/wutong-paas/wutong/util/http"
)

// ChargesVerifyController service charges verify
// swagger:operation GET /v2/tenants/{tenant_name}/envs/{tenant_env_name}/chargesverify v2 chargesverify
//
// 应用扩大资源申请接口，公有云云市验证，私有云不验证
//
// service charges verify
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
//	  description: 状态码非200，表示验证过程发生错误。状态码200，msg代表实际状态：success, illegal_quantity, missing_tenantEnv, owned_fee, region_unauthorized, lack_of_memory
func ChargesVerifyController(w http.ResponseWriter, r *http.Request) {
	tenantEnv := r.Context().Value(ctxutil.ContextKey("tenantEnv")).(*model.TenantEnvs)
	if tenantEnv.EID == "" {
		eid := r.FormValue("eid")
		if eid == "" {
			httputil.ReturnError(r, w, 400, "enterprise id can not found")
			return
		}
		tenantEnv.EID = eid
		db.GetManager().TenantEnvDao().UpdateModel(tenantEnv)
	}
	quantity := r.FormValue("quantity")
	if quantity == "" {
		httputil.ReturnError(r, w, 400, "quantity  can not found")
		return
	}
	quantityInt, err := strconv.Atoi(quantity)
	if err != nil {
		httputil.ReturnError(r, w, 400, "quantity type must be int")
		return
	}

	if publicCloud := os.Getenv("PUBLIC_CLOUD"); publicCloud != "true" {
		err := cloud.PriChargeSverify(r.Context(), tenantEnv, quantityInt)
		if err != nil {
			err.Handle(r, w)
			return
		}
		httputil.ReturnSuccess(r, w, nil)
	} else {
		reason := r.FormValue("reason")
		if err := cloud.PubChargeSverify(tenantEnv, quantityInt, reason); err != nil {
			err.Handle(r, w)
			return
		}
		httputil.ReturnSuccess(r, w, nil)
	}
}
