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

package handler

import (
	"context"
	"fmt"
	"os"
	"path"

	"github.com/coreos/etcd/clientv3"
	"github.com/sirupsen/logrus"
	"github.com/wutong-paas/wutong/api/model"
	"github.com/wutong-paas/wutong/db"
	dbmodel "github.com/wutong-paas/wutong/db/model"
	eventdb "github.com/wutong-paas/wutong/eventlog/db"
	"github.com/wutong-paas/wutong/util/constants"
)

// LogAction  log action struct
type LogAction struct {
	EtcdCli *clientv3.Client
	eventdb *eventdb.EventFilePlugin
}

// CreateLogManager get log manager
func CreateLogManager(cli *clientv3.Client) *LogAction {
	return &LogAction{
		EtcdCli: cli,
		eventdb: &eventdb.EventFilePlugin{
			HomePath: "/wtdata/logs/",
		},
	}
}

// GetEvents get target logs
func (l *LogAction) GetEvents(target, targetID string, page, size int) ([]*dbmodel.ServiceEvent, int, error) {
	if target == "tenant_env" {
		return db.GetManager().ServiceEventDao().GetEventsByTenantEnvID(targetID, (page-1)*size, size)
	}
	return db.GetManager().ServiceEventDao().GetEventsByTarget(target, targetID, (page-1)*size, size)
}

// GetLogList get log list
func (l *LogAction) GetLogList(serviceAlias string) ([]*model.HistoryLogFile, error) {
	logDIR := path.Join(constants.WTDataLogPath, serviceAlias)
	_, err := os.Stat(logDIR)
	if os.IsNotExist(err) {
		return nil, err
	}
	fileList, err := os.ReadDir(logDIR)
	if err != nil {
		return nil, err
	}

	var logFiles []*model.HistoryLogFile
	for _, file := range fileList {
		logfile := &model.HistoryLogFile{
			Filename:     file.Name(),
			RelativePath: path.Join("logs", serviceAlias, file.Name()),
		}
		logFiles = append(logFiles, logfile)
	}
	return logFiles, nil
}

// GetLogFile GetLogFile
func (l *LogAction) GetLogFile(serviceAlias, fileName string) (string, string, error) {
	logPath := path.Join(constants.WTDataLogPath, serviceAlias)
	fullPath := path.Join(logPath, fileName)
	_, err := os.Stat(fullPath)
	if os.IsNotExist(err) {
		return "", "", err
	}
	return logPath, fullPath, err
}

// GetLogInstance get log web socket instance
func (l *LogAction) GetLogInstance(serviceID string) (string, error) {
	value, err := l.EtcdCli.Get(context.Background(), fmt.Sprintf("/event/dockerloginstacne/%s", serviceID))
	if err != nil {
		return "", err
	}
	if len(value.Kvs) > 0 {
		return string(value.Kvs[0].Value), nil
	}

	return "", nil
}

// GetLevelLog get event log
func (l *LogAction) GetLevelLog(eventID string, level string) (*model.DataLog, error) {
	re, err := l.eventdb.GetMessages(eventID, level, 0)
	if err != nil {
		logrus.Errorf("get event log error: %s", err.Error())
		return nil, err
	}
	if re != nil {
		messageList, ok := re.(eventdb.MessageDataList)
		if ok {
			return &model.DataLog{
				Status: "success",
				Data:   messageList,
			}, nil
		}
	}
	return &model.DataLog{
		Status: "success",
		Data:   nil,
	}, nil
}
