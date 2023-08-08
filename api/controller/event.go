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
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/bitly/go-simplejson"
	"github.com/go-chi/chi"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
	"github.com/wutong-paas/wutong/api/handler"
	"github.com/wutong-paas/wutong/db"
	httputil "github.com/wutong-paas/wutong/util/http"
)

// Event GetLogs
func (t *TenantEnvStruct) Event(w http.ResponseWriter, r *http.Request) {
	// swagger:operation GET  /v2/tenants/{tenant_name}/envs/{tenant_env_name}/event v2 getevents
	//
	// 获取指定event_ids详细信息
	//
	// get events
	//
	// ---
	// produces:
	// - application/json
	// - application/xml
	//
	// responses:
	//   default:
	//     schema:
	//       "$ref": "#/responses/commandResponse"
	//     description: 统一返回格式
	b, _ := io.ReadAll(r.Body)
	defer r.Body.Close()
	j, err := simplejson.NewJson(b)
	if err != nil {
		logrus.Errorf("error decode json,details %s", err.Error())
		httputil.ReturnError(r, w, 400, "bad request")
		return
	}
	eventIDs, err := j.Get("event_ids").StringArray()
	if err != nil {
		logrus.Errorf("error get event_id in json,details %s", err.Error())
		httputil.ReturnError(r, w, 400, "bad request")
		return
	}

	events, err := handler.GetServiceEventHandler().ListByEventIDs(eventIDs)
	if err != nil {
		httputil.ReturnBcodeError(r, w, err)
		return
	}

	httputil.ReturnSuccess(r, w, events)
}

// GetNotificationEvents GetNotificationEvent
// support query from start and end time or all
// swagger:operation GET  /v2/notificationEvent v2/notificationEvent getevents
//
// 获取数据中心通知事件
//
// get events
//
// ---
// produces:
// - application/json
// - application/xml
//
// responses:
//
//	default:
//	  schema:
//	    "$ref": "#/responses/commandResponse"
//	  description: 统一返回格式
func GetNotificationEvents(w http.ResponseWriter, r *http.Request) {
	var startTime, endTime time.Time
	start := r.FormValue("start")
	end := r.FormValue("end")
	if si, err := strconv.Atoi(start); err == nil {
		startTime = time.Unix(int64(si), 0)
	}
	if ei, err := strconv.Atoi(end); err == nil {
		endTime = time.Unix(int64(ei), 0)
	}
	res, err := db.GetManager().NotificationEventDao().GetNotificationEventByTime(startTime, endTime)
	if err != nil {
		logrus.Errorf(err.Error())
		httputil.ReturnError(r, w, 500, err.Error())
		return
	}
	for _, v := range res {
		service, err := db.GetManager().TenantEnvServiceDao().GetServiceByID(v.KindID)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				v.ServiceName = ""
				v.TenantEnvName = ""
				continue
			} else {
				logrus.Errorf(err.Error())
				httputil.ReturnError(r, w, 500, err.Error())
				return
			}
		}
		tenantEnv, err := db.GetManager().TenantEnvDao().GetTenantEnvByUUID(service.TenantEnvID)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				v.ServiceName = ""
				v.TenantEnvName = ""
				continue
			} else {
				logrus.Errorf(err.Error())
				httputil.ReturnError(r, w, 500, err.Error())
				return
			}
		}
		v.ServiceName = service.ServiceAlias
		v.TenantEnvName = tenantEnv.Name
	}
	httputil.ReturnSuccess(r, w, res)
}

// Handle Handle
// swagger:parameters handlenotify
type Handle struct {
	Body struct {
		//in: body
		//handle message
		HandleMessage string `json:"handle_message" validate:"handle_message"`
	}
}

// HandleNotificationEvent HandleNotificationEvent
// swagger:operation PUT  /v2/notificationEvent/{hash} v2/notificationEvent handlenotify
//
// 处理通知事件
//
// get events
//
// ---
// produces:
// - application/json
// - application/xml
//
// responses:
//
//	default:
//	  schema:
//	    "$ref": "#/responses/commandResponse"
//	  description: 统一返回格式
func HandleNotificationEvent(w http.ResponseWriter, r *http.Request) {
	serviceAlias := chi.URLParam(r, "serviceAlias")
	if serviceAlias == "" {
		httputil.ReturnError(r, w, 400, "ServiceAlias id do not empty")
		return
	}
	var handle Handle
	ok := httputil.ValidatorRequestStructAndErrorResponse(r, w, &handle.Body, nil)
	if !ok {
		return
	}
	service, err := db.GetManager().TenantEnvServiceDao().GetServiceByServiceAlias(serviceAlias)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			httputil.ReturnError(r, w, 404, "not found")
			return
		}
		httputil.ReturnError(r, w, 500, err.Error())
		return
	}
	eventList, err := db.GetManager().NotificationEventDao().GetNotificationEventByKind("service", service.ServiceID)
	if err != nil {
		httputil.ReturnError(r, w, 500, err.Error())
		return
	}
	for _, event := range eventList {
		event.IsHandle = true
		event.HandleMessage = handle.Body.HandleMessage
		err = db.GetManager().NotificationEventDao().UpdateModel(event)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				httputil.ReturnError(r, w, 404, "not found")
				return
			}
			httputil.ReturnError(r, w, 500, err.Error())
			return
		}
	}

	httputil.ReturnSuccess(r, w, nil)
}

// GetNotificationEvent GetNotificationEvent
// swagger:operation GET  /v2/notificationEvent/{hash} v2/notificationEvent getevents
//
// 获取通知事件
//
// get events
//
// ---
// produces:
// - application/json
// - application/xml
//
// responses:
//
//	default:
//	  schema:
//	    "$ref": "#/responses/commandResponse"
//	  description: 统一返回格式
func GetNotificationEvent(w http.ResponseWriter, r *http.Request) {

	hash := chi.URLParam(r, "hash")
	if hash == "" {
		httputil.ReturnError(r, w, 400, "hash id do not empty")
		return
	}
	event, err := db.GetManager().NotificationEventDao().GetNotificationEventByHash(hash)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			httputil.ReturnError(r, w, 404, "not found")
			return
		}
		httputil.ReturnError(r, w, 500, err.Error())
		return
	}
	httputil.ReturnSuccess(r, w, event)
}
