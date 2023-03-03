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

package mysql

import (
	"github.com/jinzhu/gorm"

	"github.com/wutong-paas/wutong/db/dao"
	mysqldao "github.com/wutong-paas/wutong/db/mysql/dao"
)

// VolumeTypeDao volumeTypeDao
func (m *Manager) VolumeTypeDao() dao.VolumeTypeDao {
	return &mysqldao.VolumeTypeDaoImpl{
		DB: m.db,
	}
}

// LicenseDao LicenseDao
func (m *Manager) LicenseDao() dao.LicenseDao {
	return &mysqldao.LicenseDaoImpl{
		DB: m.db,
	}
}

// TenantEnvDao 租户数据
func (m *Manager) TenantEnvDao() dao.TenantEnvDao {
	return &mysqldao.TenantEnvDaoImpl{
		DB: m.db,
	}
}

// TenantEnvDaoTransactions 租户数据，带操作事务
func (m *Manager) TenantEnvDaoTransactions(db *gorm.DB) dao.TenantEnvDao {
	return &mysqldao.TenantEnvDaoImpl{
		DB: db,
	}
}

// TenantEnvServiceDao TenantEnvServiceDao
func (m *Manager) TenantEnvServiceDao() dao.TenantEnvServiceDao {
	return &mysqldao.TenantEnvServicesDaoImpl{
		DB: m.db,
	}
}

// TenantEnvServiceDaoTransactions TenantEnvServiceDaoTransactions
func (m *Manager) TenantEnvServiceDaoTransactions(db *gorm.DB) dao.TenantEnvServiceDao {
	return &mysqldao.TenantEnvServicesDaoImpl{
		DB: db,
	}
}

// TenantEnvServiceDeleteDao TenantEnvServiceDeleteDao
func (m *Manager) TenantEnvServiceDeleteDao() dao.TenantEnvServiceDeleteDao {
	return &mysqldao.TenantEnvServicesDeleteImpl{
		DB: m.db,
	}
}

// TenantEnvServiceDeleteDaoTransactions TenantEnvServiceDeleteDaoTransactions
func (m *Manager) TenantEnvServiceDeleteDaoTransactions(db *gorm.DB) dao.TenantEnvServiceDeleteDao {
	return &mysqldao.TenantEnvServicesDeleteImpl{
		DB: db,
	}
}

// TenantEnvServicesPortDao TenantEnvServicesPortDao
func (m *Manager) TenantEnvServicesPortDao() dao.TenantEnvServicesPortDao {
	return &mysqldao.TenantEnvServicesPortDaoImpl{
		DB: m.db,
	}
}

// TenantEnvServicesPortDaoTransactions TenantEnvServicesPortDaoTransactions
func (m *Manager) TenantEnvServicesPortDaoTransactions(db *gorm.DB) dao.TenantEnvServicesPortDao {
	return &mysqldao.TenantEnvServicesPortDaoImpl{
		DB: db,
	}
}

// TenantEnvServiceRelationDao TenantEnvServiceRelationDao
func (m *Manager) TenantEnvServiceRelationDao() dao.TenantEnvServiceRelationDao {
	return &mysqldao.TenantEnvServiceRelationDaoImpl{
		DB: m.db,
	}
}

// TenantEnvServiceRelationDaoTransactions TenantEnvServiceRelationDaoTransactions
func (m *Manager) TenantEnvServiceRelationDaoTransactions(db *gorm.DB) dao.TenantEnvServiceRelationDao {
	return &mysqldao.TenantEnvServiceRelationDaoImpl{
		DB: db,
	}
}

// TenantEnvServiceEnvVarDao TenantEnvServiceEnvVarDao
func (m *Manager) TenantEnvServiceEnvVarDao() dao.TenantEnvServiceEnvVarDao {
	return &mysqldao.TenantEnvServiceEnvVarDaoImpl{
		DB: m.db,
	}
}

// TenantEnvServiceEnvVarDaoTransactions TenantEnvServiceEnvVarDaoTransactions
func (m *Manager) TenantEnvServiceEnvVarDaoTransactions(db *gorm.DB) dao.TenantEnvServiceEnvVarDao {
	return &mysqldao.TenantEnvServiceEnvVarDaoImpl{
		DB: db,
	}
}

