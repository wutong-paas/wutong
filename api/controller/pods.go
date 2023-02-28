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
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi"
	"github.com/sirupsen/logrus"
	"github.com/wutong-paas/wutong/api/handler"
	ctxutil "github.com/wutong-paas/wutong/api/util/ctx"
	"github.com/wutong-paas/wutong/db"
	"github.com/wutong-paas/wutong/db/model"
	httputil "github.com/wutong-paas/wutong/util/http"
	"github.com/wutong-paas/wutong/worker/server"
)

// PodController is an implementation of PodInterface
type PodController struct{}

// Pods get some service pods
// swagger:operation GET /v2/tenants/{tenant_name}/envs/{tenant_env_name}/pods v2/tenants/{tenant_name}/envs pods
//
// 获取一些应用的Pod信息
//
// get some service pods
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
//	  description: get some service pods
func Pods(w http.ResponseWriter, r *http.Request) {
	serviceIDs := strings.Split(r.FormValue("service_ids"), ",")
	if len(serviceIDs) == 0 {
		tenantEnv := r.Context().Value(ctxutil.ContextKey("tenantEnv")).(*model.TenantEnvs)
		services, _ := db.GetManager().TenantEnvServiceDao().GetServicesByTenantEnvID(tenantEnv.UUID)
		for _, s := range services {
			serviceIDs = append(serviceIDs, s.ServiceID)
		}
	}
	var allpods []*handler.K8sPodInfo
	podinfo, err := handler.GetServiceManager().GetMultiServicePods(serviceIDs)
	if err != nil {
		logrus.Errorf("get service pod failure %s", err.Error())
	}
	if podinfo != nil {
		var pods []*handler.K8sPodInfo
		if podinfo.OldPods != nil {
			pods = append(podinfo.NewPods, podinfo.OldPods...)
		} else {
			pods = podinfo.NewPods
		}
		allpods = append(allpods, pods...)
	}
	httputil.ReturnSuccess(r, w, allpods)
}

// PodNums reutrns the number of pods for components.
func PodNums(w http.ResponseWriter, r *http.Request) {
	componentIDs := strings.Split(r.FormValue("service_ids"), ",")
	podNums, err := handler.GetServiceManager().GetComponentPodNums(r.Context(), componentIDs)
	if err != nil {
		httputil.ReturnBcodeError(r, w, err)
		return
	}
	httputil.ReturnSuccess(r, w, podNums)
}

// PodDetail -
func (p *PodController) PodDetail(w http.ResponseWriter, r *http.Request) {
	podName := chi.URLParam(r, "pod_name")
	tenantEnv := r.Context().Value(ctxutil.ContextKey("tenantEnv")).(*model.TenantEnvs)
	pd, err := handler.GetPodHandler().PodDetail(tenantEnv.Namespace, podName)
	if err != nil {
		logrus.Errorf("error getting pod detail: %v", err)
		if err == server.ErrPodNotFound {
			httputil.ReturnError(r, w, 404, fmt.Sprintf("error getting pod detail: %v", err))
			return
		}
		httputil.ReturnError(r, w, 500, fmt.Sprintf("error getting pod detail: %v", err))
		return
	}
	httputil.ReturnSuccess(r, w, pd)
}
