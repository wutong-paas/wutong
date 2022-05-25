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
	"github.com/wutong-paas/wutong/pkg/helm"
	httputil "github.com/wutong-paas/wutong/util/http"
)

// HelmAppsController is an implementation of HelmAppsInterface
type HelmAppsController struct{}

//ListHelmApps - get helm apps
func (c *HelmAppsController) ListHelmApps(w http.ResponseWriter, r *http.Request) {
	helmNamespace := chi.URLParam(r, "helm_namespace")
	if strings.TrimSpace(helmNamespace) == "" {
		httputil.ReturnError(r, w, 400, "helm_amespace is request")
		return
	}

	releases, err := helm.AllReleases(helmNamespace)
	if err != nil {
		logrus.Errorf("list helm apps failure %s", err.Error())
		httputil.ReturnError(r, w, http.StatusInternalServerError, fmt.Sprintf("list helm apps failure: %v", err))
		return
	}
	httputil.ReturnSuccess(r, w, releases)
}

//ListHelmAppResources - list helm resources
func (c *HelmAppsController) ListHelmAppResources(w http.ResponseWriter, r *http.Request) {
	helmNamespace := chi.URLParam(r, "helm_namespace")
	if strings.TrimSpace(helmNamespace) == "" {
		httputil.ReturnError(r, w, 400, "helm_amespace is request")
		return
	}
	helmName := chi.URLParam(r, "helm_name")
	if strings.TrimSpace(helmName) == "" {
		httputil.ReturnError(r, w, 400, "helm_name is request")
		return
	}

	resources, err := helm.AllResources(helmName, helmNamespace)
	if err != nil {
		logrus.Errorf("list helm resources failure %s", err.Error())
		httputil.ReturnError(r, w, http.StatusInternalServerError, fmt.Sprintf("list helm resources failure: %v", err))
		return
	}
	httputil.ReturnSuccess(r, w, resources)
}
