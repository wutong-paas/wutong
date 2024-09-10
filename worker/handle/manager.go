// Copyright (C) 2nilfmt.Errorf("a")4-2nilfmt.Errorf("a")8 Wutong Co., Ltd.
// WUTONG, component Management Platform

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

package handle

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/wutong-paas/wutong/cmd/worker/option"
	"github.com/wutong-paas/wutong/db"
	dbmodel "github.com/wutong-paas/wutong/db/model"
	"github.com/wutong-paas/wutong/event"
	"github.com/wutong-paas/wutong/util"
	"github.com/wutong-paas/wutong/worker/appm/controller"
	"github.com/wutong-paas/wutong/worker/appm/conversion"
	"github.com/wutong-paas/wutong/worker/appm/store"
	v1 "github.com/wutong-paas/wutong/worker/appm/types/v1"
	"github.com/wutong-paas/wutong/worker/discover/model"
	"github.com/wutong-paas/wutong/worker/gc"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Manager manager
type Manager struct {
	ctx               context.Context
	cfg               option.Config
	store             store.Storer
	dbmanager         db.Manager
	controllerManager *controller.Manager
	garbageCollector  *gc.GarbageCollector
}

// NewManager now handle
func NewManager(ctx context.Context,
	config option.Config,
	store store.Storer,
	controllerManager *controller.Manager,
	garbageCollector *gc.GarbageCollector) *Manager {

	return &Manager{
		ctx:               ctx,
		cfg:               config,
		dbmanager:         db.GetManager(),
		store:             store,
		controllerManager: controllerManager,
		garbageCollector:  garbageCollector,
	}
}

// ErrCallback do not handle this task
var ErrCallback = fmt.Errorf("callback task to mq")

func (m *Manager) checkCount() bool {
	return m.controllerManager.GetControllerSize() > m.cfg.MaxTasks
}

// AnalystToExec analyst exec
func (m *Manager) AnalystToExec(task *model.Task) error {
	if task == nil {
		return nil
	}
	//max worker count check
	if m.checkCount() {
		return ErrCallback
	}
	if !m.store.Ready() {
		return ErrCallback
	}
	switch task.Type {
	case "start":
		logrus.Info("start a 'start' task worker")
		return m.startExec(task)
	case "stop":
		logrus.Info("start a 'stop' task worker")
		return m.stopExec(task)
	case "restart":
		logrus.Info("start a 'restart' task worker")
		return m.restartExec(task)
	case "horizontal_scaling":
		logrus.Info("start a 'horizontal_scaling' task worker")
		return m.horizontalScalingExec(task)
	case "vertical_scaling":
		logrus.Info("start a 'vertical_scaling' task worker")
		return m.verticalScalingExec(task)
	case "rolling_upgrade":
		logrus.Info("start a 'rolling_upgrade' task worker")
		return m.rollingUpgradeExec(task)
	case "apply_rule":
		logrus.Info("start a 'apply_rule' task worker")
		return m.applyRuleExec(task)
	case "apply_plugin_config":
		logrus.Info("start a 'apply_plugin_config' task worker")
		return m.applyPluginConfig(task)
	case "service_gc":
		logrus.Info("start the 'service_gc' task")
		return m.ExecServiceGCTask(task)
	case "delete_tenant_env":
		logrus.Info("start a 'delete_tenant_env' task worker")
		return m.deleteTenantEnv(task)
	case "refreshhpa":
		logrus.Info("start a 'refreshhpa' task worker")
		return m.ExecRefreshHPATask(task)
	case "apply_registry_auth_secret":
		logrus.Info("start a 'apply_registry_auth_secret' task worker")
		return m.ExecApplyRegistryAuthSecretTask(task)
	case "export_helm_chart":
		logrus.Info("start a 'export_helm_chart' task worker")
		return m.ExecExportHelmChartTask(task)
	case "export_k8s_yaml":
		logrus.Info("start a 'export_k8s_yaml' task worker")
		return m.ExecExportK8sYamlTask(task)
	default:
		logrus.Warning("task can not execute because no type is identified")
		return nil
	}
}

