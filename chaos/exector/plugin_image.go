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
	"strings"

	"github.com/pquerna/ffjson/ffjson"
	"github.com/sirupsen/logrus"
	"github.com/wutong-paas/wutong/chaos"
	"github.com/wutong-paas/wutong/chaos/model"
	"github.com/wutong-paas/wutong/db"
	"github.com/wutong-paas/wutong/event"
	"github.com/wutong-paas/wutong/mq/api/grpc/pb"
)

func (e *exectorManager) pluginImageBuild(task *pb.TaskMessage) {
	var tb model.BuildPluginTaskBody
	if err := ffjson.Unmarshal(task.TaskBody, &tb); err != nil {
		logrus.Errorf("unmarshal taskbody error, %v", err)
		return
	}
	eventID := tb.EventID
	logger := event.GetLogger(eventID)
	logger.Info("从镜像构建插件任务开始执行", map[string]string{"step": "builder-exector", "status": "starting"})

	logrus.Info("start exec build plugin from image worker")
	defer event.CloseLogger(eventID)
	for retry := 0; retry < 2; retry++ {
		err := e.run(&tb, logger)
		if err != nil {
			logrus.Errorf("exec plugin build from image error:%s", err.Error())
			logger.Info("镜像构建插件任务执行失败，开始重试", map[string]string{"step": "builder-exector", "status": "failure"})
		} else {
			return
		}
	}
	version, err := db.GetManager().TenantEnvPluginBuildVersionDao().GetBuildVersionByDeployVersion(tb.PluginID, tb.VersionID, tb.DeployVersion)
	if err != nil {
		logrus.Errorf("get version error, %v", err)
		return
	}
	tenantEnvPlugin, err := db.GetManager().TenantEnvPluginDao().GetPluginByID(tb.PluginID, tb.TenantEnvID)
	if err != nil {
		logrus.Errorf("get plugin error, %v", err)
		return
	}
	if tenantEnvPlugin.PluginType != "sys" {
		version.Status = "failure"
		if err := db.GetManager().TenantEnvPluginBuildVersionDao().UpdateModel(version); err != nil {
			logrus.Errorf("update version error, %v", err)
		}
	}
	MetricErrorTaskNum++
	logger.Error("镜像构建插件任务执行失败", map[string]string{"step": "callback", "status": "failure"})
}

func (e *exectorManager) run(t *model.BuildPluginTaskBody, logger event.Logger) error {
	var syncImage = true
	image := t.ImageURL
	if strings.HasPrefix(t.ImageURL, chaos.REGISTRYDOMAIN) {
		syncImage = false
	}
	if t.ImageInfo.HubUser == "" {
		syncImage = false
	}
	if syncImage {
		hubUser, hubPass := chaos.GetImageUserInfoV2(t.ImageURL, t.ImageInfo.HubUser, t.ImageInfo.HubPassword)
		if _, err := e.imageClient.ImagePull(t.ImageURL, hubUser, hubPass, logger, 10); err != nil {
			logrus.Errorf("pull image %v error, %v", t.ImageURL, err)
			logger.Error("拉取镜像失败，错误信息："+err.Error(), map[string]string{"step": "builder-exector", "status": "failure"})
			return err
		}
		logger.Info("拉取镜像完成", map[string]string{"step": "build-exector", "status": "complete"})
		image = createPluginImageTag(t.ImageURL, t.PluginID, t.DeployVersion)
		err := e.imageClient.ImageTag(t.ImageURL, image, logger, 1)
		if err != nil {
			logrus.Errorf("set plugin image tag error, %v", err)
			logger.Error("修改镜像 Tag 失败，错误信息："+err.Error(), map[string]string{"step": "builder-exector", "status": "failure"})
			return err
		}
		logger.Info("修改镜像Tag完成", map[string]string{"step": "build-exector", "status": "complete"})
		if err := e.imageClient.ImagePush(image, chaos.REGISTRYUSER, chaos.REGISTRYPASS, logger, 10); err != nil {
			logrus.Errorf("push image %s error, %v", image, err)
			logger.Error("推送镜像失败，错误信息："+err.Error(), map[string]string{"step": "builder-exector", "status": "failure"})
			return err
		}
	}

	version, err := db.GetManager().TenantEnvPluginBuildVersionDao().GetBuildVersionByDeployVersion(t.PluginID, t.VersionID, t.DeployVersion)
	if err != nil {
		logger.Error("更新插件版本信息失败，错误信息："+err.Error(), map[string]string{"step": "builder-exector", "status": "failure"})
		return err
	}
	version.BuildLocalImage = image
	version.Status = "complete"
	if err := db.GetManager().TenantEnvPluginBuildVersionDao().UpdateModel(version); err != nil {
		logger.Error("更新插件版本信息失败，错误信息："+err.Error(), map[string]string{"step": "builder-exector", "status": "failure"})
		return err
	}
	logger.Info("从镜像构建插件完成", map[string]string{"step": "last", "status": "success"})
	return nil
}

func createPluginImageTag(image string, pluginid, version string) string {
	//alias is pluginID
	mm := strings.Split(image, "/")
	tag := "latest"
	iName := ""
	if strings.Contains(mm[len(mm)-1], ":") {
		nn := strings.Split(mm[len(mm)-1], ":")
		tag = nn[1]
		iName = nn[0]
	} else {
		iName = image
	}
	if strings.HasPrefix(iName, "plugin") {
		return fmt.Sprintf("%s/%s:%s_%s", chaos.REGISTRYDOMAIN, iName, pluginid, version)
	}
	return fmt.Sprintf("%s/plugin_%s_%s:%s_%s", chaos.REGISTRYDOMAIN, iName, pluginid, tag, version)
}