// TenantEnvServiceMountRelationDao TenantEnvServiceMountRelationDao
func (m *Manager) TenantEnvServiceMountRelationDao() dao.TenantEnvServiceMountRelationDao {
	return &mysqldao.TenantEnvServiceMountRelationDaoImpl{
		DB: m.db,
	}
}

// TenantEnvServiceMountRelationDaoTransactions TenantEnvServiceMountRelationDaoTransactions
func (m *Manager) TenantEnvServiceMountRelationDaoTransactions(db *gorm.DB) dao.TenantEnvServiceMountRelationDao {
	return &mysqldao.TenantEnvServiceMountRelationDaoImpl{
		DB: db,
	}
}

// TenantEnvServiceVolumeDao TenantEnvServiceVolumeDao
func (m *Manager) TenantEnvServiceVolumeDao() dao.TenantEnvServiceVolumeDao {
	return &mysqldao.TenantEnvServiceVolumeDaoImpl{
		DB: m.db,
	}
}

// TenantEnvServiceVolumeDaoTransactions TenantEnvServiceVolumeDaoTransactions
func (m *Manager) TenantEnvServiceVolumeDaoTransactions(db *gorm.DB) dao.TenantEnvServiceVolumeDao {
	return &mysqldao.TenantEnvServiceVolumeDaoImpl{
		DB: db,
	}
}

// TenantEnvServiceConfigFileDao TenantEnvServiceConfigFileDao
func (m *Manager) TenantEnvServiceConfigFileDao() dao.TenantEnvServiceConfigFileDao {
	return &mysqldao.TenantEnvServiceConfigFileDaoImpl{
		DB: m.db,
	}
}

// TenantEnvServiceConfigFileDaoTransactions -
func (m *Manager) TenantEnvServiceConfigFileDaoTransactions(db *gorm.DB) dao.TenantEnvServiceConfigFileDao {
	return &mysqldao.TenantEnvServiceConfigFileDaoImpl{
		DB: db,
	}
}

// TenantEnvServiceLabelDao TenantEnvServiceLabelDao
func (m *Manager) TenantEnvServiceLabelDao() dao.TenantEnvServiceLabelDao {
	return &mysqldao.ServiceLabelDaoImpl{
		DB: m.db,
	}
}

// TenantEnvServiceLabelDaoTransactions TenantEnvServiceLabelDaoTransactions
func (m *Manager) TenantEnvServiceLabelDaoTransactions(db *gorm.DB) dao.TenantEnvServiceLabelDao {
	return &mysqldao.ServiceLabelDaoImpl{
		DB: db,
	}
}

// ServiceProbeDao ServiceProbeDao
func (m *Manager) ServiceProbeDao() dao.ServiceProbeDao {
	return &mysqldao.ServiceProbeDaoImpl{
		DB: m.db,
	}
}

// ServiceProbeDaoTransactions ServiceProbeDaoTransactions
func (m *Manager) ServiceProbeDaoTransactions(db *gorm.DB) dao.ServiceProbeDao {
	return &mysqldao.ServiceProbeDaoImpl{
		DB: db,
	}
}

// TenantEnvServiceLBMappingPortDao TenantEnvServiceLBMappingPortDao
func (m *Manager) TenantEnvServiceLBMappingPortDao() dao.TenantEnvServiceLBMappingPortDao {
	return &mysqldao.TenantEnvServiceLBMappingPortDaoImpl{
		DB: m.db,
	}
}

// TenantEnvServiceLBMappingPortDaoTransactions TenantEnvServiceLBMappingPortDaoTransactions
func (m *Manager) TenantEnvServiceLBMappingPortDaoTransactions(db *gorm.DB) dao.TenantEnvServiceLBMappingPortDao {
	return &mysqldao.TenantEnvServiceLBMappingPortDaoImpl{
		DB: db,
	}
}

// TenantEnvPluginDao TenantEnvPluginDao
func (m *Manager) TenantEnvPluginDao() dao.TenantEnvPluginDao {
	return &mysqldao.PluginDaoImpl{
		DB: m.db,
	}
}

