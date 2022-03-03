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
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//Manager manager
type Manager struct {
	ctx               context.Context
	cfg               option.Config
	store             store.Storer
	dbmanager         db.Manager
	controllerManager *controller.Manager
	garbageCollector  *gc.GarbageCollector
}

//NewManager now handle
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

//ErrCallback do not handle this task
var ErrCallback = fmt.Errorf("callback task to mq")

func (m *Manager) checkCount() bool {
	return m.controllerManager.GetControllerSize() > m.cfg.MaxTasks
}

//AnalystToExec analyst exec
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
	case "delete_tenant":
		logrus.Info("start a 'delete_tenant' task worker")
		return m.deleteTenant(task)
	case "refreshhpa":
		logrus.Info("start a 'refreshhpa' task worker")
		return m.ExecRefreshHPATask(task)
	default:
		logrus.Warning("task can not execute because no type is identified")
		return nil
	}
}

//startExec exec start service task
func (m *Manager) startExec(task *model.Task) error {
	body, ok := task.Body.(model.StartTaskBody)
	if !ok {
		logrus.Errorf("start body convert to taskbody error")
		return fmt.Errorf("start body convert to taskbody error")
	}
	logger := event.GetManager().GetLogger(body.EventID)
	appService := m.store.GetAppService(body.ServiceID)
	if appService != nil && !appService.IsClosed() {
		logger.Info("component is not closed, can not start", event.GetLastLoggerOption())
		event.GetManager().ReleaseLogger(logger)
		return nil
	}
	newAppService, err := conversion.InitAppService(m.dbmanager, body.ServiceID, body.Configs)
	if err != nil {
		logrus.Errorf("component init create failure:%s", err.Error())
		logger.Error("component init create failure", event.GetCallbackLoggerOption())
		event.GetManager().ReleaseLogger(logger)
		return fmt.Errorf("application init create failure")
	}
	newAppService.Logger = logger
	//regist new app service
	m.store.RegistAppService(newAppService)
	err = m.controllerManager.StartController(controller.TypeStartController, *newAppService)
	if err != nil {
		logrus.Errorf("component run start controller failure:%s", err.Error())
		logger.Error("component run start controller failure", event.GetCallbackLoggerOption())
		event.GetManager().ReleaseLogger(logger)
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
	logger := event.GetManager().GetLogger(body.EventID)
	appService := m.store.GetAppService(body.ServiceID)
	if appService == nil {
		logger.Info("component is closed, can not stop", event.GetLastLoggerOption())
		event.GetManager().ReleaseLogger(logger)
		return nil
	}
	appService.Logger = logger
	for k, v := range body.Configs {
		appService.ExtensionSet[k] = v
	}
	err := m.controllerManager.StartController(controller.TypeStopController, *appService)
	if err != nil {
		logrus.Errorf("component run  stop controller failure:%s", err.Error())
		logger.Info("component run stop controller failure", event.GetCallbackLoggerOption())
		event.GetManager().ReleaseLogger(logger)
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
	logger := event.GetManager().GetLogger(body.EventID)
	appService := m.store.GetAppService(body.ServiceID)
	if appService == nil {
		logger.Info("component is closed, can not stop", event.GetLastLoggerOption())
		event.GetManager().ReleaseLogger(logger)
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
		logger.Info("component run restart controller failure", event.GetCallbackLoggerOption())
		event.GetManager().ReleaseLogger(logger)
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

	logger := event.GetManager().GetLogger(body.EventID)
	service, err := db.GetManager().TenantServiceDao().GetServiceByID(body.ServiceID)
	if err != nil {
		logger.Error("Get app base info failure", event.GetCallbackLoggerOption())
		event.GetManager().ReleaseLogger(logger)
		logrus.Errorf("horizontal_scaling get rc error. %v", err)
		err = fmt.Errorf("a")
		return
	}
	appService := m.store.GetAppService(body.ServiceID)
	if appService == nil || appService.IsClosed() {
		logger.Info("service is closed, no need handle", event.GetLastLoggerOption())
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
		scalingRecord := &dbmodel.TenantServiceScalingRecords{
			ServiceID:   body.ServiceID,
			EventName:   util.NewUUID(),
			RecordType:  "manual",
			Reason:      reason,
			Count:       1,
			Description: desc,
			Operator:    body.Username,
			LastTime:    time.Now(),
		}
		if err := db.GetManager().TenantServiceScalingRecordsDao().AddModel(scalingRecord); err != nil {
			logrus.Warningf("save scaling record: %v", err)
		}
	}()

	appService.Logger = logger
	appService.Replicas = service.Replicas
	err = m.controllerManager.StartController(controller.TypeScalingController, *appService)
	if err != nil {
		logrus.Errorf("component run  scaling controller failure:%s", err.Error())
		logger.Info("component run scaling controller failure", event.GetCallbackLoggerOption())
		event.GetManager().ReleaseLogger(logger)
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
	logger := event.GetManager().GetLogger(body.EventID)
	service, err := db.GetManager().TenantServiceDao().GetServiceByID(body.ServiceID)
	if err != nil {
		logrus.Errorf("vertical_scaling get rc error. %v", err)
		logger.Error("Get app base info failure", event.GetCallbackLoggerOption())
		event.GetManager().ReleaseLogger(logger)
		return fmt.Errorf("vertical_scaling get rc error. %v", err)
	}
	appService := m.store.GetAppService(body.ServiceID)
	if appService == nil || appService.IsClosed() {
		logger.Info("service is closed,no need handle", event.GetLastLoggerOption())
		event.GetManager().ReleaseLogger(logger)
		return nil
	}
	appService.ContainerCPU = service.ContainerCPU
	appService.ContainerMemory = service.ContainerMemory
	appService.ContainerGPU = service.ContainerGPU
	appService.Logger = logger
	newAppService, err := conversion.InitAppService(m.dbmanager, body.ServiceID, nil)
	if err != nil {
		logrus.Errorf("component init create failure:%s", err.Error())
		logger.Error("component init create failure", event.GetCallbackLoggerOption())
		event.GetManager().ReleaseLogger(logger)
		return fmt.Errorf("application init create failure")
	}
	newAppService.Logger = logger
	appService.SetUpgradePatch(newAppService)
	err = m.controllerManager.StartController(controller.TypeUpgradeController, *newAppService)
	if err != nil {
		logrus.Errorf("component run  vertical scaling(upgrade) controller failure:%s", err.Error())
		logger.Info("component run vertical scaling(upgrade) controller failure", event.GetCallbackLoggerOption())
		event.GetManager().ReleaseLogger(logger)
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
	logger := event.GetManager().GetLogger(body.EventID)
	newAppService, err := conversion.InitAppService(m.dbmanager, body.ServiceID, body.Configs)
	if err != nil {
		logrus.Errorf("component init create failure:%s", err.Error())
		logger.Error("component init create failure", event.GetCallbackLoggerOption())
		event.GetManager().ReleaseLogger(logger)
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
			logger.Info("component run start controller failure", event.GetCallbackLoggerOption())
			event.GetManager().ReleaseLogger(logger)
			return fmt.Errorf("component start failure")
		}
		logrus.Infof("service(%s) %s working is running.", body.ServiceID, "start")
		return nil
	}
	if err := oldAppService.SetUpgradePatch(newAppService); err != nil {
		if err.Error() == "no upgrade" {
			logger.Info("component no change no need upgrade.", event.GetLastLoggerOption())
			return nil
		}
		logrus.Errorf("component get upgrade info error:%s", err.Error())
		logger.Error(fmt.Sprintf("component get upgrade info error:%s", err.Error()), event.GetCallbackLoggerOption())
		return nil
	}
	//if service already deploy,upgrade it:
	err = m.controllerManager.StartController(controller.TypeUpgradeController, *newAppService)
	if err != nil {
		logrus.Errorf("component run  upgrade controller failure:%s", err.Error())
		logger.Info("component run upgrade controller failure", event.GetCallbackLoggerOption())
		event.GetManager().ReleaseLogger(logger)
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
	svc, err := db.GetManager().TenantServiceDao().GetServiceByID(body.ServiceID)
	if err != nil {
		logrus.Errorf("error get TenantServices: %v", err)
		return fmt.Errorf("error get TenantServices: %v", err)
	}
	logger := event.GetManager().GetLogger(body.EventID)
	oldAppService := m.store.GetAppService(body.ServiceID)
	logrus.Debugf("body action: %s", body.Action)
	if svc.Kind != dbmodel.ServiceKindThirdParty.String() && !strings.HasPrefix(body.Action, "port") {
		if oldAppService == nil || oldAppService.IsClosed() {
			logrus.Debugf("service is closed, no need handle")
			logger.Info("service is closed,no need handle", event.GetLastLoggerOption())
			event.GetManager().ReleaseLogger(logger)
			return nil
		}
	}
	var newAppService *v1.AppService
	if svc.Kind == dbmodel.ServiceKindThirdParty.String() {
		newAppService, err = conversion.InitAppService(m.dbmanager, body.ServiceID, nil,
			"ServiceSource", "TenantServiceBase", "TenantServiceRegist")
	} else {
		newAppService, err = conversion.InitAppService(m.dbmanager, body.ServiceID, nil)
	}
	if err != nil {
		logrus.Errorf("component init create failure:%s", err.Error())
		logger.Error("component init create failure", event.GetCallbackLoggerOption())
		event.GetManager().ReleaseLogger(logger)
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

//applyPluginConfig apply service plugin config
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
	newApp, err := conversion.InitAppService(m.dbmanager, body.ServiceID, nil, "ServiceSource", "TenantServiceBase", "TenantServicePlugin")
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

func (m *Manager) deleteTenant(task *model.Task) (err error) {
	body, ok := task.Body.(*model.DeleteTenantTaskBody)
	if !ok {
		logrus.Errorf("can't convert %s to *model.DeleteTenantTaskBody", reflect.TypeOf(task.Body))
		err = fmt.Errorf("can't convert %s to *model.DeleteTenantTaskBody", reflect.TypeOf(task.Body))
		return
	}

	defer func() {
		if err == nil {
			return
		}
		logrus.Errorf("failed to delete tenant: %v", err)
		var tenant *dbmodel.Tenants
		tenant, err = db.GetManager().TenantDao().GetTenantByUUID(body.TenantID)
		if err != nil {
			err = fmt.Errorf("tenant id: %s; find tenant: %v", body.TenantID, err)
			return
		}
		tenant.Status = dbmodel.TenantStatusDeleteFailed.String()
		err := db.GetManager().TenantDao().UpdateModel(tenant)
		if err != nil {
			logrus.Errorf("update tenant_status to '%s': %v", tenant.Status, err)
			return
		}
	}()
	tenant, err := db.GetManager().TenantDao().GetTenantByUUID(body.TenantID)
	if err != nil {
		err = fmt.Errorf("tenant id: %s; find tenant: %v", body.TenantID, err)
		return
	}
	if err = m.cfg.KubeClient.CoreV1().Namespaces().Delete(context.Background(), tenant.Namespace, metav1.DeleteOptions{
		GracePeriodSeconds: util.Int64(0),
	}); err != nil && !k8sErrors.IsNotFound(err) {
		err = fmt.Errorf("delete namespace: %v", err)
		return
	}

	err = db.GetManager().TenantDao().DelByTenantID(body.TenantID)
	if err != nil {
		err = fmt.Errorf("delete tenant: %v", err)
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

	logger := event.GetManager().GetLogger(body.EventID)

	oldAppService := m.store.GetAppService(body.ServiceID)
	if oldAppService != nil && oldAppService.IsClosed() {
		logger.Info("application is closed, ignore task 'refreshhpa'", event.GetLastLoggerOption())
		event.GetManager().ReleaseLogger(logger)
		return nil
	}

	newAppService, err := conversion.InitAppService(m.dbmanager, body.ServiceID, nil)
	if err != nil {
		logrus.Errorf("component init create failure:%s", err.Error())
		logger.Error("component init create failure", event.GetCallbackLoggerOption())
		event.GetManager().ReleaseLogger(logger)
		return fmt.Errorf("component init create failure")
	}
	newAppService.Logger = logger
	newAppService.SetDeletedResources(oldAppService)

	err = m.controllerManager.StartController(controller.TypeControllerRefreshHPA, *newAppService)
	if err != nil {
		logrus.Errorf("component run  refreshhpa controller failure: %s", err.Error())
		logger.Error("component run refreshhpa controller failure", event.GetCallbackLoggerOption())
		event.GetManager().ReleaseLogger(logger)
		return fmt.Errorf("refresh hpa: %v", err)
	}

	logrus.Infof("rule id: %s; successfully refresh hpa", body.RuleID)
	return nil
}
