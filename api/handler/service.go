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
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/dustin/go-humanize"
	veleroversioned "github.com/vmware-tanzu/velero/pkg/generated/clientset/versioned"
	"github.com/wutong-paas/wutong/util/constants"

	"github.com/coreos/etcd/clientv3"
	"github.com/google/uuid"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	"github.com/pquerna/ffjson/ffjson"
	"github.com/sirupsen/logrus"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"github.com/wutong-paas/wutong/api/client/kube"
	"github.com/wutong-paas/wutong/api/client/prometheus"
	api_model "github.com/wutong-paas/wutong/api/model"
	apiutil "github.com/wutong-paas/wutong/api/util"
	"github.com/wutong-paas/wutong/api/util/bcode"
	"github.com/wutong-paas/wutong/chaos/parser"
	"github.com/wutong-paas/wutong/cmd/api/option"
	"github.com/wutong-paas/wutong/db"
	dberr "github.com/wutong-paas/wutong/db/errors"
	dbmodel "github.com/wutong-paas/wutong/db/model"
	"github.com/wutong-paas/wutong/event"
	gclient "github.com/wutong-paas/wutong/mq/client"
	"github.com/wutong-paas/wutong/pkg/generated/clientset/versioned"
	"github.com/wutong-paas/wutong/util"
	typesv1 "github.com/wutong-paas/wutong/worker/appm/types/v1"
	"github.com/wutong-paas/wutong/worker/client"
	"github.com/wutong-paas/wutong/worker/discover/model"
	"github.com/wutong-paas/wutong/worker/server"
	"github.com/wutong-paas/wutong/worker/server/pb"
	corev1 "k8s.io/api/core/v1"
	apiextclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apiserver/pkg/util/flushwriter"
	"k8s.io/client-go/kubernetes"
)

// ErrServiceNotClosed -
var ErrServiceNotClosed = errors.New("Service has not been closed")

// ServiceAction service act
type ServiceAction struct {
	MQClient      gclient.MQClient
	EtcdCli       *clientv3.Client
	statusCli     *client.AppRuntimeSyncClient
	prometheusCli prometheus.Interface
	conf          option.Config
	wutongClient  versioned.Interface
	kubeClient    kubernetes.Interface
	apiextClient  apiextclient.Interface
	veleroClient  veleroversioned.Interface
}

type dCfg struct {
	Type     string   `json:"type"`
	Servers  []string `json:"servers"`
	Key      string   `json:"key"`
	Username string   `json:"username"`
	Password string   `json:"password"`
}

// CreateManager create Manger
func CreateManager(conf option.Config,
	mqClient gclient.MQClient,
	etcdCli *clientv3.Client,
	statusCli *client.AppRuntimeSyncClient,
	prometheusCli prometheus.Interface,
	wutongClient versioned.Interface,
	kubeClient kubernetes.Interface,
	apiextClient apiextclient.Interface,
	veleroClient veleroversioned.Interface) *ServiceAction {
	return &ServiceAction{
		MQClient:      mqClient,
		EtcdCli:       etcdCli,
		statusCli:     statusCli,
		conf:          conf,
		prometheusCli: prometheusCli,
		wutongClient:  wutongClient,
		kubeClient:    kubeClient,
		apiextClient:  apiextClient,
		veleroClient:  veleroClient,
	}
}

// ServiceBuild service build
func (s *ServiceAction) ServiceBuild(tenantEnvID, serviceID string, r *api_model.BuildServiceStruct) error {
	eventID := r.Body.EventID
	logger := event.GetManager().GetLogger(eventID)
	defer event.CloseManager()
	service, err := db.GetManager().TenantEnvServiceDao().GetServiceByID(serviceID)
	db.GetManager().TenantEnvServiceDao().UpdateModel(service)
	if err != nil {
		return err
	}
	if r.Body.Kind == "" {
		r.Body.Kind = "source"
	}
	switch r.Body.Kind {
	case "build_from_image":
		if err := s.buildFromImage(r, service); err != nil {
			logger.Error("The image build application task failed to send: "+err.Error(), map[string]string{"step": "callback", "status": "failure"})
			return err
		}
		logger.Info("The mirror build application task successed to send ", map[string]string{"step": "image-service", "status": "starting"})
		return nil
	case "build_from_source_code":
		if err := s.buildFromSourceCode(r, service); err != nil {
			logger.Error("The source code build application task failed to send "+err.Error(), map[string]string{"step": "callback", "status": "failure"})
			return err
		}
		logger.Info("The source code build application task successed to send ", map[string]string{"step": "source-service", "status": "starting"})
		return nil
	case "build_from_market_image":
		if err := s.buildFromImage(r, service); err != nil {
			logger.Error("The cloud image build application task failed to send "+err.Error(), map[string]string{"step": "callback", "status": "failure"})
			return err
		}
		logger.Info("The cloud image build application task successed to send ", map[string]string{"step": "image-service", "status": "starting"})
		return nil
	case "build_from_market_slug":
		if err := s.buildFromMarketSlug(r, service); err != nil {
			logger.Error("The cloud slug build application task failed to send "+err.Error(), map[string]string{"step": "callback", "status": "failure"})
			return err
		}
		logger.Info("The cloud slug build application task successed to send ", map[string]string{"step": "image-service", "status": "starting"})
		return nil
	default:
		return fmt.Errorf("unexpect kind")
	}
}
func (s *ServiceAction) buildFromMarketSlug(r *api_model.BuildServiceStruct, service *dbmodel.TenantEnvServices) error {
	body := make(map[string]interface{})
	if r.Body.Operator == "" {
		body["operator"] = "define"
	} else {
		body["operator"] = r.Body.Operator
	}
	body["deploy_version"] = r.Body.DeployVersion
	body["event_id"] = r.Body.EventID
	body["action"] = r.Body.Action
	body["tenant_env_name"] = r.Body.TenantEnvName
	body["tenant_env_id"] = service.TenantEnvID
	body["service_id"] = service.ServiceID
	body["service_alias"] = r.Body.ServiceAlias
	body["slug_info"] = r.Body.SlugInfo

	topic := gclient.BuilderTopic
	if s.isWindowsService(service.ServiceID) {
		topic = gclient.WindowsBuilderTopic
	}
	return s.MQClient.SendBuilderTopic(gclient.TaskStruct{
		Topic:    topic,
		TaskType: "build_from_market_slug",
		TaskBody: body,
		Operator: r.Body.Operator,
	})
}

func (s *ServiceAction) buildFromImage(r *api_model.BuildServiceStruct, service *dbmodel.TenantEnvServices) error {
	dependIds, err := db.GetManager().TenantEnvServiceRelationDao().GetTenantEnvServiceRelations(service.ServiceID)
	if err != nil {
		return err
	}
	body := make(map[string]interface{})
	if r.Body.Operator == "" {
		body["operator"] = "define"
	} else {
		body["operator"] = r.Body.Operator
	}
	body["image"] = r.Body.ImageURL
	body["service_id"] = service.ServiceID
	body["deploy_version"] = r.Body.DeployVersion
	body["namespace"] = service.Namespace
	body["operator"] = r.Body.Operator
	body["event_id"] = r.Body.EventID
	body["tenant_env_name"] = r.Body.TenantEnvName
	body["service_alias"] = r.Body.ServiceAlias
	body["action"] = r.Body.Action
	body["dep_sids"] = dependIds
	body["code_from"] = "image_manual"
	if r.Body.User != "" && r.Body.Password != "" {
		body["user"] = r.Body.User
		body["password"] = r.Body.Password
	}
	topic := gclient.BuilderTopic
	if s.isWindowsService(service.ServiceID) {
		topic = gclient.WindowsBuilderTopic
	}
	return s.MQClient.SendBuilderTopic(gclient.TaskStruct{
		Topic:    topic,
		TaskType: "build_from_image",
		TaskBody: body,
		Operator: r.Body.Operator,
	})
}

func (s *ServiceAction) buildFromSourceCode(r *api_model.BuildServiceStruct, service *dbmodel.TenantEnvServices) error {
	logrus.Debugf("build_from_source_code")
	if r.Body.RepoURL == "" || r.Body.Branch == "" || r.Body.DeployVersion == "" || r.Body.EventID == "" {
		return fmt.Errorf("args error")
	}
	body := make(map[string]interface{})
	if r.Body.Operator == "" {
		body["operator"] = "define"
	} else {
		body["operator"] = r.Body.Operator
	}
	body["tenant_env_id"] = service.TenantEnvID
	body["service_id"] = service.ServiceID
	body["repo_url"] = r.Body.RepoURL
	body["action"] = r.Body.Action
	body["lang"] = r.Body.Lang
	body["runtime"] = r.Body.Runtime
	body["deploy_version"] = r.Body.DeployVersion
	body["event_id"] = r.Body.EventID
	body["envs"] = r.Body.ENVS
	body["tenant_env_name"] = r.Body.TenantEnvName
	body["branch"] = r.Body.Branch
	body["server_type"] = r.Body.ServerType
	body["service_alias"] = r.Body.ServiceAlias
	if r.Body.User != "" && r.Body.Password != "" {
		body["user"] = r.Body.User
		body["password"] = r.Body.Password
	}
	body["expire"] = 180
	topic := gclient.BuilderTopic
	if s.isWindowsService(service.ServiceID) {
		topic = gclient.WindowsBuilderTopic
	}
	return s.MQClient.SendBuilderTopic(gclient.TaskStruct{
		Topic:    topic,
		TaskType: "build_from_source_code",
		TaskBody: body,
		Operator: r.Body.Operator,
	})
}

func (s *ServiceAction) isWindowsService(serviceID string) bool {
	label, err := db.GetManager().TenantEnvServiceLabelDao().GetLabelByNodeSelectorKey(serviceID, "windows")
	if label == nil || err != nil {
		return false
	}
	return true
}

// AddLabel add labels
func (s *ServiceAction) AddLabel(l *api_model.LabelsStruct, serviceID string) error {

	tx := db.GetManager().Begin()
	defer func() {
		if r := recover(); r != nil {
			logrus.Errorf("Unexpected panic occurred, rollback transaction: %v", r)
			tx.Rollback()
		}
	}()
	//V5.2: do not support service type label
	for _, label := range l.Labels {
		labelModel := dbmodel.TenantEnvServiceLable{
			ServiceID:  serviceID,
			LabelKey:   label.LabelKey,
			LabelValue: label.LabelValue,
		}
		if err := db.GetManager().TenantEnvServiceLabelDaoTransactions(tx).AddModel(&labelModel); err != nil {
			tx.Rollback()
			return err
		}
	}
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return err
	}
	return nil
}

// UpdateLabel updates labels
func (s *ServiceAction) UpdateLabel(l *api_model.LabelsStruct, serviceID string) error {
	tx := db.GetManager().Begin()
	defer func() {
		if r := recover(); r != nil {
			logrus.Errorf("Unexpected panic occurred, rollback transaction: %v", r)
			tx.Rollback()
		}
	}()
	for _, label := range l.Labels {
		// delete old labels
		err := db.GetManager().TenantEnvServiceLabelDaoTransactions(tx).
			DelTenantEnvServiceLabelsByServiceIDKey(serviceID, label.LabelKey)
		if err != nil {
			logrus.Errorf("error deleting old labels: %v", err)
			tx.Rollback()
			return err
		}
		// V5.2 do not support service type label
		// add new labels
		labelModel := dbmodel.TenantEnvServiceLable{
			ServiceID:  serviceID,
			LabelKey:   label.LabelKey,
			LabelValue: label.LabelValue,
		}
		if err := db.GetManager().TenantEnvServiceLabelDaoTransactions(tx).AddModel(&labelModel); err != nil {
			logrus.Errorf("error adding new labels: %v", err)
			tx.Rollback()
			return err

		}
	}
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return err
	}
	return nil
}

// DeleteLabel deletes label
func (s *ServiceAction) DeleteLabel(l *api_model.LabelsStruct, serviceID string) error {
	tx := db.GetManager().Begin()
	defer func() {
		if r := recover(); r != nil {
			logrus.Errorf("Unexpected panic occurred, rollback transaction: %v", r)
			tx.Rollback()
		}
	}()
	for _, label := range l.Labels {
		err := db.GetManager().TenantEnvServiceLabelDaoTransactions(tx).
			DelTenantEnvServiceLabelsByServiceIDKeyValue(serviceID, label.LabelKey, label.LabelValue)
		if err != nil {
			logrus.Errorf("error deleting label: %v", err)
			tx.Rollback()
			return err
		}
	}
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return err
	}
	return nil
}

// StartStopService start service
func (s *ServiceAction) StartStopService(sss *api_model.StartStopStruct) error {
	services, err := db.GetManager().TenantEnvServiceDao().GetServiceByID(sss.ServiceID)
	if err != nil {
		logrus.Errorf("get service by id error, %v", err)
		return err
	}
	TaskBody := model.StopTaskBody{
		TenantEnvID:   sss.TenantEnvID,
		ServiceID:     sss.ServiceID,
		DeployVersion: services.DeployVersion,
		EventID:       sss.EventID,
	}
	err = s.MQClient.SendBuilderTopic(gclient.TaskStruct{
		TaskType: sss.TaskType,
		TaskBody: TaskBody,
		Topic:    gclient.WorkerTopic,
	})
	if err != nil {
		logrus.Errorf("equque mq error, %v", err)
		return err
	}
	logrus.Debugf("equeue mq startstop task success")
	return nil
}

// ServiceVertical vertical service
func (s *ServiceAction) ServiceVertical(ctx context.Context, vs *model.VerticalScalingTaskBody) error {
	service, err := db.GetManager().TenantEnvServiceDao().GetServiceByID(vs.ServiceID)
	if err != nil {
		logrus.Errorf("get service by id %s error, %s", vs.ServiceID, err)
		db.GetManager().ServiceEventDao().SetEventStatus(ctx, dbmodel.EventStatusFailure)
		return err
	}
	oldMemory := service.ContainerMemory
	oldCPU := service.ContainerCPU
	oldGPUType := service.ContainerGPUType
	oldGPU := service.ContainerGPU
	var rollback = func() {
		service.ContainerMemory = oldMemory
		service.ContainerCPU = oldCPU
		service.ContainerGPUType = oldGPUType
		service.ContainerGPU = oldGPU
		_ = db.GetManager().TenantEnvServiceDao().UpdateModel(service)
	}
	if vs.ContainerCPU != nil {
		service.ContainerCPU = *vs.ContainerCPU
	}
	if vs.ContainerMemory != nil {
		service.ContainerMemory = *vs.ContainerMemory
	}
	if vs.ContainerGPUType != nil {
		service.ContainerGPUType = *vs.ContainerGPUType
	}
	if vs.ContainerGPU != nil {
		service.ContainerGPU = *vs.ContainerGPU
	}
	// licenseInfo := license.ReadLicense()
	// if licenseInfo == nil || !licenseInfo.HaveFeature("GPU") {
	// 	service.ContainerGPU = 0
	// }
	if service.ContainerMemory == oldMemory && service.ContainerCPU == oldCPU && service.ContainerGPUType == oldGPUType && service.ContainerGPU == oldGPU {
		db.GetManager().ServiceEventDao().SetEventStatus(ctx, dbmodel.EventStatusSuccess)
		return nil
	}
	err = db.GetManager().TenantEnvServiceDao().UpdateModel(service)
	if err != nil {
		db.GetManager().ServiceEventDao().SetEventStatus(ctx, dbmodel.EventStatusFailure)
		logrus.Errorf("update service memory and cpu failure. %v", err)
		return fmt.Errorf("vertical service faliure:%s", err.Error())
	}
	err = s.MQClient.SendBuilderTopic(gclient.TaskStruct{
		TaskType: "vertical_scaling",
		TaskBody: vs,
		Topic:    gclient.WorkerTopic,
	})
	if err != nil {
		// roll back service
		rollback()
		logrus.Errorf("equque mq error, %v", err)
		db.GetManager().ServiceEventDao().SetEventStatus(ctx, dbmodel.EventStatusFailure)
		return err
	}
	logrus.Debugf("equeue mq vertical task success")
	return nil
}

// ServiceHorizontal Service Horizontal
func (s *ServiceAction) ServiceHorizontal(hs *model.HorizontalScalingTaskBody) error {
	service, err := db.GetManager().TenantEnvServiceDao().GetServiceByID(hs.ServiceID)
	if err != nil {
		logrus.Errorf("get service by id %s error, %s", hs.ServiceID, err)
		return err
	}

	// for rollback database
	oldReplicas := service.Replicas
	pods, err := s.statusCli.GetServicePods(service.ServiceID)
	if err != nil {
		logrus.Errorf("get service pods error: %v", err)
		return fmt.Errorf("horizontal service faliure:%s", err.Error())
	}
	if int32(len(pods.NewPods)) == hs.Replicas {
		return bcode.ErrHorizontalDueToNoChange
	}

	service.Replicas = int(hs.Replicas)
	err = db.GetManager().TenantEnvServiceDao().UpdateModel(service)
	if err != nil {
		logrus.Errorf("updtae service replicas failure. %v", err)
		return fmt.Errorf("horizontal service faliure:%s", err.Error())
	}

	var rollback = func() {
		service.Replicas = oldReplicas
		_ = db.GetManager().TenantEnvServiceDao().UpdateModel(service)
	}

	err = s.MQClient.SendBuilderTopic(gclient.TaskStruct{
		TaskType: "horizontal_scaling",
		TaskBody: hs,
		Topic:    gclient.WorkerTopic,
	})
	if err != nil {
		// roll back service
		rollback()
		logrus.Errorf("equque mq error, %v", err)
		return err
	}

	// if send task success, return nil
	logrus.Debugf("enqueue mq horizontal task success")
	return nil
}

