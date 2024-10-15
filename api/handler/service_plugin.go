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
	"fmt"

	"github.com/wutong-paas/wutong/worker/discover/model"

	"github.com/jinzhu/gorm"

	"github.com/pquerna/ffjson/ffjson"
	"github.com/sirupsen/logrus"
	api_model "github.com/wutong-paas/wutong/api/model"
	"github.com/wutong-paas/wutong/api/util"
	"github.com/wutong-paas/wutong/db"
	dbmodel "github.com/wutong-paas/wutong/db/model"
	gclient "github.com/wutong-paas/wutong/mq/client"
)

// GetTenantEnvServicePluginRelation GetTenantEnvServicePluginRelation
func (s *ServiceAction) GetTenantEnvServicePluginRelation(serviceID string) ([]*dbmodel.TenantEnvServicePluginRelation, *util.APIHandleError) {
	gps, err := db.GetManager().TenantEnvServicePluginRelationDao().GetALLRelationByServiceID(serviceID)
	if err != nil {
		return nil, util.CreateAPIHandleErrorFromDBError("get service relation by ID", err)
	}
	return gps, nil
}

// TenantEnvServiceDeletePluginRelation uninstall plugin for app
func (s *ServiceAction) TenantEnvServiceDeletePluginRelation(tenantEnvID, serviceID, pluginID string) *util.APIHandleError {
	tx := db.GetManager().Begin()
	defer func() {
		if r := recover(); r != nil {
			logrus.Errorf("Unexpected panic occurred, rollback transaction: %v", r)
			tx.Rollback()
		}
	}()
	deleteFunclist := []func(serviceID, pluginID string) error{
		db.GetManager().TenantEnvServicePluginRelationDaoTransactions(tx).DeleteRelationByServiceIDAndPluginID,
		db.GetManager().TenantEnvPluginVersionENVDaoTransactions(tx).DeleteEnvByPluginID,
		db.GetManager().TenantEnvPluginVersionConfigDaoTransactions(tx).DeletePluginConfig,
	}
	for _, del := range deleteFunclist {
		if err := del(serviceID, pluginID); err != nil {
			if err != gorm.ErrRecordNotFound {
				tx.Rollback()
				return util.CreateAPIHandleErrorFromDBError("delete plugin relation", err)
			}
		}
	}
	if err := s.deletePluginConfig(nil, serviceID, pluginID); err != nil {
		tx.Rollback()
		return util.CreateAPIHandleErrorFromDBError("delete service plugin config failure", err)
	}
	plugin, _ := db.GetManager().TenantEnvPluginDao().GetPluginByID(pluginID, tenantEnvID)
	if plugin != nil && checkPluginHaveInbound(plugin.PluginModel) {
		if err := db.GetManager().TenantEnvServicesStreamPluginPortDaoTransactions(tx).DeleteAllPluginMappingPortByServiceID(serviceID); err != nil {
			if err != gorm.ErrRecordNotFound {
				tx.Rollback()
				return util.CreateAPIHandleErrorFromDBError("delete upstream plugin mapping port", err)
			}
		}
	}
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return util.CreateAPIHandleErrorFromDBError("commit delete err", err)
	}
	return nil
}

