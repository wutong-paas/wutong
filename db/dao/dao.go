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

package dao

import (
	"context"
	"errors"
	"time"

	"github.com/wutong-paas/wutong/db/model"
)

var (
	// ErrVolumeNotFound volume not found error, happens when haven't find any matched data
	ErrVolumeNotFound = errors.New("volume not found")
)

// Dao 数据持久化层接口
type Dao interface {
	AddModel(model.Interface) error
	UpdateModel(model.Interface) error
}

// DelDao 删除接口
type DelDao interface {
	DeleteModel(serviceID string, arg ...interface{}) error
}

// EnterpriseDao enterprise dao
type EnterpriseDao interface {
	GetEnterpriseTenantEnvs(enterpriseID string) ([]*model.TenantEnvs, error)
}

// TenantEnvDao tenant env dao
type TenantEnvDao interface {
	Dao
	GetAllTenantEnvs(query string) ([]*model.TenantEnvs, error)
	GetTenantEnvByUUID(uuid string) (*model.TenantEnvs, error)
	GetTenantEnvIDByName(tenantName, tenantEnvName string) (*model.TenantEnvs, error)
	GetTenantEnvs(tenantName, query string) ([]*model.TenantEnvs, error)
	GetTenantEnvByEid(eid, query string) ([]*model.TenantEnvs, error)
	GetPagedTenantEnvs(offset, len int) ([]*model.TenantEnvs, error)
	GetTenantEnvIDsByNames(tenantName string, tenantEnvNames []string) ([]string, error)
	GetTenantEnvLimitsByNames(tenantName string, tenantEnvNames []string) (map[string]int, error)
	GetTenantEnvByUUIDIsExist(uuid string) bool
	DelByTenantEnvID(tenantEnvID string) error
}

// AppDao tenant env dao
type AppDao interface {
	Dao
	GetByEventId(eventID string) (*model.AppStatus, error)
	DeleteModelByEventId(eventID string) error
}

// ApplicationDao tenant env Application Dao
type ApplicationDao interface {
	Dao
	ListApps(tenantEnvID, appName string, page, pageSize int) ([]*model.Application, int64, error)
	GetAppByID(appID string) (*model.Application, error)
	DeleteApp(appID string) error
	GetByServiceID(sid string) (*model.Application, error)
	ListByAppIDs(appIDs []string) ([]*model.Application, error)
	IsK8sAppDuplicate(tenantEnvID, AppID, k8sApp string) bool
}

// AppConfigGroupDao Application config group Dao
type AppConfigGroupDao interface {
	Dao
	GetConfigGroupByID(appID, configGroupName string) (*model.ApplicationConfigGroup, error)
	ListByServiceID(sid string) ([]*model.ApplicationConfigGroup, error)
	GetConfigGroupsByAppID(appID string, page, pageSize int) ([]*model.ApplicationConfigGroup, int64, error)
	DeleteConfigGroup(appID, configGroupName string) error
	DeleteByAppID(appID string) error
	CreateOrUpdateConfigGroupsInBatch(cgroups []*model.ApplicationConfigGroup) error
}

// AppConfigGroupServiceDao service config group Dao
type AppConfigGroupServiceDao interface {
	Dao
	GetConfigGroupServicesByID(appID, configGroupName string) ([]*model.ConfigGroupService, error)
	DeleteConfigGroupService(appID, configGroupName string) error
	DeleteEffectiveServiceByServiceID(serviceID string) error
	DeleteByComponentIDs(componentIDs []string) error
	CreateOrUpdateConfigGroupServicesInBatch(cgservices []*model.ConfigGroupService) error
	DeleteByAppID(appID string) error
}

// AppConfigGroupItemDao Application config item group Dao
type AppConfigGroupItemDao interface {
	Dao
	GetConfigGroupItemsByID(appID, configGroupName string) ([]*model.ConfigGroupItem, error)
	ListByServiceID(sid string) ([]*model.ConfigGroupItem, error)
	DeleteConfigGroupItem(appID, configGroupName string) error
	DeleteByAppID(appID string) error
	CreateOrUpdateConfigGroupItemsInBatch(cgitems []*model.ConfigGroupItem) error
}

