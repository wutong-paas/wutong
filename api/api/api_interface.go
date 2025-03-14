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

package api

import (
	"net/http"
)

// ClusterInterface -
type ClusterInterface interface {
	GetClusterInfo(w http.ResponseWriter, r *http.Request)
	SetClusterInfo(w http.ResponseWriter, r *http.Request)
	GetClusterEvents(w http.ResponseWriter, r *http.Request)
	MavenSettingList(w http.ResponseWriter, r *http.Request)
	MavenSettingAdd(w http.ResponseWriter, r *http.Request)
	MavenSettingUpdate(w http.ResponseWriter, r *http.Request)
	MavenSettingDelete(w http.ResponseWriter, r *http.Request)
	MavenSettingDetail(w http.ResponseWriter, r *http.Request)
	Features(w http.ResponseWriter, r *http.Request)
	ListStorageClasses(w http.ResponseWriter, r *http.Request)
}

type NodeInterface interface {
	ListNodes(w http.ResponseWriter, r *http.Request)
	GetNode(w http.ResponseWriter, r *http.Request)
	GetNodeLabels(w http.ResponseWriter, r *http.Request)
	GetCommonLabels(w http.ResponseWriter, r *http.Request)
	GetVMSchedulingLabels(w http.ResponseWriter, r *http.Request)
	SetNodeLabel(w http.ResponseWriter, r *http.Request)
	DeleteNodeLabel(w http.ResponseWriter, r *http.Request)
	GetNodeAnnotations(w http.ResponseWriter, r *http.Request)
	SetNodeAnnotation(w http.ResponseWriter, r *http.Request)
	DeleteNodeAnnotation(w http.ResponseWriter, r *http.Request)
	GetNodeTaints(w http.ResponseWriter, r *http.Request)
	TaintNode(w http.ResponseWriter, r *http.Request)
	DeleteTaintNode(w http.ResponseWriter, r *http.Request)
	CordonNode(w http.ResponseWriter, r *http.Request)
	UncordonNode(w http.ResponseWriter, r *http.Request)
	SetVMSchedulingLabel(w http.ResponseWriter, r *http.Request)
	DeleteVMSchedulingLabel(w http.ResponseWriter, r *http.Request)
}

type SchedulingInterface interface {
	ListSchedulingNodes(w http.ResponseWriter, r *http.Request)
	ListSchedulingTaints(w http.ResponseWriter, r *http.Request)
	ListVMSchedulingLabels(w http.ResponseWriter, r *http.Request)
	ListSchedulingLabels(w http.ResponseWriter, r *http.Request)
}

// TenantEnvInterface interface
type TenantEnvInterface interface {
	TenantEnvInterfaceWithV1
	AllTenantEnvResources(w http.ResponseWriter, r *http.Request)
	TenantEnvResources(w http.ResponseWriter, r *http.Request)
	ServiceResources(w http.ResponseWriter, r *http.Request)
	TenantEnv(w http.ResponseWriter, r *http.Request)
	AddTenantEnv(w http.ResponseWriter, r *http.Request)
	GetAllTenantEnvs(w http.ResponseWriter, r *http.Request)
	GetTenantEnvs(w http.ResponseWriter, r *http.Request)
	ServicesInfo(w http.ResponseWriter, r *http.Request)
	TenantEnvsWithResource(w http.ResponseWriter, r *http.Request)
	TenantEnvsQuery(w http.ResponseWriter, r *http.Request)
	TenantEnvsGetByName(w http.ResponseWriter, r *http.Request)
	SumTenantEnvs(w http.ResponseWriter, r *http.Request)
	SingleTenantEnvResources(w http.ResponseWriter, r *http.Request)
	GetSupportProtocols(w http.ResponseWriter, r *http.Request)
	TransPlugins(w http.ResponseWriter, r *http.Request)
	ServicesCount(w http.ResponseWriter, r *http.Request)
	GetManyDeployVersion(w http.ResponseWriter, r *http.Request)
	LimitTenantEnvMemory(w http.ResponseWriter, r *http.Request)
	TenantEnvResourcesStatus(w http.ResponseWriter, r *http.Request)
	CheckResourceName(w http.ResponseWriter, r *http.Request)
	Log(w http.ResponseWriter, r *http.Request)
	GetKubeConfig(w http.ResponseWriter, r *http.Request)
	GetTenantEnvKubeResources(w http.ResponseWriter, r *http.Request)

	// kubevirt
	CreateVM(w http.ResponseWriter, r *http.Request)
	GetVM(w http.ResponseWriter, r *http.Request)
	GetVMConditions(w http.ResponseWriter, r *http.Request)
	StartVM(w http.ResponseWriter, r *http.Request)
	StopVM(w http.ResponseWriter, r *http.Request)
	RestartVM(w http.ResponseWriter, r *http.Request)
	UpdateVM(w http.ResponseWriter, r *http.Request)
	AddVMPort(w http.ResponseWriter, r *http.Request)
	GetVMPorts(w http.ResponseWriter, r *http.Request)
	EnableVMPort(w http.ResponseWriter, r *http.Request)
	DisableVMPort(w http.ResponseWriter, r *http.Request)
	CreateVMPortGateway(w http.ResponseWriter, r *http.Request)
	UpdateVMPortGateway(w http.ResponseWriter, r *http.Request)
	DeleteVMPortGateway(w http.ResponseWriter, r *http.Request)
	DeleteVMPort(w http.ResponseWriter, r *http.Request)
	DeleteVM(w http.ResponseWriter, r *http.Request)
	ListVMs(w http.ResponseWriter, r *http.Request)
	ListVMVolumes(w http.ResponseWriter, r *http.Request)
	AddVMVolume(w http.ResponseWriter, r *http.Request)
	DeleteVMVolume(w http.ResponseWriter, r *http.Request)
	RemoveBootDisk(w http.ResponseWriter, r *http.Request)
	CloneVM(w http.ResponseWriter, r *http.Request)
	CreateVMSnapshot(w http.ResponseWriter, r *http.Request)
	ListVMSnapshots(w http.ResponseWriter, r *http.Request)
	DeleteVMSnapshot(w http.ResponseWriter, r *http.Request)
	CreateVMRestore(w http.ResponseWriter, r *http.Request)
	ListVMRestores(w http.ResponseWriter, r *http.Request)
	DeleteVMRestore(w http.ResponseWriter, r *http.Request)
	ExportVM(w http.ResponseWriter, r *http.Request)
	GetVMExportStatus(w http.ResponseWriter, r *http.Request)
	DownloadVMExport(w http.ResponseWriter, r *http.Request)
}

