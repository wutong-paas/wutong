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

package version2

import (
	"github.com/go-chi/chi"
	"github.com/wutong-paas/wutong/api/controller"
	"github.com/wutong-paas/wutong/api/middleware"
	"github.com/wutong-paas/wutong/cmd/api/option"
	dbmodel "github.com/wutong-paas/wutong/db/model"
)

// V2 v2
type V2 struct {
	Cfg *option.Config
}

// Routes routes
func (v2 *V2) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/show", controller.GetManager().Show)
	r.Post("/show", controller.GetManager().Show)
	r.Get("/tenants/envs", controller.GetManager().GetAllTenantEnvs)
	r.Mount("/tenants/{tenant_name}/envs", v2.tenantEnvRouter())
	r.Mount("/cluster", v2.clusterRouter())
	r.Mount("/notificationEvent", v2.notificationEventRouter())
	r.Mount("/resources", v2.resourcesRouter())
	r.Mount("/prometheus", v2.prometheusRouter())
	r.Get("/event", controller.GetManager().Event)
	r.Mount("/app", v2.appRouter())
	r.Get("/health", controller.GetManager().Health)
	r.Post("/alertmanager-webhook", controller.GetManager().AlertManagerWebHook)
	r.Get("/version", controller.GetManager().Version)
	// deprecated use /gateway/ports
	r.Mount("/port", v2.portRouter())
	// deprecated, use /events/<event_id>/log
	r.Get("/event-log", controller.GetManager().LogByAction)
	r.Mount("/events", v2.eventsRouter())
	r.Get("/gateway/ips", controller.GetGatewayIPs)
	r.Get("/gateway/ports", controller.GetManager().GetAvailablePort)
	r.Get("/volume-options", controller.VolumeOptions)
	r.Get("/volume-options/page/{page}/size/{pageSize}", controller.ListVolumeType)
	r.Post("/volume-options", controller.VolumeSetVar)
	r.Delete("/volume-options/{volume_type}", controller.DeleteVolumeType)
	r.Put("/volume-options/{volume_type}", controller.UpdateVolumeType)
	r.Mount("/enterprise", v2.enterpriseRouter())
	r.Mount("/monitor", v2.monitorRouter())
	r.Post("/sys-plugin", controller.GetManager().SysPluginAction)
	r.Mount("/sys-plugin/{plugin_id}", v2.sysPluginRouter())
	// helm resources
	r.Get("/helm/{helm_namespace}/apps", controller.GetManager().ListHelmApps)
	r.Get("/helm/{helm_namespace}/apps/{helm_name}/resources", controller.GetManager().ListHelmAppResources)

	return r
}

func (v2 *V2) sysPluginRouter() chi.Router {
	r := chi.NewRouter()
	r.Use(middleware.InitSysPlugin)
	r.Put("/", controller.GetManager().SysPluginAction)
	r.Delete("/", controller.GetManager().SysPluginAction)
	r.Post("/build", controller.GetManager().SysPluginBuild)
	return r
}

func (v2 *V2) monitorRouter() chi.Router {
	r := chi.NewRouter()
	r.Get("/metrics", controller.GetMonitorMetrics)
	return r
}

func (v2 *V2) enterpriseRouter() chi.Router {
	r := chi.NewRouter()
	r.Get("/running-services", controller.GetRunningServices)
	r.Get("/services/status", controller.GetServicesStatus)
	return r
}

func (v2 *V2) eventsRouter() chi.Router {
	r := chi.NewRouter()
	// get target's event list with page
	r.Get("/", controller.GetManager().Events)
	// get target's event content
	r.Get("/{eventID}/log", controller.GetManager().EventLog)
	return r
}

func (v2 *V2) clusterRouter() chi.Router {
	r := chi.NewRouter()
	r.Get("/", controller.GetManager().GetClusterInfo)
	r.Get("/storageclasses", controller.GetManager().ListStorageClasses)
	r.Mount("/nodes", v2.nodeRouter())
	r.Mount("/scheduling", v2.schedulingRouter())
	r.Get("/events", controller.GetManager().GetClusterEvents)
	r.Get("/builder/mavensetting", controller.GetManager().MavenSettingList)
	r.Post("/builder/mavensetting", controller.GetManager().MavenSettingAdd)
	r.Get("/builder/mavensetting/{name}", controller.GetManager().MavenSettingDetail)
	r.Put("/builder/mavensetting/{name}", controller.GetManager().MavenSettingUpdate)
	r.Delete("/builder/mavensetting/{name}", controller.GetManager().MavenSettingDelete)

	// features
	r.Get("/features", controller.GetManager().Features)
	return r
}