// VolumeTypeDao volume type dao
type VolumeTypeDao interface {
	Dao
	DeleteModelByVolumeTypes(volumeType string) error
	GetAllVolumeTypes() ([]*model.TenantEnvServiceVolumeType, error)
	GetAllVolumeTypesByPage(page int, pageSize int) ([]*model.TenantEnvServiceVolumeType, error)
	GetVolumeTypeByType(vt string) (*model.TenantEnvServiceVolumeType, error)
	CreateOrUpdateVolumeType(vt *model.TenantEnvServiceVolumeType) (*model.TenantEnvServiceVolumeType, error)
}

// LicenseDao LicenseDao
type LicenseDao interface {
	Dao
	//DeleteLicense(token string) error
	ListLicenses() ([]*model.LicenseInfo, error)
}

// TenantEnvServiceDao TenantEnvServiceDao
type TenantEnvServiceDao interface {
	Dao
	GetServiceByID(serviceID string) (*model.TenantEnvServices, error)
	GetServiceByServiceAlias(serviceAlias string) (*model.TenantEnvServices, error)
	GetServiceByIDs(serviceIDs []string) ([]*model.TenantEnvServices, error)
	GetServiceAliasByIDs(uids []string) ([]*model.TenantEnvServices, error)
	GetServiceByTenantEnvIDAndServiceAlias(tenantEnvID, serviceName string) (*model.TenantEnvServices, error)
	SetTenantEnvServiceStatus(serviceID, status string) error
	GetServicesByTenantEnvID(tenantEnvID string) ([]*model.TenantEnvServices, error)
	GetServicesByTenantEnvIDs(tenantEnvIDs []string) ([]*model.TenantEnvServices, error)
	GetServicesAllInfoByTenantEnvID(tenantEnvID string) ([]*model.TenantEnvServices, error)
	GetServicesInfoByAppID(appID string, page, pageSize int) ([]*model.TenantEnvServices, int64, error)
	CountServiceByAppID(appID string) (int64, error)
	GetServiceIDsByAppID(appID string) (re []model.ServiceID)
	GetServicesByServiceIDs(serviceIDs []string) ([]*model.TenantEnvServices, error)
	DeleteServiceByServiceID(serviceID string) error
	GetServiceMemoryByTenantEnvIDs(tenantEnvIDs, serviceIDs []string) (map[string]map[string]interface{}, error)
	GetServiceMemoryByServiceIDs(serviceIDs []string) (map[string]map[string]interface{}, error)
	GetPagedTenantEnvService(offset, len int, serviceIDs []string) ([]map[string]interface{}, int, error)
	GetAllServicesID() ([]*model.TenantEnvServices, error)
	UpdateDeployVersion(serviceID, deployversion string) error
	ListThirdPartyServices() ([]*model.TenantEnvServices, error)
	ListServicesByTenantEnvID(tenantEnvID string) ([]*model.TenantEnvServices, error)
	GetServiceTypeByID(serviceID string) (*model.TenantEnvServices, error)
	ListByAppID(appID string) ([]*model.TenantEnvServices, error)
	BindAppByServiceIDs(appID string, serviceIDs []string) error
	CreateOrUpdateComponentsInBatch(components []*model.TenantEnvServices) error
	DeleteByComponentIDs(tenantEnvID, appID string, componentIDs []string) error
	IsK8sComponentNameDuplicate(appID, serviceID, k8sComponentName string) bool
}

// TenantEnvServiceDeleteDao TenantEnvServiceDeleteDao
type TenantEnvServiceDeleteDao interface {
	Dao
	GetTenantEnvServicesDeleteByCreateTime(createTime time.Time) ([]*model.TenantEnvServicesDelete, error)
	DeleteTenantEnvServicesDelete(record *model.TenantEnvServicesDelete) error
	List() ([]*model.TenantEnvServicesDelete, error)
}