// startExec exec start service task
func (m *Manager) startExec(task *model.Task) error {
	body, ok := task.Body.(model.StartTaskBody)
	if !ok {
		logrus.Errorf("start body convert to taskbody error")
		return fmt.Errorf("start body convert to taskbody error")
	}
	logger := event.GetLogger(body.EventID)
	appService := m.store.GetAppService(body.ServiceID)
	if appService != nil && !appService.IsClosed() {
		logger.Info("应用组件尚未关闭，无法启动", event.GetLastLoggerOption())
		event.CloseLogger(body.EventID)
		return nil
	}
	newAppService, err := conversion.InitAppService(m.dbmanager, body.ServiceID, body.Configs)
	if err != nil {
		logrus.Errorf("component init create failure:%s", err.Error())
		logger.Error("应用组件初始创建失败", event.GetCallbackLoggerOption())
		event.CloseLogger(body.EventID)
		return fmt.Errorf("application init create failure")
	}
	newAppService.Logger = logger
	//regist new app service
	m.store.RegistAppService(newAppService)
	err = m.controllerManager.StartController(controller.TypeStartController, *newAppService)
	if err != nil {
		logrus.Errorf("component run start controller failure:%s", err.Error())
		logger.Error("运行应用组件启动控制器失败", event.GetCallbackLoggerOption())
		event.CloseLogger(body.EventID)
		return fmt.Errorf("component start failure")
	}
	logrus.Infof("component(%s) %s working is running.", body.ServiceID, "start")
	return nil
}

func (m *Manager) stopExec(task *model.Task) error {
	body, ok := task.Body.(model.StopTaskBody)
	if !ok {
		logrus.Errorf("stop body convert to taskbody error")
		return fmt.Errorf("stop body convert to taskbody error")
	}
	logger := event.GetLogger(body.EventID)
	appService := m.store.GetAppService(body.ServiceID)
	if appService == nil {
		logger.Info("应用组件已经关闭", event.GetLastLoggerOption())
		event.CloseLogger(body.EventID)
		return nil
	}
	appService.Logger = logger
	for k, v := range body.Configs {
		appService.ExtensionSet[k] = v
	}
	err := m.controllerManager.StartController(controller.TypeStopController, *appService)
	if err != nil {
		logrus.Errorf("component run  stop controller failure:%s", err.Error())
		logger.Info("运行应用组件关闭控制器失败", event.GetCallbackLoggerOption())
		event.CloseLogger(body.EventID)
		return fmt.Errorf("component stop failure")
	}
	logrus.Infof("service(%s) %s working is running.", body.ServiceID, "stop")
	return nil
}

func (m *Manager) restartExec(task *model.Task) error {
	body, ok := task.Body.(model.RestartTaskBody)
	if !ok {
		logrus.Errorf("stop body convert to taskbody error")
		return fmt.Errorf("stop body convert to taskbody error")
	}
	logger := event.GetLogger(body.EventID)
	appService := m.store.GetAppService(body.ServiceID)
	if appService == nil {
		logger.Info("应用组件已关闭", event.GetLastLoggerOption())
		event.CloseLogger(body.EventID)
		return nil
	}
	appService.Logger = logger
	for k, v := range body.Configs {
		appService.ExtensionSet[k] = v
	}
	//first stop app
	err := m.controllerManager.StartController(controller.TypeRestartController, *appService)
	if err != nil {
		logrus.Errorf("component run restart controller failure:%s", err.Error())
		logger.Info("运行应用组件重启控制器失败", event.GetCallbackLoggerOption())
		event.CloseLogger(body.EventID)
		return fmt.Errorf("component restart failure")
	}
	logrus.Infof("service(%s) %s working is running.", body.ServiceID, "restart")
	return nil
}

