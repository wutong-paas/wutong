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
	"github.com/wutong-paas/wutong/builder/exector"
	dbmodel "github.com/wutong-paas/wutong/db/model"
	"github.com/wutong-paas/wutong/worker/discover/model"
	"github.com/wutong-paas/wutong/worker/server/pb"
)

// ServiceHandler service handler
type ServiceHandler interface {
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
	GetMultiServicePods(serviceIDs []string) (*K8sPodInfos, error)
	GetComponentPodNums(ctx context.Context, componentIDs []string) (map[string]int32, error)
	TransServieToDelete(ctx context.Context, tenantEnvID, serviceID string) error
	TenantEnvServiceDeletePluginRelation(tenantEnvID, serviceID, pluginID string) *util.APIHandleError
	GetTenantEnvServicePluginRelation(serviceID string) ([]*dbmodel.TenantEnvServicePluginRelation, *util.APIHandleError)
	SetTenantEnvServicePluginRelation(tenantEnvID, serviceID string, pss *api_model.PluginSetStruct) (*dbmodel.TenantEnvServicePluginRelation, *util.APIHandleError)
	UpdateTenantEnvServicePluginRelation(serviceID string, pss *api_model.PluginSetStruct) (*dbmodel.TenantEnvServicePluginRelation, *util.APIHandleError)
	UpdateVersionEnv(uve *api_model.SetVersionEnv) *util.APIHandleError
	DeletePluginConfig(serviceID, pluginID string) *util.APIHandleError
	ServiceCheck(*api_model.ServiceCheckStruct) (string, string, *util.APIHandleError)
	GetServiceCheckInfo(uuid string) (*exector.ServiceCheckResult, *util.APIHandleError)
	GetServiceDeployInfo(tenantEnvID, serviceID string) (*pb.DeployInfo, *util.APIHandleError)
	ListVersionInfo(serviceID string) (*api_model.BuildListRespVO, error)

	AddAutoscalerRule(req *api_model.AutoscalerRuleReq) error
	UpdAutoscalerRule(req *api_model.AutoscalerRuleReq) error
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

	GetKubeResources(namespace, serviceID string, customSetting api_model.KubeResourceCustomSetting) string
}