// TenantEnvServicesPortDao TenantEnvServicesPortDao
type TenantEnvServicesPortDao interface {
	Dao
	DelDao
	GetByTenantEnvAndName(tenantEnvID, name string) (*model.TenantEnvServicesPort, error)
	GetPortsByServiceID(serviceID string) ([]*model.TenantEnvServicesPort, error)
	GetOuterPorts(serviceID string) ([]*model.TenantEnvServicesPort, error)
	GetInnerPorts(serviceID string) ([]*model.TenantEnvServicesPort, error)
	GetPort(serviceID string, port int) (*model.TenantEnvServicesPort, error)
	GetOpenedPorts(serviceID string) ([]*model.TenantEnvServicesPort, error)
	//GetDepUDPPort get all depend service udp port info
	GetDepUDPPort(serviceID string) ([]*model.TenantEnvServicesPort, error)
	DELPortsByServiceID(serviceID string) error
	HasOpenPort(sid string) bool
	DelByServiceID(sid string) error
	ListInnerPortsByServiceIDs(serviceIDs []string) ([]*model.TenantEnvServicesPort, error)
	ListByK8sServiceNames(serviceIDs []string) ([]*model.TenantEnvServicesPort, error)
	CreateOrUpdatePortsInBatch(ports []*model.TenantEnvServicesPort) error
	DeleteByComponentIDs(componentIDs []string) error
}

// TenantEnvPluginDao TenantEnvPluginDao
type TenantEnvPluginDao interface {
	Dao
	GetPluginByID(pluginID, tenantEnvID string) (*model.TenantEnvPlugin, error)
	DeletePluginByID(pluginID, tenantEnvID string) error
	GetPluginsByTenantEnvID(tenantEnvID string) ([]*model.TenantEnvPlugin, error)
	ListByIDs(ids []string) ([]*model.TenantEnvPlugin, error)
	ListByTenantEnvID(tenantEnvID string) ([]*model.TenantEnvPlugin, error)
	CreateOrUpdatePluginsInBatch(plugins []*model.TenantEnvPlugin) error
}

// TenantEnvPluginDefaultENVDao TenantEnvPluginDefaultENVDao
type TenantEnvPluginDefaultENVDao interface {
	Dao
	GetDefaultENVByName(pluginID, name, versionID string) (*model.TenantEnvPluginDefaultENV, error)
	GetDefaultENVSByPluginID(pluginID, versionID string) ([]*model.TenantEnvPluginDefaultENV, error)
	//GetDefaultENVSByPluginIDCantBeSet(pluginID string) ([]*model.TenantEnvPluginDefaultENV, error)
	DeleteDefaultENVByName(pluginID, name, versionID string) error
	DeleteAllDefaultENVByPluginID(PluginID string) error
	DeleteDefaultENVByPluginIDAndVersionID(pluginID, versionID string) error
	GetALLMasterDefultENVs(pluginID string) ([]*model.TenantEnvPluginDefaultENV, error)
	GetDefaultEnvWhichCanBeSetByPluginID(pluginID, versionID string) ([]*model.TenantEnvPluginDefaultENV, error)
}

// TenantEnvPluginBuildVersionDao TenantEnvPluginBuildVersionDao
type TenantEnvPluginBuildVersionDao interface {
	Dao
	DeleteBuildVersionByVersionID(versionID string) error
	DeleteBuildVersionByPluginID(pluginID string) error
	GetBuildVersionByPluginID(pluginID string) ([]*model.TenantEnvPluginBuildVersion, error)
	GetBuildVersionByVersionID(pluginID, versionID string) (*model.TenantEnvPluginBuildVersion, error)
	GetLastBuildVersionByVersionID(pluginID, versionID string) (*model.TenantEnvPluginBuildVersion, error)
	GetBuildVersionByDeployVersion(pluginID, versionID, deployVersion string) (*model.TenantEnvPluginBuildVersion, error)
	ListSuccessfulOnesByPluginIDs(pluginIDs []string) ([]*model.TenantEnvPluginBuildVersion, error)
	CreateOrUpdatePluginBuildVersionsInBatch(buildVersions []*model.TenantEnvPluginBuildVersion) error
}

// TenantEnvPluginVersionEnvDao TenantEnvPluginVersionEnvDao
type TenantEnvPluginVersionEnvDao interface {
	Dao
	DeleteEnvByEnvName(envName, pluginID, serviceID string) error
	DeleteEnvByPluginID(serviceID, pluginID string) error
	DeleteEnvByServiceID(serviceID string) error
	GetVersionEnvByServiceID(serviceID string, pluginID string) ([]*model.TenantEnvPluginVersionEnv, error)
	ListByServiceID(serviceID string) ([]*model.TenantEnvPluginVersionEnv, error)
	GetVersionEnvByEnvName(serviceID, pluginID, envName string) (*model.TenantEnvPluginVersionEnv, error)
	DeleteByComponentIDs(componentIDs []string) error
	CreateOrUpdatePluginVersionEnvsInBatch(versionEnvs []*model.TenantEnvPluginVersionEnv) error
}

