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
	"errors"
	"io"
	"net/http"
	"os/exec"
	"strings"

	"github.com/bitly/go-simplejson"
	"github.com/go-chi/chi"
	"github.com/sirupsen/logrus"
	"github.com/wutong-paas/wutong/chaos"
	"github.com/wutong-paas/wutong/db"
	httputil "github.com/wutong-paas/wutong/util/http"
)

func GetVersionByEventID(w http.ResponseWriter, r *http.Request) {
	eventID := strings.TrimSpace(chi.URLParam(r, "eventID"))

	version, err := db.GetManager().VersionInfoDao().GetVersionByEventID(eventID)
	if err != nil {
		httputil.ReturnError(r, w, 404, err.Error())
	}
	httputil.ReturnSuccess(r, w, version)
}

func UpdateVersionByEventID(w http.ResponseWriter, r *http.Request) {
	eventID := strings.TrimSpace(chi.URLParam(r, "eventID"))

	version, err := db.GetManager().VersionInfoDao().GetVersionByEventID(eventID)
	if err != nil {
		httputil.ReturnError(r, w, 404, err.Error())
		return
	}
	in, _ := io.ReadAll(r.Body)
	json, err := simplejson.NewJson(in)
	if err != nil {
		httputil.ReturnError(r, w, 400, err.Error())
		return
	}

	if author, err := json.Get("code_commit_author").String(); err != nil || author == "" {
		logrus.Debugf("error get code_commit_author from version body ,details %s", err.Error())
	} else {
		version.Author = author
	}

	if msg, err := json.Get("code_commit_msg").String(); err != nil || msg == "" {
		logrus.Debugf("error get code_commit_msg from version body ,details %s", err.Error())
	} else {
		version.CommitMsg = msg
	}
	if cVersion, err := json.Get("code_version").String(); err != nil || cVersion == "" {
		logrus.Debugf("error get code_version from version body ,details %s", err.Error())
	} else {
		version.CodeVersion = cVersion
	}

	if status, err := json.Get("final_status").String(); err != nil || status == "" {
		logrus.Debugf("error get final_status from version body ,details %s", err.Error())
	} else {
		version.FinalStatus = status
	}
	err = db.GetManager().VersionInfoDao().UpdateModel(version)
	if err != nil {
		httputil.ReturnError(r, w, 500, err.Error())
		return
	}
	httputil.ReturnSuccess(r, w, nil)
}
func GetVersionByServiceID(w http.ResponseWriter, r *http.Request) {
	serviceID := strings.TrimSpace(chi.URLParam(r, "serviceID"))

	versions, err := db.GetManager().VersionInfoDao().GetVersionByServiceID(serviceID)
	if err != nil {
		httputil.ReturnError(r, w, 404, err.Error())
	}
	httputil.ReturnSuccess(r, w, versions)
}
func DeleteVersionByEventID(w http.ResponseWriter, r *http.Request) {
	eventID := strings.TrimSpace(chi.URLParam(r, "eventID"))

	versionInfo, _ := db.GetManager().VersionInfoDao().GetVersionByEventID(eventID)
	if versionInfo.DeliveredType == "" || versionInfo.DeliveredPath == "" {
		httputil.ReturnError(r, w, 500, errors.New("交付物类型及交付路径为空").Error())
		return
	}
	if versionInfo.DeliveredType == "code" {
		//todo 挺危险的。
	} else {
		cmd := exec.Command("docker", "rmi", versionInfo.DeliveredPath)
		err := cmd.Start()
		if err != nil {
			logrus.Errorf("error delete image :%s ,details %s", versionInfo.DeliveredPath, err.Error())
		}
	}
	err := db.GetManager().VersionInfoDao().DeleteVersionByEventID(eventID)
	if err != nil {
		httputil.ReturnError(r, w, 404, err.Error())
		return
	}
	httputil.ReturnSuccess(r, w, nil)
}
func UpdateDeliveredPath(w http.ResponseWriter, r *http.Request) {
	in, err := io.ReadAll(r.Body)
	if err != nil {
		httputil.ReturnError(r, w, 400, err.Error())
		return
	}
	logrus.Infof("update build info to %s", string(in))
	jsonc, err := simplejson.NewJson(in)
	if err != nil {
		httputil.ReturnError(r, w, 400, err.Error())
		return
	}
	event, err := jsonc.Get("event_id").String()
	if err != nil {
		httputil.ReturnError(r, w, 400, err.Error())
		return
	}
	dt, err := jsonc.Get("type").String()
	if err != nil {
		httputil.ReturnError(r, w, 400, err.Error())
		return
	}
	dp, err := jsonc.Get("path").String()
	if err != nil {
		httputil.ReturnError(r, w, 400, err.Error())
		return
	}
	version, err := db.GetManager().VersionInfoDao().GetVersionByEventID(event)
	if err != nil {
		httputil.ReturnError(r, w, 404, err.Error())
		return
	}

	version.DeliveredType = dt
	version.DeliveredPath = dp
	if version.DeliveredType == "slug" {
		version.ImageName = chaos.RUNNERIMAGENAME
	} else {
		version.ImageName = dp
	}
	err = db.GetManager().VersionInfoDao().UpdateModel(version)
	if err != nil {
		httputil.ReturnError(r, w, 500, err.Error())
		return
	}
	httputil.ReturnSuccess(r, w, nil)
}
