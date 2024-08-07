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
	"fmt"
	"strings"

	pkgerr "github.com/pkg/errors"
	gormbulkups "github.com/wutong-paas/gorm-bulk-upsert"

	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
	"github.com/wutong-paas/wutong/db/errors"
	"github.com/wutong-paas/wutong/db/model"
)

// PluginDaoImpl PluginDaoImpl
type PluginDaoImpl struct {
	DB *gorm.DB
}

// AddModel 创建插件
func (t *PluginDaoImpl) AddModel(mo model.Interface) error {
	plugin := mo.(*model.TenantEnvPlugin)
	var oldPlugin model.TenantEnvPlugin
	if ok := t.DB.Where("plugin_id = ? and tenant_env_id = ?", plugin.PluginID, plugin.TenantEnvID).Find(&oldPlugin).RecordNotFound(); ok {
		if err := t.DB.Create(plugin).Error; err != nil {
			return err
		}
	} else {
		logrus.Infof("plugin id: %s; tenant env id: %s; tenant env plugin already exist", plugin.PluginID, plugin.TenantEnvID)
		return errors.ErrRecordAlreadyExist
	}
	return nil
}

// UpdateModel 更新插件
func (t *PluginDaoImpl) UpdateModel(mo model.Interface) error {
	plugin := mo.(*model.TenantEnvPlugin)
	if err := t.DB.Save(plugin).Error; err != nil {
		return err
	}
	return nil
}

// GetPluginByID GetPluginByID
func (t *PluginDaoImpl) GetPluginByID(id, tenantEnvID string) (*model.TenantEnvPlugin, error) {
	var plugin model.TenantEnvPlugin
	// if err := t.DB.Where("plugin_id = ? and tenant_env_id = ?", id, tenantEnvID).Find(&plugin).Error; err != nil {
	if err := t.DB.Where("plugin_id = ?", id).Find(&plugin).Error; err != nil {
		return nil, err
	}
	return &plugin, nil
}

// ListByIDs returns the list of plugins based on the given plugin ids.
func (t *PluginDaoImpl) ListByIDs(ids []string) ([]*model.TenantEnvPlugin, error) {
	var plugins []*model.TenantEnvPlugin
	if err := t.DB.Where("plugin_id in (?)", ids).Find(&plugins).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return plugins, nil
}

// DeletePluginByID DeletePluginByID
func (t *PluginDaoImpl) DeletePluginByID(id, tenantEnvID string) error {
	var plugin model.TenantEnvPlugin
	if tenantEnvID == "" {
		return t.DB.Where("plugin_id=?", id).Delete(&plugin).Error
	} else {
		return t.DB.Where("plugin_id=? and tenant_env_id=?", id, tenantEnvID).Delete(&plugin).Error
	}
}

// GetPluginsByTenantEnvID GetPluginsByTenantEnvID
func (t *PluginDaoImpl) GetPluginsByTenantEnvID(tenantEnvID string) ([]*model.TenantEnvPlugin, error) {
	var plugins []*model.TenantEnvPlugin
	if err := t.DB.Where("tenant_env_id=?", tenantEnvID).Find(&plugins).Error; err != nil {
		return nil, err
	}
	return plugins, nil
}

// ListByTenantEnvID -
func (t *PluginDaoImpl) ListByTenantEnvID(tenantEnvID string) ([]*model.TenantEnvPlugin, error) {
	var plugins []*model.TenantEnvPlugin
	if err := t.DB.Where("tenant_env_id=?", tenantEnvID).Find(&plugins).Error; err != nil {
		return nil, err
	}

	return plugins, nil
}

// CreateOrUpdatePluginsInBatch -
func (t *PluginDaoImpl) CreateOrUpdatePluginsInBatch(plugins []*model.TenantEnvPlugin) error {
	var objects []interface{}
	for _, plugin := range plugins {
		objects = append(objects, *plugin)
	}
	if err := gormbulkups.BulkUpsert(t.DB, objects, 2000); err != nil {
		return pkgerr.Wrap(err, "create or update plugins in batch")
	}
	return nil
}

// PluginDefaultENVDaoImpl PluginDefaultENVDaoImpl
type PluginDefaultENVDaoImpl struct {
	DB *gorm.DB
}

// AddModel 添加插件默认变量
func (t *PluginDefaultENVDaoImpl) AddModel(mo model.Interface) error {
	env := mo.(*model.TenantEnvPluginDefaultENV)
	var oldENV model.TenantEnvPluginDefaultENV
	if ok := t.DB.Where("plugin_id=? and env_name = ? and version_id = ?",
		env.PluginID,
		env.ENVName,
		env.VersionID).Find(&oldENV).RecordNotFound(); ok {
		if err := t.DB.Create(env).Error; err != nil {
			return err
		}
	} else {
		return fmt.Errorf("env %s is exist", env.ENVName)
	}
	return nil
}