// TenantEnvPluginVersionConfigDao service plugin config that can be dynamic discovery dao interface
type TenantEnvPluginVersionConfigDao interface {
	Dao
	GetPluginConfig(serviceID, pluginID string) (*model.TenantEnvPluginVersionDiscoverConfig, error)
	GetPluginConfigs(serviceID string) ([]*model.TenantEnvPluginVersionDiscoverConfig, error)
	DeletePluginConfig(serviceID, pluginID string) error
	DeletePluginConfigByServiceID(serviceID string) error
	DeleteByComponentIDs(componentIDs []string) error
	CreateOrUpdatePluginVersionConfigsInBatch(versionConfigs []*model.TenantEnvPluginVersionDiscoverConfig) error
}

// TenantEnvServicePluginRelationDao TenantEnvServicePluginRelationDao
type TenantEnvServicePluginRelationDao interface {
	Dao
	DeleteRelationByServiceIDAndPluginID(serviceID, pluginID string) error
	DeleteALLRelationByServiceID(serviceID string) error
	DeleteALLRelationByPluginID(pluginID string) error
	GetALLRelationByServiceID(serviceID string) ([]*model.TenantEnvServicePluginRelation, error)
	GetRelateionByServiceIDAndPluginID(serviceID, pluginID string) (*model.TenantEnvServicePluginRelation, error)
	CheckSomeModelPluginByServiceID(serviceID, pluginModel string) (bool, error)
	// CheckSomeModelLikePluginByServiceID(serviceID, pluginModel string) (bool, error)
	CheckPluginBeforeInstall(serviceID, pluginModel string) (bool, error)
	DeleteByComponentIDs(componentIDs []string) error
	CreateOrUpdatePluginRelsInBatch(relations []*model.TenantEnvServicePluginRelation) error
}

// TenantEnvServiceRelationDao TenantEnvServiceRelationDao
type TenantEnvServiceRelationDao interface {
	Dao
	DelDao
	GetTenantEnvServiceRelations(serviceID string) ([]*model.TenantEnvServiceRelation, error)
	ListByServiceIDs(serviceIDs []string) ([]*model.TenantEnvServiceRelation, error)
	GetTenantEnvServiceRelationsByDependServiceID(dependServiceID string) ([]*model.TenantEnvServiceRelation, error)
	HaveRelations(serviceID string) bool
	DELRelationsByServiceID(serviceID string) error
	DeleteRelationByDepID(serviceID, depID string) error
	DeleteByComponentIDs(componentIDs []string) error
	CreateOrUpdateRelationsInBatch(relations []*model.TenantEnvServiceRelation) error
}

// TenantEnvServicesStreamPluginPortDao TenantEnvServicesStreamPluginPortDao
type TenantEnvServicesStreamPluginPortDao interface {
	Dao
	GetPluginMappingPorts(serviceID string) ([]*model.TenantEnvServicesStreamPluginPort, error)
	SetPluginMappingPort(
		tenantEnvID string,
		serviceID string,
		pluginModel string,
		containerPort int,
	) (int, error)
	DeletePluginMappingPortByContainerPort(
		serviceID string,
		pluginModel string,
		containerPort int,
	) error
	DeleteAllPluginMappingPortByServiceID(serviceID string) error
	GetPluginMappingPortByServiceIDAndContainerPort(
		serviceID string,
		pluginModel string,
		containerPort int,
	) (*model.TenantEnvServicesStreamPluginPort, error)
	ListByServiceID(sid string) ([]*model.TenantEnvServicesStreamPluginPort, error)
	DeleteByComponentIDs(componentIDs []string) error
	CreateOrUpdateStreamPluginPortsInBatch(spPorts []*model.TenantEnvServicesStreamPluginPort) error
}

// TenantEnvServiceEnvVarDao TenantEnvServiceEnvVarDao
type TenantEnvServiceEnvVarDao interface {
	Dao
	DelDao
	//service_id__in=sids, scope__in=("outer", "both")
	GetDependServiceEnvs(serviceIDs []string, scopes []string) ([]*model.TenantEnvServiceEnvVar, error)
	GetServiceEnvs(serviceID string, scopes []string) ([]*model.TenantEnvServiceEnvVar, error)
	GetEnv(serviceID, envName string) (*model.TenantEnvServiceEnvVar, error)
	DELServiceEnvsByServiceID(serviceID string) error
	DelByServiceIDAndScope(sid, scope string) error
	CreateOrUpdateEnvsInBatch(envs []*model.TenantEnvServiceEnvVar) error
	DeleteByComponentIDs(componentIDs []string) error
}