func (m *Manager) horizontalScalingExec(task *model.Task) (err error) {
	body, ok := task.Body.(model.HorizontalScalingTaskBody)
	if !ok {
		logrus.Errorf("horizontal_scaling body convert to taskbody error")
		err = fmt.Errorf("a")
		return
	}

	logger := event.GetLogger(body.EventID)
	service, err := db.GetManager().TenantEnvServiceDao().GetServiceByID(body.ServiceID)
	if err != nil {
		logger.Error("获取应用组件基础信息失败", event.GetCallbackLoggerOption())
		event.CloseLogger(body.EventID)
		logrus.Errorf("horizontal_scaling get rc error. %v", err)
		err = fmt.Errorf("a")
		return
	}
	appService := m.store.GetAppService(body.ServiceID)
	if appService == nil || appService.IsClosed() {
		logger.Info("应用组件已关闭", event.GetLastLoggerOption())
		return
	}
	oldReplicas, newReplicas := appService.Replicas, service.Replicas

	defer func() {
		desc := "the replicas is scaling from %d to %d successfully"
		desc = fmt.Sprintf(desc, oldReplicas, newReplicas)
		reason := "SuccessfulRescale"
		if err != nil {
			desc = "the replicas is scaling from %d to %d: %v"
			desc = fmt.Sprintf(desc, oldReplicas, newReplicas, err)
			reason = "FailedRescale"
		}
		scalingRecord := &dbmodel.TenantEnvServiceScalingRecords{
			ServiceID:   body.ServiceID,
			EventName:   util.NewUUID(),
			RecordType:  "manual",
			Reason:      reason,
			Count:       1,
			Description: desc,
			Operator:    body.Username,
			LastTime:    time.Now(),
		}
		if err := db.GetManager().TenantEnvServiceScalingRecordsDao().AddModel(scalingRecord); err != nil {
			logrus.Warningf("save scaling record: %v", err)
		}
	}()

	appService.Logger = logger
	appService.Replicas = service.Replicas
	err = m.controllerManager.StartController(controller.TypeScalingController, *appService)
	if err != nil {
		logrus.Errorf("component run scaling controller failure:%s", err.Error())
		logger.Info("运行应用组件水平伸缩控制器失败", event.GetCallbackLoggerOption())
		event.CloseLogger(body.EventID)
		return
	}
	logrus.Infof("service(%s) %s working is running.", body.ServiceID, "scaling")
	return nil
}

func (m *Manager) verticalScalingExec(task *model.Task) error {
	body, ok := task.Body.(model.VerticalScalingTaskBody)
	if !ok {
		logrus.Errorf("vertical_scaling body convert to taskbody error")
		return fmt.Errorf("vertical_scaling body convert to taskbody error")
	}
	logger := event.GetLogger(body.EventID)
	service, err := db.GetManager().TenantEnvServiceDao().GetServiceByID(body.ServiceID)
	if err != nil {
		logrus.Errorf("vertical_scaling get rc error. %v", err)
		logger.Error("获取应用组件基础信息失败", event.GetCallbackLoggerOption())
		event.CloseLogger(body.EventID)
		return fmt.Errorf("vertical_scaling get rc error. %v", err)
	}
	appService := m.store.GetAppService(body.ServiceID)
	if appService == nil || appService.IsClosed() {
		logger.Info("应用组件已关闭", event.GetLastLoggerOption())
		event.CloseLogger(body.EventID)
		return nil
	}
	appService.ContainerRequestCPU = service.ContainerRequestCPU
	appService.ContainerCPU = service.ContainerCPU
	appService.ContainerRequestMemory = service.ContainerRequestMemory
	appService.ContainerMemory = service.ContainerMemory
	appService.ContainerGPUType = service.ContainerGPUType
	appService.ContainerGPU = service.ContainerGPU
	appService.Logger = logger
	newAppService, err := conversion.InitAppService(m.dbmanager, body.ServiceID, nil)
	if err != nil {
		logrus.Errorf("component init create failure:%s", err.Error())
		logger.Error("应用组件初始创建失败", event.GetCallbackLoggerOption())
		event.CloseLogger(body.EventID)
		return fmt.Errorf("application init create failure")
	}
	newAppService.Logger = logger
	appService.SetUpgradePatch(newAppService)
	err = m.controllerManager.StartController(controller.TypeUpgradeController, *newAppService)
	if err != nil {
		logrus.Errorf("component run  vertical scaling(upgrade) controller failure:%s", err.Error())
		logger.Info("运行应用组件资源更新控制器失败", event.GetCallbackLoggerOption())
		event.CloseLogger(body.EventID)
		return fmt.Errorf("application vertical scaling(upgrade) failure")
	}
	logrus.Infof("service(%s) %s working is running.", body.ServiceID, "vertical scaling")
	return nil
}