// SetTenantEnvServicePluginRelation SetTenantEnvServicePluginRelation
func (s *ServiceAction) SetTenantEnvServicePluginRelation(tenantEnvID, serviceID string, pss *api_model.PluginSetStruct) (*dbmodel.TenantEnvServicePluginRelation, *util.APIHandleError) {
	plugin, err := db.GetManager().TenantEnvPluginDao().GetPluginByID(pss.Body.PluginID, tenantEnvID)
	if err != nil {
		return nil, util.CreateAPIHandleErrorFromDBError("get plugin by plugin id", err)
	}
	// crt, err := db.GetManager().TenantEnvServicePluginRelationDao().CheckSomeModelLikePluginByServiceID(
	// 	serviceID,
	// 	plugin.PluginModel,
	// )
	// if err != nil {
	// 	return nil, util.CreateAPIHandleErrorFromDBError("check plugin model", err)
	// }
	// if crt {
	// 	return nil, util.CreateAPIHandleError(400, fmt.Errorf("can not add this kind plugin, a same kind plugin has been linked"))
	// }

	crt, err := db.GetManager().TenantEnvServicePluginRelationDao().CheckPluginBeforeInstall(
		serviceID,
		plugin.PluginModel,
	)
	if err != nil {
		return nil, util.CreateAPIHandleErrorFromDBError("check plugin model", err)
	}
	if !crt {
		return nil, util.CreateAPIHandleError(400, fmt.Errorf("can not add this kind plugin, a same kind plugin has been linked"))
	}
	pluginversion, err := db.GetManager().TenantEnvPluginBuildVersionDao().GetBuildVersionByVersionID(plugin.PluginID, pss.Body.VersionID)
	if err != nil {
		return nil, util.CreateAPIHandleErrorFromDBError("plugin version get error ", err)
	}
	var openPorts = make(map[int]bool)
	if checkPluginHaveInbound(plugin.PluginModel) {
		ports, err := db.GetManager().TenantEnvServicesPortDao().GetPortsByServiceID(serviceID)
		if err != nil {
			return nil, util.CreateAPIHandleErrorFromDBError("get ports by service id", err)
		}
		for _, p := range ports {
			if *p.IsInnerService || *p.IsOuterService {
				openPorts[p.ContainerPort] = true
			}
		}
	}
	tx := db.GetManager().Begin()
	defer func() {
		if r := recover(); r != nil {
			logrus.Errorf("Unexpected panic occurred, rollback transaction: %v", r)
			tx.Rollback()
		}
	}()
	if configs := pss.Body.ConfigEnvs.ComplexEnvs; configs != nil {
		if configs.BasePorts != nil && checkPluginHaveInbound(plugin.PluginModel) {
			for _, p := range configs.BasePorts {
				pluginPort, err := db.GetManager().TenantEnvServicesStreamPluginPortDaoTransactions(tx).SetPluginMappingPort(
					tenantEnvID,
					serviceID,
					dbmodel.InBoundNetPlugin,
					p.Port,
				)
				if err != nil {
					tx.Rollback()
					logrus.Errorf(fmt.Sprintf("set upstream port %d error, %v", p.Port, err))
					return nil, util.CreateAPIHandleErrorFromDBError(
						fmt.Sprintf("set upstream port %d error ", p.Port),
						err,
					)
				}
				logrus.Debugf("set plugin upstream port %d->%d", p.Port, pluginPort)
				p.ListenPort = pluginPort
			}
		}
		if err := s.SavePluginConfig(serviceID, plugin.PluginID, pss.Body.ConfigEnvs.ComplexEnvs); err != nil {
			tx.Rollback()
			return nil, util.CreateAPIHandleError(500, fmt.Errorf("set complex error, %v", err))
		}
	}
	if err := s.normalEnvs(tx, serviceID, plugin.PluginID, pss.Body.ConfigEnvs.NormalEnvs); err != nil {
		tx.Rollback()
		return nil, util.CreateAPIHandleErrorFromDBError("set service plugin env error ", err)
	}
	tsprCPU := pluginversion.ContainerCPU
	tsprMemory := pluginversion.ContainerMemory
	if pss.Body.PluginCPU >= 0 {
		tsprCPU = pss.Body.PluginCPU
	}
	if pss.Body.PluginMemory >= 0 {
		tsprMemory = pss.Body.PluginMemory
	}
	relation := &dbmodel.TenantEnvServicePluginRelation{
		VersionID:       pss.Body.VersionID,
		ServiceID:       serviceID,
		PluginID:        pss.Body.PluginID,
		Switch:          pss.Body.Switch,
		PluginModel:     plugin.PluginModel,
		ContainerCPU:    tsprCPU,
		ContainerMemory: tsprMemory,
	}
	if err := db.GetManager().TenantEnvServicePluginRelationDaoTransactions(tx).AddModel(relation); err != nil {
		tx.Rollback()
		return nil, util.CreateAPIHandleErrorFromDBError("set service plugin relation", err)
	}
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return nil, util.CreateAPIHandleErrorFromDBError("commit set service plugin relation", err)
	}
	return relation, nil
}