// ServiceInterface ServiceInterface
type ServiceInterface interface {
	SetLanguage(w http.ResponseWriter, r *http.Request)
	SingleServiceInfo(w http.ResponseWriter, r *http.Request)
	CheckCode(w http.ResponseWriter, r *http.Request)
	Event(w http.ResponseWriter, r *http.Request)
	BuildList(w http.ResponseWriter, r *http.Request)
	CreateService(w http.ResponseWriter, r *http.Request)
	UpdateService(w http.ResponseWriter, r *http.Request)
	Dependency(w http.ResponseWriter, r *http.Request)
	AddDependencies(w http.ResponseWriter, r *http.Request)
	DeleteDependencies(w http.ResponseWriter, r *http.Request)
	Env(w http.ResponseWriter, r *http.Request)
	DeleteAllEnvs(w http.ResponseWriter, r *http.Request)
	DeleteAllInnerEnvs(w http.ResponseWriter, r *http.Request)
	Ports(w http.ResponseWriter, r *http.Request)
	DeleteAllPorts(w http.ResponseWriter, r *http.Request)
	PutPorts(w http.ResponseWriter, r *http.Request)
	PortOuterController(w http.ResponseWriter, r *http.Request)
	PortInnerController(w http.ResponseWriter, r *http.Request)
	RollBack(w http.ResponseWriter, r *http.Request)
	AddVolume(w http.ResponseWriter, r *http.Request)
	UpdVolume(w http.ResponseWriter, r *http.Request)
	DeleteVolume(w http.ResponseWriter, r *http.Request)
	DeleteAllVolumes(w http.ResponseWriter, r *http.Request)
	Pods(w http.ResponseWriter, r *http.Request)
	ListServiceInstances(w http.ResponseWriter, r *http.Request)
	ListServiceInstanceContainers(w http.ResponseWriter, r *http.Request)
	ListServiceInstanceLogs(w http.ResponseWriter, r *http.Request)
	ListServiceInstanceStaticLogs(w http.ResponseWriter, r *http.Request)
	GetServiceInstanceDescription(w http.ResponseWriter, r *http.Request)
	ListServiceInstanceContainerOptions(w http.ResponseWriter, r *http.Request)
	ListServiceInstanceEvents(w http.ResponseWriter, r *http.Request)
	VolumeDependency(w http.ResponseWriter, r *http.Request)
	Probe(w http.ResponseWriter, r *http.Request)
	Label(w http.ResponseWriter, r *http.Request)
	Share(w http.ResponseWriter, r *http.Request)
	ShareResult(w http.ResponseWriter, r *http.Request)
	BuildVersionInfo(w http.ResponseWriter, r *http.Request)
	GetDeployVersion(w http.ResponseWriter, r *http.Request)
	AutoscalerRules(w http.ResponseWriter, r *http.Request)
	ScalingRecords(w http.ResponseWriter, r *http.Request)
	AddServiceMonitors(w http.ResponseWriter, r *http.Request)
	DeleteServiceMonitors(w http.ResponseWriter, r *http.Request)
	UpdateServiceMonitors(w http.ResponseWriter, r *http.Request)
	GetServiceKubeResources(w http.ResponseWriter, r *http.Request)
	CreateBackup(w http.ResponseWriter, r *http.Request)
	DownloadBackup(w http.ResponseWriter, r *http.Request)
	CreateBackupSchedule(w http.ResponseWriter, r *http.Request)
	UpdateBackupSchedule(w http.ResponseWriter, r *http.Request)
	DeleteBackupSchedule(w http.ResponseWriter, r *http.Request)
	DeleteBackup(w http.ResponseWriter, r *http.Request)
	CreateRestore(w http.ResponseWriter, r *http.Request)
	DeleteRestore(w http.ResponseWriter, r *http.Request)
	BackupRecords(w http.ResponseWriter, r *http.Request)
	RestoreRecords(w http.ResponseWriter, r *http.Request)
	GetBackupSchedule(w http.ResponseWriter, r *http.Request)

	// Scheduling
	GetServiceSchedulingDetails(w http.ResponseWriter, r *http.Request)
	AddServiceSchedulingLabel(w http.ResponseWriter, r *http.Request)
	UpdateServiceSchedulingLabel(w http.ResponseWriter, r *http.Request)
	DeleteServiceSchedulingLabel(w http.ResponseWriter, r *http.Request)
	SetServiceSchedulingNode(w http.ResponseWriter, r *http.Request)
	AddServiceSchedulingToleration(w http.ResponseWriter, r *http.Request)
	UpdateServiceSchedulingToleration(w http.ResponseWriter, r *http.Request)
	DeleteServiceSchedulingToleration(w http.ResponseWriter, r *http.Request)

	ChangeServiceApp(w http.ResponseWriter, r *http.Request)
}