func (m *Manager) rollingUpgradeExec(task *model.Task) error {
	body, ok := task.Body.(model.RollingUpgradeTaskBody)
	if !ok {
		logrus.Error("rolling_upgrade body convert to taskbody error", task.Body)
		return fmt.Errorf("rolling_upgrade body convert to taskbody error")
	}
	logger := event.GetLogger(body.EventID)
	newAppService, err := conversion.InitAppService(m.dbmanager, body.ServiceID, body.Configs)
	if err != nil {
		logrus.Errorf("component init create failure:%s", err.Error())
		logger.Error("应用组件初始创建失败", event.GetCallbackLoggerOption())
		event.CloseLogger(body.EventID)
		return fmt.Errorf("component init create failure")
	}
	newAppService.Logger = logger
	oldAppService := m.store.GetAppService(body.ServiceID)
	// if service not deploy,start it
	if oldAppService == nil || oldAppService.IsClosed() {
		//regist new app service
		m.store.RegistAppService(newAppService)
		err = m.controllerManager.StartController(controller.TypeStartController, *newAppService)
		if err != nil {
			logrus.Errorf("component run  start controller failure:%s", err.Error())
			logger.Info("运行应用组件启动控制器失败", event.GetCallbackLoggerOption())
			event.CloseLogger(body.EventID)
			return fmt.Errorf("component start failure")
		}
		logrus.Infof("service(%s) %s working is running.", body.ServiceID, "start")
		return nil
	}
	if err := oldAppService.SetUpgradePatch(newAppService); err != nil {
		if err.Error() == "no upgrade" {
			logger.Info("应用组件无需更新", event.GetLastLoggerOption())
			return nil
		}
		logrus.Errorf("component get upgrade info error:%s", err.Error())
		logger.Error(fmt.Sprintf("获取应用组件更新信息失败，错误信息：%s", err.Error()), event.GetCallbackLoggerOption())
		return nil
	}
	//if service already deploy,upgrade it:
	err = m.controllerManager.StartController(controller.TypeUpgradeController, *newAppService)
	if err != nil {
		logrus.Errorf("component run  upgrade controller failure:%s", err.Error())
		logger.Info("运行应用组件更新控制器失败", event.GetCallbackLoggerOption())
		event.CloseLogger(body.EventID)
		return fmt.Errorf("component upgrade failure")
	}
	logrus.Infof("service(%s) %s working is running.", body.ServiceID, "upgrade")
	return nil
}

func (m *Manager) applyRuleExec(task *model.Task) error {
	body, ok := task.Body.(*model.ApplyRuleTaskBody)
	if !ok {
		logrus.Errorf("Can't convert %s to *model.ApplyRuleTaskBody", reflect.TypeOf(task.Body))
		return fmt.Errorf("can't convert %s to *model.ApplyRuleTaskBody", reflect.TypeOf(task.Body))
	}
	svc, err := db.GetManager().TenantEnvServiceDao().GetServiceByID(body.ServiceID)
	if err != nil {
		logrus.Errorf("error get TenantEnvServices: %v", err)
		return fmt.Errorf("error get TenantEnvServices: %v", err)
	}
	logger := event.GetLogger(body.EventID)
	oldAppService := m.store.GetAppService(body.ServiceID)
	logrus.Debugf("body action: %s", body.Action)
	if svc.Kind != dbmodel.ServiceKindThirdParty.String() && !strings.HasPrefix(body.Action, "port") {
		if oldAppService == nil || oldAppService.IsClosed() {
			logrus.Debugf("service is closed, no need handle")
			logger.Info("应用组件已关闭", event.GetLastLoggerOption())
			event.CloseLogger(body.EventID)
			return nil
		}
	}
	var newAppService *v1.AppService
	if svc.Kind == dbmodel.ServiceKindThirdParty.String() {
		newAppService, err = conversion.InitAppService(m.dbmanager, body.ServiceID, nil,
			"ServiceSource", "TenantEnvServiceBase", "TenantEnvServiceRegist")
	} else {
		newAppService, err = conversion.InitAppService(m.dbmanager, body.ServiceID, nil)
	}
	if err != nil {
		logrus.Errorf("component init create failure:%s", err.Error())
		logger.Error("应用组件初始创建失败", event.GetCallbackLoggerOption())
		event.CloseLogger(body.EventID)
		return fmt.Errorf("component init create failure")
	}
	newAppService.Logger = logger
	newAppService.SetDeletedResources(m.store.GetAppService(body.ServiceID))
	// update k8s resources
	newAppService.CustomParams = body.Limit
	err = m.controllerManager.StartController(controller.TypeApplyRuleController, *newAppService)
	if err != nil {
		logrus.Errorf("component apply rule controller failure:%s", err.Error())
		return fmt.Errorf("component apply rule controller failure:%s", err.Error())
	}

	return nil
}