// UpdateModel 更新插件默认变量
func (t *PluginDefaultENVDaoImpl) UpdateModel(mo model.Interface) error {
	env := mo.(*model.TenantEnvPluginDefaultENV)
	if err := t.DB.Save(env).Error; err != nil {
		return err
	}
	return nil
}

// GetALLMasterDefultENVs GetALLMasterDefultENVs
func (t *PluginDefaultENVDaoImpl) GetALLMasterDefultENVs(pluginID string) ([]*model.TenantEnvPluginDefaultENV, error) {
	var envs []*model.TenantEnvPluginDefaultENV
	if err := t.DB.Where("plugin_id=? and version_id=?", pluginID, "master_rb").Find(&envs).Error; err != nil {
		return nil, err
	}
	return envs, nil
}

// GetDefaultENVByName GetDefaultENVByName
func (t *PluginDefaultENVDaoImpl) GetDefaultENVByName(pluginID, name, versionID string) (*model.TenantEnvPluginDefaultENV, error) {
	var env model.TenantEnvPluginDefaultENV
	if err := t.DB.Where("plugin_id=? and env_name=? and version_id=?",
		pluginID,
		name,
		versionID).Find(&env).Error; err != nil {
		return nil, err
	}
	return &env, nil
}

// GetDefaultENVSByPluginID GetDefaultENVSByPluginID
func (t *PluginDefaultENVDaoImpl) GetDefaultENVSByPluginID(pluginID, versionID string) ([]*model.TenantEnvPluginDefaultENV, error) {
	var envs []*model.TenantEnvPluginDefaultENV
	if err := t.DB.Where("plugin_id=? and version_id=?", pluginID, versionID).Find(&envs).Error; err != nil {
		return nil, err
	}
	return envs, nil
}

// DeleteDefaultENVByName DeleteDefaultENVByName
func (t *PluginDefaultENVDaoImpl) DeleteDefaultENVByName(pluginID, name, versionID string) error {
	relation := &model.TenantEnvPluginDefaultENV{
		ENVName: name,
	}
	if err := t.DB.Where("plugin_id=? and env_name=? and version_id=?",
		pluginID, name, versionID).Delete(relation).Error; err != nil {
		return err
	}
	return nil
}

// DeleteDefaultENVByPluginIDAndVersionID DeleteDefaultENVByPluginIDAndVersionID
func (t *PluginDefaultENVDaoImpl) DeleteDefaultENVByPluginIDAndVersionID(pluginID, versionID string) error {
	relation := &model.TenantEnvPluginDefaultENV{
		PluginID: pluginID,
	}
	if err := t.DB.Where("plugin_id=? and version_id=?", pluginID, versionID).Delete(relation).Error; err != nil {
		return err
	}
	return nil
}

// DeleteAllDefaultENVByPluginID DeleteAllDefaultENVByPluginID
func (t *PluginDefaultENVDaoImpl) DeleteAllDefaultENVByPluginID(pluginID string) error {
	relation := &model.TenantEnvPluginDefaultENV{
		PluginID: pluginID,
	}
	if err := t.DB.Where("plugin_id=?", pluginID).Delete(relation).Error; err != nil {
		return err
	}
	return nil
}

// GetDefaultEnvWhichCanBeSetByPluginID GetDefaultEnvWhichCanBeSetByPluginID
func (t *PluginDefaultENVDaoImpl) GetDefaultEnvWhichCanBeSetByPluginID(pluginID, versionID string) ([]*model.TenantEnvPluginDefaultENV, error) {
	var envs []*model.TenantEnvPluginDefaultENV
	if err := t.DB.Where("plugin_id=? and is_change=? and version_id=?", pluginID, true, versionID).Find(&envs).Error; err != nil {
		return nil, err
	}
	return envs, nil
}

// PluginBuildVersionDaoImpl PluginBuildVersionDaoImpl
type PluginBuildVersionDaoImpl struct {
	DB *gorm.DB
}