func (v2 *V2) nodeRouter() chi.Router {
	r := chi.NewRouter()
	r.Get("/", controller.GetManager().ListNodes)
	r.Mount("/{node_name}", v2.nodeNameRouter())
	// Deprecated, use /v2/cluster/scheduling/vm/labels
	r.Get("/vm-selector-labels", controller.GetManager().ListVMSchedulingLabels)
	return r
}

func (v2 *V2) schedulingRouter() chi.Router {
	r := chi.NewRouter()
	r.Get("/nodes", controller.GetManager().ListSchedulingNodes)
	r.Get("/labels", controller.GetManager().ListSchedulingLabels)
	r.Get("/vm/labels", controller.GetManager().ListVMSchedulingLabels)
	r.Get("/taints", controller.GetManager().ListSchedulingTaints)
	return r
}

func (v2 *V2) nodeNameRouter() chi.Router {
	r := chi.NewRouter()
	r.Get("/", controller.GetManager().GetNode)
	r.Get("/label", controller.GetManager().GetNodeLabels)
	r.Get("/common/label", controller.GetManager().GetCommonLabels)
	r.Put("/label", controller.GetManager().SetNodeLabel)
	r.Delete("/label", controller.GetManager().DeleteNodeLabel)
	r.Get("/annotation", controller.GetManager().GetNodeAnnotations)
	r.Put("/annotation", controller.GetManager().SetNodeAnnotation)
	r.Delete("/annotation", controller.GetManager().DeleteNodeAnnotation)
	r.Get("/taint", controller.GetManager().GetNodeTaints)
	r.Put("/taint", controller.GetManager().TaintNode)
	r.Delete("/taint", controller.GetManager().DeleteTaintNode)
	r.Put("/cordon", controller.GetManager().CordonNode)
	r.Put("/uncordon", controller.GetManager().UncordonNode)
	r.Get("/scheduling/vm/label", controller.GetManager().GetVMSchedulingLabels)
	r.Put("/scheduling/vm/label", controller.GetManager().SetVMSchedulingLabel)
	r.Delete("/scheduling/vm/label", controller.GetManager().DeleteVMSchedulingLabel)
	return r
}

func (v2 *V2) tenantEnvRouter() chi.Router {
	r := chi.NewRouter()
	r.Post("/", controller.GetManager().AddTenantEnv)
	r.Mount("/{tenant_env_name}", v2.tenantEnvNameRouter())
	r.Get("/", controller.GetManager().GetTenantEnvs)
	r.Get("/services-count", controller.GetManager().ServicesCount)
	return r
}