// ServiceUpgrade service upgrade
func (s *ServiceAction) ServiceUpgrade(ru *model.RollingUpgradeTaskBody) error {
	services, err := db.GetManager().TenantEnvServiceDao().GetServiceByID(ru.ServiceID)
	if err != nil {
		logrus.Errorf("get service by id %s error %s", ru.ServiceID, err.Error())
		return err
	}
	version, err := db.GetManager().VersionInfoDao().GetVersionByDeployVersion(ru.NewDeployVersion, ru.ServiceID)
	if err != nil {
		logrus.Errorf("get service version by id %s version %s error, %s", ru.ServiceID, ru.NewDeployVersion, err.Error())
		return err
	}
	oldDeployVersion := services.DeployVersion
	var rollback = func() {
		services.DeployVersion = oldDeployVersion
		_ = db.GetManager().TenantEnvServiceDao().UpdateModel(services)
	}
	if version.FinalStatus != "success" {
		logrus.Warnf("deploy version %s is not build success,can not change deploy version in this upgrade event", ru.NewDeployVersion)
	} else {
		services.DeployVersion = ru.NewDeployVersion
		err = db.GetManager().TenantEnvServiceDao().UpdateModel(services)
		if err != nil {
			logrus.Errorf("update service deploy version error. %v", err)
			return fmt.Errorf("horizontal service faliure:%s", err.Error())
		}
	}
	err = s.MQClient.SendBuilderTopic(gclient.TaskStruct{
		TaskBody: ru,
		TaskType: "rolling_upgrade",
		Topic:    gclient.WorkerTopic,
	})
	if err != nil {
		// roll back service deploy version
		rollback()
		logrus.Errorf("equque upgrade message error, %v", err)
		return err
	}
	return nil
}

// ServiceCreate create service
func (s *ServiceAction) ServiceCreate(sc *api_model.ServiceStruct) error {
	jsonSC, err := ffjson.Marshal(sc)
	if err != nil {
		logrus.Errorf("trans service struct to json failed. %v", err)
		return err
	}
	var ts dbmodel.TenantEnvServices
	if err := ffjson.Unmarshal(jsonSC, &ts); err != nil {
		logrus.Errorf("trans json to tenant env service error, %v", err)
		return err
	}
	if ts.ServiceName == "" {
		ts.ServiceName = ts.ServiceAlias
	}
	if ts.ContainerCPU < 0 {
		ts.ContainerCPU = 0
	}
	if ts.ContainerMemory < 0 {
		ts.ContainerMemory = 0
	}
	if ts.ContainerGPU < 0 {
		ts.ContainerGPU = 0
	}
	if ts.K8sComponentName != "" {
		if db.GetManager().TenantEnvServiceDao().IsK8sComponentNameDuplicate(ts.AppID, ts.ServiceID, ts.K8sComponentName) {
			return bcode.ErrK8sComponentNameExists
		}
	}
	ts.UpdateTime = time.Now()
	var (
		ports         = sc.PortsInfo
		envs          = sc.EnvsInfo
		volumns       = sc.VolumesInfo
		dependVolumes = sc.DepVolumesInfo
		dependIds     = sc.DependIDs
		probes        = sc.ComponentProbes
		monitors      = sc.ComponentMonitors
		httpRules     = sc.HTTPRules
		tcpRules      = sc.TCPRules
	)
	ts.AppID = sc.AppID
	ts.DeployVersion = ""
	tx := db.GetManager().Begin()
	defer func() {
		if r := recover(); r != nil {
			logrus.Errorf("Unexpected panic occurred, rollback transaction: %v", r)
			tx.Rollback()
		}
	}()
	//create app
	if err := db.GetManager().TenantEnvServiceDaoTransactions(tx).AddModel(&ts); err != nil {
		logrus.Errorf("add service error, %v", err)
		tx.Rollback()
		return err
	}
	//set app envs
	if len(envs) > 0 {
		var batchEnvs []*dbmodel.TenantEnvServiceEnvVar
		for _, env := range envs {
			env := env
			env.ServiceID = ts.ServiceID
			env.TenantEnvID = ts.TenantEnvID
			batchEnvs = append(batchEnvs, &env)
		}
		if err := db.GetManager().TenantEnvServiceEnvVarDaoTransactions(tx).CreateOrUpdateEnvsInBatch(batchEnvs); err != nil {
			logrus.Errorf("batch add env error, %v", err)
			tx.Rollback()
			return err
		}
	}
	//set app port
	if len(ports) > 0 {
		var batchPorts []*dbmodel.TenantEnvServicesPort
		for _, port := range ports {
			port := port
			port.ServiceID = ts.ServiceID
			port.TenantEnvID = ts.TenantEnvID
			batchPorts = append(batchPorts, &port)
		}
		if err := db.GetManager().TenantEnvServicesPortDaoTransactions(tx).CreateOrUpdatePortsInBatch(batchPorts); err != nil {
			logrus.Errorf("batch add port error, %v", err)
			tx.Rollback()
			return err
		}
	}
	//set app volumns
	if len(volumns) > 0 {
		localPath := os.Getenv("LOCAL_DATA_PATH")
		sharePath := os.Getenv("SHARE_DATA_PATH")
		if localPath == "" {
			localPath = "/wtlocaldata"
		}
		if sharePath == "" {
			sharePath = "/wtdata"
		}

		for _, volumn := range volumns {
			v := dbmodel.TenantEnvServiceVolume{
				ServiceID:      ts.ServiceID,
				Category:       volumn.Category,
				VolumeType:     volumn.VolumeType,
				VolumeName:     volumn.VolumeName,
				HostPath:       volumn.HostPath,
				VolumePath:     volumn.VolumePath,
				IsReadOnly:     volumn.IsReadOnly,
				VolumeCapacity: volumn.VolumeCapacity,
				// AccessMode 读写模式（Important! A volume can only be mounted using one access mode at a time, even if it supports many. For example, a GCEPersistentDisk can be mounted as ReadWriteOnce by a single node or ReadOnlyMany by many nodes, but not at the same time. #https://kubernetes.io/docs/concepts/storage/persistent-volumes/#access-modes）
				AccessMode: volumn.AccessMode,
				// SharePolicy 共享模式
				SharePolicy: volumn.SharePolicy,
				// BackupPolicy 备份策略
				BackupPolicy: volumn.BackupPolicy,
				// ReclaimPolicy 回收策略
				ReclaimPolicy: volumn.ReclaimPolicy,
				// AllowExpansion 是否支持扩展
				AllowExpansion: volumn.AllowExpansion,
				// VolumeProviderName 使用的存储驱动别名
				VolumeProviderName: volumn.VolumeProviderName,
			}
			v.ServiceID = ts.ServiceID
			if volumn.VolumeType == "" {
				v.VolumeType = dbmodel.ShareFileVolumeType.String()
			}
			if volumn.HostPath == "" {
				//step 1 设置主机目录
				switch volumn.VolumeType {
				//共享文件存储
				case dbmodel.ShareFileVolumeType.String():
					v.HostPath = fmt.Sprintf("%s/tenantEnv/%s/service/%s%s", sharePath, sc.TenantEnvID, ts.ServiceID, volumn.VolumePath)
				//本地文件存储
				case dbmodel.LocalVolumeType.String():
					if !dbmodel.ServiceType(sc.ExtendMethod).IsState() {
						tx.Rollback()
						return apiutil.CreateAPIHandleError(400, fmt.Errorf("local volume type only support state component"))
					}
					v.HostPath = fmt.Sprintf("%s/tenantEnv/%s/service/%s%s", localPath, sc.TenantEnvID, ts.ServiceID, volumn.VolumePath)
				case dbmodel.ConfigFileVolumeType.String(), dbmodel.MemoryFSVolumeType.String():
					logrus.Debug("simple volume type : ", volumn.VolumeType)
				default:
					if !dbmodel.ServiceType(sc.ExtendMethod).IsState() {
						tx.Rollback()
						return apiutil.CreateAPIHandleError(400, fmt.Errorf("custom volume type only support state component"))
					}
				}
			}
			if volumn.VolumeName == "" {
				v.VolumeName = uuid.New().String()
			}
			if err := db.GetManager().TenantEnvServiceVolumeDaoTransactions(tx).AddModel(&v); err != nil {
				logrus.Errorf("add volumn %v error, %v", volumn.HostPath, err)
				tx.Rollback()
				return err
			}
			if volumn.FileContent != "" {
				cf := &dbmodel.TenantEnvServiceConfigFile{
					ServiceID:   sc.ServiceID,
					VolumeName:  volumn.VolumeName,
					FileContent: volumn.FileContent,
				}
				if err := db.GetManager().TenantEnvServiceConfigFileDaoTransactions(tx).AddModel(cf); err != nil {
					tx.Rollback()
					return apiutil.CreateAPIHandleErrorFromDBError("error creating config file", err)
				}
			}
		}
	}
	//set app dependVolumes
	if len(dependVolumes) > 0 {
		for _, depVolume := range dependVolumes {
			depVolume.ServiceID = ts.ServiceID
			depVolume.TenantEnvID = ts.TenantEnvID
			volume, err := db.GetManager().TenantEnvServiceVolumeDao().GetVolumeByServiceIDAndName(depVolume.DependServiceID, depVolume.VolumeName)
			if err != nil {
				tx.Rollback()
				return fmt.Errorf("find volume %s error %s", depVolume.VolumeName, err.Error())
			}
			depVolume.VolumeType = volume.VolumeType
			depVolume.HostPath = volume.HostPath
			if err := db.GetManager().TenantEnvServiceMountRelationDaoTransactions(tx).AddModel(&depVolume); err != nil {
				tx.Rollback()
				return fmt.Errorf("add dep volume %s error %s", depVolume.VolumeName, err.Error())
			}
		}
	}
	//set app depends
	if len(dependIds) > 0 {
		for _, id := range dependIds {
			if err := db.GetManager().TenantEnvServiceRelationDaoTransactions(tx).AddModel(&id); err != nil {
				logrus.Errorf("add depend_id %v error, %v", id.DependServiceID, err)
				tx.Rollback()
				return err
			}
		}
	}
	//set app label
	if sc.OSType == "windows" {
		if err := db.GetManager().TenantEnvServiceLabelDaoTransactions(tx).AddModel(&dbmodel.TenantEnvServiceLable{
			ServiceID:  ts.ServiceID,
			LabelKey:   dbmodel.LabelKeyNodeSelector,
			LabelValue: sc.OSType,
		}); err != nil {
			logrus.Errorf("add label %s=%s  %v error, %v", dbmodel.LabelKeyNodeSelector, sc.OSType, ts.ServiceID, err)
			tx.Rollback()
			return err
		}
	}
	// sc.Endpoints can't be nil
	// sc.Endpoints.Discovery or sc.Endpoints.Static can't be nil
	if sc.Kind == dbmodel.ServiceKindThirdParty.String() { // TODO: validate request data
		if sc.Endpoints == nil {
			tx.Rollback()
			return fmt.Errorf("endpoints can not be empty for third-party service")
		}
		if sc.Endpoints.Kubernetes != nil {
			c := &dbmodel.ThirdPartySvcDiscoveryCfg{
				ServiceID:   sc.ServiceID,
				Type:        string(dbmodel.DiscorveryTypeKubernetes),
				Namespace:   sc.Endpoints.Kubernetes.Namespace,
				ServiceName: sc.Endpoints.Kubernetes.ServiceName,
			}
			if err := db.GetManager().ThirdPartySvcDiscoveryCfgDaoTransactions(tx).
				AddModel(c); err != nil {
				logrus.Errorf("error saving discover center configuration: %v", err)
				tx.Rollback()
				return err
			}
		}
		if sc.Endpoints.Static != nil {
			for _, o := range sc.Endpoints.Static {
				ep := &dbmodel.Endpoint{
					ServiceID: sc.ServiceID,
					UUID:      util.NewUUID(),
				}
				address := o
				port := 0
				prefix := ""
				if strings.HasPrefix(address, "https://") {
					address = strings.Split(address, "https://")[1]
					prefix = "https://"
				}
				if strings.HasPrefix(address, "http://") {
					address = strings.Split(address, "http://")[1]
					prefix = "http://"
				}
				if strings.Contains(address, ":") {
					addressL := strings.Split(address, ":")
					address = addressL[0]
					port, _ = strconv.Atoi(addressL[1])
				}
				ep.IP = prefix + address
				ep.Port = port

				logrus.Debugf("add new endpoint: %v", ep)

				if err := db.GetManager().EndpointsDaoTransactions(tx).AddModel(ep); err != nil {
					tx.Rollback()
					logrus.Errorf("error saving o endpoint: %v", err)
					return err
				}
			}
		}
	}
	if len(probes) > 0 {
		for _, pb := range probes {
			probe := s.convertProbeModel(&pb, ts.ServiceID)
			if err := db.GetManager().ServiceProbeDaoTransactions(tx).AddModel(probe); err != nil {
				logrus.Errorf("add probe %v error, %v", probe.ProbeID, err)
				tx.Rollback()
				return err
			}
		}
	}
	if len(monitors) > 0 {
		for _, m := range monitors {
			monitor := dbmodel.TenantEnvServiceMonitor{
				Name:            m.Name,
				TenantEnvID:     ts.TenantEnvID,
				ServiceID:       ts.ServiceID,
				ServiceShowName: m.ServiceShowName,
				Port:            m.Port,
				Path:            m.Path,
				Interval:        m.Interval,
			}
			if err := db.GetManager().TenantEnvServiceMonitorDaoTransactions(tx).AddModel(&monitor); err != nil {
				logrus.Errorf("add monitor %v error, %v", monitor.Name, err)
				tx.Rollback()
				return err
			}
		}
	}
	if len(httpRules) > 0 {
		for _, httpRule := range httpRules {
			if err := GetGatewayHandler().CreateHTTPRule(tx, &httpRule); err != nil {
				logrus.Errorf("add service http rule error %v", err)
				tx.Rollback()
				return err
			}
		}
	}
	if len(tcpRules) > 0 {
		for _, tcpRule := range tcpRules {
			if GetGatewayHandler().TCPIPPortExists(tcpRule.IP, tcpRule.Port) {
				logrus.Debugf("tcp rule %v:%v exists", tcpRule.IP, tcpRule.Port)
				continue
			}
			if err := GetGatewayHandler().CreateTCPRule(tx, &tcpRule); err != nil {
				logrus.Errorf("add service tcp rule error %v", err)
				tx.Rollback()
				return err
			}
		}
	}
	labelModel := dbmodel.TenantEnvServiceLable{
		ServiceID:  ts.ServiceID,
		LabelKey:   dbmodel.LabelKeyServiceType,
		LabelValue: util.StatelessServiceType,
	}
	if ts.IsState() {
		labelModel.LabelValue = util.StatefulServiceType
	}
	if err := db.GetManager().TenantEnvServiceLabelDaoTransactions(tx).AddModel(&labelModel); err != nil {
		tx.Rollback()
		return err
	}

	// TODO: create default probe for third-party service.
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return err
	}
	logrus.Debugf("create a new app %s success", ts.ServiceAlias)

	return nil
}

func (s *ServiceAction) convertProbeModel(req *api_model.ServiceProbe, serviceID string) *dbmodel.TenantEnvServiceProbe {
	return &dbmodel.TenantEnvServiceProbe{
		ServiceID:          serviceID,
		Cmd:                req.Cmd,
		FailureThreshold:   req.FailureThreshold,
		HTTPHeader:         req.HTTPHeader,
		InitialDelaySecond: req.InitialDelaySecond,
		IsUsed:             &req.IsUsed,
		Mode:               req.Mode,
		Path:               req.Path,
		PeriodSecond:       req.PeriodSecond,
		Port:               req.Port,
		ProbeID:            req.ProbeID,
		Scheme:             req.Scheme,
		SuccessThreshold:   req.SuccessThreshold,
		TimeoutSecond:      req.TimeoutSecond,
		FailureAction:      req.FailureAction,
	}
}

// ServiceUpdate update service
func (s *ServiceAction) ServiceUpdate(sc map[string]interface{}) error {
	ts, err := db.GetManager().TenantEnvServiceDao().GetServiceByID(sc["service_id"].(string))
	if err != nil {
		return err
	}
	if memory, ok := sc["container_memory"].(int); ok && memory >= 0 {
		ts.ContainerMemory = memory
	}
	if cpu, ok := sc["container_cpu"].(int); ok && cpu >= 0 {
		ts.ContainerCPU = cpu
	}
	if gpuType, ok := sc["container_gpu_type"].(string); ok {
		ts.ContainerGPUType = gpuType
	}
	if gpu, ok := sc["container_gpu"].(int); ok {
		ts.ContainerCPU = gpu
	}
	if name, ok := sc["service_name"].(string); ok && name != "" {
		ts.ServiceName = name
	}
	if appID, ok := sc["app_id"].(string); ok && appID != "" {
		ts.AppID = appID
	}
	if k8sComponentName, ok := sc["k8s_component_name"].(string); ok && k8sComponentName != "" {
		if db.GetManager().TenantEnvServiceDao().IsK8sComponentNameDuplicate(ts.AppID, ts.ServiceID, k8sComponentName) {
			return bcode.ErrK8sComponentNameExists
		}
		ts.K8sComponentName = k8sComponentName
	}
	if sc["extend_method"] != nil {
		extendMethod := sc["extend_method"].(string)
		ts.ExtendMethod = extendMethod
		// if component replicas is more than 1, so can't change service type to singleton
		if ts.Replicas > 1 && ts.IsSingleton() {
			err := fmt.Errorf("service[%s] replicas > 1, can't change service typ to stateless_singleton", ts.ServiceAlias)
			return err
		}
		volumes, err := db.GetManager().TenantEnvServiceVolumeDao().GetTenantEnvServiceVolumesByServiceID(ts.ServiceID)
		if err != nil {
			return err
		}
		for _, vo := range volumes {
			if vo.VolumeType == dbmodel.ShareFileVolumeType.String() || vo.VolumeType == dbmodel.MemoryFSVolumeType.String() {
				continue
			}
			if vo.VolumeType == dbmodel.LocalVolumeType.String() && !ts.IsState() {
				err := fmt.Errorf("service[%s] has local volume type, can't change type to stateless", ts.ServiceAlias)
				return err
			}
			// if component use volume, what it accessMode is rwo, can't change volume type to stateless
			if vo.AccessMode == "RWO" && !ts.IsState() {
				err := fmt.Errorf("service[%s] volume[%s] access_mode is RWO, can't change type to stateless", ts.ServiceAlias, vo.VolumeName)
				return err
			}
		}
		ts.ExtendMethod = extendMethod
		ts.ServiceType = extendMethod
	}
	//update component
	if err := db.GetManager().TenantEnvServiceDao().UpdateModel(ts); err != nil {
		logrus.Errorf("update service error, %v", err)
		return err
	}
	return nil
}

