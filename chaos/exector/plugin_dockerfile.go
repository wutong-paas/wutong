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

	"github.com/pquerna/ffjson/ffjson"
	"github.com/sirupsen/logrus"
	"github.com/wutong-paas/wutong/chaos"
	"github.com/wutong-paas/wutong/chaos/model"
	"github.com/wutong-paas/wutong/chaos/sources"
	"github.com/wutong-paas/wutong/db"
	"github.com/wutong-paas/wutong/event"
	"github.com/wutong-paas/wutong/mq/api/grpc/pb"
	"github.com/wutong-paas/wutong/util"
)

const (
	cloneTimeout    = 60
	buildingTimeout = 180
	formatSourceDir = "/cache/build/%s/source/%s"
)

func (e *exectorManager) pluginDockerfileBuild(task *pb.TaskMessage) {
	var tb model.BuildPluginTaskBody
	if err := ffjson.Unmarshal(task.TaskBody, &tb); err != nil {
		logrus.Errorf("unmarshal taskbody error, %v", err)
		return
	}
	eventID := tb.EventID
	logger := event.GetManager().GetLogger(eventID)
	logger.Info("从 Dockerfile 构建插件任务开始执行...", map[string]string{"step": "builder-exector", "status": "starting"})
	logrus.Info("start exec build plugin from image worker")
	defer event.GetManager().ReleaseLogger(logger)
	for retry := 0; retry < 2; retry++ {
		err := e.runD(&tb, logger)
		if err != nil {
			logrus.Errorf("exec plugin build from dockerfile error:%s", err.Error())
			logger.Info("Dockerfile 构建插件任务执行失败，开始重试...", map[string]string{"step": "builder-exector", "status": "failure"})
		} else {
			return
		}
	}
	version, err := db.GetManager().TenantEnvPluginBuildVersionDao().GetBuildVersionByDeployVersion(tb.PluginID, tb.VersionID, tb.DeployVersion)
	if err != nil {
		logrus.Errorf("get version error, %v", err)
		return
	}
	version.Status = "failure"
	if err := db.GetManager().TenantEnvPluginBuildVersionDao().UpdateModel(version); err != nil {
		logrus.Errorf("update version error, %v", err)
	}
	MetricErrorTaskNum++
	logger.Error("Dockerfile 构建插件任务执行失败", map[string]string{"step": "callback", "status": "failure"})
}

func (e *exectorManager) runD(t *model.BuildPluginTaskBody, logger event.Logger) error {
	logger.Info("开始拉取代码...", map[string]string{"step": "build-exector"})
	sourceDir := fmt.Sprintf(formatSourceDir, t.TenantEnvID, t.VersionID)
	if t.Repo == "" {
		t.Repo = "master"
	}
	if !util.DirIsEmpty(sourceDir) {
		os.RemoveAll(sourceDir)
	}
	if err := util.CheckAndCreateDir(sourceDir); err != nil {
		return err
	}
	if _, err := sources.GitClone(sources.CodeSourceInfo{RepositoryURL: t.GitURL, Branch: t.Repo, User: t.GitUsername, Password: t.GitPassword}, sourceDir, logger, 4); err != nil {
		logger.Error("拉取代码失败，错误信息："+err.Error(), map[string]string{"step": "builder-exector", "status": "failure"})
		logrus.Errorf("[plugin]git clone code error %v", err)
		return err
	}
	if !checkDockerfile(sourceDir) {
		logger.Error("代码未检测到 Dockerfile，暂不支持构建，任务即将退出", map[string]string{"step": "builder-exector", "status": "failure"})
		logrus.Error("代码未检测到 Dockerfile")
		return fmt.Errorf("have no dockerfile")
	}

	logger.Info("代码检测为 Dockerfile，开始构建插件镜像...", map[string]string{"step": "build-exector"})
	mm := strings.Split(t.GitURL, "/")
	n1 := strings.Split(mm[len(mm)-1], ".")[0]
	buildImageName := fmt.Sprintf(chaos.REGISTRYDOMAIN+"/plugin_%s_%s:%s", n1, t.PluginID, t.DeployVersion)

	err := sources.ImageBuild(sourceDir, "wt-system", t.PluginID, t.DeployVersion, logger, "plug-build", "", e.KanikoImage)
	if err != nil {
		logger.Error(fmt.Sprintf("构建插件镜像 %s 失败，可以在 wt-chaos 组件日志中查看详情", buildImageName), map[string]string{"step": "builder-exector", "status": "failure"})
		logrus.Errorf("[plugin]build image error: %s", err.Error())
		return err
	}
	logger.Info("构建插件镜像成功，开始推送镜像到本地镜像仓库", map[string]string{"step": "builder-exector"})

	logger.Info("推送镜像成功", map[string]string{"step": "build-exector"})
	version, err := db.GetManager().TenantEnvPluginBuildVersionDao().GetBuildVersionByDeployVersion(t.PluginID, t.VersionID, t.DeployVersion)
	if err != nil {
		logrus.Errorf("get version error, %v", err)
		return err
	}
	version.BuildLocalImage = buildImageName
	version.Status = "complete"
	if err := db.GetManager().TenantEnvPluginBuildVersionDao().UpdateModel(version); err != nil {
		logrus.Errorf("update version error, %v", err)
		return err
	}
	logger.Info("通过 Dockerfile 构建插件镜像成功", map[string]string{"step": "last", "status": "success"})
	return nil
}

func checkDockerfile(sourceDir string) bool {
	if _, err := os.Stat(fmt.Sprintf("%s/Dockerfile", sourceDir)); os.IsNotExist(err) {
		return false
	}
	return true
}