func (v2 *V2) tenantEnvNameRouter() chi.Router {
	r := chi.NewRouter()
	//初始化租户和服务信
	r.Use(middleware.InitTenantEnv)
	r.Put("/", controller.GetManager().TenantEnv)
	r.Get("/", controller.GetManager().TenantEnv)
	r.Delete("/", controller.GetManager().TenantEnv)
	//租户中的日志
	r.Post("/event-log", controller.GetManager().TenantEnvLogByAction)
	r.Get("/protocols", controller.GetManager().GetSupportProtocols)
	//插件预安装
	r.Post("/transplugins", controller.GetManager().TransPlugins)
	//代码检测
	r.Post("/code-check", controller.GetManager().CheckCode)
	r.Post("/servicecheck", controller.Check)
	r.Get("/servicecheck/{uuid}", controller.GetServiceCheckInfo)
	r.Get("/resources", controller.GetManager().SingleTenantEnvResources)
	r.Get("/services", controller.GetManager().ServicesInfo)
	//创建应用
	r.Post("/services", middleware.WrapEL(controller.GetManager().CreateService, dbmodel.TargetTypeService, "create-service", dbmodel.SYNEVENTTYPE))
	r.Post("/plugin", controller.GetManager().PluginAction)
	r.Post("/plugins/{plugin_id}/share", controller.GetManager().SharePlugin)
	r.Get("/plugins/{plugin_id}/share/{share_id}", controller.GetManager().SharePluginResult)
	r.Get("/plugin", controller.GetManager().PluginAction)
	// batch install and build plugins
	r.Post("/plugins", controller.GetManager().BatchInstallPlugins)
	r.Post("/batch-build-plugins", controller.GetManager().BatchBuildPlugins)
	r.Post("/services_status", controller.GetManager().StatusServiceList)
	r.Mount("/services/{service_alias}", v2.serviceRouter())
	r.Mount("/plugin/{plugin_id}", v2.pluginRouter())
	r.Get("/event", controller.GetManager().Event) //tenant envapp
	r.Get("/pods/{pod_name}", controller.GetManager().PodDetail)
	r.Post("/apps", controller.GetManager().CreateApp)
	r.Post("/batch_create_apps", controller.GetManager().BatchCreateApp)
	r.Get("/apps", controller.GetManager().ListApps)
	r.Post("/checkResourceName", controller.GetManager().CheckResourceName)
	r.Get("/appstatuses", controller.GetManager().ListAppStatuses)
	r.Mount("/apps/{app_id}", v2.applicationRouter())
	//get some service pod info
	r.Get("/pods", controller.Pods)
	r.Get("/pod_nums", controller.PodNums)
	//app backup
	r.Get("/groupapp/backups", controller.Backups)
	r.Post("/groupapp/backups", controller.NewBackups)
	r.Post("/groupapp/backupcopy", controller.BackupCopy)
	r.Get("/groupapp/backups/{backup_id}", controller.GetBackup)
	r.Delete("/groupapp/backups/{backup_id}", controller.DeleteBackup)
	r.Post("/groupapp/backups/{backup_id}/restore", controller.Restore)
	r.Get("/groupapp/backups/{backup_id}/restore/{restore_id}", controller.RestoreResult)
	r.Post("/deployversions", controller.GetManager().GetManyDeployVersion)
	//团队资源限制
	r.Post("/limit_memory", controller.GetManager().LimitTenantEnvMemory)
	r.Get("/limit_memory", controller.GetManager().TenantEnvResourcesStatus)

	// Gateway
	r.Post("/http-rule", controller.GetManager().HTTPRule)
	r.Delete("/http-rule", controller.GetManager().HTTPRule)
	r.Put("/http-rule", controller.GetManager().HTTPRule)
	r.Post("/tcp-rule", controller.GetManager().TCPRule)
	r.Delete("/tcp-rule", controller.GetManager().TCPRule)
	r.Put("/tcp-rule", controller.GetManager().TCPRule)
	r.Mount("/gateway", v2.gatewayRouter())

	//batch operation
	r.Post("/batchoperation", controller.BatchOperation)

	// registry auth secret
	r.Post("/registry/auth", controller.GetManager().RegistryAuthSecret)
	r.Put("/registry/auth", controller.GetManager().RegistryAuthSecret)
	r.Delete("/registry/auth", controller.GetManager().RegistryAuthSecret)

	// kubeconfig
	r.Get("/kubeconfig", controller.GetManager().GetKubeConfig)

	r.Get("/kube-resources", controller.GetManager().GetTenantEnvKubeResources)

	// virtual machine
	r.Mount("/vms", v2.vmRouter())

	return r
}

func (v2 *V2) gatewayRouter() chi.Router {
	r := chi.NewRouter()
	r.Put("/certificate", controller.GetManager().Certificate)

	return r
}

