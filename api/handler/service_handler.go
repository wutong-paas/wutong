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
	"context"
	"net/http"

	"github.com/jinzhu/gorm"
	api_model "github.com/wutong-paas/wutong/api/model"
	"github.com/wutong-paas/wutong/api/util"
	"github.com/wutong-paas/wutong/chaos/exector"
	dbmodel "github.com/wutong-paas/wutong/db/model"
	"github.com/wutong-paas/wutong/worker/discover/model"
	"github.com/wutong-paas/wutong/worker/server/pb"
	"k8s.io/client-go/kubernetes"
)

// ServiceHandler service handler
type ServiceHandler interface {
	KubeClient() kubernetes.Interface

	ServiceBuild(tenantEnvID, serviceID string, r *api_model.BuildServiceStruct) error
	AddLabel(l *api_model.LabelsStruct, serviceID string) error
	DeleteLabel(l *api_model.LabelsStruct, serviceID string) error
	UpdateLabel(l *api_model.LabelsStruct, serviceID string) error
	StartStopService(s *api_model.StartStopStruct) error
	ServiceVertical(ctx context.Context, v *model.VerticalScalingTaskBody) error
	ServiceHorizontal(h *model.HorizontalScalingTaskBody) error
	ServiceUpgrade(r *model.RollingUpgradeTaskBody) error
	ServiceCreate(ts *api_model.ServiceStruct) error
	ServiceUpdate(sc map[string]interface{}) error
	LanguageSet(langS *api_model.LanguageSet) error
	GetService(tenantEnvID string) ([]*dbmodel.TenantEnvServices, error)
	GetServicesByAppID(appID string, page, pageSize int) (*api_model.ListServiceResponse, error)
	GetPagedTenantEnvRes(offset, len int) ([]*api_model.TenantEnvResource, int, error)
	GetTenantEnvRes(uuid string) (*api_model.TenantEnvResource, error)
	CodeCheck(c *api_model.CheckCodeStruct) error
	ServiceDepend(action string, ds *api_model.DependService) error
	EnvAttr(action string, at *dbmodel.TenantEnvServiceEnvVar) error
	PortVar(action string, tenantEnvID, serviceID string, vp *api_model.ServicePorts, oldPort int) error
	CreatePorts(tenantEnvID, serviceID string, vps *api_model.ServicePorts) error
	PortOuter(tenantEnvName, serviceID string, containerPort int, servicePort *api_model.ServicePortInnerOrOuter) (*dbmodel.TenantEnvServiceLBMappingPort, string, error)
	PortInner(tenantEnvName, serviceID, operation string, port int) error
	VolumnVar(avs *dbmodel.TenantEnvServiceVolume, tenantEnvID, fileContent, action string) *util.APIHandleError
	UpdVolume(sid string, req *api_model.UpdVolumeReq) error
	VolumeDependency(tsr *dbmodel.TenantEnvServiceMountRelation, action string) *util.APIHandleError
	GetDepVolumes(serviceID string) ([]*dbmodel.TenantEnvServiceMountRelation, *util.APIHandleError)
	GetVolumes(serviceID string) ([]*api_model.VolumeWithStatusStruct, *util.APIHandleError)
	ServiceProbe(tsp *dbmodel.TenantEnvServiceProbe, action string) error
	RollBack(rs *api_model.RollbackStruct) error
	GetStatus(serviceID string) (*api_model.StatusList, error)
	GetServicesStatus(tenantEnvID string, services []string) []map[string]interface{}
	GetAllRunningServices() ([]string, *util.APIHandleError)
	GetAllServicesStatus() (*ServicesStatus, *util.APIHandleError)
	CreateTenantEnv(*dbmodel.TenantEnvs) error
	CreateTenantEnvIDAndName() (string, string, error)
	GetPods(serviceID string) (*K8sPodInfos, error)
	ListServiceInstances(namespace, serviceID string) (ServiceInstances, error)
	ListServiceInstanceContainers(service *dbmodel.TenantEnvServices, namespace, instance string) (ServiceInstanceContainers, error)
	ListServiceInstanceContainerOptions(service *dbmodel.TenantEnvServices, namespace string) (ServiceInstanceContainerOptions, error)
	ListServiceInstanceEvents(namespace, instance string) (ServiceInstanceEvents, error)
	GetMultiServicePods(serviceIDs []string) (*K8sPodInfos, error)
	GetComponentPodNums(ctx context.Context, componentIDs []string) (map[string]int32, error)
	TransServieToDelete(ctx context.Context, tenantEnvID, serviceID string) error
	TenantEnvServiceDeletePluginRelation(tenantEnvID, serviceID, pluginID string) *util.APIHandleError
	GetTenantEnvServicePluginRelation(serviceID string) ([]*dbmodel.TenantEnvServicePluginRelation, *util.APIHandleError)
	SetTenantEnvServicePluginRelation(tenantEnvID, serviceID string, pss *api_model.PluginSetStruct) (*dbmodel.TenantEnvServicePluginRelation, *util.APIHandleError)
	UpdateTenantEnvServicePluginRelation(serviceID string, pss *api_model.PluginSetStruct) (*dbmodel.TenantEnvServicePluginRelation, *util.APIHandleError)
	UpdateVersionEnv(uve *api_model.SetVersionEnv) *util.APIHandleError
	UpdateComponentPluginConfig(req *api_model.UpdateComponentPluginConfigRequest) *util.APIHandleError
	ToggleComponentPlugin(req *api_model.ToggleComponentPluginRequest) *util.APIHandleError
	DeletePluginConfig(serviceID, pluginID string) *util.APIHandleError
	ServiceCheck(*api_model.ServiceCheckStruct) (string, string, *util.APIHandleError)
	GetServiceCheckInfo(uuid string) (*exector.ServiceCheckResult, *util.APIHandleError)
	GetServiceDeployInfo(tenantEnvID, serviceID string) (*pb.DeployInfo, *util.APIHandleError)
	ListVersionInfo(serviceID string) (*api_model.BuildListRespVO, error)

	AddAutoscalerRule(req *api_model.AutoscalerRuleReq) error
	UpdAutoscalerRule(req *api_model.AutoscalerRuleReq) error
	DeleteAutoscalerRule(ruleID string) error
	ListScalingRecords(serviceID string, page, pageSize int) ([]*dbmodel.TenantEnvServiceScalingRecords, int, error)

	UpdateServiceMonitor(tenantEnvID, serviceID, name string, update api_model.UpdateServiceMonitorRequestStruct) (*dbmodel.TenantEnvServiceMonitor, error)
	DeleteServiceMonitor(tenantEnvID, serviceID, name string) (*dbmodel.TenantEnvServiceMonitor, error)
	AddServiceMonitor(tenantEnvID, serviceID string, add api_model.AddServiceMonitorRequestStruct) (*dbmodel.TenantEnvServiceMonitor, error)

	SyncComponentBase(tx *gorm.DB, app *dbmodel.Application, components []*api_model.Component) error
	SyncComponentMonitors(tx *gorm.DB, app *dbmodel.Application, components []*api_model.Component) error
	SyncComponentPorts(tx *gorm.DB, app *dbmodel.Application, components []*api_model.Component) error
	SyncComponentRelations(tx *gorm.DB, app *dbmodel.Application, components []*api_model.Component) error
	SyncComponentEnvs(tx *gorm.DB, app *dbmodel.Application, components []*api_model.Component) error
	SyncComponentVolumeRels(tx *gorm.DB, app *dbmodel.Application, components []*api_model.Component) error
	SyncComponentVolumes(tx *gorm.DB, components []*api_model.Component) error
	SyncComponentConfigFiles(tx *gorm.DB, components []*api_model.Component) error
	SyncComponentProbes(tx *gorm.DB, components []*api_model.Component) error
	SyncComponentLabels(tx *gorm.DB, components []*api_model.Component) error
	SyncComponentPlugins(tx *gorm.DB, app *dbmodel.Application, components []*api_model.Component) error
	SyncComponentScaleRules(tx *gorm.DB, components []*api_model.Component) error
	SyncComponentEndpoints(tx *gorm.DB, components []*api_model.Component) error

	Log(w http.ResponseWriter, r *http.Request, component *dbmodel.TenantEnvServices, podName, containerName string, follow bool) error

	GetKubeResources(namespace, serviceID string, customSetting api_model.KubeResourceCustomSetting) (string, error)

	// Velero integration
	CreateBackup(tenantEnvID, serviceID string, req api_model.CreateBackupRequest) error
	CreateBackupSchedule(tenantEnvID, serviceID string, req api_model.CreateBackupScheduleRequest) error
	UpdateBackupSchedule(tenantEnvID, serviceID string, req api_model.UpdateBackupScheduleRequest) error
	DeleteBackupSchedule(serviceID string) error
	DownloadBackup(serviceID, backupID string) ([]byte, error)
	DeleteBackup(serviceID, backupID string) error
	CreateRestore(tenantEnvID, serviceID string, req api_model.CreateRestoreRequest) error
	DeleteRestore(serviceID, restoreID string) error
	BackupRecords(tenantEnvID, serviceID string) ([]*api_model.BackupRecord, error)
	RestoreRecords(tenantEnvID, serviceID string) ([]*api_model.RestoreRecord, error)
	GetBackupSchedule(tenantEnvID, serviceID string) (*api_model.BackupSchedule, bool)

	// Kubevirt integration
	CreateVM(tenantEnv *dbmodel.TenantEnvs, req *api_model.CreateVMRequest) (*api_model.CreateVMResponse, error)
	GetVM(tenantEnv *dbmodel.TenantEnvs, vmID string) (*api_model.GetVMResponse, error)
	GetVMConditions(tenantEnv *dbmodel.TenantEnvs, vmID string) (*api_model.GetVMConditionsResponse, error)
	UpdateVM(tenantEnv *dbmodel.TenantEnvs, vmID string, req *api_model.UpdateVMRequest) (*api_model.UpdateVMResponse, error)
	StartVM(tenantEnv *dbmodel.TenantEnvs, vmID string) (*api_model.StartVMResponse, error)
	StopVM(tenantEnv *dbmodel.TenantEnvs, vmID string) (*api_model.StopVMResponse, error)
	RestartVM(tenantEnv *dbmodel.TenantEnvs, vmID string) (*api_model.RestartVMResponse, error)
	AddVMPort(tenantEnv *dbmodel.TenantEnvs, vmID string, req *api_model.AddVMPortRequest) error
	GetVMPorts(tenantEnv *dbmodel.TenantEnvs, vmID string) (*api_model.GetVMPortsResponse, error)
	EnableVMPort(tenantEnv *dbmodel.TenantEnvs, vmID string, req *api_model.EnableVMPortRequest) error
	DisableVMPort(tenantEnv *dbmodel.TenantEnvs, vmID string, req *api_model.DisableVMPortRequest) error
	CreateVMPortGateway(tenantEnv *dbmodel.TenantEnvs, vmID string, req *api_model.CreateVMPortGatewayRequest) error
	UpdateVMPortGateway(tenantEnv *dbmodel.TenantEnvs, vmID, gatewayID string, req *api_model.UpdateVMPortGatewayRequest) error
	DeleteVMPortGateway(tenantEnv *dbmodel.TenantEnvs, vmID, gatewayID string) error
	DeleteVMPort(tenantEnv *dbmodel.TenantEnvs, vmID string, req *api_model.DeleteVMPortRequest) error
	DeleteVM(tenantEnv *dbmodel.TenantEnvs, vmID string) error
	ListVMs(tenantEnv *dbmodel.TenantEnvs) (*api_model.ListVMsResponse, error)
	ListVMVolumes(tenantEnv *dbmodel.TenantEnvs, vmID string) (*api_model.ListVMVolumesResponse, error)
	AddVMVolume(tenantEnv *dbmodel.TenantEnvs, vmID string, req *api_model.AddVMVolumeRequest) error
	DeleteVMVolume(tenantEnv *dbmodel.TenantEnvs, vmID, volumeName string) error
	RemoveBootDisk(tenantEnv *dbmodel.TenantEnvs, vmID string) error
	CloneVM(tenantEnv *dbmodel.TenantEnvs, vmID string, req *api_model.CloneVMRequest) error
	CreateVMSnapshot(tenantEnv *dbmodel.TenantEnvs, vmID string, req *api_model.CreateVMSnapshotRequest) error
	ListVMSnapshots(tenantEnv *dbmodel.TenantEnvs, vmID string) (*api_model.ListVMSnapshotsResponse, error)
	DeleteVMSnapshot(tenantEnv *dbmodel.TenantEnvs, vmID, snapshotID string) error
	CreateVMRestore(tenantEnv *dbmodel.TenantEnvs, vmID, snapshotID string) error
	ListVMRestores(tenantEnv *dbmodel.TenantEnvs, vmID string) (*api_model.ListVMRestoresResponse, error)
	DeleteVMRestore(tenantEnv *dbmodel.TenantEnvs, vmID, restoreID string) error

	// Scheduling
	GetServiceSchedulingDetails(serviceID string) (*api_model.GetServiceSchedulingDetailsResponse, error)
	AddServiceSchedulingLabel(serviceID string, req *api_model.AddServiceSchedulingLabelRequest) error
	UpdateServiceSchedulingLabel(serviceID string, req *api_model.UpdateServiceSchedulingLabelRequest) error
	DeleteServiceSchedulingLabel(serviceID string, req *api_model.DeleteServiceSchedulingLabelRequest) error
	SetServiceSchedulingNode(serviceID string, req *api_model.SetServiceSchedulingNodeRequest) error
	AddServiceSchedulingToleration(serviceID string, req *api_model.AddServiceSchedulingTolerationRequest) error
	UpdateServiceSchedulingToleration(serviceID string, req *api_model.UpdateServiceSchedulingTolerationRequest) error
	DeleteServiceSchedulingToleration(serviceID string, req *api_model.DeleteServiceSchedulingTolerationRequest) error

	ChangeServiceApp(serviceID string, req *api_model.ChangeServiceAppRequest) error
}