// applyPluginConfig apply service plugin config
func (m *Manager) applyPluginConfig(task *model.Task) error {
	body, ok := task.Body.(*model.ApplyPluginConfigTaskBody)
	if !ok {
		logrus.Errorf("Can't convert %s to *model.ApplyPluginConfigTaskBody", reflect.TypeOf(task.Body))
		return fmt.Errorf("can't convert %s to *model.ApplyPluginConfigTaskBody", reflect.TypeOf(task.Body))
	}
	oldAppService := m.store.GetAppService(body.ServiceID)
	if oldAppService == nil || oldAppService.IsClosed() {
		logrus.Debugf("service is closed,no need handle")
		return nil
	}
	newApp, err := conversion.InitAppService(m.dbmanager, body.ServiceID, nil, "ServiceSource", "TenantEnvServiceBase", "TenantEnvServicePlugin")
	if err != nil {
		logrus.Errorf("component apply plugin config controller failure:%s", err.Error())
		return err
	}
	err = m.controllerManager.StartController(controller.TypeApplyConfigController, *newApp)
	if err != nil {
		logrus.Errorf("component apply plugin config controller failure:%s", err.Error())
		return fmt.Errorf("component apply plugin config controller failure:%s", err.Error())
	}
	return nil
}

// ExecServiceGCTask executes the 'service_gc' task
func (m *Manager) ExecServiceGCTask(task *model.Task) error {
	serviceGCReq, ok := task.Body.(model.ServiceGCTaskBody)
	if !ok {
		return fmt.Errorf("can not convert the request body to 'ServiceGCTaskBody'")
	}

	m.garbageCollector.DelLogFile(serviceGCReq)
	m.garbageCollector.DelPvPvcByServiceID(serviceGCReq)
	m.garbageCollector.DelVolumeData(serviceGCReq)
	m.garbageCollector.DelKubernetesObjects(serviceGCReq)
	return nil
}

func (m *Manager) deleteTenantEnv(task *model.Task) (err error) {
	body, ok := task.Body.(*model.DeleteTenantEnvTaskBody)
	if !ok {
		logrus.Errorf("can't convert %s to *model.DeleteTenantEnvTaskBody", reflect.TypeOf(task.Body))
		err = fmt.Errorf("can't convert %s to *model.DeleteTenantEnvTaskBody", reflect.TypeOf(task.Body))
		return
	}

	defer func() {
		if err == nil {
			return
		}
		logrus.Errorf("failed to delete tenantEnv: %v", err)
		var tenantEnv *dbmodel.TenantEnvs
		tenantEnv, err = db.GetManager().TenantEnvDao().GetTenantEnvByUUID(body.TenantEnvID)
		if err != nil {
			err = fmt.Errorf("tenant env id: %s; find tenantEnv: %v", body.TenantEnvID, err)
			return
		}
		tenantEnv.Status = dbmodel.TenantEnvStatusDeleteFailed.String()
		err := db.GetManager().TenantEnvDao().UpdateModel(tenantEnv)
		if err != nil {
			logrus.Errorf("update tenant_env_status to '%s': %v", tenantEnv.Status, err)
			return
		}
	}()
	tenantEnv, err := db.GetManager().TenantEnvDao().GetTenantEnvByUUID(body.TenantEnvID)
	if err != nil {
		err = fmt.Errorf("tenant env id: %s; find tenantEnv: %v", body.TenantEnvID, err)
		return
	}
	if err = m.cfg.KubeClient.CoreV1().Namespaces().Delete(context.Background(), tenantEnv.Namespace, metav1.DeleteOptions{
		GracePeriodSeconds: util.Int64(0),
	}); err != nil && !k8sErrors.IsNotFound(err) {
		err = fmt.Errorf("delete namespace: %v", err)
		return
	}

	err = db.GetManager().TenantEnvDao().DelByTenantEnvID(body.TenantEnvID)
	if err != nil {
		err = fmt.Errorf("delete tenantEnv: %v", err)
		return
	}

	return
}

