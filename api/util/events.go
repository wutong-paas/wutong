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

package util

import (
	"time"

	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
	"github.com/wutong-paas/wutong/db"
	dbmodel "github.com/wutong-paas/wutong/db/model"
	"github.com/wutong-paas/wutong/util"
)

// CanDoEvent check can do event or not
func CanDoEvent(optType string, synType int, target, targetID string, componentKind string) bool {
	if synType == dbmodel.SyncEventType || componentKind == "third_party" {
		return true
	}
	event, err := db.GetManager().ServiceEventDao().GetLastASyncEvent(target, targetID)
	if err != nil {
		if err.Error() == gorm.ErrRecordNotFound.Error() {
			return true
		}
		logrus.Error("get event by targetID error:", err)
		return false
	}
	if event == nil || event.FinalStatus != "" {
		return true
	}
	if !checkTimeout(event) {
		return false
	}
	return true
}

func checkTimeout(event *dbmodel.ServiceEvent) bool {
	if event.SynType == dbmodel.AsyncEventType {
		if event.FinalStatus == "" {
			startTime := event.StartTime
			start, err := time.ParseInLocation(time.RFC3339, startTime, time.Local)
			if err != nil {
				return false
			}
			// end := start.Add(3 * time.Minute)
			// if time.Now().After(end) {
			// 	event.FinalStatus = dbmodel.EventFinalStatusTimeout.String()
			// 	err = db.GetManager().ServiceEventDao().UpdateModel(event)
			// 	if err != nil {
			// 		logrus.Error("check event timeout error : ", err.Error())
			// 		return false
			// 	}
			// 	return true
			// }
			// latest event is still processing on
			// return false

			// 改成 10s 后允许再次操作，但是不更新上一个事件的状态
			if time.Now().After(start.Add(10 * time.Second)) {
				return true
			}
			return false
		}
	}
	return true
}

// CreateEvent save event
func CreateEvent(target, optType, targetID, tenantEnvID, reqBody, userName string, synType int) (*dbmodel.ServiceEvent, error) {
	if len(reqBody) > 1024 {
		reqBody = reqBody[0:1024]
	}
	event := dbmodel.ServiceEvent{
		EventID:     util.NewUUID(),
		TenantEnvID: tenantEnvID,
		Target:      target,
		TargetID:    targetID,
		RequestBody: reqBody,
		UserName:    userName,
		StartTime:   time.Now().Format(time.RFC3339),
		SynType:     synType,
		OptType:     optType,
	}
	err := db.GetManager().ServiceEventDao().AddModel(&event)
	return &event, err
}

// UpdateEvent update event
func UpdateEvent(eventID string, statusCode int) {
	event, err := db.GetManager().ServiceEventDao().GetEventByEventID(eventID)
	if err != nil && err != gorm.ErrRecordNotFound {
		logrus.Errorf("find event by eventID error : %s", err.Error())
		return
	}
	if err == gorm.ErrRecordNotFound {
		logrus.Errorf("do not found event by eventID %s", eventID)
		return
	}
	event.FinalStatus = dbmodel.EventFinalStatusComplete.String()
	event.EndTime = time.Now().Format(time.RFC3339)
	if statusCode < 400 { // status code 2XX/3XX all equal to success
		event.Status = dbmodel.EventStatusSuccess.String()
	} else {
		event.Status = dbmodel.EventStatusFailure.String()
	}
	err = db.GetManager().ServiceEventDao().UpdateModel(event)
	if err != nil {
		logrus.Errorf("update event status failure %s", err.Error())
		retry := 2
		for retry > 0 {
			if err = db.GetManager().ServiceEventDao().UpdateModel(event); err != nil {
				retry--
			} else {
				break
			}
		}
	}
}