// AddModel 添加插件构建版本信息
func (t *PluginBuildVersionDaoImpl) AddModel(mo model.Interface) error {
	version := mo.(*model.TenantEnvPluginBuildVersion)
	var oldVersion model.TenantEnvPluginBuildVersion
	if ok := t.DB.Where("plugin_id =? and version_id = ? and deploy_version=?", version.PluginID, version.VersionID, version.DeployVersion).Find(&oldVersion).RecordNotFound(); ok {
		if err := t.DB.Create(version).Error; err != nil {
			return err
		}
	} else {
		logrus.Infof("plugin id: %s; version_id: %s; deploy_version: %s; tenant env plugin build versoin already exist", version.PluginID, version.VersionID, version.DeployVersion)
		return errors.ErrRecordAlreadyExist
	}
	return nil
}

// UpdateModel 更新插件默认变量
// 主体信息一般不变更，仅构建的本地镜像名与status需要变更
func (t *PluginBuildVersionDaoImpl) UpdateModel(mo model.Interface) error {
	version := mo.(*model.TenantEnvPluginBuildVersion)
	if version.ID == 0 {
		return fmt.Errorf("id can not be empty when update build verion")
	}
	if err := t.DB.Save(version).Error; err != nil {
		return err
	}
	return nil
}

// DeleteBuildVersionByVersionID DeleteBuildVersionByVersionID
func (t *PluginBuildVersionDaoImpl) DeleteBuildVersionByVersionID(versionID string) error {
	relation := &model.TenantEnvPluginBuildVersion{
		VersionID: versionID,
	}
	if err := t.DB.Where("version_id=?", versionID).Delete(relation).Error; err != nil {
		return err
	}
	return nil
}

// DeleteBuildVersionByPluginID DeleteBuildVersionByPluginID
func (t *PluginBuildVersionDaoImpl) DeleteBuildVersionByPluginID(pluginID string) error {
	relation := &model.TenantEnvPluginBuildVersion{
		PluginID: pluginID,
	}
	if err := t.DB.Where("plugin_id=?", pluginID).Delete(relation).Error; err != nil {
		return err
	}
	return nil
}

// GetBuildVersionByPluginID GetBuildVersionByPluginID
func (t *PluginBuildVersionDaoImpl) GetBuildVersionByPluginID(pluginID string) ([]*model.TenantEnvPluginBuildVersion, error) {
	var versions []*model.TenantEnvPluginBuildVersion
	if err := t.DB.Where("plugin_id = ? and status= ?", pluginID, "complete").Find(&versions).Error; err != nil {
		return nil, err
	}
	return versions, nil
}

// GetBuildVersionByVersionID GetBuildVersionByVersionID
func (t *PluginBuildVersionDaoImpl) GetBuildVersionByVersionID(pluginID, versionID string) (*model.TenantEnvPluginBuildVersion, error) {
	var version model.TenantEnvPluginBuildVersion
	if err := t.DB.Where("plugin_id=? and version_id = ? ", pluginID, versionID).Find(&version).Error; err != nil {
		return nil, err
	}
	return &version, nil
}

// GetBuildVersionByDeployVersion GetBuildVersionByDeployVersion
func (t *PluginBuildVersionDaoImpl) GetBuildVersionByDeployVersion(pluginID, versionID, deployVersion string) (*model.TenantEnvPluginBuildVersion, error) {
	var version model.TenantEnvPluginBuildVersion
	if err := t.DB.Where("plugin_id=? and version_id = ? and deploy_version=?", pluginID, versionID, deployVersion).Find(&version).Error; err != nil {
		return nil, err
	}
	return &version, nil
}

// ListSuccessfulOnesByPluginIDs returns the list of successful build versions,
func (t *PluginBuildVersionDaoImpl) ListSuccessfulOnesByPluginIDs(pluginIDs []string) ([]*model.TenantEnvPluginBuildVersion, error) {
	var version []*model.TenantEnvPluginBuildVersion
	if err := t.DB.Where("ID in (?) ", t.DB.Table("tenant_env_plugin_build_version").Select("max(id)").Where("plugin_id in (?) and status=?", pluginIDs, "complete").Group("plugin_id").QueryExpr()).Find(&version).Error; err != nil {
		return nil, err
	}
	return version, nil
}

// GetLastBuildVersionByVersionID get last success build version
func (t *PluginBuildVersionDaoImpl) GetLastBuildVersionByVersionID(pluginID, versionID string) (*model.TenantEnvPluginBuildVersion, error) {
	var version model.TenantEnvPluginBuildVersion
	if err := t.DB.Where("plugin_id=? and version_id = ? and status=?", pluginID, versionID, "complete").Order("ID desc").Limit("1").Find(&version).Error; err != nil {
		return nil, err
	}
	return &version, nil
}

