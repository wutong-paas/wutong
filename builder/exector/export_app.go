// WUTONG, Application Management Platform
// Copyright (C) 2014-2017 Wutong Co., Ltd.

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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
	"github.com/wutong-paas/wutong-oam/pkg/export"
	"github.com/wutong-paas/wutong-oam/pkg/ram/v1alpha1"
	"github.com/wutong-paas/wutong/builder"
	"github.com/wutong-paas/wutong/builder/sources"
	"github.com/wutong-paas/wutong/db"
	"github.com/wutong-paas/wutong/event"
)

// var re = regexp.MustCompile(`\s`)

// ExportApp Export app to specified format(wutong-app or dockercompose)
type ExportApp struct {
	EventID       string `json:"event_id"`
	Format        string `json:"format"`
	SourceDir     string `json:"source_dir"`
	WithImageData bool   `json:"with_image_data"`
	Logger        event.Logger
	ImageClient   sources.ImageClient
}

func init() {
	RegisterWorker("export_app", NewExportApp)
}

// NewExportApp create
func NewExportApp(in []byte, m *exectorManager) (TaskWorker, error) {
	eventID := gjson.GetBytes(in, "event_id").String()
	logger := event.GetManager().GetLogger(eventID)
	return &ExportApp{
		Format:        gjson.GetBytes(in, "format").String(),
		SourceDir:     gjson.GetBytes(in, "source_dir").String(),
		Logger:        logger,
		EventID:       eventID,
		ImageClient:   m.imageClient,
		WithImageData: gjson.GetBytes(in, "with_image_data").Bool(),
	}, nil
}

// Run Run
func (i *ExportApp) Run(timeout time.Duration) error {
	defer os.RemoveAll(i.SourceDir)

	// Delete the old application group directory and then regenerate the application package
	if i.Format != "helm_chart" && i.Format != "yaml" {
		if err := i.CleanSourceDir(); err != nil {
			return err
		}
	}

	ram, err := i.parseRAM()
	if err != nil {
		return err
	}
	i.handleDefaultRepo(ram)
	var re *export.Result
	if i.Format == "wutong-app" {
		re, err = i.exportWutongAPP(*ram)
		if err != nil {
			logrus.Errorf("export wutong app package failure %s", err.Error())
			i.updateStatus("failed", "")
			return err
		}
	} else if i.Format == "docker-compose" {
		re, err = i.exportDockerCompose(*ram)
		if err != nil {
			logrus.Errorf("export docker compose app package failure %s", err.Error())
			i.updateStatus("failed", "")
			return err
		}
	} else if i.Format == "slug" {
		re, err = i.exportSlug(*ram)
		if err != nil {
			logrus.Errorf("export slug app package failure %s", err.Error())
			i.updateStatus("failed", "")
			return err
		}
	} else if i.Format == "helm_chart" {
		re, err = i.exportHelmChart(*ram)
		if err != nil {
			logrus.Errorf("export helm chart package failure %s", err.Error())
			i.updateStatus("failed", "")
			return err
		}
	} else if i.Format == "yaml" {
		re, err = i.exportK8sYaml(*ram)
		if err != nil {
			logrus.Errorf("export k8s yaml package failure %s", err.Error())
			i.updateStatus("failed", "")
			return err
		}
	} else {
		return errors.New("Unsupported the format: " + i.Format)
	}
	if re != nil {
		// move package file to download dir
		downloadPath := path.Dir(i.SourceDir)
		os.Rename(re.PackagePath, path.Join(downloadPath, re.PackageName))
		packageDownloadPath := path.Join("/v2/app/download/", i.Format, re.PackageName)
		// update export event status
		if err := i.updateStatus("success", packageDownloadPath); err != nil {
			return err
		}
		logrus.Infof("move export package %s to download dir success", re.PackageName)
		i.cacheMd5()
	}

	return nil
}

func (i *ExportApp) handleDefaultRepo(ram *v1alpha1.WutongApplicationConfig) {
	for i := range ram.Components {
		com := ram.Components[i]
		com.AppImage.HubUser, com.AppImage.HubPassword = builder.GetImageUserInfoV2(
			com.ShareImage, com.AppImage.HubUser, com.AppImage.HubPassword)
	}
	for i := range ram.Plugins {
		plugin := ram.Plugins[i]
		plugin.PluginImage.HubUser, plugin.PluginImage.HubPassword = builder.GetImageUserInfoV2(
			plugin.ShareImage, plugin.PluginImage.HubUser, plugin.PluginImage.HubPassword)
	}
}

// create md5 file
func (i *ExportApp) cacheMd5() {
	metadataFile := fmt.Sprintf("%s/metadata.json", i.SourceDir)
	if err := exec.Command("sh", "-c", fmt.Sprintf("md5sum %s > %s.md5", metadataFile, metadataFile)).Run(); err != nil {
		err = errors.New(fmt.Sprintf("Failed to create md5 file: %v", err))
		logrus.Error(err)
	}
	logrus.Infof("create md5 file success")
}