// LanguageSet language set
func (s *ServiceAction) LanguageSet(langS *api_model.LanguageSet) error {
	logrus.Debugf("service id is %s, language is %s", langS.ServiceID, langS.Language)
	services, err := db.GetManager().TenantEnvServiceDao().GetServiceByID(langS.ServiceID)
	if err != nil {
		logrus.Errorf("get service by id error, %v, %v", services, err)
		return err
	}
	if langS.Language == "java" {
		services.ContainerMemory = 512
		if err := db.GetManager().TenantEnvServiceDao().UpdateModel(services); err != nil {
			logrus.Errorf("update tenant env service error %v", err)
			return err
		}
	}
	return nil
}

// GetService get service(s)
func (s *ServiceAction) GetService(tenantEnvID string) ([]*dbmodel.TenantEnvServices, error) {
	services, err := db.GetManager().TenantEnvServiceDao().GetServicesAllInfoByTenantEnvID(tenantEnvID)
	if err != nil {
		logrus.Errorf("get service by id error, %v, %v", services, err)
		return nil, err
	}
	var serviceIDs []string
	for _, s := range services {
		serviceIDs = append(serviceIDs, s.ServiceID)
	}
	status := s.statusCli.GetStatuss(strings.Join(serviceIDs, ","))
	for _, s := range services {
		if status, ok := status[s.ServiceID]; ok {
			s.CurStatus = status
		}
	}
	return services, nil
}

// GetServicesByAppID get service(s) by appID
func (s *ServiceAction) GetServicesByAppID(appID string, page, pageSize int) (*api_model.ListServiceResponse, error) {
	var resp api_model.ListServiceResponse
	services, total, err := db.GetManager().TenantEnvServiceDao().GetServicesInfoByAppID(appID, page, pageSize)
	if err != nil {
		logrus.Errorf("get service by application id error, %v, %v", services, err)
		return nil, err
	}
	var serviceIDs []string
	for _, s := range services {
		serviceIDs = append(serviceIDs, s.ServiceID)
	}
	status := s.statusCli.GetStatuss(strings.Join(serviceIDs, ","))
	for _, s := range services {
		if status, ok := status[s.ServiceID]; ok {
			s.CurStatus = status
		}
	}
	if services != nil {
		resp.Services = services
	} else {
		resp.Services = make([]*dbmodel.TenantEnvServices, 0)
	}

	resp.Page = page
	resp.Total = total
	resp.PageSize = pageSize
	return &resp, nil
}

// GetPagedTenantEnvRes get pagedTenantEnvServiceRes(s)
func (s *ServiceAction) GetPagedTenantEnvRes(offset, len int) ([]*api_model.TenantEnvResource, int, error) {
	allstatus := s.statusCli.GetAllStatus()
	var serviceIDs []string
	for k, v := range allstatus {
		if !s.statusCli.IsClosedStatus(v) {
			serviceIDs = append(serviceIDs, k)
		}
	}
	services, count, err := db.GetManager().TenantEnvServiceDao().GetPagedTenantEnvService(offset, len, serviceIDs)
	if err != nil {
		logrus.Errorf("get service by id error, %v, %v", services, err)
		return nil, count, err
	}
	var result []*api_model.TenantEnvResource
	for _, v := range services {
		var res api_model.TenantEnvResource
		res.UUID, _ = v["tenant_env"].(string)
		res.Name, _ = v["tenant_env_name"].(string)
		res.AllocatedCPU, _ = v["capcpu"].(int)
		res.AllocatedMEM, _ = v["capmem"].(int)
		res.UsedCPU, _ = v["usecpu"].(int)
		res.UsedMEM, _ = v["usemem"].(int)
		result = append(result, &res)
	}
	return result, count, nil
}

// GetTenantEnvRes get pagedTenantEnvServiceRes(s)
func (s *ServiceAction) GetTenantEnvRes(uuid string) (*api_model.TenantEnvResource, error) {
	if logrus.IsLevelEnabled(logrus.DebugLevel) {
		defer util.Elapsed("[ServiceAction] get tenant env resource")()
	}

	tenantEnv, err := db.GetManager().TenantEnvDao().GetTenantEnvByUUID(uuid)
	if err != nil {
		logrus.Errorf("get tenant env %s info failure %v", uuid, err.Error())
		return nil, err
	}
	services, err := db.GetManager().TenantEnvServiceDao().GetServicesByTenantEnvID(uuid)
	if err != nil {
		logrus.Errorf("get service by id error, %v, %v", services, err.Error())
		return nil, err
	}
	var serviceIDs string
	var AllocatedCPU, AllocatedMEM int
	for _, ser := range services {
		if serviceIDs == "" {
			serviceIDs += ser.ServiceID
		} else {
			serviceIDs += "," + ser.ServiceID
		}
		AllocatedCPU += ser.ContainerCPU * ser.Replicas
		AllocatedMEM += ser.ContainerMemory * ser.Replicas
	}
	tenantEnvResUesd, err := s.statusCli.GetTenantEnvResource(uuid)
	if err != nil {
		logrus.Errorf("get tenant env %s resource failure %s", uuid, err.Error())
	}
	disks := GetServicesDiskDeprecated(strings.Split(serviceIDs, ","), s.prometheusCli)
	var value float64
	for _, v := range disks {
		value += v
	}
	var res api_model.TenantEnvResource
	res.UUID = uuid
	res.Name = tenantEnv.Name
	res.AllocatedCPU = AllocatedCPU
	res.AllocatedMEM = AllocatedMEM
	if tenantEnvResUesd != nil {
		res.UsedCPU = int(tenantEnvResUesd.CpuRequest)
		res.UsedMEM = int(tenantEnvResUesd.MemoryRequest)
	}
	res.UsedDisk = value
	return &res, nil
}

// // GetTenantEnvMemoryCPU get pagedTenantEnvServiceRes(s)
// func (s *ServiceAction) GetAllocableResources(tenantEnvID string) (*api_model.TenantEnvResource, error) {
// 	if logrus.IsLevelEnabled(logrus.DebugLevel) {
// 		defer util.Elapsed("[ServiceAction] get allocable resources")()
// 	}

// 	tenantEnv, err := db.GetManager().TenantEnvDao().GetTenantEnvByUUID(tenantEnvID)
// 	if err != nil {
// 		return nil, err
// 	}

// 	services, err := db.GetManager().TenantEnvServiceDao().GetServicesByTenantEnvID(tenantEnvID)
// 	if err != nil {
// 		logrus.Errorf("get service by id error, %v, %v", services, err.Error())
// 		return nil, err
// 	}

// 	var serviceIDs string
// 	var allocatedCPU, allocatedMEM int
// 	for _, svc := range services {
// 		allocatedCPU += svc.ContainerCPU * svc.Replicas
// 		allocatedMEM += svc.ContainerMemory * svc.Replicas
// 	}
// 	usedResource, err := s.statusCli.GetTenantEnvResource(tenantEnvID)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return &res, nil
// }

// GetServicesDiskDeprecated get service disk
//
// Deprecated
func GetServicesDiskDeprecated(ids []string, prometheusCli prometheus.Interface) map[string]float64 {
	if logrus.IsLevelEnabled(logrus.DebugLevel) {
		defer util.Elapsed("[GetServicesDiskDeprecated] get tenant env resource")()
	}

	result := make(map[string]float64)
	//query disk used in prometheus
	query := fmt.Sprintf(`max(app_resource_appfs{service_id=~"%s"}) by(service_id)`, strings.Join(ids, "|"))
	metric := prometheusCli.GetMetric(query, time.Now())
	for _, re := range metric.MetricData.MetricValues {
		var serviceID = re.Metadata["service_id"]
		if re.Sample != nil {
			result[serviceID] = re.Sample.Value()
		}
	}
	return result
}

// CodeCheck code check
func (s *ServiceAction) CodeCheck(c *api_model.CheckCodeStruct) error {
	err := s.MQClient.SendBuilderTopic(gclient.TaskStruct{
		TaskType: "code_check",
		TaskBody: c.Body,
		Topic:    gclient.BuilderTopic,
	})
	if err != nil {
		logrus.Errorf("equque mq error, %v", err)
		return err
	}
	return nil
}

// ServiceDepend service depend
func (s *ServiceAction) ServiceDepend(action string, ds *api_model.DependService) error {
	switch action {
	case "add":
		tsr := &dbmodel.TenantEnvServiceRelation{
			TenantEnvID:       ds.TenantEnvID,
			ServiceID:         ds.ServiceID,
			DependServiceID:   ds.DepServiceID,
			DependServiceType: ds.DepServiceType,
			DependOrder:       1,
		}
		if err := db.GetManager().TenantEnvServiceRelationDao().AddModel(tsr); err != nil {
			logrus.Errorf("add depend error, %v", err)
			if err == dberr.ErrRecordAlreadyExist {
				return nil
			}
			return err
		}
	case "delete":
		logrus.Debugf("serviceid is %v, depid is %v", ds.ServiceID, ds.DepServiceID)
		if err := db.GetManager().TenantEnvServiceRelationDao().DeleteRelationByDepID(ds.ServiceID, ds.DepServiceID); err != nil {
			logrus.Errorf("delete depend error, %v", err)
			return err
		}
	}
	return nil
}

// EnvAttr env attr
func (s *ServiceAction) EnvAttr(action string, at *dbmodel.TenantEnvServiceEnvVar) error {
	switch action {
	case "add":
		if err := db.GetManager().TenantEnvServiceEnvVarDao().AddModel(at); err != nil {
			if err == dberr.ErrRecordAlreadyExist {
				if err = db.GetManager().TenantEnvServiceEnvVarDao().UpdateModel(at); err == nil {
					return nil
				}
			}
			logrus.Errorf("add env %v error, %v", at.AttrName, err)
			return err
		}
	case "delete":
		if err := db.GetManager().TenantEnvServiceEnvVarDao().DeleteModel(at.ServiceID, at.AttrName); err != nil {
			if err == gorm.ErrRecordNotFound {
				return nil
			}

			logrus.Errorf("delete env %v error, %v", at.AttrName, err)
			return err
		}
	case "update":
		if err := db.GetManager().TenantEnvServiceEnvVarDao().UpdateModel(at); err != nil {
			logrus.Errorf("update env %v error,%v", at.AttrName, err)
			return err
		}
	}
	return nil
}

// CreatePorts -
func (s *ServiceAction) CreatePorts(tenantEnvID, serviceID string, vps *api_model.ServicePorts) error {
	tx := db.GetManager().Begin()
	defer func() {
		if r := recover(); r != nil {
			logrus.Errorf("Unexpected panic occurred, rollback transaction: %v", r)
			tx.Rollback()
		}
	}()

	for _, vp := range vps.Port {
		// make sure K8sServiceName is unique
		if vp.K8sServiceName != "" {
			port, err := db.GetManager().TenantEnvServicesPortDao().GetByTenantEnvAndName(tenantEnvID, vp.K8sServiceName)
			if err != nil && err != gorm.ErrRecordNotFound {
				tx.Rollback()
				return err
			}
			if port != nil {
				tx.Rollback()
				return bcode.ErrK8sServiceNameExists
			}
		}

		var vpD dbmodel.TenantEnvServicesPort
		vpD.ServiceID = serviceID
		vpD.TenantEnvID = tenantEnvID
		vpD.IsInnerService = &vp.IsInnerService
		vpD.IsOuterService = &vp.IsOuterService
		vpD.ContainerPort = vp.ContainerPort
		vpD.MappingPort = vp.MappingPort
		vpD.Protocol = vp.Protocol
		vpD.PortAlias = vp.PortAlias
		vpD.K8sServiceName = vp.K8sServiceName
		if err := db.GetManager().TenantEnvServicesPortDaoTransactions(tx).AddModel(&vpD); err != nil {
			tx.Rollback()
			return err
		}
	}

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return err
	}

	return nil
}

func (s *ServiceAction) deletePorts(componentID string, ports *api_model.ServicePorts) error {
	return db.GetManager().DB().Transaction(func(tx *gorm.DB) error {
		for _, port := range ports.Port {
			if err := db.GetManager().TenantEnvServicesPortDaoTransactions(tx).DeleteModel(componentID, port.ContainerPort); err != nil {
				return err
			}

			// delete related ingress rules
			if err := GetGatewayHandler().DeleteIngressRulesByComponentPort(tx, componentID, port.ContainerPort); err != nil {
				return err
			}
		}

		return nil
	})
}

// SyncComponentPorts -
func (s *ServiceAction) SyncComponentPorts(tx *gorm.DB, app *dbmodel.Application, components []*api_model.Component) error {
	var (
		componentIDs []string
		ports        []*dbmodel.TenantEnvServicesPort
	)
	for _, component := range components {
		if component.Ports == nil {
			continue
		}
		componentIDs = append(componentIDs, component.ComponentBase.ComponentID)
		for _, port := range component.Ports {
			ports = append(ports, port.DbModel(app.TenantEnvID, component.ComponentBase.ComponentID))
		}
	}
	if err := db.GetManager().TenantEnvServicesPortDaoTransactions(tx).DeleteByComponentIDs(componentIDs); err != nil {
		return err
	}
	return db.GetManager().TenantEnvServicesPortDaoTransactions(tx).CreateOrUpdatePortsInBatch(ports)
}

// PortVar port var
func (s *ServiceAction) PortVar(action, tenantEnvID, serviceID string, vps *api_model.ServicePorts, oldPort int) error {
	crt, err := db.GetManager().TenantEnvServicePluginRelationDao().CheckSomeModelPluginByServiceID(
		serviceID,
		dbmodel.InBoundNetPlugin,
	)
	if err != nil {
		return err
	}
	switch action {
	case "delete":
		return s.deletePorts(serviceID, vps)
	case "update":
		tx := db.GetManager().Begin()
		defer func() {
			if r := recover(); r != nil {
				logrus.Errorf("Unexpected panic occurred, rollback transaction: %v", r)
				tx.Rollback()
			}
		}()
		for _, vp := range vps.Port {
			//port更新单个请求
			if oldPort == 0 {
				oldPort = vp.ContainerPort
			}
			vpD, err := db.GetManager().TenantEnvServicesPortDao().GetPort(serviceID, oldPort)
			if err != nil {
				tx.Rollback()
				return err
			}
			// make sure K8sServiceName is unique
			if vp.K8sServiceName != "" {
				port, err := db.GetManager().TenantEnvServicesPortDao().GetByTenantEnvAndName(tenantEnvID, vp.K8sServiceName)
				if err != nil && err != gorm.ErrRecordNotFound {
					tx.Rollback()
					return err
				}
				if port != nil && vpD.K8sServiceName != vp.K8sServiceName {
					tx.Rollback()
					return bcode.ErrK8sServiceNameExists
				}
			}

			vpD.ServiceID = serviceID
			vpD.TenantEnvID = tenantEnvID
			vpD.IsInnerService = &vp.IsInnerService
			vpD.IsOuterService = &vp.IsOuterService
			vpD.ContainerPort = vp.ContainerPort
			vpD.MappingPort = vp.MappingPort
			vpD.Protocol = vp.Protocol
			vpD.PortAlias = vp.PortAlias
			vpD.K8sServiceName = vp.K8sServiceName
			if err := db.GetManager().TenantEnvServicesPortDaoTransactions(tx).UpdateModel(vpD); err != nil {
				logrus.Errorf("update port var error, %v", err)
				tx.Rollback()
				return err
			}
			if crt {
				pluginPort, err := db.GetManager().TenantEnvServicesStreamPluginPortDao().GetPluginMappingPortByServiceIDAndContainerPort(
					serviceID,
					dbmodel.InBoundNetPlugin,
					oldPort,
				)
				goon := true
				if err != nil {
					if strings.Contains(err.Error(), "record not found") {
						goon = false
					} else {
						logrus.Errorf("get plugin mapping port error:(%s)", err)
						tx.Rollback()
						return err
					}
				}
				if goon {
					pluginPort.ContainerPort = vp.ContainerPort
					if err := db.GetManager().TenantEnvServicesStreamPluginPortDaoTransactions(tx).UpdateModel(pluginPort); err != nil {
						logrus.Errorf("update plugin mapping port error:(%s)", err)
						tx.Rollback()
						return err
					}
				}
			}
		}
		if err := tx.Commit().Error; err != nil {
			tx.Rollback()
			logrus.Debugf("commit update port error, %v", err)
			return err
		}
	}
	return nil
}

