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

	"github.com/sirupsen/logrus"
	"github.com/wutong-paas/wutong/api/handler"
	"github.com/wutong-paas/wutong/api/model"
	ctxutil "github.com/wutong-paas/wutong/api/util/ctx"
	"github.com/wutong-paas/wutong/db"
	"github.com/wutong-paas/wutong/db/errors"
	dbmodel "github.com/wutong-paas/wutong/db/model"
	validation "github.com/wutong-paas/wutong/util/endpoint"
	httputil "github.com/wutong-paas/wutong/util/http"
)

// ThirdPartyServiceController implements ThirdPartyServicer
type ThirdPartyServiceController struct{}

// Endpoints POST->add endpoints, PUT->update endpoints, DELETE->delete endpoints
func (t *ThirdPartyServiceController) Endpoints(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		t.addEndpoints(w, r)
	case "PUT":
		t.updEndpoints(w, r)
	case "DELETE":
		t.delEndpoints(w, r)
	case "GET":
		t.listEndpoints(w, r)
	}
}

func (t *ThirdPartyServiceController) addEndpoints(w http.ResponseWriter, r *http.Request) {
	var data model.AddEndpiontsReq
	if !httputil.ValidatorRequestStructAndErrorResponse(r, w, &data, nil) {
		return
	}
	// if address is not ip, and then it is domain
	address := validation.SplitEndpointAddress(data.Address)
	tenantEnv := r.Context().Value(ctxutil.ContextKey("tenant_env")).(*dbmodel.TenantEnvs)
	sid := r.Context().Value(ctxutil.ContextKey("service_id")).(string)

	// 服务实例为域名类型
	if !canAddDomainEndpoint(sid, validation.IsDomainNotIP(address)) {
		logrus.Warningf("new endpoint addres[%s] is domian", address)
		httputil.ReturnError(r, w, 400, "服务实例不允许同时存在域名和IP类型的端点，并且只允许一个域名类型的端点")
		return
	}

	if err := handler.Get3rdPartySvcHandler().AddEndpoints(tenantEnv, sid, &data); err != nil {
		if err == errors.ErrRecordAlreadyExist {
			httputil.ReturnError(r, w, 400, err.Error())
			return
		}
		httputil.ReturnError(r, w, 500, err.Error())
		return
	}
	httputil.ReturnSuccess(r, w, "success")
}

// canAddDomainEndpoint 检测是否可以添加域名类型的端点
// 1. 服务实例不允许同时存在域名和IP类型的端点
// 2. 服务实例只允许一个域名类型的端点
func canAddDomainEndpoint(sid string, isDomain bool) bool {
	endpoints, err := db.GetManager().EndpointsDao().List(sid)
	if err != nil {
		logrus.Errorf("find endpoints by sid[%s], error: %s", sid, err.Error())
		return false
	}

	if len(endpoints) > 0 && isDomain {
		// 已经存在服务实例，新添加了域名类型的端点
		return false
	}
	if !isDomain {
		for _, ep := range endpoints {
			address := validation.SplitEndpointAddress(ep.IP)
			if validation.IsDomainNotIP(address) {
				// 已经有一个域名类型的端点，不允许添加新的IP类型的端点
				return false
			}
		}
	}
	return true
}

func (t *ThirdPartyServiceController) updEndpoints(w http.ResponseWriter, r *http.Request) {
	var data model.UpdEndpiontsReq
	if !httputil.ValidatorRequestStructAndErrorResponse(r, w, &data, nil) {
		return
	}

	tenantEnv := r.Context().Value(ctxutil.ContextKey("tenant_env")).(*dbmodel.TenantEnvs)
	sid := r.Context().Value(ctxutil.ContextKey("service_id")).(string)
	if err := handler.Get3rdPartySvcHandler().UpdEndpoints(tenantEnv, sid, &data); err != nil {
		httputil.ReturnError(r, w, 500, err.Error())
		return
	}
	httputil.ReturnSuccess(r, w, "success")
}

func (t *ThirdPartyServiceController) delEndpoints(w http.ResponseWriter, r *http.Request) {
	var data model.DelEndpiontsReq
	if !httputil.ValidatorRequestStructAndErrorResponse(r, w, &data, nil) {
		return
	}
	tenantEnv := r.Context().Value(ctxutil.ContextKey("tenant_env")).(*dbmodel.TenantEnvs)
	sid := r.Context().Value(ctxutil.ContextKey("service_id")).(string)
	if err := handler.Get3rdPartySvcHandler().DelEndpoints(tenantEnv, data.EpID, sid); err != nil {
		httputil.ReturnError(r, w, 500, err.Error())
		return
	}
	httputil.ReturnSuccess(r, w, "success")
}

func (t *ThirdPartyServiceController) listEndpoints(w http.ResponseWriter, r *http.Request) {
	sid := r.Context().Value(ctxutil.ContextKey("service_id")).(string)
	res, err := handler.Get3rdPartySvcHandler().ListEndpoints(sid)
	if err != nil {
		httputil.ReturnError(r, w, 500, err.Error())
		return
	}
	if len(res) == 0 {
		httputil.ReturnSuccess(r, w, []*model.ThirdEndpoint{})
		return
	}
	httputil.ReturnSuccess(r, w, res)
}
