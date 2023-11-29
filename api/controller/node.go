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

	httputil "github.com/wutong-paas/wutong/util/http"
)

// NodeController -
type NodeController struct {
}

// GetClusterInfo -
func (t *ClusterController) ListVMNodeSelectorLabels(w http.ResponseWriter, r *http.Request) {
	labels, err := handler.GetNodeHandler().ListVMNodeSelectorLabels()
	if err != nil {
		logrus.Errorf("get vm node selector labels: %v", err)
		httputil.ReturnError(r, w, 500, err.Error())
		return
	}

	httputil.ReturnSuccess(r, w, labels)
}