func (v2 *V2) serviceRouter() chi.Router {
	r := chi.NewRouter()
	//初始化应用信息
	r.Use(middleware.InitService)
	r.Put("/", middleware.WrapEL(controller.GetManager().UpdateService, dbmodel.TargetTypeService, "update-service", dbmodel.SYNEVENTTYPE))
	// component build
	r.Post("/build", middleware.WrapEL(controller.GetManager().BuildService, dbmodel.TargetTypeService, "build-service", dbmodel.ASYNEVENTTYPE))
	// component start
	r.Post("/start", middleware.WrapEL(controller.GetManager().StartService, dbmodel.TargetTypeService, "start-service", dbmodel.ASYNEVENTTYPE))
	// component stop event set to synchronous event, not wait.
	r.Post("/stop", middleware.WrapEL(controller.GetManager().StopService, dbmodel.TargetTypeService, "stop-service", dbmodel.SYNEVENTTYPE))
	r.Post("/restart", middleware.WrapEL(controller.GetManager().RestartService, dbmodel.TargetTypeService, "restart-service", dbmodel.ASYNEVENTTYPE))
	//应用伸缩
	r.Put("/vertical", middleware.WrapEL(controller.GetManager().VerticalService, dbmodel.TargetTypeService, "vertical-service", dbmodel.ASYNEVENTTYPE))
	r.Put("/horizontal", middleware.WrapEL(controller.GetManager().HorizontalService, dbmodel.TargetTypeService, "horizontal-service", dbmodel.ASYNEVENTTYPE))

	//设置应用语言(act)
	r.Post("/language", middleware.WrapEL(controller.GetManager().SetLanguage, dbmodel.TargetTypeService, "set-language", dbmodel.SYNEVENTTYPE))
	//应用信息获取修改与删除(source)
	r.Get("/", controller.GetManager().SingleServiceInfo)
	r.Delete("/", middleware.WrapEL(controller.GetManager().SingleServiceInfo, dbmodel.TargetTypeService, "delete-service", dbmodel.SYNEVENTTYPE))
	//应用升级(act)
	r.Post("/upgrade", middleware.WrapEL(controller.GetManager().UpgradeService, dbmodel.TargetTypeService, "upgrade-service", dbmodel.ASYNEVENTTYPE))
	//应用状态获取(act)
	r.Get("/status", controller.GetManager().StatusService)
	//构建版本列表
	r.Get("/build-list", controller.GetManager().BuildList)
	//构建版本操作
	r.Get("/build-version/{build_version}", controller.GetManager().BuildVersionInfo)
	r.Put("/build-version/{build_version}", controller.GetManager().BuildVersionInfo)
	r.Get("/deployversion", controller.GetManager().GetDeployVersion)
	r.Delete("/build-version/{build_version}", middleware.WrapEL(controller.GetManager().BuildVersionInfo, dbmodel.TargetTypeService, "delete-buildversion", dbmodel.SYNEVENTTYPE))
	//应用分享
	r.Post("/share", middleware.WrapEL(controller.GetManager().Share, dbmodel.TargetTypeService, "share-service", dbmodel.SYNEVENTTYPE))
	r.Get("/share/{share_id}", controller.GetManager().ShareResult)
	r.Get("/logs", controller.GetManager().HistoryLogs)
	r.Get("/log-file", controller.GetManager().LogList)
	r.Get("/log-instance", controller.GetManager().LogSocket)
	r.Post("/event-log", controller.GetManager().LogByAction)

	//应用依赖关系增加与删除(source)
	r.Post("/dependency", middleware.WrapEL(controller.GetManager().Dependency, dbmodel.TargetTypeService, "add-service-dependency", dbmodel.SYNEVENTTYPE))
	r.Delete("/dependency", middleware.WrapEL(controller.GetManager().Dependency, dbmodel.TargetTypeService, "delete-service-dependency", dbmodel.SYNEVENTTYPE))
	r.Delete("/dependencies", middleware.WrapEL(controller.GetManager().DeleteDependencies, dbmodel.TargetTypeService, "delete-service-dependencies", dbmodel.SYNEVENTTYPE))
	//环境变量增删改(source)
	r.Post("/env", middleware.WrapEL(controller.GetManager().Env, dbmodel.TargetTypeService, "add-service-env", dbmodel.SYNEVENTTYPE))
	r.Put("/env", middleware.WrapEL(controller.GetManager().Env, dbmodel.TargetTypeService, "update-service-env", dbmodel.SYNEVENTTYPE))
	r.Delete("/env", middleware.WrapEL(controller.GetManager().Env, dbmodel.TargetTypeService, "delete-service-env", dbmodel.SYNEVENTTYPE))
	r.Delete("/envs/inner", middleware.WrapEL(controller.GetManager().DeleteAllInnerEnvs, dbmodel.TargetTypeService, "delete-service-all-inner-envs", dbmodel.SYNEVENTTYPE))
	r.Delete("/envs", middleware.WrapEL(controller.GetManager().DeleteAllEnvs, dbmodel.TargetTypeService, "delete-service-all-envs", dbmodel.SYNEVENTTYPE))
	//端口变量增删改(source)
	r.Post("/ports", middleware.WrapEL(controller.GetManager().Ports, dbmodel.TargetTypeService, "add-service-port", dbmodel.SYNEVENTTYPE))
	r.Put("/ports", middleware.WrapEL(controller.GetManager().PutPorts, dbmodel.TargetTypeService, "update-service-port-old", dbmodel.SYNEVENTTYPE))
	r.Put("/ports/{port}", middleware.WrapEL(controller.GetManager().Ports, dbmodel.TargetTypeService, "update-service-port", dbmodel.SYNEVENTTYPE))
	r.Delete("/ports/{port}", middleware.WrapEL(controller.GetManager().Ports, dbmodel.TargetTypeService, "delete-service-port", dbmodel.SYNEVENTTYPE))
	r.Delete("/allports", middleware.WrapEL(controller.GetManager().DeleteAllPorts, dbmodel.TargetTypeService, "delete-service-all-ports", dbmodel.SYNEVENTTYPE))
	r.Put("/ports/{port}/outer", middleware.WrapEL(controller.GetManager().PortOuterController, dbmodel.TargetTypeService, "handle-service-outerport", dbmodel.SYNEVENTTYPE))
	r.Put("/ports/{port}/inner", middleware.WrapEL(controller.GetManager().PortInnerController, dbmodel.TargetTypeService, "handle-service-innerport", dbmodel.SYNEVENTTYPE))

	//应用版本回滚(act)
	r.Post("/rollback", middleware.WrapEL(controller.GetManager().RollBack, dbmodel.TargetTypeService, "rollback-service", dbmodel.ASYNEVENTTYPE))

	//持久化信息API v2.1 支持多种持久化格式
	r.Post("/volumes", middleware.WrapEL(controller.AddVolume, dbmodel.TargetTypeService, "add-service-volume", dbmodel.SYNEVENTTYPE))
	r.Put("/volumes", middleware.WrapEL(controller.GetManager().UpdVolume, dbmodel.TargetTypeService, "update-service-volume", dbmodel.SYNEVENTTYPE))
	r.Get("/volumes", controller.GetVolume)
	r.Delete("/volumes/{volume_name}", middleware.WrapEL(controller.DeleteVolume, dbmodel.TargetTypeService, "delete-service-volume", dbmodel.SYNEVENTTYPE))
	r.Post("/depvolumes", middleware.WrapEL(controller.AddVolumeDependency, dbmodel.TargetTypeService, "add-service-depvolume", dbmodel.SYNEVENTTYPE))
	r.Delete("/depvolumes", middleware.WrapEL(controller.DeleteVolumeDependency, dbmodel.TargetTypeService, "delete-service-depvolume", dbmodel.SYNEVENTTYPE))
	r.Get("/depvolumes", controller.GetDepVolume)
	//持久化信息API v2
	r.Post("/volume-dependency", middleware.WrapEL(controller.GetManager().VolumeDependency, dbmodel.TargetTypeService, "add-service-depvolume", dbmodel.SYNEVENTTYPE))
	r.Delete("/volume-dependency", middleware.WrapEL(controller.GetManager().VolumeDependency, dbmodel.TargetTypeService, "delete-service-depvolume", dbmodel.SYNEVENTTYPE))
	r.Post("/volume", middleware.WrapEL(controller.GetManager().AddVolume, dbmodel.TargetTypeService, "add-service-volume", dbmodel.SYNEVENTTYPE))
	r.Delete("/volume", middleware.WrapEL(controller.GetManager().DeleteVolume, dbmodel.TargetTypeService, "delete-service-volume", dbmodel.SYNEVENTTYPE))
	r.Delete("/volumes", middleware.WrapEL(controller.GetManager().DeleteAllVolumes, dbmodel.TargetTypeService, "delete-service-all-volumes", dbmodel.SYNEVENTTYPE))

	//获取应用实例情况(source)
	r.Get("/pods", controller.GetManager().Pods)

	//应用探针 增 删 改(surce)
	r.Post("/probe", middleware.WrapEL(controller.GetManager().Probe, dbmodel.TargetTypeService, "add-service-probe", dbmodel.SYNEVENTTYPE))
	r.Put("/probe", middleware.WrapEL(controller.GetManager().Probe, dbmodel.TargetTypeService, "update-service-probe", dbmodel.SYNEVENTTYPE))
	r.Delete("/probe", middleware.WrapEL(controller.GetManager().Probe, dbmodel.TargetTypeService, "delete-service-probe", dbmodel.SYNEVENTTYPE))

	r.Mount("/scheduling", v2.serviceSchedulingRouter())
	r.Post("/label", middleware.WrapEL(controller.GetManager().Label, dbmodel.TargetTypeService, "add-service-label", dbmodel.SYNEVENTTYPE))
	r.Put("/label", middleware.WrapEL(controller.GetManager().Label, dbmodel.TargetTypeService, "update-service-label", dbmodel.SYNEVENTTYPE))
	r.Delete("/label", middleware.WrapEL(controller.GetManager().Label, dbmodel.TargetTypeService, "delete-service-label", dbmodel.SYNEVENTTYPE))

	//插件
	r.Mount("/plugin", v2.serviceRelatePluginRouter())

	//rule
	r.Mount("/net-rule", v2.rulesRouter())
	r.Get("/deploy-info", controller.GetServiceDeployInfo)

	// third-party service
	r.Post("/endpoints", middleware.WrapEL(controller.GetManager().Endpoints, dbmodel.TargetTypeService, "add-thirdpart-service", dbmodel.SYNEVENTTYPE))
	r.Put("/endpoints", middleware.WrapEL(controller.GetManager().Endpoints, dbmodel.TargetTypeService, "update-thirdpart-service", dbmodel.SYNEVENTTYPE))
	r.Delete("/endpoints", middleware.WrapEL(controller.GetManager().Endpoints, dbmodel.TargetTypeService, "delete-thirdpart-service", dbmodel.SYNEVENTTYPE))
	r.Get("/endpoints", controller.GetManager().Endpoints)

	// gateway
	r.Put("/rule-config", middleware.WrapEL(controller.GetManager().RuleConfig, dbmodel.TargetTypeService, "update-service-gateway-rule", dbmodel.SYNEVENTTYPE))
	r.Put("/tcprule-config", middleware.WrapEL(controller.GetManager().TCPRuleConfig, dbmodel.TargetTypeService, "update-service-gateway-tcprule", dbmodel.SYNEVENTTYPE))

	// app restore
	r.Post("/app-restore/envs", middleware.WrapEL(controller.GetManager().RestoreEnvs, dbmodel.TargetTypeService, "app-restore-envs", dbmodel.SYNEVENTTYPE))
	r.Post("/app-restore/ports", middleware.WrapEL(controller.GetManager().RestorePorts, dbmodel.TargetTypeService, "app-restore-ports", dbmodel.SYNEVENTTYPE))
	r.Post("/app-restore/volumes", middleware.WrapEL(controller.GetManager().RestoreVolumes, dbmodel.TargetTypeService, "app-restore-volumes", dbmodel.SYNEVENTTYPE))
	r.Post("/app-restore/probe", middleware.WrapEL(controller.GetManager().RestoreProbe, dbmodel.TargetTypeService, "app-restore-probe", dbmodel.SYNEVENTTYPE))
	r.Post("/app-restore/deps", middleware.WrapEL(controller.GetManager().RestoreDeps, dbmodel.TargetTypeService, "app-restore-deps", dbmodel.SYNEVENTTYPE))
	r.Post("/app-restore/depvols", middleware.WrapEL(controller.GetManager().RestoreDepVols, dbmodel.TargetTypeService, "app-restore-depvols", dbmodel.SYNEVENTTYPE))
	r.Post("/app-restore/plugins", middleware.WrapEL(controller.GetManager().RestorePlugins, dbmodel.TargetTypeService, "app-restore-plugins", dbmodel.SYNEVENTTYPE))

	r.Get("/pods/{pod_name}/detail", controller.GetManager().PodDetail)

	// autoscaler
	r.Post("/xparules", middleware.WrapEL(controller.GetManager().AutoscalerRules, dbmodel.TargetTypeService, "add-app-autoscaler-rule", dbmodel.SYNEVENTTYPE))
	r.Put("/xparules", middleware.WrapEL(controller.GetManager().AutoscalerRules, dbmodel.TargetTypeService, "update-app-autoscaler-rule", dbmodel.SYNEVENTTYPE))
	r.Delete("/xparules/{rule_id}", middleware.WrapEL(controller.GetManager().AutoscalerRules, dbmodel.TargetTypeService, "delete-app-autoscaler-rule", dbmodel.SYNEVENTTYPE))
	r.Get("/xparecords", controller.GetManager().ScalingRecords)

	//service monitor
	r.Post("/service-monitors", middleware.WrapEL(controller.GetManager().AddServiceMonitors, dbmodel.TargetTypeService, "add-app-service-monitor", dbmodel.SYNEVENTTYPE))
	r.Put("/service-monitors/{name}", middleware.WrapEL(controller.GetManager().UpdateServiceMonitors, dbmodel.TargetTypeService, "update-app-service-monitor", dbmodel.SYNEVENTTYPE))
	r.Delete("/service-monitors/{name}", middleware.WrapEL(controller.GetManager().DeleteServiceMonitors, dbmodel.TargetTypeService, "delete-app-service-monitor", dbmodel.SYNEVENTTYPE))

	r.Get("/log", controller.GetManager().Log)

	r.Get("/kube-resources", controller.GetManager().GetServiceKubeResources)

	// velero backup and restore
	r.Mount("/backup", v2.backupRouter())
	r.Mount("/restore", v2.restoreRouter())

	return r
}

