// Copyright (C) 2014-2018 Wutong Co., Ltd.
// WUTONG, Application Management Platform

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
	"github.com/wutong-paas/wutong/api/handler/group"
	"github.com/wutong-paas/wutong/api/handler/share"
	"github.com/wutong-paas/wutong/cmd/api/option"
	"github.com/wutong-paas/wutong/db"
	"github.com/wutong-paas/wutong/pkg/component/etcd"
	"github.com/wutong-paas/wutong/pkg/component/grpc"
	"github.com/wutong-paas/wutong/pkg/component/k8s"
	"github.com/wutong-paas/wutong/pkg/component/mq"
	"github.com/wutong-paas/wutong/pkg/component/prom"
)

// InitHandle 初始化handle
// func InitHandle(conf option.Config,
// 	etcdClientArgs *etcdutil.ClientArgs,
// 	statusCli *client.AppRuntimeSyncClient,
// 	etcdcli *clientv3.Client,
// 	config *rest.Config,
// 	kubeClient kubernetes.Interface,
// 	wutongClient versioned.Interface,
// 	k8sClient k8sclient.Client,
// 	dynamicClient dynamic.Interface,
// 	apiextClient apiextclient.Interface,
// 	veleroClient veleroversioned.Interface,
// ) error {
// 	mq := api_db.MQManager{
// 		EtcdClientArgs: etcdClientArgs,
// 		DefaultServer:  conf.MQAPI,
// 	}
// 	mqClient, errMQ := mq.NewMQManager()
// 	if errMQ != nil {
// 		logrus.Errorf("new MQ manager failed, %v", errMQ)
// 		return errMQ
// 	}
// 	prometheusCli, err := prometheus.NewPrometheus(&prometheus.Options{
// 		Endpoint: conf.PrometheusEndpoint,
// 	})
// 	if err != nil {
// 		logrus.Errorf("new prometheus client failure, %v", err)
// 		return err
// 	}
// 	dbmanager := db.GetManager()
// 	defaultServieHandler = CreateManager(conf, mqClient, etcdcli, statusCli, prometheusCli, wutongClient, kubeClient, dynamicClient, apiextClient, veleroClient)
// 	defaultPluginHandler = CreatePluginManager(mqClient)
// 	defaultAppHandler = CreateAppManager(mqClient)
// 	defaultTenantEnvHandler = CreateTenantEnvManager(mqClient, statusCli, &conf, config, kubeClient, prometheusCli, k8sClient)
// 	defaultNetRulesHandler = CreateNetRulesManager(etcdcli)
// 	defaultAPPBackupHandler = group.CreateBackupHandle(mqClient, statusCli, etcdcli)
// 	defaultEventHandler = CreateLogManager(etcdcli)
// 	shareHandler = &share.ServiceShareHandle{MQClient: mqClient, EtcdCli: etcdcli}
// 	pluginShareHandler = &share.PluginShareHandle{MQClient: mqClient, EtcdCli: etcdcli}
// 	defaultGatewayHandler = CreateGatewayManager(dbmanager, mqClient, etcdcli)
// 	def3rdPartySvcHandler = Create3rdPartySvcHandler(dbmanager, statusCli)
// 	operationHandler = CreateOperationHandler(mqClient)
// 	batchOperationHandler = CreateBatchOperationHandler(mqClient, statusCli, operationHandler)
// 	defaultAppRestoreHandler = NewAppRestoreHandler()
// 	defPodHandler = NewPodHandler(statusCli)
// 	defClusterHandler = NewClusterHandler(kubeClient, conf.WtNamespace, conf.PrometheusEndpoint)
// 	defNodeHandler = NewNodeHandler(kubeClient, prometheusCli)
// 	defSchedulingHandler = NewSchedulingHandler(kubeClient)
// 	defaultVolumeTypeHandler = CreateVolumeTypeManger(statusCli)
// 	defaultEtcdHandler = NewEtcdHandler(etcdcli)
// 	defaultmonitorHandler = NewMonitorHandler(prometheusCli)
// 	defServiceEventHandler = NewServiceEventHandler()
// 	defApplicationHandler = NewApplicationHandler(statusCli, prometheusCli, wutongClient, kubeClient)
// 	defRegistryAuthSecretHandler = CreateRegistryAuthSecretManager(dbmanager, mqClient, etcdcli)
// 	return nil
// }