// CreateOrUpdatePluginBuildVersionsInBatch -
func (t *PluginBuildVersionDaoImpl) CreateOrUpdatePluginBuildVersionsInBatch(buildVersions []*model.TenantEnvPluginBuildVersion) error {
	var objects []interface{}
	for _, version := range buildVersions {
		objects = append(objects, *version)
	}
	if err := gormbulkups.BulkUpsert(t.DB, objects, 2000); err != nil {
		return pkgerr.Wrap(err, "create or update plugin build versions in batch")
	}
	return nil
}

// PluginVersionEnvDaoImpl PluginVersionEnvDaoImpl
type PluginVersionEnvDaoImpl struct {
	DB *gorm.DB
}

// AddModel 添加插件默认变量
func (t *PluginVersionEnvDaoImpl) AddModel(mo model.Interface) error {
	env := mo.(*model.TenantEnvPluginVersionEnv)
	var oldENV model.TenantEnvPluginVersionEnv
	if ok := t.DB.Where("service_id=? and plugin_id=? and env_name = ?", env.ServiceID, env.PluginID, env.EnvName).Find(&oldENV).RecordNotFound(); ok {
		if err := t.DB.Create(env).Error; err != nil {
			return err
		}
	} else {
		return fmt.Errorf("env %s is exist", env.EnvName)
	}
	return nil
}

// UpdateModel 更新插件默认变量
func (t *PluginVersionEnvDaoImpl) UpdateModel(mo model.Interface) error {
	env := mo.(*model.TenantEnvPluginVersionEnv)
	if env.ID == 0 || env.ServiceID == "" || env.PluginID == "" {
		return fmt.Errorf("id can not be empty when update plugin version env")
	}
	if err := t.DB.Save(env).Error; err != nil {
		return err
	}
	return nil
}

// DeleteEnvByEnvName 删除单个env
func (t *PluginVersionEnvDaoImpl) DeleteEnvByEnvName(envName, pluginID, serviceID string) error {
	env := &model.TenantEnvPluginVersionEnv{
		PluginID:  pluginID,
		EnvName:   envName,
		ServiceID: serviceID,
	}
	return t.DB.Where("env_name=? and plugin_id=? and service_id=?", envName, pluginID, serviceID).Delete(env).Error
}

// DeleteEnvByPluginID 删除插件依赖关系时，需要操作删除对应env
func (t *PluginVersionEnvDaoImpl) DeleteEnvByPluginID(serviceID, pluginID string) error {
	env := &model.TenantEnvPluginVersionEnv{
		PluginID:  pluginID,
		ServiceID: serviceID,
	}
	return t.DB.Where("plugin_id=? and service_id= ?", pluginID, serviceID).Delete(env).Error
}

// DeleteEnvByServiceID 删除应用时，需要进行此操作
func (t *PluginVersionEnvDaoImpl) DeleteEnvByServiceID(serviceID string) error {
	env := &model.TenantEnvPluginVersionEnv{
		ServiceID: serviceID,
	}
	return t.DB.Where("service_id=?", serviceID).Delete(env).Error
}

// GetVersionEnvByServiceID 获取该应用下使用的某个插件依赖的插件变量
func (t *PluginVersionEnvDaoImpl) GetVersionEnvByServiceID(serviceID string, pluginID string) ([]*model.TenantEnvPluginVersionEnv, error) {
	var envs []*model.TenantEnvPluginVersionEnv
	if err := t.DB.Where("service_id=? and plugin_id=?", serviceID, pluginID).Find(&envs).Error; err != nil {
		return nil, err
	}
	return envs, nil
}

// GetVersionEnvByEnvName GetVersionEnvByEnvName
func (t *PluginVersionEnvDaoImpl) GetVersionEnvByEnvName(serviceID, pluginID, envName string) (*model.TenantEnvPluginVersionEnv, error) {
	var env model.TenantEnvPluginVersionEnv
	if err := t.DB.Where("service_id=? and plugin_id=? and env_name=?", serviceID, pluginID, envName).Find(&env).Error; err != nil {
		return nil, err
	}
	return &env, nil
}

// ListByServiceID returns the list of environment variables for the plugin via serviceID
func (t *PluginVersionEnvDaoImpl) ListByServiceID(serviceID string) ([]*model.TenantEnvPluginVersionEnv, error) {
	var envs []*model.TenantEnvPluginVersionEnv
	if err := t.DB.Where("service_id=?", serviceID).Find(&envs).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return envs, nil
}

// DeleteByComponentIDs -
func (t *PluginVersionEnvDaoImpl) DeleteByComponentIDs(componentIDs []string) error {
	return t.DB.Where("service_id in (?)", componentIDs).Delete(&model.TenantEnvPluginVersionEnv{}).Error
}