// TenantEnvServiceMountRelationDao TenantEnvServiceMountRelationDao
type TenantEnvServiceMountRelationDao interface {
	Dao
	GetTenantEnvServiceMountRelationsByService(serviceID string) ([]*model.TenantEnvServiceMountRelation, error)
	DElTenantEnvServiceMountRelationByServiceAndName(serviceID, mntDir string) error
	DELTenantEnvServiceMountRelationByServiceID(serviceID string) error
	DElTenantEnvServiceMountRelationByDepService(serviceID, depServiceID string) error
	DeleteByComponentIDs(componentIDs []string) error
	CreateOrUpdateVolumeRelsInBatch(volRels []*model.TenantEnvServiceMountRelation) error
}

// TenantEnvServiceVolumeDao TenantEnvServiceVolumeDao
type TenantEnvServiceVolumeDao interface {
	Dao
	DelDao
	GetTenantEnvServiceVolumesByServiceID(serviceID string) ([]*model.TenantEnvServiceVolume, error)
	DeleteTenantEnvServiceVolumesByServiceID(serviceID string) error
	DeleteByServiceIDAndVolumePath(serviceID string, volumePath string) error
	GetVolumeByServiceIDAndName(serviceID, name string) (*model.TenantEnvServiceVolume, error)
	GetAllVolumes() ([]*model.TenantEnvServiceVolume, error)
	GetVolumeByID(id int) (*model.TenantEnvServiceVolume, error)
	DelShareableBySID(sid string) error
	ListVolumesByComponentIDs(componentIDs []string) ([]*model.TenantEnvServiceVolume, error)
	DeleteByVolumeIDs(volumeIDs []uint) error
	DeleteByComponentIDs(componentIDs []string) error
	CreateOrUpdateVolumesInBatch(volumes []*model.TenantEnvServiceVolume) error
}

// TenantEnvServiceConfigFileDao tenant env service config file dao interface
type TenantEnvServiceConfigFileDao interface {
	Dao
	GetConfigFileByServiceID(serviceID string) ([]*model.TenantEnvServiceConfigFile, error)
	GetByVolumeName(sid, volumeName string) (*model.TenantEnvServiceConfigFile, error)
	DelByVolumeID(sid string, volumeName string) error
	DelByServiceID(sid string) error
	DeleteByComponentIDs(componentIDs []string) error
	CreateOrUpdateConfigFilesInBatch(configFiles []*model.TenantEnvServiceConfigFile) error
}

// TenantEnvServiceLBMappingPortDao vs lb mapping port dao
type TenantEnvServiceLBMappingPortDao interface {
	Dao
	GetTenantEnvServiceLBMappingPort(serviceID string, containerPort int) (*model.TenantEnvServiceLBMappingPort, error)
	GetLBMappingPortByServiceIDAndPort(serviceID string, port int) (*model.TenantEnvServiceLBMappingPort, error)
	GetTenantEnvServiceLBMappingPortByService(serviceID string) ([]*model.TenantEnvServiceLBMappingPort, error)
	GetLBPortsASC() ([]*model.TenantEnvServiceLBMappingPort, error)
	CreateTenantEnvServiceLBMappingPort(serviceID string, containerPort int) (*model.TenantEnvServiceLBMappingPort, error)
	DELServiceLBMappingPortByServiceID(serviceID string) error
	DELServiceLBMappingPortByServiceIDAndPort(serviceID string, lbPort int) error
	GetLBPortByTenantEnvAndPort(tenantEnvID string, lbport int) (*model.TenantEnvServiceLBMappingPort, error)
	PortExists(port int) bool
}

