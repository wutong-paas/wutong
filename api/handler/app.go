package handler

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/wutong-paas/wutong/db"
	dbmodel "github.com/wutong-paas/wutong/db/model"
	mqclient "github.com/wutong-paas/wutong/mq/client"
	dmodel "github.com/wutong-paas/wutong/worker/discover/model"

	"regexp"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
	"github.com/wutong-paas/wutong/api/model"
	"github.com/wutong-paas/wutong/api/util"
)

var re = regexp.MustCompile(`\s`)

// AppAction app action
type AppAction struct {
	MQClient   mqclient.MQClient
	staticDir  string
	appName    string
	appVersion string
	end        bool
}

// SetExportAppInfoParameter pass the export app info parameter
func (a *AppAction) SetExportAppInfoParameter(appName, appVersion string, end bool) {
	a.appName = appName
	a.appVersion = appVersion
	a.end = end
}

// GetStaticDir get static dir
func (a *AppAction) GetStaticDir() string {
	return a.staticDir
}

// CreateAppManager create app manager
func CreateAppManager(mqClient mqclient.MQClient) *AppAction {
	staticDir := "/wtdata/app"
	if os.Getenv("LOCAL_APP_CACHE_DIR") != "" {
		staticDir = os.Getenv("LOCAL_APP_CACHE_DIR")
	}
	return &AppAction{
		MQClient:  mqClient,
		staticDir: staticDir,
	}
}

// Complete Complete
func (a *AppAction) Complete(tr *model.ExportAppStruct) error {
	appName := gjson.Get(tr.Body.GroupMetadata, "group_name").String()
	if appName == "" {
		err := errors.New("Failed to get group name form metadata")
		logrus.Error(err)
		return err
	}

	if tr.Body.Format != "wutong-app" &&
		tr.Body.Format != "docker-compose" &&
		tr.Body.Format != "slug" &&
		tr.Body.Format != "helm_chart" &&
		tr.Body.Format != "yaml" {
		err := errors.New("Unsupported the format: " + tr.Body.Format)
		logrus.Error(err)
		return err
	}

	version := gjson.Get(tr.Body.GroupMetadata, "group_version").String()

	components := gjson.Get(tr.Body.GroupMetadata, "apps").Array()

	appName = unicode2zh(appName)
	tr.SourceDir = fmt.Sprintf("%s/%s/%s-%s", a.staticDir, tr.Body.Format, appName, version)

	if tr.Body.Format == "helm_chart" || tr.Body.Format == "yaml" {
		for i, v := range components {
			a.SetExportAppInfoParameter(appName, version, i == len(components)-1)
			serviceID := v.Get("service_id").String()
			service, err := db.GetManager().TenantEnvServiceDao().GetServiceByID(serviceID)
			if err != nil {
				return err
			}
			err = a.exportHelmChartOrK8sYaml(tr.Body.Format, service)
			if err != nil {
				log.Printf("Failed to export helm chart or k8s yaml: %v", err)
				return err
			}
		}
	}

	return nil
}

// ExportApp ExportApp
func (a *AppAction) ExportApp(tr *model.ExportAppStruct) error {
	// 保存元数据到组目录
	if err := a.saveMetadata(tr); err != nil {
		return util.CreateAPIHandleErrorFromDBError("Failed to export app", err)
	}
	err := a.MQClient.SendBuilderTopic(mqclient.TaskStruct{
		TaskBody: model.BuildMQBodyFrom(tr),
		TaskType: "export_app",
		Topic:    mqclient.BuilderTopic,
	})
	if err != nil {
		logrus.Error("Failed to Enqueue MQ for ExportApp:", err)
		return err
	}

	return nil
}

func (a *AppAction) exportHelmChartOrK8sYaml(format string, service *dbmodel.TenantEnvServices) error {
	body := dmodel.ExportHelmChartOrK8sYamlTaskBody{
		TenantEnvID: service.TenantEnvID,
		ServiceID:   service.ServiceID,
		AppVersion: a.appVersion,
		AppName:    a.appName,
		End:        a.end,
	}
	var taskType string
	switch format {
	case "helm_chart":
		taskType = model.ExportHelmChart
	case "yaml":
		taskType = model.ExportK8sYaml
	default:
		return fmt.Errorf("unsupported the export mode: %s", format)
	}

	return a.MQClient.SendBuilderTopic(mqclient.TaskStruct{
		Topic:    mqclient.WorkerTopic,
		TaskType: taskType,
		TaskBody: body,
	})
}

// ImportApp import app
func (a *AppAction) ImportApp(importApp *model.ImportAppStruct) error {

	err := a.MQClient.SendBuilderTopic(mqclient.TaskStruct{
		TaskBody: importApp,
		TaskType: "import_app",
		Topic:    mqclient.BuilderTopic,
	})
	if err != nil {
		logrus.Error("Failed to MQ Enqueue for ImportApp:", err)
		return err
	}
	logrus.Debugf("equeue mq build plugin from image success")

	return nil
}

func (a *AppAction) saveMetadata(tr *model.ExportAppStruct) error {
	retry := true
	// 创建应用组目录
	os.MkdirAll(tr.SourceDir, 0755)

	if tr.Body.Format == "helm_chart" {
		exportApp := fmt.Sprintf("%v-%v", a.appName, a.appVersion)
		exportPath := fmt.Sprintf("/wtdata/app/%s/%v/%v-helm/%v", tr.Body.Format, exportApp, exportApp, a.appName)
		os.MkdirAll(exportPath, 0755)
	}
	if tr.Body.Format == "yaml" {
		exportApp := fmt.Sprintf("%v-%v", a.appName, a.appVersion)
		exportPath := fmt.Sprintf("/wtdata/app/%s/%v/%v-yaml/%v", tr.Body.Format, exportApp, exportApp, a.appName)
		os.MkdirAll(exportPath, 0755)
	}

	// 写入元数据到文件
	if err := ioutil.WriteFile(fmt.Sprintf("%s/metadata.json", tr.SourceDir), []byte(tr.Body.GroupMetadata), 0644); err != nil {
		if retry && strings.Contains(err.Error(), "no such file or directory") {
			os.MkdirAll(tr.SourceDir, 0755)
			if err := ioutil.WriteFile(fmt.Sprintf("%s/metadata.json", tr.SourceDir), []byte(tr.Body.GroupMetadata), 0644); err != nil {
				logrus.Error("Failed to write metadata: ", err)
				return err
			}
		} else {
			logrus.Error("Failed to save metadata: ", err)
			return err
		}
	}

	return nil
}

// unicode2zh 将unicode转为中文，并去掉空格
func unicode2zh(uText string) (context string) {
	for i, char := range strings.Split(uText, `\\u`) {
		if i < 1 {
			context = char
			continue
		}

		length := len(char)
		if length > 3 {
			pre := char[:4]
			zh, err := strconv.ParseInt(pre, 16, 32)
			if err != nil {
				context += char
				continue
			}

			context += fmt.Sprintf("%c", zh)

			if length > 4 {
				context += char[4:]
			}
		}
	}

	context = re.ReplaceAllString(context, "")

	return context
}