// exportWutongAPP export offline wutong app
func (i *ExportApp) exportWutongAPP(ram v1alpha1.WutongApplicationConfig) (*export.Result, error) {
	ramExporter, err := export.New(export.RAM, i.SourceDir, ram, i.ImageClient.GetContainerdClient(), i.ImageClient.GetDockerClient(), logrus.StandardLogger())
	if err != nil {
		return nil, err
	}
	return ramExporter.Export()
}

// exportDockerCompose export app to docker compose app
func (i *ExportApp) exportDockerCompose(ram v1alpha1.WutongApplicationConfig) (*export.Result, error) {
	ramExporter, err := export.New(export.DC, i.SourceDir, ram, i.ImageClient.GetContainerdClient(), i.ImageClient.GetDockerClient(), logrus.StandardLogger())
	if err != nil {
		return nil, err
	}
	return ramExporter.Export()
}

// exportDockerCompose export app to docker compose app
func (i *ExportApp) exportSlug(ram v1alpha1.WutongApplicationConfig) (*export.Result, error) {
	slugExporter, err := export.New(export.SLG, i.SourceDir, ram, i.ImageClient.GetContainerdClient(), i.ImageClient.GetDockerClient(), logrus.StandardLogger())
	if err != nil {
		return nil, err
	}
	return slugExporter.Export()
}

func (i *ExportApp) exportHelmChart(ram v1alpha1.WutongApplicationConfig) (*export.Result, error) {
	helmChartExporter, err := export.New(export.HELM, i.SourceDir, ram, i.ImageClient.GetContainerdClient(), i.ImageClient.GetDockerClient(), logrus.StandardLogger())
	if err != nil {
		return nil, err
	}
	return helmChartExporter.Export()
}

func (i *ExportApp) exportK8sYaml(ram v1alpha1.WutongApplicationConfig) (*export.Result, error) {
	k8sYamlExporter, err := export.New(export.YAML, i.SourceDir, ram, i.ImageClient.GetContainerdClient(), i.ImageClient.GetDockerClient(), logrus.StandardLogger())
	if err != nil {
		return nil, err
	}
	return k8sYamlExporter.Export()
}

// Stop stop
func (i *ExportApp) Stop() error {
	return nil
}

// Name return worker name
func (i *ExportApp) Name() string {
	return "export_app"
}

// GetLogger GetLogger
func (i *ExportApp) GetLogger() event.Logger {
	return i.Logger
}

// CleanSourceDir clean export dir
func (i *ExportApp) CleanSourceDir() error {
	logrus.Debug("Ready clean the source directory.")
	metaFile := fmt.Sprintf("%s/metadata.json", i.SourceDir)

	data, err := ioutil.ReadFile(metaFile)
	if err != nil {
		logrus.Error("Failed to read metadata file: ", err)
		return err
	}

	os.RemoveAll(i.SourceDir)
	os.MkdirAll(i.SourceDir, 0755)

	if err := ioutil.WriteFile(metaFile, data, 0644); err != nil {
		logrus.Error("Failed to write metadata file: ", err)
		return err
	}

	return nil
}
func (i *ExportApp) parseRAM() (*v1alpha1.WutongApplicationConfig, error) {
	data, err := ioutil.ReadFile(fmt.Sprintf("%s/metadata.json", i.SourceDir))
	if err != nil {
		i.Logger.Error("导出应用失败，没有找到应用信息", map[string]string{"step": "read-metadata", "status": "failure"})
		logrus.Error("Failed to read metadata file: ", err)
		return nil, err
	}
	var ram v1alpha1.WutongApplicationConfig
	if err := json.Unmarshal(data, &ram); err != nil {
		return nil, err
	}

	ram.WithImageData = i.WithImageData

	return &ram, nil
}

func (i *ExportApp) updateStatus(status, filePath string) error {
	logrus.Debug("Update app status in database to: ", status)
	res, err := db.GetManager().AppDao().GetByEventId(i.EventID)
	if err != nil {
		err = errors.New(fmt.Sprintf("Failed to get app %s from db: %v", i.EventID, err))
		logrus.Error(err)
		return err
	}
	res.Status = status
	if filePath != "" {
		res.TarFileHref = filePath
	}
	if err := db.GetManager().AppDao().UpdateModel(res); err != nil {
		err = errors.New(fmt.Sprintf("Failed to update app %s: %v", i.EventID, err))
		logrus.Error(err)
		return err
	}
	return nil
}

// ErrorCallBack if run error will callback
func (i *ExportApp) ErrorCallBack(err error) {
	i.updateStatus("failed", "")
}