// ExecRefreshHPATask executes a 'refresh hpa' task.
func (m *Manager) ExecRefreshHPATask(task *model.Task) error {
	body, ok := task.Body.(*model.RefreshHPATaskBody)
	if !ok {
		logrus.Errorf("exec task 'refreshhpa'; wrong type: %v", reflect.TypeOf(task))
		return fmt.Errorf("exec task 'refreshhpa': wrong input")
	}

	logger := event.GetLogger(body.EventID)

	oldAppService := m.store.GetAppService(body.ServiceID)
	if oldAppService != nil && oldAppService.IsClosed() {
		logger.Info("应用组件已关闭，水平伸缩任务已忽略", event.GetLastLoggerOption())
		event.CloseLogger(body.EventID)
		return nil
	}

	newAppService, err := conversion.InitAppService(m.dbmanager, body.ServiceID, nil)
	if err != nil {
		logrus.Errorf("component init create failure:%s", err.Error())
		logger.Error("应用组件初始创建失败", event.GetCallbackLoggerOption())
		event.CloseLogger(body.EventID)
		return fmt.Errorf("component init create failure")
	}
	newAppService.Logger = logger
	newAppService.SetDeletedResources(oldAppService)

	err = m.controllerManager.StartController(controller.TypeControllerRefreshHPA, *newAppService)
	if err != nil {
		logrus.Errorf("component run  refreshhpa controller failure: %s", err.Error())
		logger.Error("运行应用组件水平伸缩控制器失败", event.GetCallbackLoggerOption())
		event.CloseLogger(body.EventID)
		return fmt.Errorf("refresh hpa: %v", err)
	}

	logrus.Infof("rule id: %s; successfully refresh hpa", body.RuleID)
	return nil
}

func (m *Manager) ExecApplyRegistryAuthSecretTask(task *model.Task) error {
	body, ok := task.Body.(*model.ApplyRegistryAuthSecretTaskBody)
	if !ok {
		return fmt.Errorf("can't convert %s to *model.ApplyRegistryAuthSecretTaskBody", reflect.TypeOf(task.Body))
	}
	tenantEnv, err := m.dbmanager.TenantEnvDao().GetTenantEnvByUUID(body.TenantEnvID)
	if err != nil {
		logrus.Debugf("cant get tenant env by uuid: %s", body.TenantEnvID)
		return err
	}

	secretNameFrom := func(secretID string) string {
		return fmt.Sprintf("wt-registry-auth-%s", secretID)
	}

	secret, err := m.cfg.KubeClient.CoreV1().Secrets(tenantEnv.Namespace).Get(m.ctx, secretNameFrom(body.SecretID), metav1.GetOptions{})
	switch body.Action {
	case "apply":
		if err != nil {
			if k8sErrors.IsNotFound(err) {
				secret = &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      secretNameFrom(body.SecretID),
						Namespace: tenantEnv.Namespace,
						Labels: map[string]string{
							"tenant_id":                      tenantEnv.TenantID,
							"tenant_name":                    tenantEnv.TenantName,
							"tenant_env_id":                  tenantEnv.UUID,
							"tenant_env_name":                tenantEnv.Name,
							"creator":                        "Wutong",
							"wutong.io/registry-auth-secret": "true",
						},
					},
					Data: map[string][]byte{
						"Domain":   []byte(body.Domain),
						"Username": []byte(body.Username),
						"Password": []byte(body.Password),
					},
					Type: corev1.SecretTypeOpaque,
				}
				_, err = m.cfg.KubeClient.CoreV1().Secrets(tenantEnv.Namespace).Create(m.ctx, secret, metav1.CreateOptions{})
			} else {
				logrus.Errorf("get secret failure: %s", err.Error())
				return err
			}
		} else {
			secret.Data["Domain"] = []byte(body.Domain)
			secret.Data["Username"] = []byte(body.Username)
			secret.Data["Password"] = []byte(body.Password)
			_, err = m.cfg.KubeClient.CoreV1().Secrets(tenantEnv.Namespace).Update(m.ctx, secret, metav1.UpdateOptions{})
		}
		if err != nil {
			logrus.Errorf("apply secret failure: %s", err.Error())
			return err
		}
	case "delete":
		err := m.cfg.KubeClient.CoreV1().Secrets(tenantEnv.Namespace).Delete(m.ctx, secretNameFrom(body.SecretID), metav1.DeleteOptions{})
		if err != nil {
			logrus.Debugf("delete secret: %s", err.Error())
		}
		return err
	}
	return nil
}