// UpdateTenantEnvServicePluginRelation UpdateTenantEnvServicePluginRelation
func (s *ServiceAction) UpdateTenantEnvServicePluginRelation(serviceID string, pss *api_model.PluginSetStruct) (*dbmodel.TenantEnvServicePluginRelation, *util.APIHandleError) {
	relation, err := db.GetManager().TenantEnvServicePluginRelationDao().GetRelateionByServiceIDAndPluginID(serviceID, pss.Body.PluginID)
	if err != nil {
		return nil, util.CreateAPIHandleErrorFromDBError("get relation by serviceid and pluginid", err)
	}
	relation.VersionID = pss.Body.VersionID
	relation.Switch = pss.Body.Switch
	if pss.Body.PluginCPU >= 0 {
		relation.ContainerCPU = pss.Body.PluginCPU
	}
	if pss.Body.PluginMemory >= 0 {
		relation.ContainerMemory = pss.Body.PluginMemory
	}
	err = db.GetManager().TenantEnvServicePluginRelationDao().UpdateModel(relation)
	if err != nil {
		return nil, util.CreateAPIHandleErrorFromDBError("update relation between plugin and service", err)
	}
	return relation, nil
}

func (s *ServiceAction) normalEnvs(tx *gorm.DB, serviceID, pluginID string, envs []*api_model.VersionEnv) error {
	for _, env := range envs {
		tpv := &dbmodel.TenantEnvPluginVersionEnv{
			PluginID:  pluginID,
			ServiceID: serviceID,
			EnvName:   env.EnvName,
			EnvValue:  env.EnvValue,
		}
		if err := db.GetManager().TenantEnvPluginVersionENVDaoTransactions(tx).AddModel(tpv); err != nil {
			return err
		}
	}
	return nil
}
func checkPluginHaveInbound(model string) bool {
	return model == dbmodel.InBoundNetPlugin || model == dbmodel.InBoundAndOutBoundNetPlugin
}

// UpdateVersionEnv UpdateVersionEnv
func (s *ServiceAction) UpdateVersionEnv(uve *api_model.SetVersionEnv) *util.APIHandleError {
	plugin, err := db.GetManager().TenantEnvPluginDao().GetPluginByID(uve.PluginID, uve.Body.TenantEnvID)
	if err != nil {
		return util.CreateAPIHandleErrorFromDBError("get plugin by plugin id", err)
	}
	tx := db.GetManager().Begin()
	defer func() {
		if r := recover(); r != nil {
			logrus.Errorf("Unexpected panic occurred, rollback transaction: %v", r)
			tx.Rollback()
		}
	}()
	if len(uve.Body.ConfigEnvs.NormalEnvs) != 0 {
		if err := s.upNormalEnvs(tx, uve.Body.ServiceID, uve.PluginID, uve.Body.ConfigEnvs.NormalEnvs); err != nil {
			tx.Rollback()
			return util.CreateAPIHandleErrorFromDBError("update version env", err)
		}
	}
	if uve.Body.ConfigEnvs.ComplexEnvs != nil {
		if uve.Body.ConfigEnvs.ComplexEnvs.BasePorts != nil && checkPluginHaveInbound(plugin.PluginModel) {
			for _, p := range uve.Body.ConfigEnvs.ComplexEnvs.BasePorts {
				pluginPort, err := db.GetManager().TenantEnvServicesStreamPluginPortDaoTransactions(tx).SetPluginMappingPort(
					uve.Body.TenantEnvID,
					uve.Body.ServiceID,
					dbmodel.InBoundNetPlugin,
					p.Port,
				)
				if err != nil {
					tx.Rollback()
					logrus.Errorf(fmt.Sprintf("set upstream port %d error, %v", p.Port, err))
					return util.CreateAPIHandleErrorFromDBError(
						fmt.Sprintf("set upstream port %d error ", p.Port),
						err,
					)
				}
				logrus.Debugf("set plugin upstream port %d->%d", p.Port, pluginPort)
				p.ListenPort = pluginPort
			}
		}
		if err := s.SavePluginConfig(uve.Body.ServiceID, uve.PluginID, uve.Body.ConfigEnvs.ComplexEnvs); err != nil {
			tx.Rollback()
			return util.CreateAPIHandleError(500, fmt.Errorf("update complex error, %v", err))
		}
	}
	// if err := s.upNormalEnvs(tx, uve.Body.ServiceID, uve.PluginID, uve.Body.ConfigEnvs.NormalEnvs); err != nil {
	// 	tx.Rollback()
	// 	return util.CreateAPIHandleError(500, fmt.Errorf("update env config error, %v", err))
	// }
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return util.CreateAPIHandleErrorFromDBError("commit set service plugin env", err)
	}
	return nil
}