// TenantEnvServiceLabelDao TenantEnvServiceLabelDao
type TenantEnvServiceLabelDao interface {
	Dao
	DelDao
	GetTenantEnvServiceLabel(serviceID string) ([]*model.TenantEnvServiceLable, error)
	DeleteLabelByServiceID(serviceID string) error
	GetTenantEnvServiceNodeSelectorLabel(serviceID string) ([]*model.TenantEnvServiceLable, error)
	GetTenantEnvNodeAffinityLabel(serviceID string) (*model.TenantEnvServiceLable, error)
	GetTenantEnvServiceAffinityLabel(serviceID string) ([]*model.TenantEnvServiceLable, error)
	GetTenantEnvServiceTypeLabel(serviceID string) (*model.TenantEnvServiceLable, error)
	DelTenantEnvServiceLabelsByLabelValuesAndServiceID(serviceID string) error
	DelTenantEnvServiceLabelsByServiceIDKey(serviceID string, labelKey string) error
	DelTenantEnvServiceLabelsByServiceIDKeyValue(serviceID string, labelKey string, labelValue string) error
	GetLabelByNodeSelectorKey(serviceID string, labelValue string) (*model.TenantEnvServiceLable, error)
	GetPrivilegedLabel(serviceID string) (*model.TenantEnvServiceLable, error)
	DeleteByComponentIDs(componentIDs []string) error
	CreateOrUpdateLabelsInBatch(labels []*model.TenantEnvServiceLable) error
}

// LocalSchedulerDao 本地调度信息
type LocalSchedulerDao interface {
	Dao
	GetLocalScheduler(serviceID string) ([]*model.LocalScheduler, error)
}

// ServiceProbeDao ServiceProbeDao
type ServiceProbeDao interface {
	Dao
	DelDao
	GetServiceProbes(serviceID string) ([]*model.TenantEnvServiceProbe, error)
	GetServiceUsedProbe(serviceID, mode string) (*model.TenantEnvServiceProbe, error)
	DELServiceProbesByServiceID(serviceID string) error
	DelByServiceID(sid string) error
	DeleteByComponentIDs(componentIDs []string) error
	CreateOrUpdateProbesInBatch(probes []*model.TenantEnvServiceProbe) error
}

// CodeCheckResultDao CodeCheckResultDao
type CodeCheckResultDao interface {
	Dao
	GetCodeCheckResult(serviceID string) (*model.CodeCheckResult, error)
	DeleteByServiceID(serviceID string) error
}

// EventDao EventDao
type EventDao interface {
	Dao
	CreateEventsInBatch(events []*model.ServiceEvent) error
	GetEventByEventID(eventID string) (*model.ServiceEvent, error)
	GetEventByEventIDs(eventIDs []string) ([]*model.ServiceEvent, error)
	GetEventByServiceID(serviceID string) ([]*model.ServiceEvent, error)
	DelEventByServiceID(serviceID string) error
	ListByTargetID(targetID string) ([]*model.ServiceEvent, error)
	GetEventsByTarget(target, targetID string, offset, liimt int) ([]*model.ServiceEvent, int, error)
	GetEventsByTenantEnvID(tenantEnvID string, offset, limit int) ([]*model.ServiceEvent, int, error)
	GetLastASyncEvent(target, targetID string) (*model.ServiceEvent, error)
	UnfinishedEvents(target, targetID string, optTypes ...string) ([]*model.ServiceEvent, error)
	LatestFailurePodEvent(podName string) (*model.ServiceEvent, error)
	UpdateReason(eventID string, reason string) error
	SetEventStatus(ctx context.Context, status model.EventStatus) error
	UpdateInBatch(events []*model.ServiceEvent) error
}

// VersionInfoDao VersionInfoDao
type VersionInfoDao interface {
	Dao
	ListSuccessfulOnes() ([]*model.VersionInfo, error)
	GetVersionByEventID(eventID string) (*model.VersionInfo, error)
	GetVersionByDeployVersion(version, serviceID string) (*model.VersionInfo, error)
	GetVersionByServiceID(serviceID string) ([]*model.VersionInfo, error)
	GetLatestScsVersion(sid string) (*model.VersionInfo, error)
	GetAllVersionByServiceID(serviceID string) ([]*model.VersionInfo, error)
	DeleteVersionByEventID(eventID string) error
	DeleteVersionByServiceID(serviceID string) error
	GetVersionInfo(timePoint time.Time, serviceIDList []string) ([]*model.VersionInfo, error)
	DeleteVersionInfo(obj *model.VersionInfo) error
	DeleteFailureVersionInfo(timePoint time.Time, status string, serviceIDList []string) error
	SearchVersionInfo() ([]*model.VersionInfo, error)
	ListByServiceIDStatus(serviceID string, finalStatus *bool) ([]*model.VersionInfo, error)
	ListVersionsByComponentIDs(componentIDs []string) ([]*model.VersionInfo, error)
}