func (v2 *V2) serviceSchedulingRouter() chi.Router {
	r := chi.NewRouter()

	r.Get("/details", controller.GetManager().GetServiceSchedulingDetails)

	r.Post("/labels", middleware.WrapEL(controller.GetManager().AddServiceSchedulingLabel, dbmodel.TargetTypeService, "配置调度标签", dbmodel.SYNEVENTTYPE))
	r.Put("/labels", middleware.WrapEL(controller.GetManager().UpdateServiceSchedulingLabel, dbmodel.TargetTypeService, "配置调度标签", dbmodel.SYNEVENTTYPE))
	r.Delete("/labels", middleware.WrapEL(controller.GetManager().DeleteServiceSchedulingLabel, dbmodel.TargetTypeService, "删除调度标签", dbmodel.SYNEVENTTYPE))

	r.Post("/nodes", middleware.WrapEL(controller.GetManager().SetServiceSchedulingNode, dbmodel.TargetTypeService, "配置调度节点", dbmodel.SYNEVENTTYPE))

	r.Post("/tolerations", middleware.WrapEL(controller.GetManager().AddServiceSchedulingToleration, dbmodel.TargetTypeService, "配置污点容忍", dbmodel.SYNEVENTTYPE))
	r.Put("/tolerations", middleware.WrapEL(controller.GetManager().UpdateServiceSchedulingToleration, dbmodel.TargetTypeService, "配置污点容忍", dbmodel.SYNEVENTTYPE))
	r.Delete("/tolerations", middleware.WrapEL(controller.GetManager().DeleteServiceSchedulingToleration, dbmodel.TargetTypeService, "删除污点容忍", dbmodel.SYNEVENTTYPE))

	return r
}

