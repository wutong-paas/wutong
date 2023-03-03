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

package db

import (
	"errors"
	"fmt"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
	"github.com/wutong-paas/wutong/db/config"
	"github.com/wutong-paas/wutong/db/dao"
	"github.com/wutong-paas/wutong/db/mysql"
)

// Manager db manager
type Manager interface {
	CloseManager() error
	Begin() *gorm.DB
	DB() *gorm.DB
	EnsureEndTransactionFunc() func(tx *gorm.DB)
	VolumeTypeDao() dao.VolumeTypeDao
	LicenseDao() dao.LicenseDao
	AppDao() dao.AppDao
	ApplicationDao() dao.ApplicationDao
	ApplicationDaoTransactions(db *gorm.DB) dao.ApplicationDao
	AppConfigGroupDao() dao.AppConfigGroupDao
	AppConfigGroupDaoTransactions(db *gorm.DB) dao.AppConfigGroupDao
	AppConfigGroupServiceDao() dao.AppConfigGroupServiceDao
	AppConfigGroupServiceDaoTransactions(db *gorm.DB) dao.AppConfigGroupServiceDao
	AppConfigGroupItemDao() dao.AppConfigGroupItemDao
	AppConfigGroupItemDaoTransactions(db *gorm.DB) dao.AppConfigGroupItemDao
	TenantEnvDao() dao.TenantEnvDao
	TenantEnvDaoTransactions(db *gorm.DB) dao.TenantEnvDao
	TenantEnvServiceDao() dao.TenantEnvServiceDao
	TenantEnvServiceDeleteDao() dao.TenantEnvServiceDeleteDao
	TenantEnvServiceDaoTransactions(db *gorm.DB) dao.TenantEnvServiceDao
	TenantEnvServiceDeleteDaoTransactions(db *gorm.DB) dao.TenantEnvServiceDeleteDao
	TenantEnvServicesPortDao() dao.TenantEnvServicesPortDao
	TenantEnvServicesPortDaoTransactions(*gorm.DB) dao.TenantEnvServicesPortDao
	TenantEnvServiceRelationDao() dao.TenantEnvServiceRelationDao
	TenantEnvServiceRelationDaoTransactions(*gorm.DB) dao.TenantEnvServiceRelationDao
	TenantEnvServiceEnvVarDao() dao.TenantEnvServiceEnvVarDao
	TenantEnvServiceEnvVarDaoTransactions(*gorm.DB) dao.TenantEnvServiceEnvVarDao
	TenantEnvServiceMountRelationDao() dao.TenantEnvServiceMountRelationDao
	TenantEnvServiceMountRelationDaoTransactions(db *gorm.DB) dao.TenantEnvServiceMountRelationDao
	TenantEnvServiceVolumeDao() dao.TenantEnvServiceVolumeDao
	TenantEnvServiceVolumeDaoTransactions(*gorm.DB) dao.TenantEnvServiceVolumeDao
	TenantEnvServiceConfigFileDao() dao.TenantEnvServiceConfigFileDao
	TenantEnvServiceConfigFileDaoTransactions(*gorm.DB) dao.TenantEnvServiceConfigFileDao
	ServiceProbeDao() dao.ServiceProbeDao
	ServiceProbeDaoTransactions(*gorm.DB) dao.ServiceProbeDao
	TenantEnvServiceLBMappingPortDao() dao.TenantEnvServiceLBMappingPortDao
	TenantEnvServiceLBMappingPortDaoTransactions(*gorm.DB) dao.TenantEnvServiceLBMappingPortDao
	TenantEnvServiceLabelDao() dao.TenantEnvServiceLabelDao
	TenantEnvServiceLabelDaoTransactions(db *gorm.DB) dao.TenantEnvServiceLabelDao
	LocalSchedulerDao() dao.LocalSchedulerDao
	TenantEnvPluginDaoTransactions(db *gorm.DB) dao.TenantEnvPluginDao
	TenantEnvPluginDao() dao.TenantEnvPluginDao
	TenantEnvPluginDefaultENVDaoTransactions(db *gorm.DB) dao.TenantEnvPluginDefaultENVDao
	TenantEnvPluginDefaultENVDao() dao.TenantEnvPluginDefaultENVDao
	TenantEnvPluginBuildVersionDao() dao.TenantEnvPluginBuildVersionDao
	TenantEnvPluginBuildVersionDaoTransactions(db *gorm.DB) dao.TenantEnvPluginBuildVersionDao
	TenantEnvPluginVersionENVDao() dao.TenantEnvPluginVersionEnvDao
	TenantEnvPluginVersionENVDaoTransactions(db *gorm.DB) dao.TenantEnvPluginVersionEnvDao
	TenantEnvPluginVersionConfigDao() dao.TenantEnvPluginVersionConfigDao
	TenantEnvPluginVersionConfigDaoTransactions(db *gorm.DB) dao.TenantEnvPluginVersionConfigDao
	TenantEnvServicePluginRelationDao() dao.TenantEnvServicePluginRelationDao
	TenantEnvServicePluginRelationDaoTransactions(db *gorm.DB) dao.TenantEnvServicePluginRelationDao
	TenantEnvServicesStreamPluginPortDao() dao.TenantEnvServicesStreamPluginPortDao
	TenantEnvServicesStreamPluginPortDaoTransactions(db *gorm.DB) dao.TenantEnvServicesStreamPluginPortDao

	CodeCheckResultDao() dao.CodeCheckResultDao
	CodeCheckResultDaoTransactions(db *gorm.DB) dao.CodeCheckResultDao

	ServiceEventDao() dao.EventDao
	ServiceEventDaoTransactions(db *gorm.DB) dao.EventDao

	VersionInfoDao() dao.VersionInfoDao
	VersionInfoDaoTransactions(db *gorm.DB) dao.VersionInfoDao

	RegionAPIClassDao() dao.RegionAPIClassDao
	RegionAPIClassDaoTransactions(db *gorm.DB) dao.RegionAPIClassDao

	NotificationEventDao() dao.NotificationEventDao
	AppBackupDao() dao.AppBackupDao
	AppBackupDaoTransactions(db *gorm.DB) dao.AppBackupDao
	ServiceSourceDao() dao.ServiceSourceDao

	// gateway
	CertificateDao() dao.CertificateDao
	CertificateDaoTransactions(db *gorm.DB) dao.CertificateDao
	RuleExtensionDao() dao.RuleExtensionDao
	RuleExtensionDaoTransactions(db *gorm.DB) dao.RuleExtensionDao
	HTTPRuleDao() dao.HTTPRuleDao
	HTTPRuleDaoTransactions(db *gorm.DB) dao.HTTPRuleDao
	HTTPRuleRewriteDao() dao.HTTPRuleRewriteDao
	HTTPRuleRewriteDaoTransactions(db *gorm.DB) dao.HTTPRuleRewriteDao
	TCPRuleDao() dao.TCPRuleDao
	TCPRuleDaoTransactions(db *gorm.DB) dao.TCPRuleDao
	GwRuleConfigDao() dao.GwRuleConfigDao
	GwRuleConfigDaoTransactions(db *gorm.DB) dao.GwRuleConfigDao

	// third-party service
	EndpointsDao() dao.EndpointsDao
	EndpointsDaoTransactions(db *gorm.DB) dao.EndpointsDao
	ThirdPartySvcDiscoveryCfgDao() dao.ThirdPartySvcDiscoveryCfgDao
	ThirdPartySvcDiscoveryCfgDaoTransactions(db *gorm.DB) dao.ThirdPartySvcDiscoveryCfgDao

	TenantEnvServceAutoscalerRulesDao() dao.TenantEnvServceAutoscalerRulesDao
	TenantEnvServceAutoscalerRulesDaoTransactions(db *gorm.DB) dao.TenantEnvServceAutoscalerRulesDao
	TenantEnvServceAutoscalerRuleMetricsDao() dao.TenantEnvServceAutoscalerRuleMetricsDao
	TenantEnvServceAutoscalerRuleMetricsDaoTransactions(db *gorm.DB) dao.TenantEnvServceAutoscalerRuleMetricsDao
	TenantEnvServiceScalingRecordsDao() dao.TenantEnvServiceScalingRecordsDao
	TenantEnvServiceScalingRecordsDaoTransactions(db *gorm.DB) dao.TenantEnvServiceScalingRecordsDao

	TenantEnvServiceMonitorDao() dao.TenantEnvServiceMonitorDao
	TenantEnvServiceMonitorDaoTransactions(db *gorm.DB) dao.TenantEnvServiceMonitorDao
}

var defaultManager Manager

var supportDrivers map[string]struct{}

func init() {
	supportDrivers = map[string]struct{}{
		"mysql":       {},
		"cockroachdb": {},
	}
}

// CreateManager 创建manager
func CreateManager(config config.Config) (err error) {
	if _, ok := supportDrivers[config.DBType]; !ok {
		return fmt.Errorf("DB drivers: %s not supported", config.DBType)
	}

	for {
		defaultManager, err = mysql.CreateManager(config)
		if err == nil {
			logrus.Infof("db manager is ready")
			break
		}
		logrus.Errorf("get db manager failed, try time is %d,%s", 10, err.Error())
		time.Sleep(10 * time.Second)
	}
	//TODO:etcd db plugin
	//defaultManager, err = etcd.CreateManager(config)
	return
}

// CloseManager close db manager
func CloseManager() error {
	if defaultManager == nil {
		return errors.New("default db manager not init")
	}
	return defaultManager.CloseManager()
}

// GetManager get db manager
func GetManager() Manager {
	return defaultManager
}

// SetTestManager sets the default manager for unit test
func SetTestManager(m Manager) {
	defaultManager = m
}