// TenantEnvPluginDaoTransactions TenantEnvPluginDaoTransactions
func (m *Manager) TenantEnvPluginDaoTransactions(db *gorm.DB) dao.TenantEnvPluginDao {
	return &mysqldao.PluginDaoImpl{
		DB: db,
	}
}

// TenantEnvPluginBuildVersionDao TenantEnvPluginBuildVersionDao
func (m *Manager) TenantEnvPluginBuildVersionDao() dao.TenantEnvPluginBuildVersionDao {
	return &mysqldao.PluginBuildVersionDaoImpl{
		DB: m.db,
	}
}

// TenantEnvPluginBuildVersionDaoTransactions TenantEnvPluginBuildVersionDaoTransactions
func (m *Manager) TenantEnvPluginBuildVersionDaoTransactions(db *gorm.DB) dao.TenantEnvPluginBuildVersionDao {
	return &mysqldao.PluginBuildVersionDaoImpl{
		DB: db,
	}
}

// TenantEnvPluginDefaultENVDao TenantEnvPluginDefaultENVDao
func (m *Manager) TenantEnvPluginDefaultENVDao() dao.TenantEnvPluginDefaultENVDao {
	return &mysqldao.PluginDefaultENVDaoImpl{
		DB: m.db,
	}
}

// TenantEnvPluginDefaultENVDaoTransactions TenantEnvPluginDefaultENVDaoTransactions
func (m *Manager) TenantEnvPluginDefaultENVDaoTransactions(db *gorm.DB) dao.TenantEnvPluginDefaultENVDao {
	return &mysqldao.PluginDefaultENVDaoImpl{
		DB: db,
	}
}

// TenantEnvPluginVersionENVDao TenantEnvPluginVersionENVDao
func (m *Manager) TenantEnvPluginVersionENVDao() dao.TenantEnvPluginVersionEnvDao {
	return &mysqldao.PluginVersionEnvDaoImpl{
		DB: m.db,
	}
}

// TenantEnvPluginVersionENVDaoTransactions TenantEnvPluginVersionENVDaoTransactions
func (m *Manager) TenantEnvPluginVersionENVDaoTransactions(db *gorm.DB) dao.TenantEnvPluginVersionEnvDao {
	return &mysqldao.PluginVersionEnvDaoImpl{
		DB: db,
	}
}

// TenantEnvPluginVersionConfigDao TenantEnvPluginVersionENVDao
func (m *Manager) TenantEnvPluginVersionConfigDao() dao.TenantEnvPluginVersionConfigDao {
	return &mysqldao.PluginVersionConfigDaoImpl{
		DB: m.db,
	}
}

// TenantEnvPluginVersionConfigDaoTransactions TenantEnvPluginVersionConfigDaoTransactions
func (m *Manager) TenantEnvPluginVersionConfigDaoTransactions(db *gorm.DB) dao.TenantEnvPluginVersionConfigDao {
	return &mysqldao.PluginVersionConfigDaoImpl{
		DB: db,
	}
}

// TenantEnvServicePluginRelationDao TenantEnvServicePluginRelationDao
func (m *Manager) TenantEnvServicePluginRelationDao() dao.TenantEnvServicePluginRelationDao {
	return &mysqldao.TenantEnvServicePluginRelationDaoImpl{
		DB: m.db,
	}
}

// TenantEnvServicePluginRelationDaoTransactions TenantEnvServicePluginRelationDaoTransactions
func (m *Manager) TenantEnvServicePluginRelationDaoTransactions(db *gorm.DB) dao.TenantEnvServicePluginRelationDao {
	return &mysqldao.TenantEnvServicePluginRelationDaoImpl{
		DB: db,
	}
}

// TenantEnvServicesStreamPluginPortDao TenantEnvServicesStreamPluginPortDao
func (m *Manager) TenantEnvServicesStreamPluginPortDao() dao.TenantEnvServicesStreamPluginPortDao {
	return &mysqldao.TenantEnvServicesStreamPluginPortDaoImpl{
		DB: m.db,
	}
}