// UpdateComponentPluginConfig 更新组件插件配置
func (s *ServiceAction) UpdateComponentPluginConfig(req *api_model.UpdateComponentPluginConfigRequest) *util.APIHandleError {
	plugin, err := db.GetManager().TenantEnvPluginDao().GetPluginByID(req.Body.PluginID, req.Body.TenantEnvID)
	if err != nil {
		return util.CreateAPIHandleErrorFromDBError("get plugin by plugin id", err)
	}
	relation, err := db.GetManager().TenantEnvServicePluginRelationDao().GetRelateionByServiceIDAndPluginID(req.Body.ServiceID, req.Body.PluginID)
	if err != nil {
		return util.CreateAPIHandleErrorFromDBError("get relation by serviceid and pluginid", err)
	}

	tx := db.GetManager().Begin()
	defer func() {
		if r := recover(); r != nil {
			logrus.Errorf("Unexpected panic occurred, rollback transaction: %v", r)
			tx.Rollback()
		}
	}()
	if len(req.Body.ConfigEnvs.NormalEnvs) != 0 {
		if err := s.upNormalEnvs(tx, req.Body.ServiceID, req.Body.PluginID, req.Body.ConfigEnvs.NormalEnvs); err != nil {
			tx.Rollback()
			return util.CreateAPIHandleErrorFromDBError("update version env", err)
		}
	}
	if req.Body.ConfigEnvs.ComplexEnvs != nil {
		if req.Body.ConfigEnvs.ComplexEnvs.BasePorts != nil && checkPluginHaveInbound(plugin.PluginModel) {
			for _, p := range req.Body.ConfigEnvs.ComplexEnvs.BasePorts {
				pluginPort, err := db.GetManager().TenantEnvServicesStreamPluginPortDaoTransactions(tx).SetPluginMappingPort(
					req.Body.TenantEnvID,
					req.Body.ServiceID,
					dbmodel.InBoundNetPlugin,
					p.Port,
				)
				if err != nil {
					tx.Rollback()
					logrus.Errorf(fmt.Sprintf("set upstream port %d error, %v", p.Port, err))
					return util.CreateAPIHandleErrorFromDBError(
						fmt.Sprintf("set upstream port %d error ", p.Port),
						err,
					)
				}
				logrus.Debugf("set plugin upstream port %d->%d", p.Port, pluginPort)
				p.ListenPort = pluginPort
			}
		}
		if err := s.SavePluginConfig(req.Body.ServiceID, req.Body.PluginID, req.Body.ConfigEnvs.ComplexEnvs); err != nil {
			tx.Rollback()
			return util.CreateAPIHandleError(500, fmt.Errorf("update complex error, %v", err))
		}
	}

	if req.Body.PluginCPU >= 0 {
		relation.ContainerCPU = req.Body.PluginCPU
	}
	if req.Body.PluginMemory >= 0 {
		relation.ContainerMemory = req.Body.PluginMemory
	}
	err = db.GetManager().TenantEnvServicePluginRelationDao().UpdateModel(relation)
	if err != nil {
		return util.CreateAPIHandleErrorFromDBError("update relation between plugin and service", err)
	}

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return util.CreateAPIHandleErrorFromDBError("commit set service plugin env", err)
	}
	return nil
}

