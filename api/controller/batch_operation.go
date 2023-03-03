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

package controller

import (
	"net/http"

	"github.com/sirupsen/logrus"
	"github.com/wutong-paas/wutong/api/model"
	httputil "github.com/wutong-paas/wutong/util/http"
)

// BatchOperation batch operation for tenant env
// support operation is : start,build,stop,update
func BatchOperation(w http.ResponseWriter, r *http.Request) {
	var build model.BatchOperationReq
	ok := httputil.ValidatorRequestStructAndErrorResponse(r, w, &build.Body, nil)
	if !ok {
		logrus.Errorf("start batch operation validate request body failure")
		return
	}

	// tenantEnv := r.Context().Value(ctxutil.ContextKey("tenant_env")).(*dbmodel.TenantEnvs)

	// var batchOpReqs []model.ComponentOpReq
	// var f func(ctx context.Context, tenant env *dbmodel.TenantEnvs, operator string, batchOpReqs model.BatchOpRequesters) (model.BatchOpResult, error)
	// switch build.Body.Operation {
	// case "build":
	// 	for _, build := range build.Body.Builds {
	// 		build.TenantEnvName = tenantEnv.Name
	// 		batchOpReqs = append(batchOpReqs, build)
	// 	}
	// 	f = handler.GetBatchOperationHandler().Build
	// case "start":
	// 	for _, start := range build.Body.Starts {
	// 		batchOpReqs = append(batchOpReqs, start)
	// 	}
	// 	f = handler.GetBatchOperationHandler().Start
	// case "stop":
	// 	for _, stop := range build.Body.Stops {
	// 		batchOpReqs = append(batchOpReqs, stop)
	// 	}
	// 	f = handler.GetBatchOperationHandler().Stop
	// case "upgrade":
	// 	for _, upgrade := range build.Body.Upgrades {
	// 		batchOpReqs = append(batchOpReqs, upgrade)
	// 	}
	// 	f = handler.GetBatchOperationHandler().Upgrade
	// default:
	// 	httputil.ReturnError(r, w, 400, fmt.Sprintf("operation %s do not support batch", build.Body.Operation))
	// 	return
	// }
	// if len(batchOpReqs) > 1024 {
	// 	batchOpReqs = batchOpReqs[0:1024]
	// }
	// res, err := f(r.Context(), tenantEnv, build.Operator, batchOpReqs)
	// if err != nil {
	// 	httputil.ReturnBcodeError(r, w, err)
	// 	return
	// }

	// // append every create event result to re and then return
	// httputil.ReturnSuccess(r, w, map[string]interface{}{
	// 	"batch_result": res,
	// })
}