// CreateOrUpdatePluginVersionEnvsInBatch -
func (t *PluginVersionEnvDaoImpl) CreateOrUpdatePluginVersionEnvsInBatch(versionEnvs []*model.TenantEnvPluginVersionEnv) error {
	var objects []interface{}
	for _, env := range versionEnvs {
		objects = append(objects, *env)
	}
	if err := gormbulkups.BulkUpsert(t.DB, objects, 2000); err != nil {
		return pkgerr.Wrap(err, "create or update plugin version env in batch")
	}
	return nil
}

// PluginVersionConfigDaoImpl PluginVersionEnvDaoImpl
type PluginVersionConfigDaoImpl struct {
	DB *gorm.DB
}

// AddModel add or update service plugin config
func (t *PluginVersionConfigDaoImpl) AddModel(mo model.Interface) error {
	config := mo.(*model.TenantEnvPluginVersionDiscoverConfig)
	var oldconfig model.TenantEnvPluginVersionDiscoverConfig
	if ok := t.DB.Where("service_id=? and plugin_id=?", config.ServiceID, config.PluginID).Find(&oldconfig).RecordNotFound(); ok {
		if err := t.DB.Create(config).Error; err != nil {
			return err
		}
	} else {
		config.ID = oldconfig.ID
		config.CreatedAt = oldconfig.CreatedAt
		return t.UpdateModel(config)
	}
	return nil
}

// UpdateModel update service plugin config
func (t *PluginVersionConfigDaoImpl) UpdateModel(mo model.Interface) error {
	env := mo.(*model.TenantEnvPluginVersionDiscoverConfig)
	if env.ID == 0 || env.ServiceID == "" || env.PluginID == "" {
		return fmt.Errorf("id can not be empty when update plugin version config")
	}
	if err := t.DB.Save(env).Error; err != nil {
		return err
	}
	return nil
}

// DeletePluginConfig delete service plugin config
func (t *PluginVersionConfigDaoImpl) DeletePluginConfig(serviceID, pluginID string) error {
	var oldconfig model.TenantEnvPluginVersionDiscoverConfig
	if err := t.DB.Where("service_id=? and plugin_id=?", serviceID, pluginID).Delete(&oldconfig).Error; err != nil {
		return err
	}
	return nil
}

// DeletePluginConfigByServiceID Batch delete config by service id
func (t *PluginVersionConfigDaoImpl) DeletePluginConfigByServiceID(serviceID string) error {
	var oldconfig model.TenantEnvPluginVersionDiscoverConfig
	if err := t.DB.Where("service_id=?", serviceID).Delete(&oldconfig).Error; err != nil {
		return err
	}
	return nil
}

// GetPluginConfig get service plugin config
func (t *PluginVersionConfigDaoImpl) GetPluginConfig(serviceID, pluginID string) (*model.TenantEnvPluginVersionDiscoverConfig, error) {
	var oldconfig model.TenantEnvPluginVersionDiscoverConfig
	if err := t.DB.Where("service_id=? and plugin_id=?", serviceID, pluginID).Find(&oldconfig).Error; err != nil {
		return nil, err
	}
	return &oldconfig, nil
}

// GetPluginConfigs get plugin configs
func (t *PluginVersionConfigDaoImpl) GetPluginConfigs(serviceID string) ([]*model.TenantEnvPluginVersionDiscoverConfig, error) {
	var oldconfigs []*model.TenantEnvPluginVersionDiscoverConfig
	if err := t.DB.Where("service_id=?", serviceID).Find(&oldconfigs).Error; err != nil {
		return nil, err
	}
	return oldconfigs, nil
}

// DeleteByComponentIDs -
func (t *PluginVersionConfigDaoImpl) DeleteByComponentIDs(componentIDs []string) error {
	return t.DB.Where("service_id in (?)", componentIDs).Delete(&model.TenantEnvPluginVersionDiscoverConfig{}).Error
}

// CreateOrUpdatePluginVersionConfigsInBatch -
func (t *PluginVersionConfigDaoImpl) CreateOrUpdatePluginVersionConfigsInBatch(versionConfigs []*model.TenantEnvPluginVersionDiscoverConfig) error {
	var objects []interface{}
	for _, config := range versionConfigs {
		objects = append(objects, *config)
	}
	if err := gormbulkups.BulkUpsert(t.DB, objects, 2000); err != nil {
		return pkgerr.Wrap(err, "create or update plugin version config in batch")
	}
	return nil
}

// TenantEnvServicePluginRelationDaoImpl TenantEnvServicePluginRelationDaoImpl
type TenantEnvServicePluginRelationDaoImpl struct {
	DB *gorm.DB
}