// ToggleComponentPlugin 启用/停用组件插件
func (s *ServiceAction) ToggleComponentPlugin(req *api_model.ToggleComponentPluginRequest) *util.APIHandleError {
	relation, err := db.GetManager().TenantEnvServicePluginRelationDao().GetRelateionByServiceIDAndPluginID(req.Body.ServiceID, req.Body.PluginID)
	if err != nil {
		return util.CreateAPIHandleErrorFromDBError("get relation by component id and plugin id", err)
	}
	relation.Switch = req.Body.Switch
	err = db.GetManager().TenantEnvServicePluginRelationDao().UpdateModel(relation)
	if err != nil {
		return util.CreateAPIHandleErrorFromDBError("update relation between plugin and service", err)
	}
	return nil
}

func (s *ServiceAction) upNormalEnvs(tx *gorm.DB, componentId, pluginId string, normalEnvs []*api_model.VersionEnv) *util.APIHandleError {
	err := db.GetManager().TenantEnvPluginVersionENVDaoTransactions(tx).DeleteEnvByPluginID(componentId, pluginId)
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			return util.CreateAPIHandleErrorFromDBError("delete version env", err)
		}
	}
	if err := s.normalEnvs(tx, componentId, pluginId, normalEnvs); err != nil {
		return util.CreateAPIHandleErrorFromDBError("update version env", err)
	}
	return nil
}

// SavePluginConfig save plugin dynamic discovery config
func (s *ServiceAction) SavePluginConfig(serviceID, pluginID string, config *api_model.ResourceSpec) *util.APIHandleError {
	if config == nil {
		return nil
	}
	v, err := ffjson.Marshal(config)
	if err != nil {
		logrus.Errorf("mashal plugin config value error, %v", err)
		return util.CreateAPIHandleError(500, err)
	}
	if err := db.GetManager().TenantEnvPluginVersionConfigDao().AddModel(&dbmodel.TenantEnvPluginVersionDiscoverConfig{
		PluginID:  pluginID,
		ServiceID: serviceID,
		ConfigStr: string(v),
	}); err != nil {
		return util.CreateAPIHandleErrorFromDBError("save plugin config failure", err)
	}
	//push message to worker
	TaskBody := model.ApplyPluginConfigTaskBody{
		ServiceID: serviceID,
		PluginID:  pluginID,
		EventID:   "system",
		Action:    "put",
	}
	err = s.MQClient.SendBuilderTopic(gclient.TaskStruct{
		TaskType: "apply_plugin_config",
		TaskBody: TaskBody,
		Topic:    gclient.WorkerTopic,
	})
	if err != nil {
		logrus.Errorf("equque mq error, %v", err)
		// not return error
		return nil
	}
	return nil
}

// DeletePluginConfig delete service plugin dynamic discovery config
func (s *ServiceAction) DeletePluginConfig(serviceID, pluginID string) *util.APIHandleError {
	tx := db.GetManager().Begin()
	err := s.deletePluginConfig(tx, serviceID, pluginID)
	if err != nil {
		tx.Rollback()
		logrus.Errorf("equque mq error, %v", err)
		return util.CreateAPIHandleErrorf(500, "send apply plugin config message failure")
	}
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return util.CreateAPIHandleErrorFromDBError("delete plugin config failure", err)
	}
	return nil
}

// DeletePluginConfig delete service plugin dynamic discovery config
func (s *ServiceAction) deletePluginConfig(tx *gorm.DB, serviceID, pluginID string) *util.APIHandleError {
	if tx != nil {
		if err := db.GetManager().TenantEnvPluginVersionConfigDaoTransactions(tx).DeletePluginConfig(serviceID, pluginID); err != nil {
			return util.CreateAPIHandleErrorFromDBError("delete plugin config failure", err)
		}
	}
	//push message to worker
	TaskBody := model.ApplyPluginConfigTaskBody{
		ServiceID: serviceID,
		PluginID:  pluginID,
		EventID:   "system",
		Action:    "delete",
	}
	err := s.MQClient.SendBuilderTopic(gclient.TaskStruct{
		TaskType: "apply_plugin_config",
		TaskBody: TaskBody,
		Topic:    gclient.WorkerTopic,
	})
	if err != nil {
		logrus.Errorf("equque mq error, %v", err)
		return util.CreateAPIHandleErrorf(500, "send apply plugin config message failure")
	}
	return nil
}
