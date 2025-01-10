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

	"github.com/go-chi/chi"
	"github.com/wutong-paas/wutong/api/handler"
	"github.com/wutong-paas/wutong/api/model"
	"github.com/wutong-paas/wutong/cmd/api/option"
	httputil "github.com/wutong-paas/wutong/util/http"
)

// AppStoreVersionStruct -
type AppStoreVersionStruct struct {
	OptCfg *option.Config
}

func (c *AppStoreVersionStruct) ExportAppStoreVersionStatus(w http.ResponseWriter, r *http.Request) {
	versionId := chi.URLParam(r, "versionID")
	if len(versionId) == 0 {
		httputil.ReturnError(r, w, 400, "versionId is required")
		return
	}

	_, status := handler.GetAppStoreVersionHandler().ExportStatus(versionId)
	httputil.ReturnSuccess(r, w, map[string]string{"status": status})
}

func (c *AppStoreVersionStruct) ExportAppStoreVersion(w http.ResponseWriter, r *http.Request) {
	versionId := chi.URLParam(r, "versionID")
	if len(versionId) == 0 {
		httputil.ReturnError(r, w, 400, "versionId is required")
		return
	}

	var req model.AppStoreVersionExportImageInfo
	if ok := httputil.ValidatorRequestStructAndErrorResponse(r, w, &req, nil); !ok {
		return
	}

	if err := handler.GetAppStoreVersionHandler().Export(versionId, &req); err != nil {
		httputil.ReturnError(r, w, 500, err.Error())
		return
	}

	httputil.ReturnSuccess(r, w, nil)
}

func (c *AppStoreVersionStruct) DownloadAppStoreVersion(w http.ResponseWriter, r *http.Request) {
	versionId := chi.URLParam(r, "versionID")
	if len(versionId) == 0 {
		httputil.ReturnError(r, w, 400, "versionId is required")
		return
	}

	file, err := handler.GetAppStoreVersionHandler().Download(versionId)
	if err != nil {
		httputil.ReturnError(r, w, 500, err.Error())
		return
	}

	if len(file) > 0 {
		http.ServeFile(w, r, file)
	} else {
		httputil.ReturnError(r, w, 500, "导出文件不存在")
	}
}