// PortOuter 端口对外服务操作
func (s *ServiceAction) PortOuter(tenantEnvName, serviceID string, containerPort int,
	servicePort *api_model.ServicePortInnerOrOuter) (*dbmodel.TenantEnvServiceLBMappingPort, string, error) {
	p, err := db.GetManager().TenantEnvServicesPortDao().GetPort(serviceID, containerPort)
	if err != nil {
		return nil, "", fmt.Errorf("find service port error:%s", err.Error())
	}
	_, err = db.GetManager().TenantEnvServiceDao().GetServiceByID(serviceID)
	if err != nil {
		return nil, "", fmt.Errorf("find service error:%s", err.Error())
	}
	hasUpStream, err := db.GetManager().TenantEnvServicePluginRelationDao().CheckSomeModelPluginByServiceID(
		serviceID,
		dbmodel.InBoundNetPlugin,
	)
	if err != nil {
		return nil, "", fmt.Errorf("get plugin relations error: %s", err.Error())
	}
	//if stream 创建vs端口
	vsPort := &dbmodel.TenantEnvServiceLBMappingPort{}
	switch servicePort.Body.Operation {
	case "close":
		if *p.IsOuterService { //如果端口已经开了对外
			falsev := false
			p.IsOuterService = &falsev
			tx := db.GetManager().Begin()
			defer func() {
				if r := recover(); r != nil {
					logrus.Errorf("Unexpected panic occurred, rollback transaction: %v", r)
					tx.Rollback()
				}
			}()
			if err = db.GetManager().TenantEnvServicesPortDaoTransactions(tx).UpdateModel(p); err != nil {
				tx.Rollback()
				return nil, "", err
			}

			if hasUpStream {
				pluginPort, err := db.GetManager().TenantEnvServicesStreamPluginPortDao().GetPluginMappingPortByServiceIDAndContainerPort(
					serviceID,
					dbmodel.InBoundNetPlugin,
					containerPort,
				)
				if err != nil {
					if err.Error() == gorm.ErrRecordNotFound.Error() {
						logrus.Debugf("outer, plugin port (%d) is not exist, do not need delete", containerPort)
						goto OUTERCLOSEPASS
					}
					tx.Rollback()
					return nil, "", fmt.Errorf("outer, get plugin mapping port error:(%s)", err)
				}
				if *p.IsInnerService {
					//发现内网未关闭则不删除该映射
					logrus.Debugf("outer, close outer, but plugin inner port (%d) is exist, do not need delete", containerPort)
					goto OUTERCLOSEPASS
				}
				if err := db.GetManager().TenantEnvServicesStreamPluginPortDaoTransactions(tx).DeletePluginMappingPortByContainerPort(
					serviceID,
					dbmodel.InBoundNetPlugin,
					containerPort,
				); err != nil {
					tx.Rollback()
					return nil, "", fmt.Errorf("outer, delete plugin mapping port %d error:(%s)", containerPort, err)
				}
				logrus.Debugf(fmt.Sprintf("outer, delete plugin port %d->%d", containerPort, pluginPort.PluginPort))
			OUTERCLOSEPASS:
			}
			if err := tx.Commit().Error; err != nil {
				tx.Rollback()
				return nil, "", err
			}
		} else {
			return nil, "", nil
		}

	case "open":
		truev := true
		p.IsOuterService = &truev
		tx := db.GetManager().Begin()
		defer func() {
			if r := recover(); r != nil {
				logrus.Errorf("Unexpected panic occurred, rollback transaction: %v", r)
				tx.Rollback()
			}
		}()
		if err = db.GetManager().TenantEnvServicesPortDaoTransactions(tx).UpdateModel(p); err != nil {
			tx.Rollback()
			return nil, "", err
		}
		if hasUpStream {
			pluginPort, err := db.GetManager().TenantEnvServicesStreamPluginPortDao().GetPluginMappingPortByServiceIDAndContainerPort(
				serviceID,
				dbmodel.InBoundNetPlugin,
				containerPort,
			)
			var pPort int
			if err != nil {
				if err.Error() == gorm.ErrRecordNotFound.Error() {
					ppPort, err := db.GetManager().TenantEnvServicesStreamPluginPortDaoTransactions(tx).SetPluginMappingPort(
						p.TenantEnvID,
						serviceID,
						dbmodel.InBoundNetPlugin,
						containerPort,
					)
					if err != nil {
						tx.Rollback()
						logrus.Errorf("outer, set plugin mapping port error:(%s)", err)
						return nil, "", fmt.Errorf("outer, set plugin mapping port error:(%s)", err)
					}
					pPort = ppPort
					goto OUTEROPENPASS
				}
				tx.Rollback()
				return nil, "", fmt.Errorf("outer, in setting plugin mapping port, get plugin mapping port error:(%s)", err)
			}
			logrus.Debugf("outer, plugin mapping port is already exist, %d->%d", pluginPort.ContainerPort, pluginPort.PluginPort)
		OUTEROPENPASS:
			logrus.Debugf("outer, set plugin mapping port %d->%d", containerPort, pPort)
		}
		if err := tx.Commit().Error; err != nil {
			tx.Rollback()
			return nil, "", err
		}
	}
	return vsPort, p.Protocol, nil
}

// PortInner 端口对内服务操作
// TODO: send task to worker
func (s *ServiceAction) PortInner(tenantEnvName, serviceID, operation string, port int) error {
	p, err := db.GetManager().TenantEnvServicesPortDao().GetPort(serviceID, port)
	if err != nil {
		return err
	}
	_, err = db.GetManager().TenantEnvServiceDao().GetServiceByID(serviceID)
	if err != nil {
		return fmt.Errorf("get service error:%s", err.Error())
	}
	hasUpStream, err := db.GetManager().TenantEnvServicePluginRelationDao().CheckSomeModelPluginByServiceID(
		serviceID,
		dbmodel.InBoundNetPlugin,
	)
	if err != nil {
		return fmt.Errorf("get plugin relations error: %s", err.Error())
	}
	tx := db.GetManager().Begin()
	defer func() {
		if r := recover(); r != nil {
			logrus.Errorf("Unexpected panic occurred, rollback transaction: %v", r)
			tx.Rollback()
		}
	}()
	switch operation {
	case "close":
		if *p.IsInnerService { //如果端口已经开了对内
			falsev := false
			p.IsInnerService = &falsev
			if err = db.GetManager().TenantEnvServicesPortDaoTransactions(tx).UpdateModel(p); err != nil {
				tx.Rollback()
				return fmt.Errorf("update service port error: %s", err.Error())
			}
			if hasUpStream {
				pluginPort, err := db.GetManager().TenantEnvServicesStreamPluginPortDao().GetPluginMappingPortByServiceIDAndContainerPort(
					serviceID,
					dbmodel.InBoundNetPlugin,
					port,
				)
				if err != nil {
					if err.Error() == gorm.ErrRecordNotFound.Error() {
						logrus.Debugf("inner, plugin port (%d) is not exist, do not need delete", port)
						goto INNERCLOSEPASS
					}
					tx.Rollback()
					return fmt.Errorf("inner, get plugin mapping port error:(%s)", err)
				}
				if *p.IsOuterService {
					logrus.Debugf("inner, close inner, but plugin outerport (%d) is exist, do not need delete", port)
					goto INNERCLOSEPASS
				}
				if err := db.GetManager().TenantEnvServicesStreamPluginPortDaoTransactions(tx).DeletePluginMappingPortByContainerPort(
					serviceID,
					dbmodel.InBoundNetPlugin,
					port,
				); err != nil {
					tx.Rollback()
					return fmt.Errorf("inner, delete plugin mapping port %d error:(%s)", port, err)
				}
				logrus.Debugf(fmt.Sprintf("inner, delete plugin port %d->%d", port, pluginPort.PluginPort))
			INNERCLOSEPASS:
			}
		} else {
			tx.Rollback()
			return fmt.Errorf("already close")
		}
	case "open":
		if *p.IsInnerService {
			tx.Rollback()
			return fmt.Errorf("already open")
		}
		truv := true
		p.IsInnerService = &truv
		if err = db.GetManager().TenantEnvServicesPortDaoTransactions(tx).UpdateModel(p); err != nil {
			tx.Rollback()
			return err
		}
		if hasUpStream {
			pluginPort, err := db.GetManager().TenantEnvServicesStreamPluginPortDao().GetPluginMappingPortByServiceIDAndContainerPort(
				serviceID,
				dbmodel.InBoundNetPlugin,
				port,
			)
			var pPort int
			if err != nil {
				if err.Error() == gorm.ErrRecordNotFound.Error() {
					ppPort, err := db.GetManager().TenantEnvServicesStreamPluginPortDaoTransactions(tx).SetPluginMappingPort(
						p.TenantEnvID,
						serviceID,
						dbmodel.InBoundNetPlugin,
						port,
					)
					if err != nil {
						tx.Rollback()
						logrus.Errorf("inner, set plugin mapping port error:(%s)", err)
						return fmt.Errorf("inner, set plugin mapping port error:(%s)", err)
					}
					pPort = ppPort
					goto INNEROPENPASS
				}
				tx.Rollback()
				return fmt.Errorf("inner, in setting plugin mapping port, get plugin mapping port error:(%s)", err)
			}
			logrus.Debugf("inner, plugin mapping port is already exist, %d->%d", pluginPort.ContainerPort, pluginPort.PluginPort)
		INNEROPENPASS:
			logrus.Debugf("inner, set plugin mapping port %d->%d", port, pPort)
		}
	}
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return err
	}
	return nil
}

// VolumnVar var volumn
func (s *ServiceAction) VolumnVar(tsv *dbmodel.TenantEnvServiceVolume, tenantEnvID, fileContent, action string) *apiutil.APIHandleError {
	localPath := os.Getenv("LOCAL_DATA_PATH")
	sharePath := os.Getenv("SHARE_DATA_PATH")
	if localPath == "" {
		localPath = "/wtlocaldata"
	}
	if sharePath == "" {
		sharePath = "/wtdata"
	}

	switch action {
	case "add":
		if tsv.HostPath == "" {
			//step 1 设置主机目录
			switch tsv.VolumeType {
			//共享文件存储
			case dbmodel.ShareFileVolumeType.String():
				tsv.HostPath = fmt.Sprintf("%s/tenantEnv/%s/service/%s%s", sharePath, tenantEnvID, tsv.ServiceID, tsv.VolumePath)
			//本地文件存储
			case dbmodel.LocalVolumeType.String():
				serviceInfo, err := db.GetManager().TenantEnvServiceDao().GetServiceTypeByID(tsv.ServiceID)
				if err != nil {
					return apiutil.CreateAPIHandleErrorFromDBError("service type", err)
				}
				// local volume just only support state component
				if serviceInfo == nil || !serviceInfo.IsState() {
					return apiutil.CreateAPIHandleError(400, fmt.Errorf("应用类型为'无状态'.不支持本地存储"))
				}
				tsv.HostPath = fmt.Sprintf("%s/tenantEnv/%s/service/%s%s", localPath, tenantEnvID, tsv.ServiceID, tsv.VolumePath)
			}
		}
		apiutil.SetVolumeDefaultValue(tsv)
		// begin transaction
		tx := db.GetManager().Begin()
		defer func() {
			if r := recover(); r != nil {
				logrus.Errorf("Unexpected panic occurred, rollback transaction: %v", r)
				tx.Rollback()
			}
		}()
		if err := db.GetManager().TenantEnvServiceVolumeDaoTransactions(tx).AddModel(tsv); err != nil {
			tx.Rollback()
			return apiutil.CreateAPIHandleErrorFromDBError("add volume", err)
		}
		if fileContent != "" {
			cf := &dbmodel.TenantEnvServiceConfigFile{
				ServiceID:   tsv.ServiceID,
				VolumeName:  tsv.VolumeName,
				FileContent: fileContent,
			}
			if err := db.GetManager().TenantEnvServiceConfigFileDaoTransactions(tx).AddModel(cf); err != nil {
				tx.Rollback()
				return apiutil.CreateAPIHandleErrorFromDBError("error creating config file", err)
			}
		}
		// end transaction
		if err := tx.Commit().Error; err != nil {
			tx.Rollback()
			return apiutil.CreateAPIHandleErrorFromDBError("error ending transaction", err)
		}
	case "delete":
		// begin transaction
		tx := db.GetManager().Begin()
		defer func() {
			if r := recover(); r != nil {
				logrus.Errorf("Unexpected panic occurred, rollback transaction: %v", r)
				tx.Rollback()
			}
		}()
		if tsv.VolumeName != "" {
			volume, err := db.GetManager().TenantEnvServiceVolumeDaoTransactions(tx).GetVolumeByServiceIDAndName(tsv.ServiceID, tsv.VolumeName)
			if err != nil {
				tx.Rollback()
				return apiutil.CreateAPIHandleErrorFromDBError("find volume", err)
			}

			if err := db.GetManager().TenantEnvServiceVolumeDaoTransactions(tx).DeleteModel(tsv.ServiceID, tsv.VolumeName); err != nil && err.Error() != gorm.ErrRecordNotFound.Error() {
				tx.Rollback()
				return apiutil.CreateAPIHandleErrorFromDBError("delete volume", err)
			}

			err = s.MQClient.SendBuilderTopic(gclient.TaskStruct{
				Topic:    gclient.WorkerTopic,
				TaskType: "volume_gc",
				TaskBody: map[string]interface{}{
					"tenant_env_id": tenantEnvID,
					"service_id":    volume.ServiceID,
					"volume_id":     volume.ID,
					"volume_path":   volume.VolumePath,
				},
			})
			if err != nil {
				logrus.Errorf("send 'volume_gc' task: %v", err)
				tx.Rollback()
				return apiutil.CreateAPIHandleErrorFromDBError("send 'volume_gc' task", err)
			}
		} else {
			if err := db.GetManager().TenantEnvServiceVolumeDaoTransactions(tx).DeleteByServiceIDAndVolumePath(tsv.ServiceID, tsv.VolumePath); err != nil && err.Error() != gorm.ErrRecordNotFound.Error() {
				tx.Rollback()
				return apiutil.CreateAPIHandleErrorFromDBError("delete volume", err)
			}
		}
		if err := db.GetManager().TenantEnvServiceConfigFileDaoTransactions(tx).DelByVolumeID(tsv.ServiceID, tsv.VolumeName); err != nil {
			tx.Rollback()
			return apiutil.CreateAPIHandleErrorFromDBError("error deleting config files", err)
		}
		// end transaction
		if err := tx.Commit().Error; err != nil {
			tx.Rollback()
			return apiutil.CreateAPIHandleErrorFromDBError("error ending transaction", err)
		}
	}
	return nil
}

// UpdVolume updates service volume.
func (s *ServiceAction) UpdVolume(sid string, req *api_model.UpdVolumeReq) error {
	tx := db.GetManager().Begin()
	defer func() {
		if r := recover(); r != nil {
			logrus.Errorf("Unexpected panic occurred, rollback transaction: %v", r)
			tx.Rollback()
		}
	}()
	v, err := db.GetManager().TenantEnvServiceVolumeDaoTransactions(tx).GetVolumeByServiceIDAndName(sid, req.VolumeName)
	if err != nil {
		tx.Rollback()
		return err
	}
	v.VolumePath = req.VolumePath
	v.Mode = req.Mode
	if err := db.GetManager().TenantEnvServiceVolumeDaoTransactions(tx).UpdateModel(v); err != nil {
		tx.Rollback()
		return err
	}
	if req.VolumeType == "config-file" {
		configfile, err := db.GetManager().TenantEnvServiceConfigFileDaoTransactions(tx).GetByVolumeName(sid, req.VolumeName)
		if err != nil {
			tx.Rollback()
			return err
		}
		configfile.FileContent = req.FileContent
		if err := db.GetManager().TenantEnvServiceConfigFileDaoTransactions(tx).UpdateModel(configfile); err != nil {
			tx.Rollback()
			return err
		}
	}
	tx.Commit()
	return nil
}

// GetVolumes 获取应用全部存储
func (s *ServiceAction) GetVolumes(serviceID string) ([]*api_model.VolumeWithStatusStruct, *apiutil.APIHandleError) {
	volumeWithStatusList := make([]*api_model.VolumeWithStatusStruct, 0)
	vs, err := db.GetManager().TenantEnvServiceVolumeDao().GetTenantEnvServiceVolumesByServiceID(serviceID)
	if err != nil && err.Error() != gorm.ErrRecordNotFound.Error() {
		return nil, apiutil.CreateAPIHandleErrorFromDBError("get volumes", err)
	}

	volumeStatusList, err := s.statusCli.GetAppVolumeStatus(serviceID)
	if err != nil {
		logrus.Warnf("get volume status error: %s", err.Error())
	}
	volumeStatus := make(map[string]pb.ServiceVolumeStatus)
	if volumeStatusList != nil && volumeStatusList.GetStatus() != nil {
		volumeStatus = volumeStatusList.GetStatus()
	}
	for _, volume := range vs {
		vws := &api_model.VolumeWithStatusStruct{
			ServiceID:          volume.ServiceID,
			Category:           volume.Category,
			VolumeType:         volume.VolumeType,
			VolumeName:         volume.VolumeName,
			HostPath:           volume.HostPath,
			VolumePath:         volume.VolumePath,
			IsReadOnly:         volume.IsReadOnly,
			VolumeCapacity:     volume.VolumeCapacity,
			AccessMode:         volume.AccessMode,
			SharePolicy:        volume.SharePolicy,
			BackupPolicy:       volume.BackupPolicy,
			ReclaimPolicy:      volume.ReclaimPolicy,
			AllowExpansion:     volume.AllowExpansion,
			VolumeProviderName: volume.VolumeProviderName,
		}
		volumeID := strconv.FormatInt(int64(volume.ID), 10)
		if phrase, ok := volumeStatus[volumeID]; ok {
			vws.Status = phrase.String()
		} else {
			vws.Status = pb.ServiceVolumeStatus_NOT_READY.String()
		}
		volumeWithStatusList = append(volumeWithStatusList, vws)
	}

	return volumeWithStatusList, nil
}

// VolumeDependency VolumeDependency
func (s *ServiceAction) VolumeDependency(tsr *dbmodel.TenantEnvServiceMountRelation, action string) *apiutil.APIHandleError {
	switch action {
	case "add":
		if tsr.VolumeName != "" {
			vm, err := db.GetManager().TenantEnvServiceVolumeDao().GetVolumeByServiceIDAndName(tsr.DependServiceID, tsr.VolumeName)
			if err != nil {
				return apiutil.CreateAPIHandleErrorFromDBError("get volume", err)
			}
			tsr.HostPath = vm.HostPath
			if err := db.GetManager().TenantEnvServiceMountRelationDao().AddModel(tsr); err != nil {
				return apiutil.CreateAPIHandleErrorFromDBError("add volume mount relation", err)
			}
		} else {
			if tsr.HostPath == "" {
				return apiutil.CreateAPIHandleError(400, fmt.Errorf("host path can not be empty when create volume dependency in api v2"))
			}
			if err := db.GetManager().TenantEnvServiceMountRelationDao().AddModel(tsr); err != nil {
				return apiutil.CreateAPIHandleErrorFromDBError("add volume mount relation", err)
			}
		}
	case "delete":
		if tsr.VolumeName != "" {
			if err := db.GetManager().TenantEnvServiceMountRelationDao().DElTenantEnvServiceMountRelationByServiceAndName(tsr.ServiceID, tsr.VolumeName); err != nil {
				return apiutil.CreateAPIHandleErrorFromDBError("delete mount relation", err)
			}
		} else {
			if err := db.GetManager().TenantEnvServiceMountRelationDao().DElTenantEnvServiceMountRelationByDepService(tsr.ServiceID, tsr.DependServiceID); err != nil {
				return apiutil.CreateAPIHandleErrorFromDBError("delete mount relation", err)
			}
		}
	}
	return nil
}

// GetDepVolumes 获取依赖存储
func (s *ServiceAction) GetDepVolumes(serviceID string) ([]*dbmodel.TenantEnvServiceMountRelation, *apiutil.APIHandleError) {
	dbManager := db.GetManager()
	mounts, err := dbManager.TenantEnvServiceMountRelationDao().GetTenantEnvServiceMountRelationsByService(serviceID)
	if err != nil {
		return nil, apiutil.CreateAPIHandleErrorFromDBError("get dep volume", err)
	}
	return mounts, nil
}