// InitHandle 初始化handle
func InitHandle(conf option.Config) error {
	// mq := api_db.MQManager{
	// 	EtcdClientArgs: etcdClientArgs,
	// 	DefaultServer:  conf.MQAPI,
	// }
	// mqClient, errMQ := mq.NewMQManager()
	// if errMQ != nil {
	// 	logrus.Errorf("new MQ manager failed, %v", errMQ)
	// 	return errMQ
	// }
	// prometheusCli, err := prometheus.NewPrometheus(&prometheus.Options{
	// 	Endpoint: conf.PrometheusEndpoint,
	// })
	// if err != nil {
	// 	logrus.Errorf("new prometheus client failure, %v", err)
	// 	return err
	// }

	etcdcli := etcd.Default().EtcdClient
	statusCli := grpc.Default().StatusClient
	kubeClient := k8s.Default().Clientset
	wutongClient := k8s.Default().WutongClient
	k8sClient := k8s.Default().K8sClient
	restconfig := k8s.Default().RestConfig
	dynamicClient := k8s.Default().DynamicClient
	// gatewayClient := k8s.Default().GatewayClient
	// kubevirtCli := k8s.Default().KubevirtCli
	apiextClient := k8s.Default().ApiExtClient
	veleroClient := k8s.Default().VeleroClient
	// mapper := k8s.Default().Mapper
	// registryCli := cr.Default().RegistryCli
	mqClient := mq.Default().MqClient
	prometheusCli := prom.Default().PrometheusCli

	dbmanager := db.GetManager()
	defaultServieHandler = CreateManager(conf, mqClient, etcdcli, statusCli, prometheusCli, wutongClient, kubeClient, dynamicClient, apiextClient, veleroClient)
	defaultPluginHandler = CreatePluginManager(mqClient)
	defaultAppHandler = CreateAppManager(mqClient)
	defaultTenantEnvHandler = CreateTenantEnvManager(mqClient, statusCli, &conf, restconfig, kubeClient, prometheusCli, k8sClient)
	defaultNetRulesHandler = CreateNetRulesManager(etcdcli)
	defaultAPPBackupHandler = group.CreateBackupHandle(mqClient, statusCli, etcdcli)
	defaultEventHandler = CreateLogManager(etcdcli)
	shareHandler = &share.ServiceShareHandle{MQClient: mqClient, EtcdCli: etcdcli}
	pluginShareHandler = &share.PluginShareHandle{MQClient: mqClient, EtcdCli: etcdcli}
	defaultGatewayHandler = CreateGatewayManager(dbmanager, mqClient, etcdcli)
	def3rdPartySvcHandler = Create3rdPartySvcHandler(dbmanager, statusCli)
	operationHandler = CreateOperationHandler(mqClient)
	batchOperationHandler = CreateBatchOperationHandler(mqClient, statusCli, operationHandler)
	defaultAppRestoreHandler = NewAppRestoreHandler()
	defPodHandler = NewPodHandler(statusCli)
	defClusterHandler = NewClusterHandler(kubeClient, k8sClient, conf.WtNamespace, conf.PrometheusEndpoint)
	defNodeHandler = NewNodeHandler(kubeClient, prometheusCli)
	defSchedulingHandler = NewSchedulingHandler(kubeClient)
	defaultVolumeTypeHandler = CreateVolumeTypeManger(statusCli)
	defaultEtcdHandler = NewEtcdHandler(etcdcli)
	defaultmonitorHandler = NewMonitorHandler(prometheusCli)
	defServiceEventHandler = NewServiceEventHandler()
	defApplicationHandler = NewApplicationHandler(statusCli, prometheusCli, wutongClient, kubeClient)
	defRegistryAuthSecretHandler = CreateRegistryAuthSecretManager(dbmanager, mqClient)
	return nil
}