// TenantEnvInterfaceWithV1 funcs for both v2 and v1
type TenantEnvInterfaceWithV1 interface {
	StartService(w http.ResponseWriter, r *http.Request)
	StopService(w http.ResponseWriter, r *http.Request)
	RestartService(w http.ResponseWriter, r *http.Request)
	VerticalService(w http.ResponseWriter, r *http.Request)
	HorizontalService(w http.ResponseWriter, r *http.Request)
	BuildService(w http.ResponseWriter, r *http.Request)
	DeployService(w http.ResponseWriter, r *http.Request)
	UpgradeService(w http.ResponseWriter, r *http.Request)
	StatusService(w http.ResponseWriter, r *http.Request)
	StatusServiceList(w http.ResponseWriter, r *http.Request)
	StatusContainerID(w http.ResponseWriter, r *http.Request)
}

// LogInterface log interface
type LogInterface interface {
	HistoryLogs(w http.ResponseWriter, r *http.Request)
	LogList(w http.ResponseWriter, r *http.Request)
	LogFile(w http.ResponseWriter, r *http.Request)
	LogSocket(w http.ResponseWriter, r *http.Request)
	LogByAction(w http.ResponseWriter, r *http.Request)
	TenantEnvLogByAction(w http.ResponseWriter, r *http.Request)
	Events(w http.ResponseWriter, r *http.Request)
	EventLog(w http.ResponseWriter, r *http.Request)
}

// PluginInterface plugin interface
type PluginInterface interface {
	PluginAction(w http.ResponseWriter, r *http.Request)
	SysPluginAction(w http.ResponseWriter, r *http.Request)
	PluginDefaultENV(w http.ResponseWriter, r *http.Request)
	PluginBuild(w http.ResponseWriter, r *http.Request)
	SysPluginBuild(w http.ResponseWriter, r *http.Request)
	GetAllPluginBuildVersions(w http.ResponseWriter, r *http.Request)
	GetPluginBuildVersion(w http.ResponseWriter, r *http.Request)
	DeletePluginBuildVersion(w http.ResponseWriter, r *http.Request)
	//plugin
	PluginSet(w http.ResponseWriter, r *http.Request)
	DeletePluginRelation(w http.ResponseWriter, r *http.Request)
	GePluginEnvWhichCanBeSet(w http.ResponseWriter, r *http.Request)
	UpdateVersionEnv(w http.ResponseWriter, r *http.Request)
	UpdateComponentPluginConfig(w http.ResponseWriter, r *http.Request)
	ToggleComponentPlugin(w http.ResponseWriter, r *http.Request)
	GetPluginDefaultEnvs(w http.ResponseWriter, r *http.Request)
	SharePlugin(w http.ResponseWriter, r *http.Request)
	SharePluginResult(w http.ResponseWriter, r *http.Request)
	BatchInstallPlugins(w http.ResponseWriter, r *http.Request)
	BatchBuildPlugins(w http.ResponseWriter, r *http.Request)
}