// AddModel 添加插件默认变量
func (t *TenantEnvServicePluginRelationDaoImpl) AddModel(mo model.Interface) error {
	relation := mo.(*model.TenantEnvServicePluginRelation)
	var oldRelation model.TenantEnvServicePluginRelation
	if ok := t.DB.Where("service_id= ? and plugin_id=?", relation.ServiceID, relation.PluginID).Find(&oldRelation).RecordNotFound(); ok {
		if err := t.DB.Create(relation).Error; err != nil {
			return err
		}
	} else {
		return errors.ErrRecordAlreadyExist
	}
	return nil
}

// UpdateModel 更新插件默认变量 更新依赖的version id
func (t *TenantEnvServicePluginRelationDaoImpl) UpdateModel(mo model.Interface) error {
	relation := mo.(*model.TenantEnvServicePluginRelation)
	if relation.ID == 0 {
		return fmt.Errorf("id can not be empty when update service plugin relation")
	}
	if err := t.DB.Save(relation).Error; err != nil {
		return err
	}
	return nil
}

// DeleteRelationByServiceIDAndPluginID 删除service plugin 对应关系
func (t *TenantEnvServicePluginRelationDaoImpl) DeleteRelationByServiceIDAndPluginID(serviceID, pluginID string) error {
	relation := &model.TenantEnvServicePluginRelation{
		ServiceID: serviceID,
		PluginID:  pluginID,
	}
	return t.DB.Where("plugin_id=? and service_id=?",
		pluginID,
		serviceID).Delete(relation).Error
}

// CheckSomeModelPluginByServiceID 检查是否绑定了某种插件且处于启用状态
func (t *TenantEnvServicePluginRelationDaoImpl) CheckSomeModelPluginByServiceID(serviceID, pluginModel string) (bool, error) {
	var relations []*model.TenantEnvServicePluginRelation
	if err := t.DB.Where("service_id=? and plugin_model=? and switch=?", serviceID, pluginModel, true).Find(&relations).Error; err != nil {
		return false, err
	}
	if len(relations) == 1 {
		return true, nil
	}
	return false, nil
}

//CheckSomeModelLikePluginByServiceID 检查是否绑定了某大类插件
// func (t *TenantEnvServicePluginRelationDaoImpl) CheckSomeModelLikePluginByServiceID(serviceID, pluginModel string) (bool, error) {
// 	var relations []*model.TenantEnvServicePluginRelation
// 	// catePlugin := "%" + pluginModel + "%"
// 	// if err := t.DB.Where("service_id=? and plugin_model LIKE ?", serviceID, catePlugin).Find(&relations).Error; err != nil {
// 	// return false, err
// 	// }
// 	if err := t.DB.Where("service_id=? and plugin_model LIKE 'net-plugin:%'", serviceID).Find(&relations).Error; err != nil {
// 		return false, err
// 	}
// 	if len(relations) == 1 {
// 		return true, nil
// 	}
// 	return false, nil
// }

// CheckPluginBeforeInstall 插件安装前的检查
// 1. 如果组件之前已经安装过网络类插件，不能继续安装同类插件
// 2. 如果组件之前已经安装过数据中间件管理插件，不能继续安装同类插件
func (t *TenantEnvServicePluginRelationDaoImpl) CheckPluginBeforeInstall(serviceID, pluginModel string) (bool, error) {
	if strings.HasPrefix(pluginModel, "net-plugin:") {
		var relations []*model.TenantEnvServicePluginRelation
		if err := t.DB.Where("service_id=? and plugin_model LIKE 'net-plugin:%'", serviceID).Find(&relations).Error; err != nil {
			return false, err
		}
		if len(relations) > 0 {
			return false, nil
		}
	}
	if pluginModel == model.DbgatePlugin {
		var relations []*model.TenantEnvServicePluginRelation
		if err := t.DB.Where("service_id=? and plugin_model=?", serviceID, model.DbgatePlugin).Find(&relations).Error; err != nil {
			return false, err
		}
		if len(relations) > 0 {
			return false, nil
		}
	}
	return true, nil
}

// DeleteALLRelationByServiceID 删除serviceID所有插件依赖 一般用于删除应用时使用
func (t *TenantEnvServicePluginRelationDaoImpl) DeleteALLRelationByServiceID(serviceID string) error {
	relation := &model.TenantEnvServicePluginRelation{
		ServiceID: serviceID,
	}
	return t.DB.Where("service_id=?", serviceID).Delete(relation).Error
}