// ServiceProbe ServiceProbe
func (s *ServiceAction) ServiceProbe(tsp *dbmodel.TenantEnvServiceProbe, action string) error {
	switch action {
	case "add":
		if err := db.GetManager().ServiceProbeDao().AddModel(tsp); err != nil {
			return err
		}
	case "update":
		if err := db.GetManager().ServiceProbeDao().UpdateModel(tsp); err != nil {
			return err
		}
	case "delete":
		if err := db.GetManager().ServiceProbeDao().DeleteModel(tsp.ServiceID, tsp.ProbeID); err != nil {
			return err
		}
	}
	return nil
}

// RollBack RollBack
func (s *ServiceAction) RollBack(rs *api_model.RollbackStruct) error {
	service, err := db.GetManager().TenantEnvServiceDao().GetServiceByID(rs.ServiceID)
	if err != nil {
		return err
	}
	oldDeployVersion := service.DeployVersion
	if service.DeployVersion == rs.DeployVersion {
		return fmt.Errorf("current version is %v, don't need rollback", rs.DeployVersion)
	}
	service.DeployVersion = rs.DeployVersion
	if err := db.GetManager().TenantEnvServiceDao().UpdateModel(service); err != nil {
		return err
	}
	//发送重启消息到MQ
	startStopStruct := &api_model.StartStopStruct{
		TenantEnvID: rs.TenantEnvID,
		ServiceID:   rs.ServiceID,
		EventID:     rs.EventID,
		TaskType:    "rolling_upgrade",
	}
	if err := GetServiceManager().StartStopService(startStopStruct); err != nil {
		// rollback
		service.DeployVersion = oldDeployVersion
		if err := db.GetManager().TenantEnvServiceDao().UpdateModel(service); err != nil {
			logrus.Warningf("error deploy version rollback: %v", err)
		}
		return err
	}
	return nil
}

// GetStatus GetStatus
func (s *ServiceAction) GetStatus(serviceID string) (*api_model.StatusList, error) {
	services, errS := db.GetManager().TenantEnvServiceDao().GetServiceByID(serviceID)
	if errS != nil {
		return nil, errS
	}
	sl := &api_model.StatusList{
		TenantEnvID:   services.TenantEnvID,
		ServiceID:     serviceID,
		ServiceAlias:  services.ServiceAlias,
		DeployVersion: services.DeployVersion,
		Replicas:      services.Replicas,
		ContainerMem:  services.ContainerMemory,
		ContainerCPU:  services.ContainerCPU,
		CurStatus:     services.CurStatus,
		StatusCN:      TransStatus(services.CurStatus),
	}
	status := s.statusCli.GetStatus(serviceID)
	if status != "" {
		sl.CurStatus = status
		sl.StatusCN = TransStatus(status)
	}
	di, err := s.statusCli.GetServiceDeployInfo(serviceID)
	if err != nil {
		logrus.Warningf("service id: %s; failed to get deploy info: %v", serviceID, err)
	} else {
		sl.StartTime = di.GetStartTime()
	}
	return sl, nil
}

// GetServicesStatus  获取一组应用状态，若 serviceIDs为空,获取租户所有应用状态
func (s *ServiceAction) GetServicesStatus(tenantEnvID string, serviceIDs []string) []map[string]interface{} {
	if len(serviceIDs) == 0 {
		services, _ := db.GetManager().TenantEnvServiceDao().GetServicesByTenantEnvID(tenantEnvID)
		for _, s := range services {
			serviceIDs = append(serviceIDs, s.ServiceID)
		}
	}
	if len(serviceIDs) == 0 {
		return []map[string]interface{}{}
	}
	statusList := s.statusCli.GetStatuss(strings.Join(serviceIDs, ","))
	var info = make([]map[string]interface{}, 0)
	for k, v := range statusList {
		serviceInfo := map[string]interface{}{"service_id": k, "status": v, "status_cn": TransStatus(v), "used_mem": 0}
		info = append(info, serviceInfo)
	}
	return info
}

// GetAllRunningServices get running services
func (s *ServiceAction) GetAllRunningServices() ([]string, *apiutil.APIHandleError) {
	var tenantEnvIDs []string
	tenantEnvs, err := db.GetManager().TenantEnvDao().GetAllTenantEnvs("")
	if err != nil {
		logrus.Errorf("list tenant env failed: %s", err.Error())
		return nil, apiutil.CreateAPIHandleErrorFromDBError("get tenant env failed", err)
	}
	if len(tenantEnvs) == 0 {
		return nil, apiutil.CreateAPIHandleErrorf(400, "not found any tenantEnvs")
	}
	for _, tenantEnv := range tenantEnvs {
		tenantEnvIDs = append(tenantEnvIDs, tenantEnv.UUID)
	}
	services, err := db.GetManager().TenantEnvServiceDao().GetServicesByTenantEnvIDs(tenantEnvIDs)
	if err != nil {
		logrus.Errorf("list tenantEnvs service failed: %s", err.Error())
		return nil, apiutil.CreateAPIHandleErrorf(500, "get service failed: %s", err.Error())
	}
	var serviceIDs []string
	for _, svc := range services {
		serviceIDs = append(serviceIDs, svc.ServiceID)
	}
	statusList := s.statusCli.GetStatuss(strings.Join(serviceIDs, ","))
	retServices := make([]string, 0, 10)
	for service, status := range statusList {
		if status == typesv1.RUNNING {
			retServices = append(retServices, service)
		}
	}
	return retServices, nil
}

type ServicesStatus struct {
	RunningServices   []string `json:"running_services"`
	UnRunningServices []string `json:"unrunning_services"`
	AbnormalServices  []string `json:"abnormal_services"`
}

func (s *ServiceAction) GetAllServicesStatus() (*ServicesStatus, *apiutil.APIHandleError) {
	var tenantEnvIDs []string
	tenantEnvs, err := db.GetManager().TenantEnvDao().GetAllTenantEnvs("")
	if err != nil {
		logrus.Errorf("list tenant env failed: %s", err.Error())
		return nil, apiutil.CreateAPIHandleErrorFromDBError("get tenant env failed", err)
	}
	if len(tenantEnvs) == 0 {
		return nil, apiutil.CreateAPIHandleErrorf(400, "not found any tenant envs")
	}
	for _, tenantEnv := range tenantEnvs {
		tenantEnvIDs = append(tenantEnvIDs, tenantEnv.UUID)
	}
	services, err := db.GetManager().TenantEnvServiceDao().GetServicesByTenantEnvIDs(tenantEnvIDs)
	if err != nil {
		logrus.Errorf("list tenant envs service failed: %s", err.Error())
		return nil, apiutil.CreateAPIHandleErrorf(500, "get service failed: %s", err.Error())
	}
	var serviceIDs []string
	for _, svc := range services {
		serviceIDs = append(serviceIDs, svc.ServiceID)
	}
	statusList := s.statusCli.GetStatuss(strings.Join(serviceIDs, ","))
	servicesStatus := &ServicesStatus{
		RunningServices:   []string{},
		UnRunningServices: []string{},
		AbnormalServices:  []string{},
	}
	for service, status := range statusList {
		switch status {
		case typesv1.RUNNING:
			servicesStatus.RunningServices = append(servicesStatus.RunningServices, service)
		case typesv1.STOPPING, typesv1.CLOSED, typesv1.BUILDING, typesv1.STARTING, typesv1.UNDEPLOY, typesv1.UPGRADE:
			servicesStatus.UnRunningServices = append(servicesStatus.UnRunningServices, service)
		case typesv1.ABNORMAL, typesv1.BUILDEFAILURE, typesv1.SOMEABNORMAL, typesv1.UNKNOW:
			servicesStatus.AbnormalServices = append(servicesStatus.AbnormalServices, service)
		}
	}
	return servicesStatus, nil
}

// CreateTenantEnv create tenantEnv
func (s *ServiceAction) CreateTenantEnv(t *dbmodel.TenantEnvs) error {
	tenantEnv, _ := db.GetManager().TenantEnvDao().GetTenantEnvIDByName(t.TenantName, t.Name)
	if tenantEnv != nil {
		return fmt.Errorf("tenant env name %s is exist", t.Name)
	}
	labels := map[string]string{
		constants.ResourceManagedByLabel:     constants.Wutong,
		constants.ResourceTenantIDLabel:      t.TenantID,
		constants.ResourceTenantNameLabel:    t.TenantName,
		constants.ResourceTenantEnvIDLabel:   t.UUID,
		constants.ResourceTenantEnvNameLabel: t.Name,
	}
	return db.GetManager().DB().Transaction(func(tx *gorm.DB) error {
		if err := db.GetManager().TenantEnvDaoTransactions(tx).AddModel(t); err != nil {
			return err
		}
		if _, err := s.kubeClient.CoreV1().Namespaces().Create(context.Background(), &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:   t.Namespace,
				Labels: labels,
			},
		}, metav1.CreateOptions{}); err != nil {
			if k8sErrors.IsAlreadyExists(err) {
				return bcode.ErrNamespaceExists
			}
			return err
		}
		return nil
	})
}

// CreateTenantEnvIDAndName create tenant_env_id and tenant_env_name
func (s *ServiceAction) CreateTenantEnvIDAndName() (string, string, error) {
	id := uuid.New().String()
	uid := strings.Replace(id, "-", "", -1)
	name := strings.Split(id, "-")[0]
	logrus.Debugf("uuid is %v, name is %v", uid, name)
	return uid, name, nil
}

// K8sPodInfos -
type K8sPodInfos struct {
	NewPods []*K8sPodInfo `json:"new_pods"`
	OldPods []*K8sPodInfo `json:"old_pods"`
}

// K8sPodInfo for api
type K8sPodInfo struct {
	PodName   string                       `json:"pod_name"`
	PodIP     string                       `json:"pod_ip"`
	PodStatus string                       `json:"pod_status"`
	ServiceID string                       `json:"service_id"`
	Container map[string]map[string]string `json:"container"`
}

// GetPods get pods
func (s *ServiceAction) GetPods(serviceID string) (*K8sPodInfos, error) {
	pods, err := s.statusCli.GetServicePods(serviceID)
	if err != nil && !strings.Contains(err.Error(), server.ErrAppServiceNotFound.Error()) &&
		!strings.Contains(err.Error(), server.ErrPodNotFound.Error()) {
		logrus.Error("GetPodByService Error:", err)
		return nil, err
	}
	if pods == nil {
		return nil, nil
	}
	convpod := func(pods []*pb.ServiceAppPod) []*K8sPodInfo {
		var podsInfoList []*K8sPodInfo
		var podNames []string
		for _, v := range pods {
			var podInfo K8sPodInfo
			podInfo.PodName = v.PodName
			podInfo.PodIP = v.PodIp
			podInfo.PodStatus = v.PodStatus
			podInfo.ServiceID = serviceID
			containerInfos := make(map[string]map[string]string, 10)
			for _, container := range v.Containers {
				containerInfos[container.ContainerName] = map[string]string{
					"memory_limit": fmt.Sprintf("%d", container.MemoryLimit),
					"memory_usage": "0",
					"cpu_limit":    fmt.Sprintf("%d", container.CpuRequest), // TODO: should use cpu limit
					"cpu_usage":    "0",
				}
			}
			podInfo.Container = containerInfos
			podNames = append(podNames, v.PodName)
			podsInfoList = append(podsInfoList, &podInfo)
		}
		containerMemInfo, _ := s.GetPodContainerMemory(podNames)
		for _, c := range podsInfoList {
			for k := range c.Container {
				if info, exist := containerMemInfo[c.PodName][k]; exist {
					c.Container[k]["memory_usage"] = info
				}
			}
		}
		containerCpuInfo, _ := s.GetPodContainerCPU(podNames)
		for _, c := range podsInfoList {
			for k := range c.Container {
				if info, exist := containerCpuInfo[c.PodName][k]; exist {
					c.Container[k]["cpu_usage"] = info
				}
			}
		}
		return podsInfoList
	}
	newpods := convpod(pods.NewPods)
	oldpods := convpod(pods.OldPods)
	return &K8sPodInfos{
		NewPods: newpods,
		OldPods: oldpods,
	}, nil
}

// GetMultiServicePods get pods
func (s *ServiceAction) GetMultiServicePods(serviceIDs []string) (*K8sPodInfos, error) {
	mpods, err := s.statusCli.GetMultiServicePods(serviceIDs)
	if err != nil && !strings.Contains(err.Error(), server.ErrAppServiceNotFound.Error()) &&
		!strings.Contains(err.Error(), server.ErrPodNotFound.Error()) {
		logrus.Error("GetPodByService Error:", err)
		return nil, err
	}
	if mpods == nil {
		return nil, nil
	}
	convpod := func(serviceID string, pods []*pb.ServiceAppPod) []*K8sPodInfo {
		var podsInfoList []*K8sPodInfo
		for _, v := range pods {
			var podInfo K8sPodInfo
			podInfo.PodName = v.PodName
			podInfo.PodIP = v.PodIp
			podInfo.PodStatus = v.PodStatus
			podInfo.ServiceID = serviceID
			podsInfoList = append(podsInfoList, &podInfo)
		}
		return podsInfoList
	}
	var re K8sPodInfos
	for serviceID, pods := range mpods.ServicePods {
		if pods != nil {
			re.NewPods = append(re.NewPods, convpod(serviceID, pods.NewPods)...)
			re.OldPods = append(re.OldPods, convpod(serviceID, pods.OldPods)...)
		}
	}
	return &re, nil
}

// GetComponentPodNums get pods
func (s *ServiceAction) GetComponentPodNums(ctx context.Context, componentIDs []string) (map[string]int32, error) {
	if logrus.IsLevelEnabled(logrus.DebugLevel) {
		defer util.Elapsed(fmt.Sprintf("[AppRuntimeSyncClient] [GetComponentPodNums] component nums: %d", len(componentIDs)))()
	}

	podNums, err := s.statusCli.GetComponentPodNums(ctx, componentIDs)
	if err != nil {
		return nil, errors.Wrap(err, "get component nums")
	}

	return podNums, nil
}

// GetPodContainerMemory Use Prometheus to query memory resources
func (s *ServiceAction) GetPodContainerMemory(podNames []string) (map[string]map[string]string, error) {
	memoryUsageMap := make(map[string]map[string]string, 10)
	queryName := strings.Join(podNames, "|")
	query := fmt.Sprintf(`container_memory_rss{pod=~"%s"}`, queryName)
	metric := s.prometheusCli.GetMetric(query, time.Now())

	for _, re := range metric.MetricData.MetricValues {
		var containerName = re.Metadata["container"]
		var podName = re.Metadata["pod"]
		var valuesBytes string
		if re.Sample != nil {
			valuesBytes = fmt.Sprintf("%d", int(re.Sample.Value()))
		}
		if _, ok := memoryUsageMap[podName]; ok {
			memoryUsageMap[podName][containerName] = valuesBytes
		} else {
			memoryUsageMap[podName] = map[string]string{
				containerName: valuesBytes,
			}
		}
	}
	return memoryUsageMap, nil
}

// GetPodContainerCPU Use Prometheus to query cpu resources
func (s *ServiceAction) GetPodContainerCPU(podNames []string) (map[string]map[string]string, error) {
	cpuUsageMap := make(map[string]map[string]string, 10)
	queryName := strings.Join(podNames, "|")
	query := fmt.Sprintf(`rate(container_cpu_usage_seconds_total{pod=~"%s"}[5m])`, queryName)
	metric := s.prometheusCli.GetMetric(query, time.Now())

	for _, re := range metric.MetricData.MetricValues {
		var containerName = re.Metadata["container"]
		var podName = re.Metadata["pod"]
		var valuesBytes string
		if re.Sample != nil {
			valuesBytes = fmt.Sprintf("%f", re.Sample.Value()*1000)
		}
		if _, ok := cpuUsageMap[podName]; ok {
			cpuUsageMap[podName][containerName] = valuesBytes
		} else {
			cpuUsageMap[podName] = map[string]string{
				containerName: valuesBytes,
			}
		}
	}
	return cpuUsageMap, nil
}

// TransServieToDelete trans service info to delete table
func (s *ServiceAction) TransServieToDelete(ctx context.Context, tenantEnvID, serviceID string) error {
	_, err := db.GetManager().TenantEnvServiceDao().GetServiceByID(serviceID)
	if err != nil && gorm.ErrRecordNotFound == err {
		logrus.Infof("service[%s] of tenant env[%s] do not exist, ignore it", serviceID, tenantEnvID)
		return nil
	}
	if err := s.isServiceClosed(serviceID); err != nil {
		return err
	}

	body, err := s.gcTaskBody(tenantEnvID, serviceID)
	if err != nil {
		return fmt.Errorf("GC task body: %v", err)
	}

	if err := s.delServiceMetadata(ctx, serviceID); err != nil {
		return fmt.Errorf("delete service-related metadata: %v", err)
	}

	// let wt-chaos remove related persistent data
	logrus.Info("let wt-chaos remove related persistent data")
	topic := gclient.WorkerTopic
	if err := s.MQClient.SendBuilderTopic(gclient.TaskStruct{
		Topic:    topic,
		TaskType: "service_gc",
		TaskBody: body,
	}); err != nil {
		logrus.Warningf("send gc task: %v", err)
	}

	return nil
}

// isServiceClosed checks if the service has been closed according to the serviceID.
func (s *ServiceAction) isServiceClosed(serviceID string) error {
	service, err := db.GetManager().TenantEnvServiceDao().GetServiceByID(serviceID)
	if err != nil {
		return err
	}
	status := s.statusCli.GetStatus(serviceID)
	if service.Kind != dbmodel.ServiceKindThirdParty.String() {
		if !s.statusCli.IsClosedStatus(status) {
			return ErrServiceNotClosed
		}
	}
	return nil
}