func (v2 *V2) backupRouter() chi.Router {
	r := chi.NewRouter()
	r.Use(middleware.InitVeleroBackupOrRestore)
	r.Post("/", controller.GetManager().CreateBackup)
	r.Post("/schedule", controller.GetManager().CreateBackupSchedule)
	r.Put("/schedule", controller.GetManager().UpdateBackupSchedule)
	r.Delete("/schedule", controller.GetManager().DeleteBackupSchedule)
	r.Get("/schedule", controller.GetManager().GetBackupSchedule)
	r.Get("/{backup_id}/download", controller.GetManager().DownloadBackup)
	r.Delete("/{backup_id}", controller.GetManager().DeleteBackup)
	r.Get("/records", controller.GetManager().BackupRecords)
	return r
}

func (v2 *V2) restoreRouter() chi.Router {
	r := chi.NewRouter()
	r.Use(middleware.InitVeleroBackupOrRestore)
	r.Post("/", controller.GetManager().CreateRestore)
	r.Delete("/{restore_id}", controller.GetManager().DeleteRestore)
	r.Get("/records", controller.GetManager().RestoreRecords)
	return r
}

func (v2 *V2) applicationRouter() chi.Router {
	r := chi.NewRouter()
	// Init Application
	r.Use(middleware.InitApplication)
	// app governance mode
	r.Get("/governance/check", controller.GetManager().CheckGovernanceMode)
	// Operation application
	r.Put("/", controller.GetManager().UpdateApp)
	r.Delete("/", controller.GetManager().DeleteApp)
	r.Put("/volumes", controller.GetManager().ChangeVolumes)
	// Get services under application
	r.Get("/services", controller.GetManager().ListServices)
	// bind components
	r.Put("/services", controller.GetManager().BatchBindService)
	// Application configuration group
	r.Post("/configgroups", controller.GetManager().AddConfigGroup)
	r.Put("/configgroups/{config_group_name}", controller.GetManager().UpdateConfigGroup)

	r.Put("/ports", controller.GetManager().BatchUpdateComponentPorts)
	r.Put("/status", controller.GetManager().GetAppStatus)
	// status
	r.Post("/install", controller.GetManager().Install)
	r.Get("/releases", controller.GetManager().ListHelmAppReleases)

	r.Delete("/configgroups/{config_group_name}", controller.GetManager().DeleteConfigGroup)
	r.Get("/configgroups", controller.GetManager().ListConfigGroups)

	// Synchronize component information, full coverage
	r.Post("/components", controller.GetManager().SyncComponents)
	r.Post("/app-config-groups", controller.GetManager().SyncAppConfigGroups)

	r.Get("/kube-resources", controller.GetManager().GetApplicationKubeResources)
	return r
}

