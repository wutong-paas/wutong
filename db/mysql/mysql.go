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
	"sync"

	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
	"github.com/wutong-paas/wutong/db/config"
	"github.com/wutong-paas/wutong/db/model"

	// import sql driver manually
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jinzhu/gorm/dialects/postgres"
)

// Manager db manager
type Manager struct {
	db      *gorm.DB
	config  config.Config
	initOne sync.Once
	models  []model.Interface
}

// CreateManager create manager
func CreateManager(config config.Config) (*Manager, error) {
	var db *gorm.DB
	if config.DBType == "mysql" {
		var err error
		db, err = gorm.Open("mysql", config.MysqlConnectionInfo+"?charset=utf8mb4&parseTime=True&loc=Local")
		if err != nil {
			return nil, err
		}
	}
	if config.DBType == "cockroachdb" {
		var err error
		addr := config.MysqlConnectionInfo
		db, err = gorm.Open("postgres", addr)
		if err != nil {
			return nil, err
		}
	}
	if config.ShowSQL {
		db = db.Debug()
	}
	manager := &Manager{
		db:      db,
		config:  config,
		initOne: sync.Once{},
	}
	db.SetLogger(manager)
	manager.RegisterTableModel()
	manager.CheckTable()
	logrus.Debug("mysql db driver create")
	return manager, nil
}

// CloseManager 关闭管理器
func (m *Manager) CloseManager() error {
	return m.db.Close()
}

// Begin begin a transaction
func (m *Manager) Begin() *gorm.DB {
	return m.db.Begin()
}

// DB returns the db.
func (m *Manager) DB() *gorm.DB {
	return m.db
}

// EnsureEndTransactionFunc -
func (m *Manager) EnsureEndTransactionFunc() func(tx *gorm.DB) {
	return func(tx *gorm.DB) {
		if r := recover(); r != nil {
			logrus.Errorf("Unexpected panic occurred, rollback transaction: %v", r)
			tx.Rollback()
		}
	}
}

// Print Print
func (m *Manager) Print(v ...interface{}) {
	logrus.Info(v...)
}

// RegisterTableModel register table model
func (m *Manager) RegisterTableModel() {
	m.models = append(m.models, &model.TenantEnvs{})
	m.models = append(m.models, &model.TenantEnvServices{})
	m.models = append(m.models, &model.TenantEnvServicesPort{})
	m.models = append(m.models, &model.TenantEnvServiceRelation{})
	m.models = append(m.models, &model.TenantEnvServiceEnvVar{})
	m.models = append(m.models, &model.TenantEnvServiceMountRelation{})
	m.models = append(m.models, &model.TenantEnvServiceVolume{})
	m.models = append(m.models, &model.TenantEnvServiceLable{})
	m.models = append(m.models, &model.TenantEnvServiceProbe{})
	m.models = append(m.models, &model.LicenseInfo{})
	m.models = append(m.models, &model.TenantEnvServicesDelete{})
	m.models = append(m.models, &model.TenantEnvServiceLBMappingPort{})
	m.models = append(m.models, &model.TenantEnvPlugin{})
	m.models = append(m.models, &model.TenantEnvPluginBuildVersion{})
	m.models = append(m.models, &model.TenantEnvServicePluginRelation{})
	m.models = append(m.models, &model.TenantEnvPluginVersionEnv{})
	m.models = append(m.models, &model.TenantEnvPluginVersionDiscoverConfig{})
	m.models = append(m.models, &model.CodeCheckResult{})
	m.models = append(m.models, &model.ServiceEvent{})
	m.models = append(m.models, &model.VersionInfo{})
	m.models = append(m.models, &model.RegionUserInfo{})
	m.models = append(m.models, &model.TenantEnvServicesStreamPluginPort{})
	m.models = append(m.models, &model.RegionAPIClass{})
	m.models = append(m.models, &model.RegionProcotols{})
	m.models = append(m.models, &model.LocalScheduler{})
	m.models = append(m.models, &model.NotificationEvent{})
	m.models = append(m.models, &model.AppStatus{})
	m.models = append(m.models, &model.AppBackup{})
	m.models = append(m.models, &model.ServiceSourceConfig{})
	m.models = append(m.models, &model.Application{})
	m.models = append(m.models, &model.ApplicationConfigGroup{})
	m.models = append(m.models, &model.ConfigGroupService{})
	m.models = append(m.models, &model.ConfigGroupItem{})
	// gateway
	m.models = append(m.models, &model.Certificate{})
	m.models = append(m.models, &model.RuleExtension{})
	m.models = append(m.models, &model.HTTPRule{})
	m.models = append(m.models, &model.HTTPRuleRewrite{})
	m.models = append(m.models, &model.TCPRule{})
	m.models = append(m.models, &model.TenantEnvServiceConfigFile{})
	m.models = append(m.models, &model.Endpoint{})
	m.models = append(m.models, &model.ThirdPartySvcDiscoveryCfg{})
	m.models = append(m.models, &model.GwRuleConfig{})

	// volumeType
	m.models = append(m.models, &model.TenantEnvServiceVolumeType{})
	// pod autoscaler
	m.models = append(m.models, &model.TenantEnvServiceAutoscalerRules{})
	m.models = append(m.models, &model.TenantEnvServiceAutoscalerRuleMetrics{})
	m.models = append(m.models, &model.TenantEnvServiceScalingRecords{})
	m.models = append(m.models, &model.TenantEnvServiceMonitor{})
}