func (s *ServiceAction) deleteComponent(tx *gorm.DB, service *dbmodel.TenantEnvServices) error {
	delService := service.ChangeDelete()
	delService.ID = 0
	if err := db.GetManager().TenantEnvServiceDeleteDaoTransactions(tx).AddModel(delService); err != nil {
		return err
	}
	var deleteServicePropertyFunc = []func(serviceID string) error{
		db.GetManager().CodeCheckResultDaoTransactions(tx).DeleteByServiceID,
		db.GetManager().TenantEnvServiceEnvVarDaoTransactions(tx).DELServiceEnvsByServiceID,
		db.GetManager().TenantEnvPluginVersionConfigDaoTransactions(tx).DeletePluginConfigByServiceID,
		db.GetManager().TenantEnvServicePluginRelationDaoTransactions(tx).DeleteALLRelationByServiceID,
		db.GetManager().TenantEnvServicesStreamPluginPortDaoTransactions(tx).DeleteAllPluginMappingPortByServiceID,
		db.GetManager().TenantEnvServiceDaoTransactions(tx).DeleteServiceByServiceID,
		db.GetManager().TenantEnvServicesPortDaoTransactions(tx).DELPortsByServiceID,
		db.GetManager().TenantEnvServiceRelationDaoTransactions(tx).DELRelationsByServiceID,
		db.GetManager().TenantEnvServiceMountRelationDaoTransactions(tx).DELTenantEnvServiceMountRelationByServiceID,
		db.GetManager().TenantEnvServiceVolumeDaoTransactions(tx).DeleteTenantEnvServiceVolumesByServiceID,
		db.GetManager().TenantEnvServiceConfigFileDaoTransactions(tx).DelByServiceID,
		db.GetManager().EndpointsDaoTransactions(tx).DeleteByServiceID,
		db.GetManager().ThirdPartySvcDiscoveryCfgDaoTransactions(tx).DeleteByServiceID,
		db.GetManager().TenantEnvServiceLabelDaoTransactions(tx).DeleteLabelByServiceID,
		db.GetManager().VersionInfoDaoTransactions(tx).DeleteVersionByServiceID,
		db.GetManager().TenantEnvPluginVersionENVDaoTransactions(tx).DeleteEnvByServiceID,
		db.GetManager().ServiceProbeDaoTransactions(tx).DELServiceProbesByServiceID,
		db.GetManager().ServiceEventDaoTransactions(tx).DelEventByServiceID,
		db.GetManager().TenantEnvServiceMonitorDaoTransactions(tx).DeleteServiceMonitorByServiceID,
		db.GetManager().AppConfigGroupServiceDaoTransactions(tx).DeleteEffectiveServiceByServiceID,
	}
	if err := GetGatewayHandler().DeleteTCPRuleByServiceIDWithTransaction(service.ServiceID, tx); err != nil {
		return err
	}
	if err := GetGatewayHandler().DeleteHTTPRuleByServiceIDWithTransaction(service.ServiceID, tx); err != nil {
		return err
	}
	for _, del := range deleteServicePropertyFunc {
		if err := del(service.ServiceID); err != nil {
			if err != gorm.ErrRecordNotFound {
				return err
			}
		}
	}
	return nil
}

// delServiceMetadata deletes service-related metadata in the database.
func (s *ServiceAction) delServiceMetadata(ctx context.Context, serviceID string) error {
	service, err := db.GetManager().TenantEnvServiceDao().GetServiceByID(serviceID)
	if err != nil {
		return err
	}
	logrus.Infof("delete service %s %s", serviceID, service.ServiceAlias)
	return db.GetManager().DB().Transaction(func(tx *gorm.DB) error {
		if err := s.deleteThirdComponent(ctx, service); err != nil {
			return err
		}
		return s.deleteComponent(tx, service)
	})
}

func (s *ServiceAction) deleteThirdComponent(ctx context.Context, component *dbmodel.TenantEnvServices) error {
	if component.Kind != "third_party" {
		return nil
	}
	tenantEnv, err := db.GetManager().TenantEnvDao().GetTenantEnvByUUID(component.TenantEnvID)
	if err != nil {
		return err
	}
	thirdPartySvcDiscoveryCfg, err := db.GetManager().ThirdPartySvcDiscoveryCfgDao().GetByServiceID(component.ServiceID)
	if err != nil {
		return err
	}
	if thirdPartySvcDiscoveryCfg == nil {
		return nil
	}
	if thirdPartySvcDiscoveryCfg.Type != string(dbmodel.DiscorveryTypeKubernetes) {
		return nil
	}

	newCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	err = s.wutongClient.WutongV1alpha1().ThirdComponents(tenantEnv.Namespace).Delete(newCtx, component.ServiceID, metav1.DeleteOptions{})
	if err != nil && !k8sErrors.IsNotFound(err) {
		return err
	}
	return nil
}

func (s *ServiceAction) gcTaskBody(tenantEnvID, serviceID string) (map[string]interface{}, error) {
	events, err := db.GetManager().ServiceEventDao().ListByTargetID(serviceID)
	if err != nil {
		logrus.Errorf("list events based on serviceID: %v", err)
	}
	var eventIDs []string
	for _, event := range events {
		eventIDs = append(eventIDs, event.EventID)
	}

	return map[string]interface{}{
		"tenant_env_id": tenantEnvID,
		"service_id":    serviceID,
		"event_ids":     eventIDs,
	}, nil
}

// GetServiceDeployInfo get service deploy info
func (s *ServiceAction) GetServiceDeployInfo(tenantEnvID, serviceID string) (*pb.DeployInfo, *apiutil.APIHandleError) {
	info, err := s.statusCli.GetServiceDeployInfo(serviceID)
	if err != nil {
		return nil, apiutil.CreateAPIHandleError(500, err)
	}
	return info, nil
}

// ListVersionInfo lists version info
func (s *ServiceAction) ListVersionInfo(serviceID string) (*api_model.BuildListRespVO, error) {
	versionInfos, err := db.GetManager().VersionInfoDao().GetAllVersionByServiceID(serviceID)
	if err != nil && err != gorm.ErrRecordNotFound {
		logrus.Errorf("error getting all version by service id: %v", err)
		return nil, fmt.Errorf("error getting all version by service id: %v", err)
	}
	svc, err := db.GetManager().TenantEnvServiceDao().GetServiceByID(serviceID)
	if err != nil {
		logrus.Errorf("error getting service by uuid: %v", err)
		return nil, fmt.Errorf("error getting service by uuid: %v", err)
	}
	b, err := json.Marshal(versionInfos)
	if err != nil {
		return nil, fmt.Errorf("error marshaling version infos: %v", err)
	}
	var bversions []*api_model.BuildVersion
	if err := json.Unmarshal(b, &bversions); err != nil {
		return nil, fmt.Errorf("error unmarshaling version infos: %v", err)
	}
	for idx := range bversions {
		bv := bversions[idx]
		if bv.Kind == "build_from_image" || bv.Kind == "build_from_market_image" {
			image := parser.ParseImageName(bv.RepoURL)
			bv.ImageDomain = image.GetDomain()
			bv.ImageRepo = image.GetRepostory()
			bv.ImageTag = image.GetTag()
		}
	}
	result := &api_model.BuildListRespVO{
		DeployVersion: svc.DeployVersion,
		List:          bversions,
	}
	return result, nil
}

// AddAutoscalerRule -
func (s *ServiceAction) AddAutoscalerRule(req *api_model.AutoscalerRuleReq) error {
	tx := db.GetManager().Begin()
	defer db.GetManager().EnsureEndTransactionFunc()

	r := &dbmodel.TenantEnvServiceAutoscalerRules{
		RuleID:      req.RuleID,
		ServiceID:   req.ServiceID,
		Enable:      req.Enable,
		XPAType:     req.XPAType,
		MinReplicas: req.MinReplicas,
		MaxReplicas: req.MaxReplicas,
	}
	if err := db.GetManager().TenantEnvServceAutoscalerRulesDaoTransactions(tx).AddModel(r); err != nil {
		tx.Rollback()
		return err
	}

	for _, metric := range req.Metrics {
		m := &dbmodel.TenantEnvServiceAutoscalerRuleMetrics{
			RuleID:            req.RuleID,
			MetricsType:       metric.MetricsType,
			MetricsName:       metric.MetricsName,
			MetricTargetType:  metric.MetricTargetType,
			MetricTargetValue: metric.MetricTargetValue,
		}
		if err := db.GetManager().TenantEnvServceAutoscalerRuleMetricsDaoTransactions(tx).AddModel(m); err != nil {
			tx.Rollback()
			return err
		}
	}

	taskbody := map[string]interface{}{
		"service_id": r.ServiceID,
		"rule_id":    r.RuleID,
	}
	if err := s.MQClient.SendBuilderTopic(gclient.TaskStruct{
		TaskType: "refreshhpa",
		TaskBody: taskbody,
		Topic:    gclient.WorkerTopic,
	}); err != nil {
		logrus.Errorf("send 'refreshhpa' task: %v", err)
		return err
	}
	logrus.Infof("rule id: %s; successfully send 'refreshhpa' task.", r.RuleID)

	return tx.Commit().Error
}

// UpdAutoscalerRule -
func (s *ServiceAction) UpdAutoscalerRule(req *api_model.AutoscalerRuleReq) error {
	rule, err := db.GetManager().TenantEnvServceAutoscalerRulesDao().GetByRuleID(req.RuleID)
	if err != nil {
		return err
	}

	rule.Enable = req.Enable
	rule.XPAType = req.XPAType
	rule.MinReplicas = req.MinReplicas
	rule.MaxReplicas = req.MaxReplicas

	tx := db.GetManager().Begin()
	defer db.GetManager().EnsureEndTransactionFunc()

	if err := db.GetManager().TenantEnvServceAutoscalerRulesDaoTransactions(tx).UpdateModel(rule); err != nil {
		tx.Rollback()
		return err
	}

	// delete metrics
	if err := db.GetManager().TenantEnvServceAutoscalerRuleMetricsDaoTransactions(tx).DeleteByRuleID(req.RuleID); err != nil {
		tx.Rollback()
		return err
	}

	for _, metric := range req.Metrics {
		m := &dbmodel.TenantEnvServiceAutoscalerRuleMetrics{
			RuleID:            req.RuleID,
			MetricsType:       metric.MetricsType,
			MetricsName:       metric.MetricsName,
			MetricTargetType:  metric.MetricTargetType,
			MetricTargetValue: metric.MetricTargetValue,
		}
		if err := db.GetManager().TenantEnvServceAutoscalerRuleMetricsDaoTransactions(tx).AddModel(m); err != nil {
			tx.Rollback()
			return err
		}
	}

	taskbody := map[string]interface{}{
		"service_id": rule.ServiceID,
		"rule_id":    rule.RuleID,
	}
	if err := s.MQClient.SendBuilderTopic(gclient.TaskStruct{
		TaskType: "refreshhpa",
		TaskBody: taskbody,
		Topic:    gclient.WorkerTopic,
	}); err != nil {
		logrus.Errorf("send 'refreshhpa' task: %v", err)
		return err
	}
	logrus.Infof("rule id: %s; successfully send 'refreshhpa' task.", rule.RuleID)

	return tx.Commit().Error
}

// ListScalingRecords -
func (s *ServiceAction) ListScalingRecords(serviceID string, page, pageSize int) ([]*dbmodel.TenantEnvServiceScalingRecords, int, error) {
	records, err := db.GetManager().TenantEnvServiceScalingRecordsDao().ListByServiceID(serviceID, (page-1)*pageSize, pageSize)
	if err != nil {
		return nil, 0, err
	}

	count, err := db.GetManager().TenantEnvServiceScalingRecordsDao().CountByServiceID(serviceID)
	if err != nil {
		return nil, 0, err
	}

	return records, count, nil
}

// SyncComponentBase -
func (s *ServiceAction) SyncComponentBase(tx *gorm.DB, app *dbmodel.Application, components []*api_model.Component) error {
	var (
		componentIDs []string
		dbComponents []*dbmodel.TenantEnvServices
	)
	for _, component := range components {
		componentIDs = append(componentIDs, component.ComponentBase.ComponentID)
	}
	oldComponents, err := db.GetManager().TenantEnvServiceDao().GetServiceByIDs(componentIDs)
	if err != nil {
		return err
	}
	existComponents := make(map[string]*dbmodel.TenantEnvServices)
	for _, oc := range oldComponents {
		existComponents[oc.ServiceID] = oc
	}
	for _, component := range components {
		var deployVersion string
		if oldComponent, ok := existComponents[component.ComponentBase.ComponentID]; ok {
			deployVersion = oldComponent.DeployVersion
		}
		dbComponents = append(dbComponents, component.ComponentBase.DbModel(app.TenantEnvID, app.AppID, deployVersion))
	}
	if err := db.GetManager().TenantEnvServiceDaoTransactions(tx).DeleteByComponentIDs(app.TenantEnvID, app.AppID, componentIDs); err != nil {
		return err
	}
	return db.GetManager().TenantEnvServiceDaoTransactions(tx).CreateOrUpdateComponentsInBatch(dbComponents)
}

// SyncComponentRelations -
func (s *ServiceAction) SyncComponentRelations(tx *gorm.DB, app *dbmodel.Application, components []*api_model.Component) error {
	var (
		componentIDs []string
		relations    []*dbmodel.TenantEnvServiceRelation
	)
	for _, component := range components {
		if component.Relations == nil {
			continue
		}
		componentIDs = append(componentIDs, component.ComponentBase.ComponentID)
		for _, relation := range component.Relations {
			relations = append(relations, relation.DbModel(app.TenantEnvID, component.ComponentBase.ComponentID))
		}
	}
	if err := db.GetManager().TenantEnvServiceRelationDaoTransactions(tx).DeleteByComponentIDs(componentIDs); err != nil {
		return err
	}
	return db.GetManager().TenantEnvServiceRelationDaoTransactions(tx).CreateOrUpdateRelationsInBatch(relations)
}

// SyncComponentEnvs -
func (s *ServiceAction) SyncComponentEnvs(tx *gorm.DB, app *dbmodel.Application, components []*api_model.Component) error {
	var (
		componentIDs []string
		envs         []*dbmodel.TenantEnvServiceEnvVar
	)
	for _, component := range components {
		if component.Envs == nil {
			continue
		}
		componentIDs = append(componentIDs, component.ComponentBase.ComponentID)
		for _, env := range component.Envs {
			envs = append(envs, env.DbModel(app.TenantEnvID, component.ComponentBase.ComponentID))
		}
	}
	if err := db.GetManager().TenantEnvServiceEnvVarDaoTransactions(tx).DeleteByComponentIDs(componentIDs); err != nil {
		return err
	}
	return db.GetManager().TenantEnvServiceEnvVarDaoTransactions(tx).CreateOrUpdateEnvsInBatch(envs)
}

// SyncComponentVolumeRels -
func (s *ServiceAction) SyncComponentVolumeRels(tx *gorm.DB, app *dbmodel.Application, components []*api_model.Component) error {
	var (
		componentIDs []string
		volRels      []*dbmodel.TenantEnvServiceMountRelation
	)
	// Get the storage of all components under the application
	appComponents, err := db.GetManager().TenantEnvServiceDao().ListByAppID(app.AppID)
	if err != nil {
		return err
	}
	var appComponentIDs []string
	for _, ac := range appComponents {
		appComponentIDs = append(appComponentIDs, ac.ServiceID)
	}
	existVolume, err := s.getExistVolumes(appComponentIDs)
	if err != nil {
		return err
	}
	// Get the storage that needs to be newly created
	for _, component := range components {
		componentID := component.ComponentBase.ComponentID
		if component.Volumes == nil {
			continue
		}
		for _, vol := range component.Volumes {
			if _, ok := existVolume[vol.Key(componentID)]; !ok {
				existVolume[vol.Key(componentID)] = vol.DbModel(componentID)
			}
		}
	}

	for _, component := range components {
		if component.VolumeRelations == nil {
			continue
		}
		componentIDs = append(componentIDs, component.ComponentBase.ComponentID)
		//The hostpath attribute should not be recorded in the mount relationship table,
		//and should be processed when the worker takes effect
		for _, volumeRelation := range component.VolumeRelations {
			if vol, ok := existVolume[volumeRelation.Key()]; ok {
				volRels = append(volRels, volumeRelation.DbModel(app.TenantEnvID, component.ComponentBase.ComponentID, vol.HostPath, vol.VolumeType))
			}
		}
	}
	if err := db.GetManager().TenantEnvServiceMountRelationDaoTransactions(tx).DeleteByComponentIDs(componentIDs); err != nil {
		return err
	}
	return db.GetManager().TenantEnvServiceMountRelationDaoTransactions(tx).CreateOrUpdateVolumeRelsInBatch(volRels)
}

// SyncComponentVolumes -
func (s *ServiceAction) SyncComponentVolumes(tx *gorm.DB, components []*api_model.Component) error {
	var (
		componentIDs []string
		volumes      []*dbmodel.TenantEnvServiceVolume
	)
	for _, component := range components {
		if component.Volumes == nil {
			continue
		}
		componentIDs = append(componentIDs, component.ComponentBase.ComponentID)
		for _, volume := range component.Volumes {
			volumes = append(volumes, volume.DbModel(component.ComponentBase.ComponentID))
		}
	}
	existVolumes, err := s.getExistVolumes(componentIDs)
	if err != nil {
		return err
	}
	deleteVolumeIDs := s.getDeleteVolumeIDs(existVolumes, volumes)
	createOrUpdates := s.getCreateOrUpdateVolumes(existVolumes, volumes)
	if err := db.GetManager().TenantEnvServiceVolumeDaoTransactions(tx).DeleteByVolumeIDs(deleteVolumeIDs); err != nil {
		return err
	}
	return db.GetManager().TenantEnvServiceVolumeDaoTransactions(tx).CreateOrUpdateVolumesInBatch(createOrUpdates)
}

func (s *ServiceAction) getExistVolumes(componentIDs []string) (existVolumes map[string]*dbmodel.TenantEnvServiceVolume, err error) {
	existVolumes = make(map[string]*dbmodel.TenantEnvServiceVolume)
	volumes, err := db.GetManager().TenantEnvServiceVolumeDao().ListVolumesByComponentIDs(componentIDs)
	if err != nil {
		return nil, err
	}
	for _, volume := range volumes {
		existVolumes[volume.Key()] = volume
	}
	return existVolumes, nil
}