// TenantEnvServicesStreamPluginPortDaoTransactions TenantEnvServicesStreamPluginPortDaoTransactions
func (m *Manager) TenantEnvServicesStreamPluginPortDaoTransactions(db *gorm.DB) dao.TenantEnvServicesStreamPluginPortDao {
	return &mysqldao.TenantEnvServicesStreamPluginPortDaoImpl{
		DB: db,
	}
}

// CodeCheckResultDao CodeCheckResultDao
func (m *Manager) CodeCheckResultDao() dao.CodeCheckResultDao {
	return &mysqldao.CodeCheckResultDaoImpl{
		DB: m.db,
	}
}

// CodeCheckResultDaoTransactions CodeCheckResultDaoTransactions
func (m *Manager) CodeCheckResultDaoTransactions(db *gorm.DB) dao.CodeCheckResultDao {
	return &mysqldao.CodeCheckResultDaoImpl{
		DB: db,
	}
}

// ServiceEventDao TenantEnvServicePluginRelationDao
func (m *Manager) ServiceEventDao() dao.EventDao {
	return &mysqldao.EventDaoImpl{
		DB: m.db,
	}
}

// ServiceEventDaoTransactions TenantEnvServicePluginRelationDaoTransactions
func (m *Manager) ServiceEventDaoTransactions(db *gorm.DB) dao.EventDao {
	return &mysqldao.EventDaoImpl{
		DB: db,
	}
}

// VersionInfoDao VersionInfoDao
func (m *Manager) VersionInfoDao() dao.VersionInfoDao {
	return &mysqldao.VersionInfoDaoImpl{
		DB: m.db,
	}
}

// VersionInfoDaoTransactions VersionInfoDaoTransactions
func (m *Manager) VersionInfoDaoTransactions(db *gorm.DB) dao.VersionInfoDao {
	return &mysqldao.VersionInfoDaoImpl{
		DB: db,
	}
}

// LocalSchedulerDao 本地调度信息
func (m *Manager) LocalSchedulerDao() dao.LocalSchedulerDao {
	return &mysqldao.LocalSchedulerDaoImpl{
		DB: m.db,
	}
}

// RegionAPIClassDao RegionAPIClassDao
func (m *Manager) RegionAPIClassDao() dao.RegionAPIClassDao {
	return &mysqldao.RegionAPIClassDaoImpl{
		DB: m.db,
	}
}

// RegionAPIClassDaoTransactions RegionAPIClassDaoTransactions
func (m *Manager) RegionAPIClassDaoTransactions(db *gorm.DB) dao.RegionAPIClassDao {
	return &mysqldao.RegionAPIClassDaoImpl{
		DB: db,
	}
}

// NotificationEventDao NotificationEventDao
func (m *Manager) NotificationEventDao() dao.NotificationEventDao {
	return &mysqldao.NotificationEventDaoImpl{
		DB: m.db,
	}
}

// AppDao app export and import info
func (m *Manager) AppDao() dao.AppDao {
	return &mysqldao.AppDaoImpl{
		DB: m.db,
	}
}

// ApplicationDao -
func (m *Manager) ApplicationDao() dao.ApplicationDao {
	return &mysqldao.ApplicationDaoImpl{
		DB: m.db,
	}
}

// ApplicationDaoTransactions -
func (m *Manager) ApplicationDaoTransactions(db *gorm.DB) dao.ApplicationDao {
	return &mysqldao.ApplicationDaoImpl{
		DB: db,
	}
}

// AppConfigGroupDao -
func (m *Manager) AppConfigGroupDao() dao.AppConfigGroupDao {
	return &mysqldao.AppConfigGroupDaoImpl{
		DB: m.db,
	}
}

// AppConfigGroupDaoTransactions -
func (m *Manager) AppConfigGroupDaoTransactions(db *gorm.DB) dao.AppConfigGroupDao {
	return &mysqldao.AppConfigGroupDaoImpl{
		DB: db,
	}
}

// AppConfigGroupServiceDao -
func (m *Manager) AppConfigGroupServiceDao() dao.AppConfigGroupServiceDao {
	return &mysqldao.AppConfigGroupServiceDaoImpl{
		DB: m.db,
	}
}