// DeleteALLRelationByPluginID 删除pluginID所有依赖 一般不要使用 会影响关联过的应用启动
func (t *TenantEnvServicePluginRelationDaoImpl) DeleteALLRelationByPluginID(pluginID string) error {
	relation := &model.TenantEnvServicePluginRelation{
		PluginID: pluginID,
	}
	return t.DB.Where("plugin_id=?", pluginID).Delete(relation).Error
}

// GetALLRelationByServiceID 获取当前应用所有的插件依赖关系
func (t *TenantEnvServicePluginRelationDaoImpl) GetALLRelationByServiceID(serviceID string) ([]*model.TenantEnvServicePluginRelation, error) {
	var relations []*model.TenantEnvServicePluginRelation
	if err := t.DB.Where("service_id=?", serviceID).Find(&relations).Error; err != nil {
		return nil, err
	}
	return relations, nil
}

// GetRelateionByServiceIDAndPluginID GetRelateionByServiceIDAndPluginID
func (t *TenantEnvServicePluginRelationDaoImpl) GetRelateionByServiceIDAndPluginID(serviceID, pluginID string) (*model.TenantEnvServicePluginRelation, error) {
	relation := &model.TenantEnvServicePluginRelation{
		PluginID:  pluginID,
		ServiceID: serviceID,
	}
	if err := t.DB.Where("plugin_id=? and service_id=?", pluginID, serviceID).Find(relation).Error; err != nil {
		return nil, err
	}
	return relation, nil
}

// DeleteByComponentIDs -
func (t *TenantEnvServicePluginRelationDaoImpl) DeleteByComponentIDs(componentIDs []string) error {
	return t.DB.Where("service_id in (?)", componentIDs).Delete(&model.TenantEnvServicePluginRelation{}).Error
}

// CreateOrUpdatePluginRelsInBatch -
func (t *TenantEnvServicePluginRelationDaoImpl) CreateOrUpdatePluginRelsInBatch(relations []*model.TenantEnvServicePluginRelation) error {
	var objects []interface{}
	for _, relation := range relations {
		objects = append(objects, *relation)
	}
	if err := gormbulkups.BulkUpsert(t.DB, objects, 2000); err != nil {
		return pkgerr.Wrap(err, "create or update plugin relation in batch")
	}
	return nil
}

// TenantEnvServicesStreamPluginPortDaoImpl TenantEnvServicesStreamPluginPortDaoImpl
type TenantEnvServicesStreamPluginPortDaoImpl struct {
	DB *gorm.DB
}

// AddModel 添加插件端口映射信息
func (t *TenantEnvServicesStreamPluginPortDaoImpl) AddModel(mo model.Interface) error {
	port := mo.(*model.TenantEnvServicesStreamPluginPort)
	var oldPort model.TenantEnvServicesStreamPluginPort
	if ok := t.DB.Where("service_id= ? and container_port= ? and plugin_model=? ",
		port.ServiceID,
		port.ContainerPort,
		port.PluginModel).Find(&oldPort).RecordNotFound(); ok {
		if err := t.DB.Create(port).Error; err != nil {
			return err
		}
	} else {
		return fmt.Errorf("plugin port %d mappint to %d is exist", port.ContainerPort, port.PluginPort)
	}
	return nil
}

// UpdateModel 更新插件端口映射信息
func (t *TenantEnvServicesStreamPluginPortDaoImpl) UpdateModel(mo model.Interface) error {
	port := mo.(*model.TenantEnvServicesStreamPluginPort)
	if port.ID == 0 {
		return fmt.Errorf("id can not be empty when update plugin mapping port")
	}
	if err := t.DB.Save(port).Error; err != nil {
		return err
	}
	return nil
}

// GetPluginMappingPorts GetPluginMappingPorts  降序排列
func (t *TenantEnvServicesStreamPluginPortDaoImpl) GetPluginMappingPorts(
	serviceID string) ([]*model.TenantEnvServicesStreamPluginPort, error) {
	var ports []*model.TenantEnvServicesStreamPluginPort
	if err := t.DB.Where("service_id=?", serviceID).Order("plugin_port asc").Find(&ports).Error; err != nil {
		return nil, err
	}
	return ports, nil
}

// GetPluginMappingPortByServiceIDAndContainerPort GetPluginMappingPortByServiceIDAndContainerPort
func (t *TenantEnvServicesStreamPluginPortDaoImpl) GetPluginMappingPortByServiceIDAndContainerPort(
	serviceID string,
	pluginModel string,
	containerPort int,
) (*model.TenantEnvServicesStreamPluginPort, error) {
	var port model.TenantEnvServicesStreamPluginPort
	if err := t.DB.Where(
		"service_id=? and plugin_model=? and container_port=?",
		serviceID,
		pluginModel,
		containerPort,
	).Find(&port).Error; err != nil {
		return nil, err
	}
	return &port, nil
}