func (s *ServiceAction) getCreateOrUpdateVolumes(existVolumes map[string]*dbmodel.TenantEnvServiceVolume, incomeVolumes []*dbmodel.TenantEnvServiceVolume) (volumes []*dbmodel.TenantEnvServiceVolume) {
	for _, incomeVolume := range incomeVolumes {
		if _, ok := existVolumes[incomeVolume.Key()]; ok {
			incomeVolume.ID = existVolumes[incomeVolume.Key()].ID
		}
		volumes = append(volumes, incomeVolume)
	}
	return volumes
}

func (s *ServiceAction) getDeleteVolumeIDs(existVolumes map[string]*dbmodel.TenantEnvServiceVolume, incomeVolumes []*dbmodel.TenantEnvServiceVolume) (deleteVolumeIDs []uint) {
	newVolumes := make(map[string]struct{})
	for _, volume := range incomeVolumes {
		newVolumes[volume.Key()] = struct{}{}
	}
	for existKey, existVolume := range existVolumes {
		if _, ok := newVolumes[existKey]; !ok {
			deleteVolumeIDs = append(deleteVolumeIDs, existVolume.ID)
		}
	}
	return deleteVolumeIDs
}

// SyncComponentConfigFiles -
func (s *ServiceAction) SyncComponentConfigFiles(tx *gorm.DB, components []*api_model.Component) error {
	var (
		componentIDs []string
		configFiles  []*dbmodel.TenantEnvServiceConfigFile
	)
	for _, component := range components {
		if component.ConfigFiles == nil {
			continue
		}
		componentIDs = append(componentIDs, component.ComponentBase.ComponentID)
		for _, configFile := range component.ConfigFiles {
			configFiles = append(configFiles, configFile.DbModel(component.ComponentBase.ComponentID))
		}
	}
	if err := db.GetManager().TenantEnvServiceConfigFileDaoTransactions(tx).DeleteByComponentIDs(componentIDs); err != nil {
		return err
	}
	return db.GetManager().TenantEnvServiceConfigFileDaoTransactions(tx).CreateOrUpdateConfigFilesInBatch(configFiles)
}

// SyncComponentProbes -
func (s *ServiceAction) SyncComponentProbes(tx *gorm.DB, components []*api_model.Component) error {
	var (
		componentIDs []string
		probes       []*dbmodel.TenantEnvServiceProbe
	)
	for _, component := range components {
		componentIDs = append(componentIDs, component.ComponentBase.ComponentID)
		modes := make(map[string]struct{})
		for _, probe := range component.Probes {
			_, ok := modes[probe.Mode]
			if ok {
				continue
			}
			probes = append(probes, probe.DbModel(component.ComponentBase.ComponentID))
			modes[probe.Mode] = struct{}{}
		}
	}
	if err := db.GetManager().ServiceProbeDaoTransactions(tx).DeleteByComponentIDs(componentIDs); err != nil {
		return err
	}
	return db.GetManager().ServiceProbeDaoTransactions(tx).CreateOrUpdateProbesInBatch(probes)
}

// SyncComponentLabels -
func (s *ServiceAction) SyncComponentLabels(tx *gorm.DB, components []*api_model.Component) error {
	var (
		componentIDs []string
		labels       []*dbmodel.TenantEnvServiceLable
	)
	for _, component := range components {
		if component.Labels == nil {
			continue
		}
		componentIDs = append(componentIDs, component.ComponentBase.ComponentID)
		for _, label := range component.Labels {
			labels = append(labels, label.DbModel(component.ComponentBase.ComponentID))
		}
	}
	if err := db.GetManager().TenantEnvServiceLabelDaoTransactions(tx).DeleteByComponentIDs(componentIDs); err != nil {
		return err
	}
	return db.GetManager().TenantEnvServiceLabelDaoTransactions(tx).CreateOrUpdateLabelsInBatch(labels)
}

// SyncComponentPlugins -
func (s *ServiceAction) SyncComponentPlugins(tx *gorm.DB, app *dbmodel.Application, components []*api_model.Component) error {
	var (
		componentIDs           []string
		portConfigComponentIDs []string
		envComponentIDs        []string
		pluginRelations        []*dbmodel.TenantEnvServicePluginRelation
		pluginVersionEnvs      []*dbmodel.TenantEnvPluginVersionEnv
		pluginVersionConfigs   []*dbmodel.TenantEnvPluginVersionDiscoverConfig
		pluginStreamPorts      []*dbmodel.TenantEnvServicesStreamPluginPort
	)
	for _, component := range components {
		if component.Plugins == nil {
			continue
		}
		componentIDs = append(componentIDs, component.ComponentBase.ComponentID)
		for _, plugin := range component.Plugins {
			pluginRelations = append(pluginRelations, plugin.DbModel(component.ComponentBase.ComponentID))
			if plugin.ConfigEnvs.NormalEnvs != nil {
				envComponentIDs = append(envComponentIDs, component.ComponentBase.ComponentID)
				for _, versionEnv := range plugin.ConfigEnvs.NormalEnvs {
					pluginVersionEnvs = append(pluginVersionEnvs, versionEnv.DbModel(component.ComponentBase.ComponentID, plugin.PluginID))
				}
			}

			if configs := plugin.ConfigEnvs.ComplexEnvs; configs != nil {
				portConfigComponentIDs = append(portConfigComponentIDs, component.ComponentBase.ComponentID)
				if configs.BasePorts != nil && checkPluginHaveInbound(plugin.PluginModel) {
					psPorts := s.handlePluginMappingPort(app.TenantEnvID, component.ComponentBase.ComponentID, plugin.PluginModel, configs.BasePorts)
					pluginStreamPorts = append(pluginStreamPorts, psPorts...)
				}
				config, err := ffjson.Marshal(configs)
				if err != nil {
					return err
				}
				pluginVersionConfigs = append(pluginVersionConfigs, &dbmodel.TenantEnvPluginVersionDiscoverConfig{
					PluginID:  plugin.PluginID,
					ServiceID: component.ComponentBase.ComponentID,
					ConfigStr: string(config),
				})
			}
		}
	}

	if err := db.GetManager().TenantEnvServicesStreamPluginPortDaoTransactions(tx).DeleteByComponentIDs(portConfigComponentIDs); err != nil {
		return err
	}
	if err := db.GetManager().TenantEnvPluginVersionConfigDaoTransactions(tx).DeleteByComponentIDs(portConfigComponentIDs); err != nil {
		return err
	}
	if err := db.GetManager().TenantEnvServicePluginRelationDaoTransactions(tx).DeleteByComponentIDs(componentIDs); err != nil {
		return err
	}
	if err := db.GetManager().TenantEnvPluginVersionENVDaoTransactions(tx).DeleteByComponentIDs(envComponentIDs); err != nil {
		return err
	}

	if err := db.GetManager().TenantEnvServicePluginRelationDaoTransactions(tx).CreateOrUpdatePluginRelsInBatch(pluginRelations); err != nil {
		return err
	}
	if err := db.GetManager().TenantEnvPluginVersionENVDaoTransactions(tx).CreateOrUpdatePluginVersionEnvsInBatch(pluginVersionEnvs); err != nil {
		return err
	}
	if err := db.GetManager().TenantEnvServicesStreamPluginPortDaoTransactions(tx).CreateOrUpdateStreamPluginPortsInBatch(pluginStreamPorts); err != nil {
		return err
	}
	return db.GetManager().TenantEnvPluginVersionConfigDaoTransactions(tx).CreateOrUpdatePluginVersionConfigsInBatch(pluginVersionConfigs)
}

// handlePluginMappingPort -
func (s *ServiceAction) handlePluginMappingPort(tenantEnvID, componentID, pluginModel string, ports []*api_model.BasePort) []*dbmodel.TenantEnvServicesStreamPluginPort {
	existPorts := make(map[int]struct{})
	for _, port := range ports {
		existPorts[port.Port] = struct{}{}
	}

	minPort := 65301
	var newPorts []*dbmodel.TenantEnvServicesStreamPluginPort
	for _, port := range ports {
		newPort := &dbmodel.TenantEnvServicesStreamPluginPort{
			TenantEnvID:   tenantEnvID,
			ServiceID:     componentID,
			PluginModel:   pluginModel,
			ContainerPort: port.Port,
		}
		if _, ok := existPorts[minPort]; ok {
			minPort = minPort + 1
		}
		newPluginPort := minPort
		if _, ok := existPorts[newPluginPort]; ok {
			minPort = minPort + 1
			newPluginPort = minPort
		}

		existPorts[newPluginPort] = struct{}{}
		port.ListenPort = newPluginPort
		newPort.PluginPort = newPluginPort
		newPorts = append(newPorts, newPort)
	}
	return newPorts
}

// SyncComponentScaleRules -
func (s *ServiceAction) SyncComponentScaleRules(tx *gorm.DB, components []*api_model.Component) error {
	var (
		componentIDs         []string
		autoScaleRuleIDs     []string
		autoScaleRules       []*dbmodel.TenantEnvServiceAutoscalerRules
		autoScaleRuleMetrics []*dbmodel.TenantEnvServiceAutoscalerRuleMetrics
	)
	for _, component := range components {
		componentIDs = append(componentIDs, component.ComponentBase.ComponentID)
		autoScaleRuleIDs = append(autoScaleRuleIDs, component.AutoScaleRule.RuleID)
		autoScaleRules = append(autoScaleRules, component.AutoScaleRule.DbModel(component.ComponentBase.ComponentID))

		for _, metric := range component.AutoScaleRule.RuleMetrics {
			autoScaleRuleMetrics = append(autoScaleRuleMetrics, metric.DbModel(component.AutoScaleRule.RuleID))
		}
	}
	if err := db.GetManager().TenantEnvServceAutoscalerRulesDaoTransactions(tx).DeleteByComponentIDs(componentIDs); err != nil {
		return err
	}
	if err := db.GetManager().TenantEnvServceAutoscalerRuleMetricsDaoTransactions(tx).DeleteByRuleIDs(autoScaleRuleIDs); err != nil {
		return err
	}
	if err := db.GetManager().TenantEnvServceAutoscalerRulesDaoTransactions(tx).CreateOrUpdateScaleRulesInBatch(autoScaleRules); err != nil {
		return err
	}
	return db.GetManager().TenantEnvServceAutoscalerRuleMetricsDaoTransactions(tx).CreateOrUpdateScaleRuleMetricsInBatch(autoScaleRuleMetrics)
}

// SyncComponentEndpoints -
func (s *ServiceAction) SyncComponentEndpoints(tx *gorm.DB, components []*api_model.Component) error {
	var (
		componentIDs               []string
		thirdPartySvcDiscoveryCfgs []*dbmodel.ThirdPartySvcDiscoveryCfg
	)
	for _, component := range components {
		if component.Endpoint == nil {
			continue
		}
		componentIDs = append(componentIDs, component.ComponentBase.ComponentID)
		if component.Endpoint.Kubernetes != nil {
			thirdPartySvcDiscoveryCfgs = append(thirdPartySvcDiscoveryCfgs, component.Endpoint.DbModel(component.ComponentBase.ComponentID))
		}
	}

	if err := db.GetManager().ThirdPartySvcDiscoveryCfgDaoTransactions(tx).DeleteByComponentIDs(componentIDs); err != nil {
		return err
	}
	return db.GetManager().ThirdPartySvcDiscoveryCfgDaoTransactions(tx).CreateOrUpdate3rdSvcDiscoveryCfgInBatch(thirdPartySvcDiscoveryCfgs)
}

// Log returns the logs reader for a container in a pod, a pod or a component.
func (s *ServiceAction) Log(w http.ResponseWriter, r *http.Request, component *dbmodel.TenantEnvServices, podName, containerName string, follow bool) error {
	// If podName and containerName is missing, return the logs reader for the component
	// If containerName is missing, return the logs reader for the pod.
	if podName == "" || containerName == "" {
		// Only support return the logs reader for a container now.
		return errors.WithStack(bcode.NewBadRequest("the field 'podName' and 'containerName' is required"))
	}
	tenantEnv, err := db.GetManager().TenantEnvDao().GetTenantEnvByUUID(component.TenantEnvID)
	if err != nil {
		return fmt.Errorf("get tenant env info failure %s", err.Error())
	}
	request := s.kubeClient.CoreV1().Pods(tenantEnv.Namespace).GetLogs(podName, &corev1.PodLogOptions{
		Container: containerName,
		Follow:    follow,
		TailLines: util.Int64(500),
	})

	out, err := request.Stream(context.TODO())
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			return errors.Wrap(bcode.ErrPodNotFound, "get pod log")
		}
		return errors.Wrap(err, "get stream from request")
	}
	defer out.Close()

	w.Header().Set("Transfer-Encoding", "chunked")
	w.WriteHeader(http.StatusOK)

	// Flush headers, if possible
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}

	writer := flushwriter.Wrap(w)

	_, err = io.Copy(writer, out)
	if err != nil {
		if strings.HasSuffix(err.Error(), "write: broken pipe") {
			return nil
		}
		logrus.Warningf("write stream to response: %v", err)
	}
	return nil
}

// GetKubeResources get kube resources for component
func (s *ServiceAction) GetKubeResources(namespace, serviceID string, customSetting api_model.KubeResourceCustomSetting) (string, error) {
	if msgs := validation.IsDNS1123Label(customSetting.Namespace); len(msgs) > 0 {
		return "", fmt.Errorf("invalid namespace name: %s", customSetting.Namespace)
	}
	selectors := []labels.Selector{
		labels.SelectorFromSet(labels.Set{"service_id": serviceID}),
	}
	resources := kube.GetResourcesYamlFormat(s.kubeClient, namespace, selectors, &customSetting)
	return resources, nil
}

// TransStatus trans service status
func TransStatus(eStatus string) string {
	switch eStatus {
	case "starting":
		return "启动中"
	case "abnormal":
		return "运行异常"
	case "upgrade":
		return "升级中"
	case "closed":
		return "已关闭"
	case "stopping":
		return "关闭中"
	case "checking":
		return "检测中"
	case "unusual":
		return "运行异常"
	case "running":
		return "运行中"
	case "failure":
		return "未知"
	case "undeploy":
		return "未部署"
	case "deployed":
		return "已部署"
	}
	return ""
}

// CreateBackup create backup for service resources and data
func (s *ServiceAction) CreateBackup(tenantEnvID, serviceID string) error {
	if !kube.IsVeleroInstalled(s.kubeClient, s.apiextClient) {
		return errors.New("集群中未检测到 Velero 服务，使用该功能请联系管理员安装 Velero 服务！")
	}
	tenantEnv, err := db.GetManager().TenantEnvDao().GetTenantEnvByUUID(tenantEnvID)
	if err != nil {
		return errors.New("环境不存在！")
	}
	volumes, _ := db.GetManager().TenantEnvServiceVolumeDao().GetTenantEnvServiceVolumesByServiceID(serviceID)
	if len(volumes) == 0 {
		return errors.New("当前组件没有挂载存储！")
	}

	// 1、如果当前组件没有处于运行中，则需要先启动组件
	pods, err := s.GetPods(serviceID)
	if err != nil {
		return errors.New("获取组件状态失败！")
	}
	if pods == nil || (len(pods.NewPods) == 0 && len(pods.OldPods) == 0) {
		return errors.New("当前组件未运行，请先启动组件！")
	}

	selector := labels.SelectorFromSet(labels.Set{
		"wutong.io/service_id": serviceID,
	})

	// 2、校验是否存在未完成的备份任务
	histories, err := kube.GetVeleroCachedResources(s.kubeClient, s.veleroClient, s.apiextClient).BackupLister.Backups("velero").List(selector)
	if err != nil {
		logrus.Errorf("get velero backup history failed, error: %s", err.Error())
		return errors.New("校验历史备份数据失败！")
	}
	for _, history := range histories {
		if history.Status.Phase != "Completed" {
			return errors.New("当前组件还存在未完成的备份任务！")
		}
	}

	backup := &velerov1.Backup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceID + "-" + time.Now().Format("20060102150405"),
			Namespace: "velero",
			Labels: map[string]string{
				"wutong.io/service_id": serviceID,
			},
		},
		Spec: velerov1.BackupSpec{
			CSISnapshotTimeout: metav1.Duration{Duration: 10 * time.Minute},
			DefaultVolumesToFsBackup: func() *bool {
				b := true
				return &b
			}(),
			IncludedNamespaces: []string{tenantEnv.Namespace},
			LabelSelector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"service_id": serviceID, // 梧桐组件的 Label
				},
			},
			StorageLocation: "default",
		},
	}
	_, err = s.veleroClient.VeleroV1().Backups("velero").Create(context.Background(), backup, metav1.CreateOptions{})
	if err != nil {
		logrus.Errorf("create velero backup failed, error: %s", err.Error())
		return errors.New("创建备份任务失败！")
	}
	return nil
}

// CreateBackupSchedule create backup schedule for service resources and data
func (s *ServiceAction) CreateBackupSchedule(tenantEnvID, serviceID, cron string) error {
	if !kube.IsVeleroInstalled(s.kubeClient, s.apiextClient) {
		return errors.New("集群中未检测到 Velero 服务，使用该功能请联系管理员安装 Velero 服务！")
	}
	tenantEnv, err := db.GetManager().TenantEnvDao().GetTenantEnvByUUID(tenantEnvID)
	if err != nil {
		return errors.New("环境不存在！")
	}
	volumes, _ := db.GetManager().TenantEnvServiceVolumeDao().GetTenantEnvServiceVolumesByServiceID(serviceID)
	if len(volumes) == 0 {
		return errors.New("当前组件没有挂载存储！")
	}

	// 1、如果当前组件没有处于运行中，则需要先启动组件
	// pods, err := s.GetPods(serviceID)
	// if err != nil {
	// 	return errors.New("获取组件状态失败！")
	// }
	// if pods == nil || (len(pods.NewPods) == 0 && len(pods.OldPods) == 0) {
	// 	return errors.New("当前组件未运行，请先启动组件！")
	// }

	// 2、校验是否存在未完成的备份定时任务
	_, err = kube.GetVeleroCachedResources(s.kubeClient, s.veleroClient, s.apiextClient).ScheduleLister.Schedules("velero").Get(serviceID)
	if err == nil {
		logrus.Error("schedule already exists")
		return errors.New("当前组件已存在定时备份计划！")
	}
	schedule := &velerov1.Schedule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceID,
			Namespace: "velero",
			Labels: map[string]string{
				"wutong.io/service_id": serviceID,
			},
		},
		Spec: velerov1.ScheduleSpec{
			Schedule: cron,
			Template: velerov1.BackupSpec{
				CSISnapshotTimeout: metav1.Duration{Duration: 10 * time.Minute},
				DefaultVolumesToFsBackup: func() *bool {
					b := true
					return &b
				}(),
				IncludedNamespaces: []string{tenantEnv.Namespace},
				LabelSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"service_id": serviceID, // 梧桐组件的 Label
					},
				},
				StorageLocation: "default",
			},
		},
	}
	_, err = s.veleroClient.VeleroV1().Schedules("velero").Create(context.Background(), schedule, metav1.CreateOptions{})
	if err != nil {
		logrus.Errorf("create velero backup schedule failed, error: %s", err.Error())
		return errors.New("创建定时备份计划失败！")
	}
	return nil
}