// RulesInterface RulesInterface
type RulesInterface interface {
	SetDownStreamRule(w http.ResponseWriter, r *http.Request)
	GetDownStreamRule(w http.ResponseWriter, r *http.Request)
	DeleteDownStreamRule(w http.ResponseWriter, r *http.Request)
	UpdateDownStreamRule(w http.ResponseWriter, r *http.Request)
}

// AppInterface app handle interface
type AppInterface interface {
	ExportApp(w http.ResponseWriter, r *http.Request)
	Download(w http.ResponseWriter, r *http.Request)
	Upload(w http.ResponseWriter, r *http.Request)
	NewUpload(w http.ResponseWriter, r *http.Request)
	ImportID(w http.ResponseWriter, r *http.Request)
	ImportApp(w http.ResponseWriter, r *http.Request)
}

// ApplicationInterface tenant envapplication interface
type ApplicationInterface interface {
	CreateApp(w http.ResponseWriter, r *http.Request)
	BatchCreateApp(w http.ResponseWriter, r *http.Request)
	UpdateApp(w http.ResponseWriter, r *http.Request)
	ListApps(w http.ResponseWriter, r *http.Request)
	ListComponents(w http.ResponseWriter, r *http.Request)
	BatchBindService(w http.ResponseWriter, r *http.Request)
	DeleteApp(w http.ResponseWriter, r *http.Request)
	AddConfigGroup(w http.ResponseWriter, r *http.Request)
	UpdateConfigGroup(w http.ResponseWriter, r *http.Request)

	BatchUpdateComponentPorts(w http.ResponseWriter, r *http.Request)
	GetAppStatus(w http.ResponseWriter, r *http.Request)
	Install(w http.ResponseWriter, r *http.Request)
	ListServices(w http.ResponseWriter, r *http.Request)
	ListHelmAppReleases(w http.ResponseWriter, r *http.Request)

	DeleteConfigGroup(w http.ResponseWriter, r *http.Request)
	ListConfigGroups(w http.ResponseWriter, r *http.Request)
	SyncComponents(w http.ResponseWriter, r *http.Request)
	SyncAppConfigGroups(w http.ResponseWriter, r *http.Request)
	ListAppStatuses(w http.ResponseWriter, r *http.Request)
	CheckGovernanceMode(w http.ResponseWriter, r *http.Request)
	ChangeVolumes(w http.ResponseWriter, r *http.Request)

	GetApplicationKubeResources(w http.ResponseWriter, r *http.Request)
}

// Gatewayer gateway api interface
type Gatewayer interface {
	HTTPRule(w http.ResponseWriter, r *http.Request)
	TCPRule(w http.ResponseWriter, r *http.Request)
	GetAvailablePort(w http.ResponseWriter, r *http.Request)
	RuleConfig(w http.ResponseWriter, r *http.Request)
	TCPRuleConfig(w http.ResponseWriter, r *http.Request)
	Certificate(w http.ResponseWriter, r *http.Request)
}

// ThirdPartyServicer is an interface for defining methods for third-party service.
type ThirdPartyServicer interface {
	Endpoints(w http.ResponseWriter, r *http.Request)
}

// Labeler is an interface for defining methods to get information of labels.
type Labeler interface {
	Labels(w http.ResponseWriter, r *http.Request)
}

// AppRestoreInterface defines api methods to restore app.
// app means market service.
type AppRestoreInterface interface {
	RestoreEnvs(w http.ResponseWriter, r *http.Request)
	RestorePorts(w http.ResponseWriter, r *http.Request)
	RestoreVolumes(w http.ResponseWriter, r *http.Request)
	RestoreProbe(w http.ResponseWriter, r *http.Request)
	RestoreDeps(w http.ResponseWriter, r *http.Request)
	RestoreDepVols(w http.ResponseWriter, r *http.Request)
	RestorePlugins(w http.ResponseWriter, r *http.Request)
}

// PodInterface defines api methods about k8s pods.
type PodInterface interface {
	PodDetail(w http.ResponseWriter, r *http.Request)
}

// HelmAppsInterface defines api methods about helm apps
type HelmAppsInterface interface {
	ListHelmAppResources(w http.ResponseWriter, r *http.Request)
	ListHelmApps(w http.ResponseWriter, r *http.Request)
}

// RegistryAuthSecretInterface registry auth secret interface
type RegistryAuthSecretInterface interface {
	RegistryAuthSecret(w http.ResponseWriter, r *http.Request)
}

// AppStoreVersionInterface app store version interface
type AppStoreVersionInterface interface {
	ExportAppStoreVersionStatus(w http.ResponseWriter, r *http.Request)
	ExportAppStoreVersion(w http.ResponseWriter, r *http.Request)
	DownloadAppStoreVersion(w http.ResponseWriter, r *http.Request)
}