func (v2 *V2) vmRouter() chi.Router {
	r := chi.NewRouter()
	// InitVM middleware
	r.Use(middleware.InitVM)
	r.Post("/", controller.GetManager().CreateVM)
	r.Get("/", controller.GetManager().ListVMs)
	r.Mount("/{vm_id}", v2.vmIDRouter())
	return r
}

func (v2 *V2) vmIDRouter() chi.Router {
	r := chi.NewRouter()
	// InitVMID middleware
	r.Use(middleware.InitVMID)
	r.Delete("/", controller.GetManager().DeleteVM)
	r.Put("/", controller.GetManager().UpdateVM)
	r.Post("/start", controller.GetManager().StartVM)
	r.Post("/stop", controller.GetManager().StopVM)
	r.Post("/restart", controller.GetManager().RestartVM)
	r.Post("/ports", controller.GetManager().AddVMPort)
	r.Get("/ports", controller.GetManager().GetVMPorts)
	r.Post("/ports/enable", controller.GetManager().EnableVMPort)
	r.Post("/ports/disable", controller.GetManager().DisableVMPort)
	r.Post("/gateways", controller.GetManager().CreateVMPortGateway)
	r.Put("/gateways/{gateway_id}", controller.GetManager().UpdateVMPortGateway)
	r.Delete("/gateways/{gateway_id}", controller.GetManager().DeleteVMPortGateway)
	r.Delete("/ports", controller.GetManager().DeleteVMPort)
	r.Get("/", controller.GetManager().GetVM)
	r.Get("/conditions", controller.GetManager().GetVMConditions)
	r.Get("/volumes", controller.GetManager().ListVMVolumes)
	r.Post("/volumes", controller.GetManager().AddVMVolume)
	r.Delete("/volumes/{volume_name}", controller.GetManager().DeleteVMVolume)
	r.Delete("/disks/boot", controller.GetManager().RemoveBootDisk)
	return r
}

