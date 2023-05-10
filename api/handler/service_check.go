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

package handler

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pquerna/ffjson/ffjson"
	"github.com/sirupsen/logrus"
	api_model "github.com/wutong-paas/wutong/api/model"
	"github.com/wutong-paas/wutong/api/util"
	"github.com/wutong-paas/wutong/builder/exector"
	client "github.com/wutong-paas/wutong/mq/client"
	tutil "github.com/wutong-paas/wutong/util"
)

// ServiceCheck check service build source
func (s *ServiceAction) ServiceCheck(scs *api_model.ServiceCheckStruct) (string, string, *util.APIHandleError) {
	checkUUID := uuid.New().String()
	scs.Body.CheckUUID = checkUUID
	if scs.Body.EventID == "" {
		scs.Body.EventID = tutil.NewUUID()
	}
	topic := client.BuilderTopic
	if tutil.StringArrayContains(s.conf.EnableFeature, "windows") {
		if scs.Body.CheckOS == "windows" {
			topic = client.WindowsBuilderTopic
		}
		if scs.Body.SourceType == "docker-run" || scs.Body.SourceType == "docker-compose" {
			if maybeIsWindowsContainerImage(scs.Body.SourceBody) {
				topic = client.WindowsBuilderTopic
			}
		}
	}
	err := s.MQClient.SendBuilderTopic(client.TaskStruct{
		TaskType: "service_check",
		TaskBody: scs.Body,
		Topic:    topic,
	})
	if err != nil {
		logrus.Errorf("enqueue service check message to mq error, %v", err)
		return "", "", util.CreateAPIHandleError(500, err)
	}
	return checkUUID, scs.Body.EventID, nil
}

var windowsKeywords = []string{"windows", "asp", "microsoft", "nanoserver"}

func maybeIsWindowsContainerImage(source string) bool {
	for _, k := range windowsKeywords {
		if strings.Contains(source, k) {
			return true
		}
	}
	return false

}

// GetServiceCheckInfo get application source detection information
func (s *ServiceAction) GetServiceCheckInfo(uuid string) (*exector.ServiceCheckResult, *util.APIHandleError) {
	k := fmt.Sprintf("/servicecheck/%s", uuid)
	var si exector.ServiceCheckResult
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	resp, err := s.EtcdCli.Get(ctx, k)
	if err != nil {
		logrus.Errorf("get etcd k %s error, %v", k, err)
		return nil, util.CreateAPIHandleError(500, err)
	}
	if resp.Count == 0 {
		return &si, nil
	}
	v := resp.Kvs[0].Value
	if err := ffjson.Unmarshal(v, &si); err != nil {
		return nil, util.CreateAPIHandleError(500, err)
	}
	if si.CheckStatus == "" {
		si.CheckStatus = "Checking"
		logrus.Debugf("checking is %v", si)
	}
	return &si, nil
}
