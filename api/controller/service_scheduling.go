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

package controller

import (
	"net/http"

	"github.com/sirupsen/logrus"
	"github.com/wutong-paas/wutong/api/handler"
	"github.com/wutong-paas/wutong/api/model"
	ctxutil "github.com/wutong-paas/wutong/api/util/ctx"
	httputil "github.com/wutong-paas/wutong/util/http"
)

func (t *TenantEnvStruct) GetServiceSchedulingDetails(w http.ResponseWriter, r *http.Request) {
	serviceID := r.Context().Value(ctxutil.ContextKey("service_id")).(string)
	result, err := handler.GetServiceManager().GetServiceSchedulingDetails(serviceID)
	if err != nil {
		httputil.ReturnError(r, w, 500, err.Error())
		return
	}
	httputil.ReturnSuccess(r, w, result)
}

func (t *TenantEnvStruct) AddServiceSchedulingLabel(w http.ResponseWriter, r *http.Request) {
	serviceID := r.Context().Value(ctxutil.ContextKey("service_id")).(string)
	var req model.AddServiceSchedulingLabelRequest
	ok := httputil.ValidatorRequestStructAndErrorResponse(r, w, &req, nil)
	if !ok {
		logrus.Errorf("start operation validate request body failure")
		return
	}

	err := handler.GetServiceManager().AddServiceSchedulingLabel(serviceID, &req)
	if err != nil {
		httputil.ReturnError(r, w, 500, err.Error())
		return
	}
	httputil.ReturnSuccess(r, w, nil)
}

func (t *TenantEnvStruct) UpdateServiceSchedulingLabel(w http.ResponseWriter, r *http.Request) {
	serviceID := r.Context().Value(ctxutil.ContextKey("service_id")).(string)
	var req model.UpdateServiceSchedulingLabelRequest
	ok := httputil.ValidatorRequestStructAndErrorResponse(r, w, &req, nil)
	if !ok {
		logrus.Errorf("start operation validate request body failure")
		return
	}

	err := handler.GetServiceManager().UpdateServiceSchedulingLabel(serviceID, &req)
	if err != nil {
		httputil.ReturnError(r, w, 500, err.Error())
		return
	}
	httputil.ReturnSuccess(r, w, nil)
}

func (t *TenantEnvStruct) DeleteServiceSchedulingLabel(w http.ResponseWriter, r *http.Request) {
	serviceID := r.Context().Value(ctxutil.ContextKey("service_id")).(string)
	var req model.DeleteServiceSchedulingLabelRequest
	ok := httputil.ValidatorRequestStructAndErrorResponse(r, w, &req, nil)
	if !ok {
		logrus.Errorf("start operation validate request body failure")
		return
	}

	err := handler.GetServiceManager().DeleteServiceSchedulingLabel(serviceID, &req)
	if err != nil {
		httputil.ReturnError(r, w, 500, err.Error())
		return
	}
	httputil.ReturnSuccess(r, w, nil)
}

func (t *TenantEnvStruct) SetServiceSchedulingNode(w http.ResponseWriter, r *http.Request) {
	serviceID := r.Context().Value(ctxutil.ContextKey("service_id")).(string)
	var req model.SetServiceSchedulingNodeRequest
	ok := httputil.ValidatorRequestStructAndErrorResponse(r, w, &req, nil)
	if !ok {
		logrus.Errorf("start operation validate request body failure")
		return
	}

	err := handler.GetServiceManager().SetServiceSchedulingNode(serviceID, &req)
	if err != nil {
		httputil.ReturnError(r, w, 500, err.Error())
		return
	}
	httputil.ReturnSuccess(r, w, nil)
}

func (t *TenantEnvStruct) AddServiceSchedulingToleration(w http.ResponseWriter, r *http.Request) {
	serviceID := r.Context().Value(ctxutil.ContextKey("service_id")).(string)
	var req model.AddServiceSchedulingTolerationRequest
	ok := httputil.ValidatorRequestStructAndErrorResponse(r, w, &req, nil)
	if !ok {
		logrus.Errorf("start operation validate request body failure")
		return
	}

	err := handler.GetServiceManager().AddServiceSchedulingToleration(serviceID, &req)
	if err != nil {
		httputil.ReturnError(r, w, 500, err.Error())
		return
	}
	httputil.ReturnSuccess(r, w, nil)
}

func (t *TenantEnvStruct) UpdateServiceSchedulingToleration(w http.ResponseWriter, r *http.Request) {
	serviceID := r.Context().Value(ctxutil.ContextKey("service_id")).(string)
	var req model.UpdateServiceSchedulingTolerationRequest
	ok := httputil.ValidatorRequestStructAndErrorResponse(r, w, &req, nil)
	if !ok {
		logrus.Errorf("start operation validate request body failure")
		return
	}

	err := handler.GetServiceManager().UpdateServiceSchedulingToleration(serviceID, &req)
	if err != nil {
		httputil.ReturnError(r, w, 500, err.Error())
		return
	}
	httputil.ReturnSuccess(r, w, nil)
}

func (t *TenantEnvStruct) DeleteServiceSchedulingToleration(w http.ResponseWriter, r *http.Request) {
	serviceID := r.Context().Value(ctxutil.ContextKey("service_id")).(string)
	var req model.DeleteServiceSchedulingTolerationRequest
	ok := httputil.ValidatorRequestStructAndErrorResponse(r, w, &req, nil)
	if !ok {
		logrus.Errorf("start operation validate request body failure")
		return
	}

	err := handler.GetServiceManager().DeleteServiceSchedulingToleration(serviceID, &req)
	if err != nil {
		httputil.ReturnError(r, w, 500, err.Error())
		return
	}
	httputil.ReturnSuccess(r, w, nil)
}