// AppConfigGroupServiceDaoTransactions -
func (m *Manager) AppConfigGroupServiceDaoTransactions(db *gorm.DB) dao.AppConfigGroupServiceDao {
	return &mysqldao.AppConfigGroupServiceDaoImpl{
		DB: db,
	}
}

// AppConfigGroupItemDao -
func (m *Manager) AppConfigGroupItemDao() dao.AppConfigGroupItemDao {
	return &mysqldao.AppConfigGroupItemDaoImpl{
		DB: m.db,
	}
}

// AppConfigGroupItemDaoTransactions -
func (m *Manager) AppConfigGroupItemDaoTransactions(db *gorm.DB) dao.AppConfigGroupItemDao {
	return &mysqldao.AppConfigGroupItemDaoImpl{
		DB: db,
	}
}

// AppBackupDao group app backup info
func (m *Manager) AppBackupDao() dao.AppBackupDao {
	return &mysqldao.AppBackupDaoImpl{
		DB: m.db,
	}
}

// AppBackupDaoTransactions -
func (m *Manager) AppBackupDaoTransactions(db *gorm.DB) dao.AppBackupDao {
	return &mysqldao.AppBackupDaoImpl{
		DB: db,
	}
}

// ServiceSourceDao service source db impl
func (m *Manager) ServiceSourceDao() dao.ServiceSourceDao {
	return &mysqldao.ServiceSourceImpl{
		DB: m.db,
	}
}

// CertificateDao CertificateDao
func (m *Manager) CertificateDao() dao.CertificateDao {
	return &mysqldao.CertificateDaoImpl{
		DB: m.db,
	}
}

// CertificateDaoTransactions CertificateDaoTransactions
func (m *Manager) CertificateDaoTransactions(db *gorm.DB) dao.CertificateDao {
	return &mysqldao.CertificateDaoImpl{
		DB: db,
	}
}

// RuleExtensionDao RuleExtensionDao
func (m *Manager) RuleExtensionDao() dao.RuleExtensionDao {
	return &mysqldao.RuleExtensionDaoImpl{
		DB: m.db,
	}
}

// RuleExtensionDaoTransactions RuleExtensionDaoTransactions
func (m *Manager) RuleExtensionDaoTransactions(db *gorm.DB) dao.RuleExtensionDao {
	return &mysqldao.RuleExtensionDaoImpl{
		DB: db,
	}
}

// HTTPRuleDao HTTPRuleDao
func (m *Manager) HTTPRuleDao() dao.HTTPRuleDao {
	return &mysqldao.HTTPRuleDaoImpl{
		DB: m.db,
	}
}

// HTTPRuleDaoTransactions -
func (m *Manager) HTTPRuleDaoTransactions(db *gorm.DB) dao.HTTPRuleDao {
	return &mysqldao.HTTPRuleDaoImpl{
		DB: db,
	}
}

// HTTPRuleRewriteDao HTTPRuleRewriteDao
func (m *Manager) HTTPRuleRewriteDao() dao.HTTPRuleRewriteDao {
	return &mysqldao.HTTPRuleRewriteDaoTmpl{
		DB: m.db,
	}
}

// HTTPRuleRewriteDaoTransactions -
func (m *Manager) HTTPRuleRewriteDaoTransactions(db *gorm.DB) dao.HTTPRuleRewriteDao {
	return &mysqldao.HTTPRuleRewriteDaoTmpl{
		DB: db,
	}
}

// TCPRuleDao TCPRuleDao
func (m *Manager) TCPRuleDao() dao.TCPRuleDao {
	return &mysqldao.TCPRuleDaoTmpl{
		DB: m.db,
	}
}

// TCPRuleDaoTransactions TCPRuleDaoTransactions
func (m *Manager) TCPRuleDaoTransactions(db *gorm.DB) dao.TCPRuleDao {
	return &mysqldao.TCPRuleDaoTmpl{
		DB: db,
	}
}

// EndpointsDao returns a new EndpointDaoImpl with default *gorm.DB.
func (m *Manager) EndpointsDao() dao.EndpointsDao {
	return &mysqldao.EndpointDaoImpl{
		DB: m.db,
	}
}