// CheckTable check and create tables
func (m *Manager) CheckTable() {
	m.initOne.Do(func() {
		for _, md := range m.models {
			if !m.db.HasTable(md) {
				if m.config.DBType == "mysql" {
					err := m.db.Set("gorm:table_options", "ENGINE=InnoDB charset=utf8mb4").CreateTable(md).Error
					if err != nil {
						logrus.Errorf("auto create table %s to db error."+err.Error(), md.TableName())
					} else {
						logrus.Infof("auto create table %s to db success", md.TableName())
					}
				} else { //cockroachdb
					err := m.db.CreateTable(md).Error
					if err != nil {
						logrus.Errorf("auto create cockroachdb table %s to db error."+err.Error(), md.TableName())
					} else {
						logrus.Infof("auto create cockroachdb table %s to db success", md.TableName())
					}
				}
			} else {
				if err := m.db.AutoMigrate(md).Error; err != nil {
					logrus.Errorf("auto Migrate table %s to db error."+err.Error(), md.TableName())
				}
			}
		}
		m.patchTable()
	})
}

func (m *Manager) patchTable() {
	if err := m.db.Exec("alter table tenant_env_services_envs modify column attr_value text;").Error; err != nil {
		logrus.Errorf("alter table tenant_env_services_envs error %s", err.Error())
	}

	if err := m.db.Exec("alter table tenant_env_services_event modify column request_body varchar(1024);").Error; err != nil {
		logrus.Errorf("alter table tenant_env_services_envent error %s", err.Error())
	}

	if err := m.db.Exec("update gateway_tcp_rule set ip=? where ip=?", "0.0.0.0", "").Error; err != nil {
		logrus.Errorf("update gateway_tcp_rule data error %s", err.Error())
	}
	if err := m.db.Exec("alter table tenant_env_services_volume modify column volume_type varchar(64);").Error; err != nil {
		logrus.Errorf("alter table tenant_env_services_volume error: %s", err.Error())
	}
	if err := m.db.Exec("update tenantEnvs set namespace=uuid where namespace is NULL;").Error; err != nil {
		logrus.Errorf("update tenantEnvs namespace error: %s", err.Error())
	}
	if err := m.db.Exec("update applications set k8s_app=concat('app-',LEFT(app_id,8)) where k8s_app is NULL;").Error; err != nil {
		logrus.Errorf("update tenantEnvs namespace error: %s", err.Error())
	}
	if err := m.db.Exec("update tenant_env_services set k8s_component_name=service_alias where k8s_component_name is NULL;").Error; err != nil {
		logrus.Errorf("update tenantEnvs namespace error: %s", err.Error())
	}
}