// RegionUserInfoDao UserRegionInfoDao
type RegionUserInfoDao interface {
	Dao
	GetALLTokenInValidityPeriod() ([]*model.RegionUserInfo, error)
	GetTokenByEid(eid string) (*model.RegionUserInfo, error)
	GetTokenByTokenID(token string) (*model.RegionUserInfo, error)
}

// RegionAPIClassDao RegionAPIClassDao
type RegionAPIClassDao interface {
	Dao
	GetPrefixesByClass(apiClass string) ([]*model.RegionAPIClass, error)
	DeletePrefixInClass(apiClass, prefix string) error
}

// NotificationEventDao NotificationEventDao
type NotificationEventDao interface {
	Dao
	GetNotificationEventByHash(hash string) (*model.NotificationEvent, error)
	GetNotificationEventByKind(kind, kindID string) ([]*model.NotificationEvent, error)
	GetNotificationEventByTime(start, end time.Time) ([]*model.NotificationEvent, error)
	GetNotificationEventNotHandle() ([]*model.NotificationEvent, error)
}

// AppBackupDao group app backup history
type AppBackupDao interface {
	Dao
	CheckHistory(groupID, version string) bool
	GetAppBackups(groupID string) ([]*model.AppBackup, error)
	DeleteAppBackup(backupID string) error
	GetAppBackup(backupID string) (*model.AppBackup, error)
	GetDeleteAppBackup(backupID string) (*model.AppBackup, error)
	GetDeleteAppBackups() ([]*model.AppBackup, error)
}

// ServiceSourceDao service source dao
type ServiceSourceDao interface {
	Dao
	GetServiceSource(serviceID string) ([]*model.ServiceSourceConfig, error)
}

// CertificateDao -
type CertificateDao interface {
	Dao
	AddOrUpdate(mo model.Interface) error
	GetCertificateByID(certificateID string) (*model.Certificate, error)
}

// RuleExtensionDao -
type RuleExtensionDao interface {
	Dao
	GetRuleExtensionByRuleID(ruleID string) ([]*model.RuleExtension, error)
	DeleteRuleExtensionByRuleID(ruleID string) error
	DeleteByRuleIDs(ruleIDs []string) error
	CreateOrUpdateRuleExtensionsInBatch(exts []*model.RuleExtension) error
}

// HTTPRuleDao -
type HTTPRuleDao interface {
	Dao
	GetHTTPRuleByID(id string) (*model.HTTPRule, error)
	GetHTTPRuleByServiceIDAndContainerPort(serviceID string, containerPort int) ([]*model.HTTPRule, error)
	GetHTTPRulesByCertificateID(certificateID string) ([]*model.HTTPRule, error)
	DeleteHTTPRuleByID(id string) error
	DeleteHTTPRuleByServiceID(serviceID string) error
	ListByServiceID(serviceID string) ([]*model.HTTPRule, error)
	ListByComponentPort(componentID string, port int) ([]*model.HTTPRule, error)
	ListByCertID(certID string) ([]*model.HTTPRule, error)
	DeleteByComponentPort(componentID string, port int) error
	DeleteByComponentIDs(componentIDs []string) error
	CreateOrUpdateHTTPRuleInBatch(httpRules []*model.HTTPRule) error
	ListByComponentIDs(componentIDs []string) ([]*model.HTTPRule, error)
}

// HTTPRuleRewriteDao -
type HTTPRuleRewriteDao interface {
	Dao
	CreateOrUpdateHTTPRuleRewriteInBatch(httpRuleRewrites []*model.HTTPRuleRewrite) error
	ListByHTTPRuleID(httpRuleID string) ([]*model.HTTPRuleRewrite, error)
	DeleteByHTTPRuleID(httpRuleID string) error
	DeleteByHTTPRuleIDs(httpRuleIDs []string) error
}

