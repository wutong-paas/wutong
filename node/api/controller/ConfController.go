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

	"github.com/wutong-paas/wutong/node/api/model"

	"github.com/sirupsen/logrus"
	httputil "github.com/wutong-paas/wutong/util/http"
)

//GetDatacenterConfig 获取数据中心配置
func GetDatacenterConfig(w http.ResponseWriter, r *http.Request) {
	c, err := datacenterConfig.GetDataCenterConfig()
	if err != nil {
		httputil.ReturnError(r, w, 500, err.Error())
		return
	}
	logrus.Infof("task details is %v", c)
	httputil.ReturnSuccess(r, w, c)
}

//PutDatacenterConfig 更新数据中心配置
func PutDatacenterConfig(w http.ResponseWriter, r *http.Request) {
	var gconfig model.GlobalConfig
	if ok := httputil.ValidatorRequestStructAndErrorResponse(r, w, &gconfig, nil); !ok {
		return
	}
	if err := datacenterConfig.PutDataCenterConfig(&gconfig); err != nil {
		httputil.ReturnError(r, w, 500, err.Error())
		return
	}
	httputil.ReturnSuccess(r, w, &gconfig)
}
