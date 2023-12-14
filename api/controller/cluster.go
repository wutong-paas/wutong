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

	"github.com/go-chi/chi"
	"github.com/sirupsen/logrus"
	"github.com/wutong-paas/wutong/api/handler"

	httputil "github.com/wutong-paas/wutong/util/http"
)

// ClusterController -
type ClusterController struct {
}

// GetClusterInfo -
func (t *ClusterController) GetClusterInfo(w http.ResponseWriter, r *http.Request) {
	nodes, err := handler.GetClusterHandler().GetClusterInfo(r.Context())
	if err != nil {
		logrus.Errorf("get cluster info: %v", err)
		httputil.ReturnError(r, w, 500, err.Error())
		return
	}

	httputil.ReturnSuccess(r, w, nodes)
}

// GetClusterEvents
func (t *ClusterController) GetClusterEvents(w http.ResponseWriter, r *http.Request) {
	events, err := handler.GetClusterHandler().GetClusterEvents(r.Context())
	if err != nil {
		logrus.Errorf("get cluster events: %v", err)
		httputil.ReturnError(r, w, 500, err.Error())
		return
	}

	httputil.ReturnSuccess(r, w, events)
}

// MavenSettingList maven setting list
func (t *ClusterController) MavenSettingList(w http.ResponseWriter, r *http.Request) {
	httputil.ReturnSuccess(r, w, handler.GetClusterHandler().MavenSettingList(r.Context()))
}

// MavenSettingAdd maven setting add
func (t *ClusterController) MavenSettingAdd(w http.ResponseWriter, r *http.Request) {
	var set handler.MavenSetting
	if ok := httputil.ValidatorRequestStructAndErrorResponse(r, w, &set, nil); !ok {
		return
	}
	if err := handler.GetClusterHandler().MavenSettingAdd(r.Context(), &set); err != nil {
		err.Handle(r, w)
		return
	}
	httputil.ReturnSuccess(r, w, &set)
}

// MavenSettingUpdate maven setting file update
func (t *ClusterController) MavenSettingUpdate(w http.ResponseWriter, r *http.Request) {
	type SettingUpdate struct {
		Content string `json:"content" validate:"required"`
	}
	var su SettingUpdate
	if ok := httputil.ValidatorRequestStructAndErrorResponse(r, w, &su, nil); !ok {
		return
	}
	set := &handler.MavenSetting{
		Name:    chi.URLParam(r, "name"),
		Content: su.Content,
	}
	if err := handler.GetClusterHandler().MavenSettingUpdate(r.Context(), set); err != nil {
		err.Handle(r, w)
		return
	}
	httputil.ReturnSuccess(r, w, set)
}

// MavenSettingDelete maven setting file delete
func (t *ClusterController) MavenSettingDelete(w http.ResponseWriter, r *http.Request) {
	err := handler.GetClusterHandler().MavenSettingDelete(r.Context(), chi.URLParam(r, "name"))
	if err != nil {
		err.Handle(r, w)
		return
	}
	httputil.ReturnSuccess(r, w, nil)
}

// MavenSettingDetail maven setting file delete
func (t *ClusterController) MavenSettingDetail(w http.ResponseWriter, r *http.Request) {
	c, err := handler.GetClusterHandler().MavenSettingDetail(r.Context(), chi.URLParam(r, "name"))
	if err != nil {
		err.Handle(r, w)
		return
	}
	httputil.ReturnSuccess(r, w, c)
}

// Features
func (t *ClusterController) Features(w http.ResponseWriter, r *http.Request) {
	httputil.ReturnSuccess(r, w, handler.GetClusterHandler().Features(r.Context()))
}
