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

package exector

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
	"github.com/wutong-paas/wutong/chaos"
	"github.com/wutong-paas/wutong/chaos/build"
	"github.com/wutong-paas/wutong/chaos/sources"
	"github.com/wutong-paas/wutong/db"
	"github.com/wutong-paas/wutong/event"
)

// ImageBuildItem ImageBuildItem
type ImageBuildItem struct {
	Namespace     string       `json:"namespace"`
	TenantEnvName string       `json:"tenant_env_name"`
	ServiceAlias  string       `json:"service_alias"`
	Image         string       `json:"image"`
	DestImage     string       `json:"dest_image"`
	Logger        event.Logger `json:"logger"`
	EventID       string       `json:"event_id"`
	ImageClient   sources.ImageClient
	TenantEnvID   string
	ServiceID     string
	DeployVersion string
	HubUser       string
	HubPassword   string
	Action        string
	Configs       map[string]gjson.Result `json:"configs"`
}

// NewImageBuildItem 创建实体
func NewImageBuildItem(in []byte) *ImageBuildItem {
	eventID := gjson.GetBytes(in, "event_id").String()
	logger := event.GetManager().GetLogger(eventID)
	return &ImageBuildItem{
		Namespace:     gjson.GetBytes(in, "namespace").String(),
		TenantEnvName: gjson.GetBytes(in, "tenant_env_name").String(),
		ServiceAlias:  gjson.GetBytes(in, "service_alias").String(),
		ServiceID:     gjson.GetBytes(in, "service_id").String(),
		Image:         gjson.GetBytes(in, "image").String(),
		DeployVersion: gjson.GetBytes(in, "deploy_version").String(),
		Action:        gjson.GetBytes(in, "action").String(),
		HubUser:       gjson.GetBytes(in, "user").String(),
		HubPassword:   gjson.GetBytes(in, "password").String(),
		Configs:       gjson.GetBytes(in, "configs").Map(),
		Logger:        logger,
		EventID:       eventID,
	}
}

// Run Run
func (i *ImageBuildItem) Run(timeout time.Duration) error {
	var syncImage = true
	image := i.Image
	if strings.HasPrefix(i.Image, chaos.REGISTRYDOMAIN) {
		syncImage = false
	}
	if len(i.HubUser) == 0 {
		syncImage = false
	}
	if syncImage {
		user, pass := chaos.GetImageUserInfoV2(i.Image, i.HubUser, i.HubPassword)
		_, err := i.ImageClient.ImagePull(i.Image, user, pass, i.Logger, 30)
		if err != nil {
			logrus.Errorf("pull image %s error: %s", i.Image, err.Error())
			i.Logger.Error(fmt.Sprintf("获取指定镜像：%s 失败！", i.Image), map[string]string{"step": "builder-exector", "status": "failure"})
			return err
		}

		image = build.CreateImageName(i.ServiceID, i.DeployVersion)
		if err := i.ImageClient.ImageTag(i.Image, image, i.Logger, 1); err != nil {
			logrus.Errorf("change image tag error: %s", err.Error())
			i.Logger.Error(fmt.Sprintf("修改镜像 Tag：%s -> %s 失败！", i.Image, image), map[string]string{"step": "builder-exector", "status": "failure"})
			return err
		}
		err = i.ImageClient.ImagePush(image, chaos.REGISTRYUSER, chaos.REGISTRYPASS, i.Logger, 30)
		if err != nil {
			logrus.Errorf("push image into registry error: %s", err.Error())
			i.Logger.Error("推送镜像至镜像仓库失败："+err.Error(), map[string]string{"step": "builder-exector", "status": "failure"})
			return err
		}

		if err := i.ImageClient.ImageRemove(image); err != nil {
			logrus.Errorf("remove image %s failure %s", image, err.Error())
		}

		if os.Getenv("DISABLE_IMAGE_CACHE") == "true" {
			if err := i.ImageClient.ImageRemove(i.Image); err != nil {
				logrus.Errorf("remove image %s failure %s", i.Image, err.Error())
			}
		}
	} else {
		i.Logger.Info("判定应用组件镜像源为来自内置镜像仓库或公开镜像，免构建...", nil)
	}

	if err := i.StorageVersionInfo(image); err != nil {
		logrus.Errorf("storage version info error, ignor it: %s", err.Error())
		i.Logger.Error("更新应用版本信息失败！", map[string]string{"step": "builder-exector", "status": "failure"})
		return err
	}
	return nil
}

// StorageVersionInfo 存储version信息
func (i *ImageBuildItem) StorageVersionInfo(image string) error {
	version, err := db.GetManager().VersionInfoDao().GetVersionByDeployVersion(i.DeployVersion, i.ServiceID)
	if err != nil {
		return err
	}
	version.DeliveredType = "image"
	version.DeliveredPath = image
	version.ImageName = image
	version.RepoURL = i.Image
	version.FinalStatus = "success"
	version.FinishTime = time.Now()
	if err := db.GetManager().VersionInfoDao().UpdateModel(version); err != nil {
		return err
	}
	return nil
}

// UpdateVersionInfo 更新任务执行结果
func (i *ImageBuildItem) UpdateVersionInfo(status string) error {
	version, err := db.GetManager().VersionInfoDao().GetVersionByEventID(i.EventID)
	if err != nil {
		return err
	}
	version.FinalStatus = status
	version.RepoURL = i.Image
	version.FinishTime = time.Now()
	if err := db.GetManager().VersionInfoDao().UpdateModel(version); err != nil {
		return err
	}
	return nil
}
