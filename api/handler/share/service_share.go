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

package share

import (
	"context"
	"fmt"

	"github.com/wutong-paas/wutong/mq/client"

	"github.com/wutong-paas/wutong/chaos/exector"

	"github.com/google/uuid"

	"github.com/pquerna/ffjson/ffjson"

	"github.com/wutong-paas/wutong/db"

	"github.com/coreos/etcd/clientv3"
	"github.com/sirupsen/logrus"
	api_model "github.com/wutong-paas/wutong/api/model"
	"github.com/wutong-paas/wutong/api/util"
)

// ServiceShareHandle service share
type ServiceShareHandle struct {
	MQClient client.MQClient
	EtcdCli  *clientv3.Client
}

// APIResult 分享接口返回
type APIResult struct {
	EventID   string `json:"event_id"`
	ShareID   string `json:"share_id"`
	ImageName string `json:"image_name,omitempty"`
	SlugPath  string `json:"slug_path,omitempty"`
}

// Share 分享应用
func (s *ServiceShareHandle) Share(serviceID string, ss api_model.ServiceShare) (*APIResult, *util.APIHandleError) {
	service, err := db.GetManager().TenantEnvServiceDao().GetServiceByID(serviceID)
	if err != nil {
		return nil, util.CreateAPIHandleErrorFromDBError("查询应用出错", err)
	}
	//查询部署版本
	version, err := db.GetManager().VersionInfoDao().GetVersionByDeployVersion(service.DeployVersion, serviceID)
	if err != nil {
		logrus.Error("query service deploy version error", err.Error())
	}
	shareID := uuid.New().String()
	var slugPath, shareImageName string
	var task client.TaskStruct
	if version.DeliveredType == "slug" {
		shareSlugInfo := ss.Body.SlugInfo
		slugPath = service.CreateShareSlug(ss.Body.ServiceKey, shareSlugInfo.Namespace, ss.Body.AppVersion)
		if ss.Body.SlugInfo.FTPHost == "" {
			slugPath = fmt.Sprintf("/wtdata/build/tenantEnv/%s", slugPath)
		}
		info := map[string]interface{}{
			"service_alias":   ss.ServiceAlias,
			"service_id":      serviceID,
			"tenant_env_name": ss.TenantEnvName,
			"share_info":      ss.Body,
			"slug_path":       slugPath,
			"share_id":        shareID,
		}
		if version != nil && version.DeliveredPath != "" {
			info["local_slug_path"] = version.DeliveredPath
		} else {
			info["local_slug_path"] = fmt.Sprintf("/wtdata/build/tenantEnv/%s/slug/%s/%s.tgz", service.TenantEnvID, service.ServiceID, service.DeployVersion)
		}
		task.TaskType = "share-slug"
		task.TaskBody = info
	} else {
		// shareImageInfo := ss.Body.ImageInfo
		// shareImageName, err = version.CreateShareImage(shareImageInfo.HubURL, shareImageInfo.Namespace, ss.Body.AppVersion)
		shareImageName = version.ImageName
		if err != nil {
			return nil, util.CreateAPIHandleError(500, err)
		}
		info := map[string]interface{}{
			"share_info":      ss.Body,
			"service_alias":   ss.ServiceAlias,
			"service_id":      serviceID,
			"tenant_env_name": ss.TenantEnvName,
			"image_name":      shareImageName,
			"share_id":        shareID,
		}
		if version != nil && version.DeliveredPath != "" {
			info["local_image_name"] = version.DeliveredPath
		}
		task.TaskType = "share-image"
		task.TaskBody = info
	}
	label, err := db.GetManager().TenantEnvServiceLabelDao().GetLabelByNodeSelectorKey(serviceID, "windows")
	if label == nil || err != nil {
		task.Topic = client.BuilderTopic
	} else {
		task.Topic = client.WindowsBuilderTopic
	}
	err = s.MQClient.SendBuilderTopic(task)
	if err != nil {
		logrus.Errorf("equque mq error, %v", err)
		return nil, util.CreateAPIHandleError(502, err)
	}
	return &APIResult{EventID: ss.Body.EventID, ShareID: shareID, ImageName: shareImageName, SlugPath: slugPath}, nil
}

// ShareResult 分享应用结果查询
func (s *ServiceShareHandle) ShareResult(shareID string) (i exector.ShareStatus, e *util.APIHandleError) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	res, err := s.EtcdCli.Get(ctx, fmt.Sprintf("/wutong/shareresult/%s", shareID))
	if err != nil {
		e = util.CreateAPIHandleError(500, err)
	} else {
		if res.Count == 0 {
			i.ShareID = shareID
		} else {
			if err := ffjson.Unmarshal(res.Kvs[0].Value, &i); err != nil {
				return i, util.CreateAPIHandleError(500, err)
			}
		}
	}
	return
}