// EndpointsDaoTransactions returns a new EndpointDaoImpl with the givem *gorm.DB.
func (m *Manager) EndpointsDaoTransactions(db *gorm.DB) dao.EndpointsDao {
	return &mysqldao.EndpointDaoImpl{
		DB: db,
	}
}

// ThirdPartySvcDiscoveryCfgDao returns a new ThirdPartySvcDiscoveryCfgDao.
func (m *Manager) ThirdPartySvcDiscoveryCfgDao() dao.ThirdPartySvcDiscoveryCfgDao {
	return &mysqldao.ThirdPartySvcDiscoveryCfgDaoImpl{
		DB: m.db,
	}
}

// ThirdPartySvcDiscoveryCfgDaoTransactions returns a new ThirdPartySvcDiscoveryCfgDao.
func (m *Manager) ThirdPartySvcDiscoveryCfgDaoTransactions(db *gorm.DB) dao.ThirdPartySvcDiscoveryCfgDao {
	return &mysqldao.ThirdPartySvcDiscoveryCfgDaoImpl{
		DB: db,
	}
}

// GwRuleConfigDao creates a new dao.GwRuleConfigDao.
func (m *Manager) GwRuleConfigDao() dao.GwRuleConfigDao {
	return &mysqldao.GwRuleConfigDaoImpl{
		DB: m.db,
	}
}

// GwRuleConfigDaoTransactions creates a new dao.GwRuleConfigDao with special transaction.
func (m *Manager) GwRuleConfigDaoTransactions(db *gorm.DB) dao.GwRuleConfigDao {
	return &mysqldao.GwRuleConfigDaoImpl{
		DB: db,
	}
}

// TenantEnvServceAutoscalerRulesDao -
func (m *Manager) TenantEnvServceAutoscalerRulesDao() dao.TenantEnvServceAutoscalerRulesDao {
	return &mysqldao.TenantEnvServceAutoscalerRulesDaoImpl{
		DB: m.db,
	}
}

// TenantEnvServceAutoscalerRulesDaoTransactions -
func (m *Manager) TenantEnvServceAutoscalerRulesDaoTransactions(db *gorm.DB) dao.TenantEnvServceAutoscalerRulesDao {
	return &mysqldao.TenantEnvServceAutoscalerRulesDaoImpl{
		DB: db,
	}
}

// TenantEnvServceAutoscalerRuleMetricsDao -
func (m *Manager) TenantEnvServceAutoscalerRuleMetricsDao() dao.TenantEnvServceAutoscalerRuleMetricsDao {
	return &mysqldao.TenantEnvServceAutoscalerRuleMetricsDaoImpl{
		DB: m.db,
	}
}

// TenantEnvServceAutoscalerRuleMetricsDaoTransactions -
func (m *Manager) TenantEnvServceAutoscalerRuleMetricsDaoTransactions(db *gorm.DB) dao.TenantEnvServceAutoscalerRuleMetricsDao {
	return &mysqldao.TenantEnvServceAutoscalerRuleMetricsDaoImpl{
		DB: db,
	}
}

// TenantEnvServiceScalingRecordsDao -
func (m *Manager) TenantEnvServiceScalingRecordsDao() dao.TenantEnvServiceScalingRecordsDao {
	return &mysqldao.TenantEnvServiceScalingRecordsDaoImpl{
		DB: m.db,
	}
}

// TenantEnvServiceScalingRecordsDaoTransactions -
func (m *Manager) TenantEnvServiceScalingRecordsDaoTransactions(db *gorm.DB) dao.TenantEnvServiceScalingRecordsDao {
	return &mysqldao.TenantEnvServiceScalingRecordsDaoImpl{
		DB: db,
	}
}

// TenantEnvServiceMonitorDao monitor dao
func (m *Manager) TenantEnvServiceMonitorDao() dao.TenantEnvServiceMonitorDao {
	return &mysqldao.TenantEnvServiceMonitorDaoImpl{
		DB: m.db,
	}
}

// TenantEnvServiceMonitorDaoTransactions monitor dao
func (m *Manager) TenantEnvServiceMonitorDaoTransactions(db *gorm.DB) dao.TenantEnvServiceMonitorDao {
	return &mysqldao.TenantEnvServiceMonitorDaoImpl{
		DB: db,
	}
}