func (v2 *V2) resourcesRouter() chi.Router {
	r := chi.NewRouter()
	r.Get("/labels", controller.GetManager().Labels)
	r.Post("/tenants/{tenant_name}/envs", controller.GetManager().TenantEnvResources)
	r.Post("/services", controller.GetManager().ServiceResources)
	r.Get("/tenants/{tenant_name}/envs/sum", controller.GetManager().SumTenantEnvs)
	//tenant envs's resource
	r.Get("/tenants/{tenant_name}/envs/res", controller.GetManager().TenantEnvsWithResource)
	r.Get("/tenants/{tenant_name}/envs/res/page/{curPage}/size/{pageLen}", controller.GetManager().TenantEnvsWithResource)
	r.Get("/tenants/{tenant_name}/envs/query/{tenant_env_name}", controller.GetManager().TenantEnvsQuery)
	r.Get("/tenants/{tenant_name}/envs/{tenant_env_name}/res", controller.GetManager().TenantEnvsGetByName)
	r.Get("/tenants/{tenant_name}/envs/kubeconfig", controller.GetManager().GetKubeConfig)
	return r
}

func (v2 *V2) prometheusRouter() chi.Router {
	r := chi.NewRouter()
	return r
}

func (v2 *V2) appRouter() chi.Router {
	r := chi.NewRouter()
	r.Post("/export", controller.GetManager().ExportApp)
	r.Get("/export/{eventID}", controller.GetManager().ExportApp)

	r.Get("/download/{format}/{fileName}", controller.GetManager().Download)
	r.Post("/upload/{eventID}", controller.GetManager().NewUpload)
	r.Options("/upload/{eventID}", controller.GetManager().NewUpload)

	r.Post("/import/ids/{eventID}", controller.GetManager().ImportID)
	r.Get("/import/ids/{eventID}", controller.GetManager().ImportID)
	r.Delete("/import/ids/{eventID}", controller.GetManager().ImportID)

	r.Post("/import", controller.GetManager().ImportApp)
	r.Get("/import/{eventID}", controller.GetManager().ImportApp)
	r.Delete("/import/{eventID}", controller.GetManager().ImportApp)
	return r
}

func (v2 *V2) notificationEventRouter() chi.Router {
	r := chi.NewRouter()
	r.Get("/", controller.GetNotificationEvents)
	r.Put("/{serviceAlias}", controller.HandleNotificationEvent)
	r.Get("/{hash}", controller.GetNotificationEvent)
	return r
}

func (v2 *V2) portRouter() chi.Router {
	r := chi.NewRouter()
	r.Get("/avail-port", controller.GetManager().GetAvailablePort)
	return r
}