// SetPluginMappingPort SetPluginMappingPort
func (t *TenantEnvServicesStreamPluginPortDaoImpl) SetPluginMappingPort(
	tenantEnvID string,
	serviceID string,
	pluginModel string,
	containerPort int) (int, error) {
	ports, err := t.GetPluginMappingPorts(serviceID)
	if err != nil {
		return 0, err
	}
	//if have been allocated,return
	for _, oldp := range ports {
		if oldp.ContainerPort == containerPort {
			return oldp.PluginPort, nil
		}
	}
	//Distribution port range
	minPort := 65301
	maxPort := 65400
	newPort := &model.TenantEnvServicesStreamPluginPort{
		TenantEnvID:   tenantEnvID,
		ServiceID:     serviceID,
		PluginModel:   pluginModel,
		ContainerPort: containerPort,
	}
	if len(ports) == 0 {
		newPort.PluginPort = minPort
		if err := t.AddModel(newPort); err != nil {
			return 0, err
		}
		return newPort.PluginPort, nil
	}
	oldMaxPort := ports[len(ports)-1]
	//已分配端口+2大于最大端口限制则从原范围内扫描端口使用
	if oldMaxPort.PluginPort > (maxPort - 2) {
		waitPort := minPort
		for _, p := range ports {
			if p.PluginPort == waitPort {
				waitPort++
				continue
			}
			newPort.PluginPort = waitPort
			if err := t.AddModel(newPort); err != nil {
				return 0, nil
			}
			continue
		}
	}
	//端口与预分配端口相同重新分配
	if containerPort == (oldMaxPort.PluginPort + 1) {
		newPort.PluginPort = oldMaxPort.PluginPort + 2
		if err := t.AddModel(newPort); err != nil {
			return 0, err
		}
		return newPort.PluginPort, nil
	}
	newPort.PluginPort = oldMaxPort.PluginPort + 1
	if err := t.AddModel(newPort); err != nil {
		return 0, err
	}
	return newPort.PluginPort, nil
}

// DeletePluginMappingPortByContainerPort DeletePluginMappingPortByContainerPort
func (t *TenantEnvServicesStreamPluginPortDaoImpl) DeletePluginMappingPortByContainerPort(
	serviceID string,
	pluginModel string,
	containerPort int) error {
	relation := &model.TenantEnvServicesStreamPluginPort{
		ServiceID:     serviceID,
		PluginModel:   pluginModel,
		ContainerPort: containerPort,
	}
	return t.DB.Where("service_id=? and plugin_model=? and container_port=?",
		serviceID,
		pluginModel,
		containerPort).Delete(relation).Error
}

// DeleteAllPluginMappingPortByServiceID DeleteAllPluginMappingPortByServiceID
func (t *TenantEnvServicesStreamPluginPortDaoImpl) DeleteAllPluginMappingPortByServiceID(serviceID string) error {
	relation := &model.TenantEnvServicesStreamPluginPort{
		ServiceID: serviceID,
	}
	return t.DB.Where("service_id=?", serviceID).Delete(relation).Error
}

// ListByServiceID returns the list of environment variables for the plugin via serviceID
func (t *TenantEnvServicesStreamPluginPortDaoImpl) ListByServiceID(sid string) ([]*model.TenantEnvServicesStreamPluginPort, error) {
	var result []*model.TenantEnvServicesStreamPluginPort
	if err := t.DB.Where("service_id=?", sid).Find(&result).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return result, nil
}

// DeleteByComponentIDs -
func (t *TenantEnvServicesStreamPluginPortDaoImpl) DeleteByComponentIDs(componentIDs []string) error {
	return t.DB.Where("service_id in (?)", componentIDs).Delete(&model.TenantEnvServicesStreamPluginPort{}).Error
}

// CreateOrUpdateStreamPluginPortsInBatch -
func (t *TenantEnvServicesStreamPluginPortDaoImpl) CreateOrUpdateStreamPluginPortsInBatch(spPorts []*model.TenantEnvServicesStreamPluginPort) error {
	var objects []interface{}
	for _, volRel := range spPorts {
		objects = append(objects, *volRel)
	}
	if err := gormbulkups.BulkUpsert(t.DB, objects, 2000); err != nil {
		return pkgerr.Wrap(err, "create or update stream plugin port failed in batch")
	}
	return nil
}
