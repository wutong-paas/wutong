// WUTONG, Application Management Platform
// Copyright (C) 2014-2017 Wutong Co., Ltd.

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
	"encoding/json"
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/wutong-paas/wutong/api/model"
	"github.com/wutong-paas/wutong/cmd/api/option"
	"github.com/wutong-paas/wutong/util/containerutil"
)

const (
	appStoreVersionExportPathPrefix = "/wtdata/appstore/version/export"
)

// AppStoreVersionAction -
type AppStoreVersionAction struct {
	OptCfg *option.Config
}

// CreateAppStoreVersionManager creates app store version manager
func CreateAppStoreVersionManager(optCfg *option.Config) *AppStoreVersionAction {
	return &AppStoreVersionAction{
		OptCfg: optCfg,
	}
}

type stat struct {
	Status string `json:"status"`
}

type exportInfo struct {
	status string
}

const (
	exportStatusNotExport  = "未导出"
	exportStatusProcessing = "导出中"
	exportStatusSuccess    = "导出完成"
	exportStatusFailed     = "导出失败"
)

// ExportStatus 获取导出状态
// 导出状态枚举：导出中、导出完成、导出失败
func (a *AppStoreVersionAction) ExportStatus(versionId string) (string, string) {
	var file, status string

	if exportInfo, err := os.ReadFile(fmt.Sprintf("%s/%s.json", appStoreVersionExportPathPrefix, versionId)); err == nil {
		var stat stat
		if err := json.Unmarshal(exportInfo, &stat); err == nil {
			status = stat.Status
		}
		file = fmt.Sprintf("%s/%s.tar", appStoreVersionExportPathPrefix, versionId)
		if status == exportStatusSuccess {
			if _, err := os.Stat(file); err != nil && os.IsNotExist(err) {
				file = ""
				status = exportStatusFailed
			}
		}
	}
	if status == "" {
		status = exportStatusNotExport
	}
	return file, status
}

// Export 导出镜像
func (a *AppStoreVersionAction) Export(versionId string, req *model.AppStoreVersionExportImageInfo) error {
	if len(req.Images) == 0 {
		return nil
	}

	containerClient, err := containerutil.NewClient(a.OptCfg.ContainerRuntime, a.OptCfg.RuntimeEndpoint)
	if err != nil {
		logrus.Errorf("create container client failed: %v", err)
		return err
	}

	_, err = os.Stat(appStoreVersionExportPathPrefix)
	if err != nil {
		if os.IsNotExist(err) {
			if err = os.MkdirAll(appStoreVersionExportPathPrefix, 0755); err != nil {
				return err
			}
		} else {
			return err
		}
	}

	if err := writeExportInfo(versionId, exportStatusProcessing); err != nil {
		logrus.Errorf("write export info failed: %v", err)
		return fmt.Errorf("导出失败")
	}

	images := func(req *model.AppStoreVersionExportImageInfo) []string {
		var result []string
		for _, image := range req.Images {
			result = append(result, image.Image)
		}
		return result
	}(req)

	// 异步导出镜像，并写入导出状态
	go func() {
		for _, image := range req.Images {
			if _, err := containerClient.ImagePull(image.Image, image.Username, image.Password, 0); err != nil {
				logrus.Errorf("pull image %s failed: %v", image.Image, err)
				if err := writeExportInfo(versionId, exportStatusFailed); err != nil {
					logrus.Errorf("write export info failed: %v", err)
				}
				return
			}
		}
		if err := containerClient.ImageSave(fmt.Sprintf("%s/%s.tar", appStoreVersionExportPathPrefix, versionId), images); err != nil {
			if err := writeExportInfo(versionId, exportStatusFailed); err != nil {
				logrus.Errorf("write export info failed: %v", err)
			}
			return
		}
		if err := writeExportInfo(versionId, exportStatusSuccess); err != nil {
			logrus.Errorf("write export info failed: %v", err)
		}
	}()

	return nil
}

// Download 下载导出的镜像
func (a *AppStoreVersionAction) Download(versionId string) (string, error) {
	file, status := a.ExportStatus(versionId)
	if status != exportStatusSuccess {
		return file, fmt.Errorf("请先导出或等待导出完成")
	}

	return fmt.Sprintf("%s/%s.tar", appStoreVersionExportPathPrefix, versionId), nil
}

func writeExportInfo(versionId, status string) error {
	stat := stat{
		Status: status,
	}
	b, err := json.Marshal(stat)
	if err != nil {
		return err
	}

	return os.WriteFile(fmt.Sprintf("%s/%s.json", appStoreVersionExportPathPrefix, versionId), b, 0644)
}