// TCPRuleDao -
type TCPRuleDao interface {
	Dao
	GetTCPRuleByServiceIDAndContainerPort(serviceID string, containerPort int) ([]*model.TCPRule, error)
	GetTCPRuleByID(id string) (*model.TCPRule, error)
	GetTCPRuleByServiceID(sid string) ([]*model.TCPRule, error)
	DeleteByID(uuid string) error
	DeleteTCPRuleByServiceID(serviceID string) error
	ListByServiceID(serviceID string) ([]*model.TCPRule, error)
	GetUsedPortsByIP(ip string) ([]*model.TCPRule, error)
	DeleteByComponentPort(componentID string, port int) error
	DeleteByComponentIDs(componentIDs []string) error
	CreateOrUpdateTCPRuleInBatch(tcpRules []*model.TCPRule) error
}

// EndpointsDao is an interface for defining method
// for operating table 3rd_party_svc_endpoints.
type EndpointsDao interface {
	Dao
	GetByUUID(uuid string) (*model.Endpoint, error)
	DelByUUID(uuid string) error
	List(sid string) ([]*model.Endpoint, error)
	DeleteByServiceID(sid string) error
}

// ThirdPartySvcDiscoveryCfgDao is an interface for defining method
// for operating table 3rd_party_svc_discovery_cfg.
type ThirdPartySvcDiscoveryCfgDao interface {
	Dao
	GetByServiceID(sid string) (*model.ThirdPartySvcDiscoveryCfg, error)
	DeleteByServiceID(sid string) error
	DeleteByComponentIDs(componentIDs []string) error
	CreateOrUpdate3rdSvcDiscoveryCfgInBatch(cfgs []*model.ThirdPartySvcDiscoveryCfg) error
}

// GwRuleConfigDao is the interface that wraps the required methods to execute
// curd for table gateway_rule_config.
type GwRuleConfigDao interface {
	Dao
	DeleteByRuleID(rid string) error
	ListByRuleID(rid string) ([]*model.GwRuleConfig, error)
	DeleteByRuleIDs(ruleIDs []string) error
	CreateOrUpdateGwRuleConfigsInBatch(ruleConfigs []*model.GwRuleConfig) error
}

// TenantEnvServceAutoscalerRulesDao -
type TenantEnvServceAutoscalerRulesDao interface {
	Dao
	GetByRuleID(ruleID string) (*model.TenantEnvServiceAutoscalerRules, error)
	ListByServiceID(serviceID string) ([]*model.TenantEnvServiceAutoscalerRules, error)
	ListEnableOnesByServiceID(serviceID string) ([]*model.TenantEnvServiceAutoscalerRules, error)
	ListByComponentIDs(componentIDs []string) ([]*model.TenantEnvServiceAutoscalerRules, error)
	DeleteByComponentIDs(componentIDs []string) error
	CreateOrUpdateScaleRulesInBatch(rules []*model.TenantEnvServiceAutoscalerRules) error
}

// TenantEnvServceAutoscalerRuleMetricsDao -
type TenantEnvServceAutoscalerRuleMetricsDao interface {
	Dao
	UpdateOrCreate(metric *model.TenantEnvServiceAutoscalerRuleMetrics) error
	ListByRuleID(ruleID string) ([]*model.TenantEnvServiceAutoscalerRuleMetrics, error)
	DeleteByRuleID(ruldID string) error
	DeleteByRuleIDs(ruleIDs []string) error
	CreateOrUpdateScaleRuleMetricsInBatch(metrics []*model.TenantEnvServiceAutoscalerRuleMetrics) error
}

// TenantEnvServiceScalingRecordsDao -
type TenantEnvServiceScalingRecordsDao interface {
	Dao
	UpdateOrCreate(new *model.TenantEnvServiceScalingRecords) error
	ListByServiceID(serviceID string, offset, limit int) ([]*model.TenantEnvServiceScalingRecords, error)
	CountByServiceID(serviceID string) (int, error)
}

// TenantEnvServiceMonitorDao -
type TenantEnvServiceMonitorDao interface {
	Dao
	GetByName(serviceID, name string) (*model.TenantEnvServiceMonitor, error)
	GetByServiceID(serviceID string) ([]*model.TenantEnvServiceMonitor, error)
	DeleteServiceMonitor(mo *model.TenantEnvServiceMonitor) error
	DeleteServiceMonitorByServiceID(serviceID string) error
	DeleteByComponentIDs(componentIDs []string) error
	CreateOrUpdateMonitorInBatch(monitors []*model.TenantEnvServiceMonitor) error
}