func (m *Manager) ExecExportHelmChartTask(task *model.Task) error {
	body, ok := task.Body.(*model.ExportHelmChartOrK8sYamlTaskBody)
	if !ok {
		logrus.Error("export_helm_chart body convert to taskbody error", task.Body)
		return fmt.Errorf("export_helm_chart body convert to taskbody error")
	}
	eventID := util.NewUUID()
	logger := event.GetLogger(eventID)
	newAppService, err := conversion.InitAppService(m.dbmanager, body.ServiceID, nil)
	if err != nil {
		logrus.Errorf("component init create failure:%s", err.Error())
		logger.Error("应用组件初始创建失败", event.GetCallbackLoggerOption())
		event.CloseLogger(eventID)
		return fmt.Errorf("component init create failure")
	}
	newAppService.Logger = logger
	oldAppService := m.store.GetAppService(body.ServiceID)
	// if service not deploy,start it
	if oldAppService == nil || oldAppService.IsClosed() {
		//regist new app service
		m.store.RegistAppService(newAppService)

	}
	err = m.controllerManager.StartExportHelmChartController(body.AppName, body.AppVersion, body.End, *newAppService)
	if err != nil {
		logrus.Errorf("component run export_helm_chart controller failure:%s", err.Error())
		logger.Info("运行应用组件导出 Helm—Chart 控制器失败", event.GetCallbackLoggerOption())
		event.CloseLogger(eventID)
		return fmt.Errorf("component export_helm_chart failure")
	}
	logrus.Infof("service(%s) %s working is running.", body.ServiceID, "export_helm_chart")
	return nil

}

func (m *Manager) ExecExportK8sYamlTask(task *model.Task) error {
	body, ok := task.Body.(*model.ExportHelmChartOrK8sYamlTaskBody)
	if !ok {
		logrus.Error("export_k8s_yaml body convert to taskbody error", task.Body)
		return fmt.Errorf("export_k8s_yaml body convert to taskbody error")
	}
	eventID := util.NewUUID()
	logger := event.GetLogger(eventID)
	newAppService, err := conversion.InitAppService(m.dbmanager, body.ServiceID, nil)
	if err != nil {
		logrus.Errorf("component init create failure:%s", err.Error())
		logger.Error("应用组件初始创建失败", event.GetCallbackLoggerOption())
		event.CloseLogger(eventID)
		return fmt.Errorf("component init create failure")
	}
	newAppService.AppServiceBase.GovernanceMode = dbmodel.GovernanceModeKubernetesNativeService
	newAppService.Logger = logger
	oldAppService := m.store.GetAppService(body.ServiceID)
	// if service not deploy,start it
	if oldAppService == nil || oldAppService.IsClosed() {
		//regist new app service
		m.store.RegistAppService(newAppService)
	}
	err = m.controllerManager.StartExportK8sYamlController(body.AppName, body.AppVersion, body.End, *newAppService)
	if err != nil {
		logrus.Errorf("component run export_k8s_yaml controller failure:%s", err.Error())
		logger.Info("运行应用组件导出 K8s-Yaml 控制器失败", event.GetCallbackLoggerOption())
		event.CloseLogger(eventID)
		return fmt.Errorf("component start failure")
	}
	logrus.Infof("service(%s) %s working is running.", body.ServiceID, "export_k8s_yaml")
	return nil
}