// DeleteBackupSchedule
func (s *ServiceAction) DeleteBackupSchedule(serviceID string) error {
	if !kube.IsVeleroInstalled(s.kubeClient, s.apiextClient) {
		return errors.New("集群中未检测到 Velero 服务，使用该功能请联系管理员安装 Velero 服务！")
	}

	err := s.veleroClient.VeleroV1().Schedules("velero").Delete(context.Background(), serviceID, metav1.DeleteOptions{})
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			return nil
		}
		return errors.New("删除备份计划失败！")
	}
	return nil
}

// DownloadBackup download backup data
func (s *ServiceAction) DownloadBackup(serviceID, backupID string) ([]byte, error) {
	if !kube.IsVeleroInstalled(s.kubeClient, s.apiextClient) {
		return nil, errors.New("集群中未检测到 Velero 服务，使用该功能请联系管理员安装 Velero 服务！")
	}

	backup, err := kube.GetVeleroCachedResources(s.kubeClient, s.veleroClient, s.apiextClient).BackupLister.Backups("velero").Get(backupID)
	if err != nil {
		logrus.Errorf("download backup data failed, error: %s", err.Error())
		return nil, errors.New("获取备份失败！")
	}

	if backup.Labels["wutong.io/service_id"] != serviceID {
		return nil, errors.New("当前备份不属于该组件！")
	}

	if backup.Status.Phase != "Completed" {
		return nil, errors.New("当前备份还未完成！")
	}

	veleroStatus := kube.GetVeleroStatus(s.kubeClient, s.veleroClient, s.apiextClient)
	if veleroStatus == nil {
		return nil, errors.New("获取 Velero 存储信息失败！")
	}

	object := fmt.Sprintf("/backups/%s/%s.tar.gz", backupID, backupID)

	// Minio client
	// useSSL := u.Scheme == "https"
	// minioClient, err := minio.NewWithRegion(u.Host, accessKey, secretKey, useSSL, region)
	// if err != nil {
	// 	logrus.Errorf("download backup data failed, bucket: %s, object: %s, error: %s", veleroStatus.S3Bucket, object, err.Error())
	// 	return nil, errors.New("下载备份数据失败！")
	// }

	// obj, err := minioClient.GetObject(bucket, object, minio.GetObjectOptions{})
	// if err != nil {
	// 	return nil, errors.New("下载备份数据失败！")
	// }
	// defer obj.Close()

	// bytes, err := io.ReadAll(obj)
	// if err != nil {
	// 	return nil, errors.New("下载备份数据失败！")
	// }

	// S3 standard client
	disableSSL := veleroStatus.S3UrlScheme != "https"
	sess, err := session.NewSession(&aws.Config{
		Endpoint:         &veleroStatus.S3Host,
		Region:           &veleroStatus.S3Region,
		S3ForcePathStyle: util.Ptr(true),
		Credentials:      credentials.NewStaticCredentials(veleroStatus.S3AccessKeyID, veleroStatus.S3SecretAccessKey, ""),
		DisableSSL:       util.Ptr(disableSSL),
	})
	if err != nil {
		logrus.Errorf("download backup data failed, bucket: %s, object: %s, error: %s", veleroStatus.S3Bucket, object, err.Error())
		return nil, errors.New("下载备份数据失败！")
	}

	out, err := s3.New(sess).GetObject(&s3.GetObjectInput{
		Bucket: util.Ptr(veleroStatus.S3Bucket),
		Key:    util.Ptr(object),
	})
	if err != nil {
		logrus.Errorf("download backup data failed, bucket: %s, object: %s, error: %s", veleroStatus.S3Bucket, object, err.Error())
		return nil, errors.New("下载备份数据失败！")
	}
	defer out.Body.Close()

	tarBuffer := new(bytes.Buffer)
	tarWriter := tar.NewWriter(tarBuffer)

	manifests, err := io.ReadAll(out.Body)
	if err != nil {
		return nil, errors.New("下载备份数据失败！")
	}
	addFileToTar(tarWriter, "manifests.tar", manifests)

	pvbl, _ := kube.GetVeleroCachedResources(s.kubeClient, s.veleroClient, s.apiextClient).PodVolumeBackupLister.List(labels.SelectorFromSet(labels.Set{
		"velero.io/backup-name": backupID,
	}))

	for _, pvb := range pvbl {
		switch pvb.Spec.UploaderType {
		case velerov1.BackupRepositoryTypeKopia:
			// TODO: 不是很好的解决方案，后续需要优化
			// 1. 如果不存在，添加 kopia 操作用户，以命名空间名作为用户名
			kopiaUser := "kopia-" + pvb.Spec.Pod.Namespace
			exec.Command("adduser", kopiaUser, "-D").Run()

			// 2. 当前用户执行 kopia 用户命令
			rawConnectCmd := fmt.Sprintf("kopia repository connect s3 --endpoint %s --region %s --bucket %s --prefix kopia/%s/", veleroStatus.S3Host, veleroStatus.S3Region, veleroStatus.S3Bucket, pvb.Spec.Pod.Namespace)
			if disableSSL {
				rawConnectCmd = rawConnectCmd + " --disable-tls"
			}
			connetctCmd := exec.Command("su", kopiaUser, "-c", rawConnectCmd)
			err := connetctCmd.Run()
			if err != nil {
				logrus.Warningf("kopia repository connect error: %s", err.Error())
				continue
			}

			tmpFile := fmt.Sprintf("/home/%s/kopia-%s.tar", kopiaUser, pvb.Status.SnapshotID)
			rawRestoreCmd := fmt.Sprintf("kopia snapshot restore %s %s", pvb.Status.SnapshotID, tmpFile)
			restoreCmd := exec.Command("su", kopiaUser, "-c", rawRestoreCmd)
			err = restoreCmd.Run()
			if err != nil {
				logrus.Warningf("kopia snapshot restore error: %s", err.Error())
				continue
			}
			volumeData, err := os.ReadFile(tmpFile)
			if err != nil {
				logrus.Warningf("read kopia snapshot file error: %s", err.Error())
				continue
			}
			if len(volumeData) > 0 {
				addFileToTar(tarWriter, fmt.Sprintf("volumes/%s.tar", pvb.Name), volumeData)
			}
			os.Remove(tmpFile)
		case velerov1.BackupRepositoryTypeRestic:
			dumpCmd := exec.Command("restic", "-r", pvb.Spec.RepoIdentifier, "--verbose", "dump", pvb.Status.SnapshotID, "/")
			volumeData, err := dumpCmd.Output()
			if err != nil {
				logrus.Warningf("restic dump error: %s", err.Error())
				continue
			}
			if len(volumeData) > 0 {
				addFileToTar(tarWriter, fmt.Sprintf("volumes/%s.tar", pvb.Name), volumeData)
			}
		}
	}

	// 关闭 tar.Writer，完成归档文件的创建
	tarWriter.Close()

	gzipBuffer := new(bytes.Buffer)
	gzipWriter := gzip.NewWriter(gzipBuffer)

	_, err = io.Copy(gzipWriter, tarBuffer)
	if err != nil {
		return nil, errors.New("下载备份数据失败！")
	}

	// 关闭 gzip.Writer，完成压缩文件的创建
	gzipWriter.Close()

	return gzipBuffer.Bytes(), nil
}

// DeleteBackup delete backup for service resources and data
func (s *ServiceAction) DeleteBackup(serviceID, backupID string) error {
	if !kube.IsVeleroInstalled(s.kubeClient, s.apiextClient) {
		return errors.New("集群中未检测到 Velero 服务，使用该功能请联系管理员安装 Velero 服务！")
	}

	backup, err := kube.GetVeleroCachedResources(s.kubeClient, s.veleroClient, s.apiextClient).BackupLister.Backups("velero").Get(backupID)
	if err != nil {
		return errors.New("获取待删除备份记录失败！")
	}

	if backup.Labels["wutong.io/service_id"] != serviceID {
		return errors.New("当前备份不属于该组件！")
	}

	dbrl, _ := kube.GetVeleroCachedResources(s.kubeClient, s.veleroClient, s.apiextClient).DeleteBackupRequestLister.List(labels.SelectorFromSet(labels.Set{
		"velero.io/backup-name": backupID,
	}))
	if len(dbrl) > 0 {
		return errors.New("当前备份记录已存在删除请求，请稍后再试！")
	}

	_, err = s.veleroClient.VeleroV1().DeleteBackupRequests("velero").Create(context.Background(), &velerov1.DeleteBackupRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceID + "-" + time.Now().Format("20060102150405"),
			Namespace: "velero",
			Labels: map[string]string{
				"velero.io/backup-name": backupID,
			},
		},
		Spec: velerov1.DeleteBackupRequestSpec{
			BackupName: backupID,
		},
	}, metav1.CreateOptions{})
	if err != nil {
		return errors.New("创建删除备份请求失败！")
	}
	return nil
}

// CreateRestore create restore for service resources and data from backup
func (s *ServiceAction) CreateRestore(tenantEnvID, serviceID, BackupID string) error {
	if !kube.IsVeleroInstalled(s.kubeClient, s.apiextClient) {
		return errors.New("集群中未检测到 Velero 服务，使用该功能请联系管理员安装 Velero 服务！")
	}

	// 1、如果当前组件处于运行中，则先关闭组件
	pods, err := s.GetPods(serviceID)
	if err != nil {
		return errors.New("获取组件状态失败！")
	}
	if pods != nil {
		if len(pods.NewPods) > 0 || len(pods.OldPods) > 0 {
			return errors.New("当前组件处于运行中，请先关闭组件！")
		}
	}

	// 2、校验备份数据是否存在
	backup, err := kube.GetVeleroCachedResources(s.kubeClient, s.veleroClient, s.apiextClient).BackupLister.Backups("velero").Get(BackupID)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			return errors.New("备份数据不存在！")
		}
		return errors.New("获取备份数据失败！")
	}

	if backup.Labels["wutong.io/service_id"] != serviceID {
		return errors.New("当前备份不属于该组件！")
	}

	restore := &velerov1.Restore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceID + "-" + time.Now().Format("20060102150405"),
			Namespace: "velero",
			Labels:    backup.Labels,
		},
		Spec: velerov1.RestoreSpec{
			BackupName: BackupID,
			ExcludedResources: []string{
				"nodes",
				"events",
				"endpoints",
				"events.events.k8s.io",
				"backups.velero.io",
				"restores.velero.io",
				"resticrepositories.velero.io",
				"csinodes.storage.k8s.io",
				"volumeattachments.storage.k8s.io",
				"backuprepositories.velero.io",
			},
		},
	}

	_, err = s.veleroClient.VeleroV1().Restores("velero").Create(context.Background(), restore, metav1.CreateOptions{})
	if err != nil {
		return errors.New("创建还原任务失败！")
	}
	return nil
}

// DeleteRestore delete restore for service resources and data from backup
func (s *ServiceAction) DeleteRestore(serviceID, restoreID string) error {
	if !kube.IsVeleroInstalled(s.kubeClient, s.apiextClient) {
		return errors.New("集群中未检测到 Velero 服务，使用该功能请联系管理员安装 Velero 服务！")
	}

	r, err := kube.GetVeleroCachedResources(s.kubeClient, s.veleroClient, s.apiextClient).RestoreLister.Restores("velero").Get(restoreID)
	if err != nil {
		return errors.New("获取待删除还原记录失败！")
	}

	if r.Labels["wutong.io/service_id"] != serviceID {
		return errors.New("当前还原记录不属于该组件！")
	}

	err = s.veleroClient.VeleroV1().Restores("velero").Delete(context.Background(), restoreID, metav1.DeleteOptions{})
	if err != nil {
		logrus.Error(err)
		return errors.New("删除恢复历史失败！")
	}
	return nil
}

// GetBackupSchedule get velero backup schedule
func (s *ServiceAction) GetBackupSchedule(tenantEnvID, serviceID string) (*api_model.BackupSchedule, bool) {
	if !kube.IsVeleroInstalled(s.kubeClient, s.apiextClient) {
		return nil, false
	}

	schedule, err := kube.GetVeleroCachedResources(s.kubeClient, s.veleroClient, s.apiextClient).ScheduleLister.Schedules("velero").Get(serviceID)
	if err != nil {
		return nil, false
	}

	result := &api_model.BackupSchedule{
		ScheduleID: serviceID,
		ServiceID:  serviceID,
		Cron:       schedule.Spec.Schedule,
	}
	return result, true
}

// BackupRecords get velero backup histories
func (s *ServiceAction) BackupRecords(tenantEnvID, serviceID string) ([]*api_model.BackupRecord, error) {
	if !kube.IsVeleroInstalled(s.kubeClient, s.apiextClient) {
		return nil, nil
	}
	var result []*api_model.BackupRecord
	backups, err := kube.GetVeleroCachedResources(s.kubeClient, s.veleroClient, s.apiextClient).BackupLister.Backups("velero").List(labels.SelectorFromSet(labels.Set{
		"wutong.io/service_id": serviceID,
	}))
	if err != nil {
		return nil, errors.New("获取历史备份数据失败！")
	}
	for _, backup := range backups {
		pvbs, _ := kube.GetVeleroCachedResources(s.kubeClient, s.veleroClient, s.apiextClient).PodVolumeBackupLister.PodVolumeBackups("velero").List(labels.SelectorFromSet(labels.Set{
			"velero.io/backup-name": backup.Name,
		}))
		var totalBytes, completedBytes int64
		for _, pvb := range pvbs {
			if pvb != nil {
				totalBytes += pvb.Status.Progress.TotalBytes
				completedBytes += pvb.Status.Progress.BytesDone
			}
		}
		var totalItems, completedItems int
		if backup.Status.Progress != nil {
			totalItems = backup.Status.Progress.TotalItems
			completedItems = backup.Status.Progress.ItemsBackedUp
		}
		result = append(result, &api_model.BackupRecord{
			BackupID:       backup.Name,
			ServiceID:      serviceID,
			Mode:           backupMode(backup),
			CreatedAt:      convertMetaV1Time(backup.Status.StartTimestamp),
			CompletedAt:    convertMetaV1Time(backup.Status.CompletionTimestamp),
			ExpiredAt:      convertMetaV1Time(backup.Status.Expiration),
			Size:           formatBytesSize(totalBytes),
			ProgressRate:   formatProcessRate(totalBytes, completedBytes),
			CompletedItems: completedItems,
			TotalItems:     totalItems,
			Status:         string(backup.Status.Phase),
		})
	}
	return result, nil
}

// RestoreRecords get velero restore histories
func (s *ServiceAction) RestoreRecords(tenantEnvID, serviceID string) ([]*api_model.RestoreRecord, error) {
	if !kube.IsVeleroInstalled(s.kubeClient, s.apiextClient) {
		return nil, nil
	}
	var result []*api_model.RestoreRecord
	restores, err := kube.GetVeleroCachedResources(s.kubeClient, s.veleroClient, s.apiextClient).RestoreLister.Restores("velero").List(labels.SelectorFromSet(labels.Set{
		"wutong.io/service_id": serviceID,
	}))
	if err != nil {
		return nil, errors.New("获取历史恢复数据失败！")
	}
	for _, restore := range restores {
		pvbs, _ := kube.GetVeleroCachedResources(s.kubeClient, s.veleroClient, s.apiextClient).PodVolumeRestoreLister.PodVolumeRestores("velero").List(labels.SelectorFromSet(labels.Set{
			"velero.io/restore-name": restore.Name,
		}))
		var totalBytes, completedBytes int64
		for _, pvb := range pvbs {
			if pvb != nil {
				totalBytes += pvb.Status.Progress.TotalBytes
				completedBytes += pvb.Status.Progress.BytesDone
			}
		}
		var totalItems, completedItems int
		if restore.Status.Progress != nil {
			totalItems = restore.Status.Progress.TotalItems
			completedItems = restore.Status.Progress.ItemsRestored
		}
		result = append(result, &api_model.RestoreRecord{
			RestoreID:      restore.Name,
			BackupID:       restore.Spec.BackupName,
			ServiceID:      serviceID,
			CreatedAt:      convertMetaV1Time(restore.Status.StartTimestamp),
			CompletedAt:    convertMetaV1Time(restore.Status.CompletionTimestamp),
			Size:           formatBytesSize(totalBytes),
			ProgressRate:   formatProcessRate(totalBytes, completedBytes),
			CompletedItems: completedItems,
			TotalItems:     totalItems,
			Status:         string(restore.Status.Phase),
		})
	}
	return result, nil
}

func backupMode(backup *velerov1.Backup) string {
	if _, ok := backup.Labels["velero.io/schedule-name"]; ok {
		return "Manual"
	}
	return "Scheduled"
}

func formatBytesSize(size int64) string {
	if size == 0 {
		return "-"
	}
	return humanize.Bytes(uint64(size))
}

func formatProcessRate(total, completed int64) string {
	if total == 0 {
		return "-"
	}
	return fmt.Sprintf("%s%%", fmt.Sprintf("%.2f", float64(completed)/float64(total)*100))
}

func convertMetaV1Time(t *metav1.Time) string {
	if t == nil {
		return "-"
	}
	return t.Format("2006-01-02 15:04:05")
}

func addFileToTar(tarWriter *tar.Writer, fileName string, content []byte) {
	header := &tar.Header{
		Name: fileName,
		Mode: 0644,
		Size: int64(len(content)),
	}
	err := tarWriter.WriteHeader(header)
	if err != nil {
		panic(err)
	}
	_, err = tarWriter.Write(content)
	if err != nil {
		panic(err)
	}
}