var defaultServieHandler ServiceHandler
var shareHandler *share.ServiceShareHandle
var pluginShareHandler *share.PluginShareHandle
var defaultmonitorHandler MonitorHandler

// GetMonitorHandle get monitor handler
func GetMonitorHandle() MonitorHandler {
	return defaultmonitorHandler
}

// GetShareHandle get share handle
func GetShareHandle() *share.ServiceShareHandle {
	return shareHandler
}

// GetPluginShareHandle get plugin share handle
func GetPluginShareHandle() *share.PluginShareHandle {
	return pluginShareHandler
}

// GetServiceManager get manager
func GetServiceManager() ServiceHandler {
	return defaultServieHandler
}

var defaultPluginHandler PluginHandler

// GetPluginManager get manager
func GetPluginManager() PluginHandler {
	return defaultPluginHandler
}

var defaultTenantEnvHandler TenantEnvHandler

// GetTenantEnvManager get manager
func GetTenantEnvManager() TenantEnvHandler {
	return defaultTenantEnvHandler
}

var defaultNetRulesHandler NetRulesHandler

// GetRulesManager get manager
func GetRulesManager() NetRulesHandler {
	return defaultNetRulesHandler
}

var defaultEventHandler EventHandler

// GetEventHandler get event handler
func GetEventHandler() EventHandler {
	return defaultEventHandler
}

var defaultAppHandler *AppAction

// GetAppHandler GetAppHandler
func GetAppHandler() *AppAction {
	return defaultAppHandler
}

var defaultAPPBackupHandler *group.BackupHandle

// GetAPPBackupHandler GetAPPBackupHandler
func GetAPPBackupHandler() *group.BackupHandle {
	return defaultAPPBackupHandler
}

var defaultGatewayHandler GatewayHandler

// GetGatewayHandler returns a default GatewayHandler
func GetGatewayHandler() GatewayHandler {
	return defaultGatewayHandler
}

var def3rdPartySvcHandler *ThirdPartyServiceHanlder

// Get3rdPartySvcHandler returns the defalut ThirdParthServiceHanlder
func Get3rdPartySvcHandler() *ThirdPartyServiceHanlder {
	return def3rdPartySvcHandler
}

var batchOperationHandler *BatchOperationHandler

// GetBatchOperationHandler get handler
func GetBatchOperationHandler() *BatchOperationHandler {
	return batchOperationHandler
}

var operationHandler *OperationHandler

// GetOperationHandler get handler
func GetOperationHandler() *OperationHandler {
	return operationHandler
}

var defaultAppRestoreHandler AppRestoreHandler

// GetAppRestoreHandler returns a default AppRestoreHandler
func GetAppRestoreHandler() AppRestoreHandler {
	return defaultAppRestoreHandler
}

var defPodHandler PodHandler

// GetPodHandler returns the defalut PodHandler
func GetPodHandler() PodHandler {
	return defPodHandler
}

var defaultEtcdHandler *EtcdHandler

// GetEtcdHandler returns the default etcd handler.
func GetEtcdHandler() *EtcdHandler {
	return defaultEtcdHandler
}

var defClusterHandler ClusterHandler

// GetClusterHandler returns the default cluster handler.
func GetClusterHandler() ClusterHandler {
	return defClusterHandler
}

var defNodeHandler NodeHandler

// GetVolumeTypeHandler returns the default volume type handler.
func GetNodeHandler() NodeHandler {
	return defNodeHandler
}

var defSchedulingHandler SchedulingHandler

func GetSchedulingHandler() SchedulingHandler {
	return defSchedulingHandler
}

var defApplicationHandler ApplicationHandler

// GetApplicationHandler  returns the default tenant env application handler.
func GetApplicationHandler() ApplicationHandler {
	return defApplicationHandler
}

var defServiceEventHandler *ServiceEventHandler

// GetServiceEventHandler -
func GetServiceEventHandler() *ServiceEventHandler {
	return defServiceEventHandler
}

var defRegistryAuthSecretHandler RegistryAuthSecretHandler

// GetRegistryAuthSecretHandler -
func GetRegistryAuthSecretHandler() RegistryAuthSecretHandler {
	return defRegistryAuthSecretHandler
}
