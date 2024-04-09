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
	"strings"
	"time"

	api_model "github.com/wutong-paas/wutong/api/model"
	"github.com/wutong-paas/wutong/api/util"
	"github.com/wutong-paas/wutong/db"
	dbmodel "github.com/wutong-paas/wutong/db/model"
	"github.com/wutong-paas/wutong/event"
	"github.com/wutong-paas/wutong/mq/client"
	core_util "github.com/wutong-paas/wutong/util"

	builder_model "github.com/wutong-paas/wutong/chaos/model"

	"github.com/sirupsen/logrus"
)

// PluginAction  plugin action struct
type PluginAction struct {
	MQClient client.MQClient
}

// CreatePluginManager get plugin manager
func CreatePluginManager(mqClient client.MQClient) *PluginAction {
	return &PluginAction{
		MQClient: mqClient,
	}
}

// BatchCreatePlugins -
func (p *PluginAction) BatchCreatePlugins(tenantEnvID string, plugins []*api_model.Plugin) *util.APIHandleError {
	var dbPlugins []*dbmodel.TenantEnvPlugin
	for _, plugin := range plugins {
		dbPlugins = append(dbPlugins, plugin.DbModel(tenantEnvID))
	}
	if err := db.GetManager().TenantEnvPluginDao().CreateOrUpdatePluginsInBatch(dbPlugins); err != nil {
		return util.CreateAPIHandleErrorFromDBError("batch create plugins", err)
	}
	return nil
}

// BatchBuildPlugins -
func (p *PluginAction) BatchBuildPlugins(req *api_model.BatchBuildPlugins, tenantEnvID string) *util.APIHandleError {
	var pluginIDs []string
	for _, buildReq := range req.Plugins {
		buildReq.TenantEnvID = tenantEnvID
		pluginIDs = append(pluginIDs, buildReq.PluginID)
	}
	plugins, err := db.GetManager().TenantEnvPluginDao().ListByIDs(pluginIDs)
	if err != nil {
		return util.CreateAPIHandleErrorFromDBError(fmt.Sprintf("get plugin by %v", pluginIDs), err)
	}
	if err := p.batchBuildPlugins(req, plugins); err != nil {
		return util.CreateAPIHandleError(500, fmt.Errorf("build plugin error"))
	}
	return nil
}

// CreatePluginAct PluginAct
func (p *PluginAction) CreatePluginAct(cps *api_model.CreatePluginStruct) *util.APIHandleError {
	tp := &dbmodel.TenantEnvPlugin{
		TenantEnvID: cps.Body.TenantEnvID,
		PluginID:    cps.Body.PluginID,
		PluginInfo:  cps.Body.PluginInfo,
		PluginModel: cps.Body.PluginModel,
		PluginName:  cps.Body.PluginName,
		ImageURL:    cps.Body.ImageURL,
		GitURL:      cps.Body.GitURL,
		BuildModel:  cps.Body.BuildModel,
		Domain:      cps.TenantEnvName,
		PluginType:  cps.Body.PluginType,
	}
	if cps.Body.PluginType == "sys" {
		tp.TenantEnvID = ""
		tp.Domain = ""
	}
	if err := db.GetManager().TenantEnvPluginDao().AddModel(tp); err != nil {
		return util.CreateAPIHandleErrorFromDBError("create plugin", err)
	}
	return nil
}

// UpdatePluginAct UpdatePluginAct
func (p *PluginAction) UpdatePluginAct(pluginID, tenantEnvID string, cps *api_model.UpdatePluginStruct) *util.APIHandleError {
	tp, err := db.GetManager().TenantEnvPluginDao().GetPluginByID(pluginID, tenantEnvID)
	if err != nil {
		return util.CreateAPIHandleErrorFromDBError("get old plugin info", err)
	}
	tp.PluginInfo = cps.Body.PluginInfo
	tp.PluginModel = cps.Body.PluginModel
	tp.PluginName = cps.Body.PluginName
	tp.ImageURL = cps.Body.ImageURL
	tp.GitURL = cps.Body.GitURL
	tp.BuildModel = cps.Body.BuildModel
	err = db.GetManager().TenantEnvPluginDao().UpdateModel(tp)
	if err != nil {
		return util.CreateAPIHandleErrorFromDBError("update plugin", err)
	}
	return nil
}

// DeletePluginAct DeletePluginAct
func (p *PluginAction) DeletePluginAct(pluginID, tenantEnvID string) *util.APIHandleError {
	tx := db.GetManager().Begin()
	defer func() {
		if r := recover(); r != nil {
			logrus.Errorf("Unexpected panic occurred, rollback transaction: %v", r)
			tx.Rollback()
		}
	}()
	//step1: delete service plugin relation
	err := db.GetManager().TenantEnvServicePluginRelationDaoTransactions(tx).DeleteALLRelationByPluginID(pluginID)
	if err != nil {
		tx.Rollback()
		return util.CreateAPIHandleErrorFromDBError("delete plugin relation", err)
	}
	//step2: delete plugin build version
	err = db.GetManager().TenantEnvPluginBuildVersionDaoTransactions(tx).DeleteBuildVersionByPluginID(pluginID)
	if err != nil {
		tx.Rollback()
		return util.CreateAPIHandleErrorFromDBError("delete plugin build version", err)
	}
	//step3: delete plugin
	err = db.GetManager().TenantEnvPluginDaoTransactions(tx).DeletePluginByID(pluginID, tenantEnvID)
	if err != nil {
		tx.Rollback()
		return util.CreateAPIHandleErrorFromDBError("delete plugin", err)
	}
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return util.CreateAPIHandleErrorFromDBError("commit delete plugin transactions", err)
	}
	return nil
}

// GetPlugins get all plugins by tenantEnvID
func (p *PluginAction) GetPlugins(tenantEnvID string) ([]*dbmodel.TenantEnvPlugin, *util.APIHandleError) {
	plugins, err := db.GetManager().TenantEnvPluginDao().GetPluginsByTenantEnvID(tenantEnvID)
	if err != nil {
		return nil, util.CreateAPIHandleErrorFromDBError("get plugins by tenant env id", err)
	}
	return plugins, nil
}

// AddDefaultEnv AddDefaultEnv
func (p *PluginAction) AddDefaultEnv(est *api_model.ENVStruct) *util.APIHandleError {
	tx := db.GetManager().Begin()
	defer func() {
		if r := recover(); r != nil {
			logrus.Errorf("Unexpected panic occurred, rollback transaction: %v", r)
			tx.Rollback()
		}
	}()
	for _, env := range est.Body.EVNInfo {
		vis := &dbmodel.TenantEnvPluginDefaultENV{
			PluginID:  est.PluginID,
			ENVName:   env.ENVName,
			ENVValue:  env.ENVValue,
			IsChange:  env.IsChange,
			VersionID: env.VersionID,
		}
		err := db.GetManager().TenantEnvPluginDefaultENVDaoTransactions(tx).AddModel(vis)
		if err != nil {
			tx.Rollback()
			return util.CreateAPIHandleErrorFromDBError(fmt.Sprintf("add default env %s", env.ENVName), err)
		}
	}
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return util.CreateAPIHandleErrorFromDBError("commit add default env transactions", err)
	}
	return nil
}

// UpdateDefaultEnv UpdateDefaultEnv
func (p *PluginAction) UpdateDefaultEnv(est *api_model.ENVStruct) *util.APIHandleError {
	for _, env := range est.Body.EVNInfo {
		vis := &dbmodel.TenantEnvPluginDefaultENV{
			ENVName:   env.ENVName,
			ENVValue:  env.ENVValue,
			IsChange:  env.IsChange,
			VersionID: env.VersionID,
		}
		err := db.GetManager().TenantEnvPluginDefaultENVDao().UpdateModel(vis)
		if err != nil {
			return util.CreateAPIHandleErrorFromDBError(fmt.Sprintf("update default env %s", env.ENVName), err)
		}
	}
	return nil
}

// DeleteDefaultEnv DeleteDefaultEnv
func (p *PluginAction) DeleteDefaultEnv(pluginID, versionID, name string) *util.APIHandleError {
	if err := db.GetManager().TenantEnvPluginDefaultENVDao().DeleteDefaultENVByName(pluginID, name, versionID); err != nil {
		return util.CreateAPIHandleErrorFromDBError(fmt.Sprintf("delete default env %s", name), err)
	}
	return nil
}

// GetDefaultEnv GetDefaultEnv
func (p *PluginAction) GetDefaultEnv(pluginID, versionID string) ([]*dbmodel.TenantEnvPluginDefaultENV, *util.APIHandleError) {
	envs, err := db.GetManager().TenantEnvPluginDefaultENVDao().GetDefaultENVSByPluginID(pluginID, versionID)
	if err != nil {
		return nil, util.CreateAPIHandleErrorFromDBError("get default env", err)
	}
	return envs, nil
}

// GetEnvsWhichCanBeSet GetEnvsWhichCanBeSet
func (p *PluginAction) GetEnvsWhichCanBeSet(serviceID, pluginID string) (interface{}, *util.APIHandleError) {
	relation, err := db.GetManager().TenantEnvServicePluginRelationDao().GetRelateionByServiceIDAndPluginID(serviceID, pluginID)
	if err != nil {
		return nil, util.CreateAPIHandleErrorFromDBError("get relation", err)
	}
	envs, err := db.GetManager().TenantEnvPluginVersionENVDao().GetVersionEnvByServiceID(serviceID, pluginID)
	if err != nil {
		return nil, util.CreateAPIHandleErrorFromDBError("get envs which can be set", err)
	}
	if len(envs) > 0 {
		return envs, nil
	}
	envD, errD := db.GetManager().TenantEnvPluginDefaultENVDao().GetDefaultEnvWhichCanBeSetByPluginID(pluginID, relation.VersionID)
	if errD != nil {
		return nil, util.CreateAPIHandleErrorFromDBError("get envs which can be set", errD)
	}
	return envD, nil
}

// BuildPluginManual BuildPluginManual
func (p *PluginAction) BuildPluginManual(bps *api_model.BuildPluginStruct) (*dbmodel.TenantEnvPluginBuildVersion, *util.APIHandleError) {
	eventID := bps.Body.EventID
	logger := event.GetManager().GetLogger(eventID)
	defer event.CloseManager()
	plugin, err := db.GetManager().TenantEnvPluginDao().GetPluginByID(bps.PluginID, bps.Body.TenantEnvID)
	if err != nil {
		return nil, util.CreateAPIHandleErrorFromDBError(fmt.Sprintf("get plugin by %v", bps.PluginID), err)
	}
	switch plugin.BuildModel {
	case "image", "dockerfile":
		pbv, err := p.buildPlugin(bps, plugin)
		if err != nil {
			logrus.Errorf("build plugin from %s error: %s", plugin.BuildModel, err.Error())
			logger.Error(fmt.Sprintf("从 %s 构建插件任务发送失败: %s", plugin.BuildModel, err.Error()), map[string]string{"step": "callback", "status": "failure"})
			return nil, util.CreateAPIHandleError(500, fmt.Errorf("build plugin from %s error", plugin.BuildModel))
		}
		logger.Info(fmt.Sprintf("从 %s 构建插件任务发送成功", plugin.BuildModel), map[string]string{"step": "image-plugin", "status": "starting"})
		plugin.ImageURL = bps.Body.BuildImage
		err = db.GetManager().TenantEnvPluginDao().UpdateModel(plugin)
		if err != nil {
			logrus.Error("update tenant env plugin image url error ", err.Error())
		}
		return pbv, nil
	default:
		return nil, util.CreateAPIHandleError(400, fmt.Errorf("unexpect kind"))
	}
}

func (p *PluginAction) checkBuildPluginParam(req interface{}, plugin *dbmodel.TenantEnvPlugin) error {
	if plugin.ImageURL == "" && plugin.BuildModel == "image" {
		return fmt.Errorf("need image url")
	}
	if plugin.GitURL == "" && plugin.BuildModel == "dockerfile" {
		return fmt.Errorf("need git repo url")
	}
	switch value := req.(type) {
	case *api_model.BuildPluginStruct:
		if value.Body.Operator == "" {
			value.Body.Operator = "define"
		}
		if value.Body.BuildVersion == "" {
			return fmt.Errorf("build version can not be empty")
		}
		if value.Body.DeployVersion == "" {
			value.Body.DeployVersion = core_util.CreateVersionByTime()
		}
	case *api_model.BuildPluginReq:
		if value.Operator == "" {
			value.Operator = "define"
		}
		if value.BuildVersion == "" {
			return fmt.Errorf("build version can not be empty")
		}
		if value.DeployVersion == "" {
			value.DeployVersion = core_util.CreateVersionByTime()
		}
	}
	return nil
}

// buildPlugin buildPlugin
func (p *PluginAction) buildPlugin(b *api_model.BuildPluginStruct, plugin *dbmodel.TenantEnvPlugin) (
	*dbmodel.TenantEnvPluginBuildVersion, error) {
	if err := p.checkBuildPluginParam(b, plugin); err != nil {
		return nil, err
	}
	pbv := &dbmodel.TenantEnvPluginBuildVersion{
		VersionID:       b.Body.BuildVersion,
		DeployVersion:   b.Body.DeployVersion,
		PluginID:        b.PluginID,
		Kind:            plugin.BuildModel,
		Repo:            b.Body.RepoURL,
		GitURL:          plugin.GitURL,
		BaseImage:       b.Body.BuildImage,
		ContainerCPU:    b.Body.PluginCPU,
		ContainerMemory: b.Body.PluginMemory,
		ContainerCMD:    b.Body.PluginCMD,
		BuildTime:       time.Now().Format(time.RFC3339),
		Info:            b.Body.Info,
		Status:          "building",
	}

	if plugin.PluginType == api_model.PluginTypeSys {
		pbv.BuildLocalImage = plugin.ImageURL
		pbv.Status = "complete"
	}

	if b.Body.PluginCPU < 0 {
		pbv.ContainerCPU = 125
	}
	if b.Body.PluginMemory < 0 {
		pbv.ContainerMemory = 64
	}
	if err := db.GetManager().TenantEnvPluginBuildVersionDao().AddModel(pbv); err != nil {
		if !strings.Contains(err.Error(), "exist") {
			logrus.Errorf("build plugin error: %s", err.Error())
			return nil, err
		}
	}
	var updateVersion = func() {
		pbv.Status = "failure"
		db.GetManager().TenantEnvPluginBuildVersionDao().UpdateModel(pbv)
	}
	taskBody := &builder_model.BuildPluginTaskBody{
		TenantEnvID:   b.Body.TenantEnvID,
		PluginID:      b.PluginID,
		Operator:      b.Body.Operator,
		DeployVersion: b.Body.DeployVersion,
		ImageURL:      b.Body.BuildImage,
		EventID:       b.Body.EventID,
		Kind:          plugin.BuildModel,
		PluginCMD:     b.Body.PluginCMD,
		PluginCPU:     b.Body.PluginCPU,
		PluginMemory:  b.Body.PluginMemory,
		VersionID:     b.Body.BuildVersion,
		ImageInfo:     b.Body.ImageInfo,
		Repo:          b.Body.RepoURL,
		GitURL:        plugin.GitURL,
		GitUsername:   b.Body.Username,
		GitPassword:   b.Body.Password,
	}
	taskType := "plugin_image_build"
	if plugin.BuildModel == "dockerfile" {
		taskType = "plugin_dockerfile_build"
	}
	err := p.MQClient.SendBuilderTopic(client.TaskStruct{
		TaskType: taskType,
		TaskBody: taskBody,
		Topic:    client.BuilderTopic,
	})
	if err != nil {
		if plugin.PluginType != api_model.PluginTypeSys {
			updateVersion()
		}
		logrus.Errorf("equque mq error, %v", err)
		return nil, err
	}
	logrus.Debugf("equeue mq build plugin from image success")
	return pbv, nil
}

// buildPlugin buildPlugin
func (p *PluginAction) batchBuildPlugins(req *api_model.BatchBuildPlugins, plugins []*dbmodel.TenantEnvPlugin) error {
	reqPluginRel := make(map[string]*dbmodel.TenantEnvPlugin)
	for _, plugin := range plugins {
		reqPluginRel[plugin.PluginID] = plugin
	}
	var errPluginIDs []string
	var pluginBuildVersions []*dbmodel.TenantEnvPluginBuildVersion
	for _, buildReq := range req.Plugins {
		if _, ok := reqPluginRel[buildReq.PluginID]; !ok {
			continue
		}
		if err := p.checkBuildPluginParam(buildReq, reqPluginRel[buildReq.PluginID]); err != nil {
			return err
		}

		pluginBuildVersion := buildReq.DbModel(reqPluginRel[buildReq.PluginID])
		// Create record before build task, or the build task cant not
		// find the record by deploy version
		if err := db.GetManager().TenantEnvPluginBuildVersionDao().AddModel(pluginBuildVersion); err != nil {
			if !strings.Contains(err.Error(), "exist") {
				logrus.Errorf("build plugin error: %s", err.Error())
				return err
			}
		}

		logger := event.GetManager().GetLogger(buildReq.EventID)
		taskBody := &builder_model.BuildPluginTaskBody{
			TenantEnvID:   buildReq.TenantEnvID,
			PluginID:      buildReq.PluginID,
			Operator:      buildReq.Operator,
			DeployVersion: buildReq.DeployVersion,
			ImageURL:      reqPluginRel[buildReq.PluginID].ImageURL,
			EventID:       buildReq.EventID,
			Kind:          reqPluginRel[buildReq.PluginID].BuildModel,
			PluginCMD:     buildReq.PluginCMD,
			PluginCPU:     buildReq.PluginCPU,
			PluginMemory:  buildReq.PluginMemory,
			VersionID:     buildReq.BuildVersion,
			ImageInfo:     buildReq.ImageInfo,
			Repo:          buildReq.RepoURL,
			GitURL:        reqPluginRel[buildReq.PluginID].GitURL,
			GitUsername:   buildReq.Username,
			GitPassword:   buildReq.Password,
		}
		taskType := "plugin_image_build"
		loggerInfo := map[string]string{"step": "image-plugin", "status": "starting"}
		if reqPluginRel[buildReq.PluginID].BuildModel == "dockerfile" {
			taskType = "plugin_dockerfile_build"
			loggerInfo = map[string]string{"step": "dockerfile-plugin", "status": "starting"}
		}
		err := p.MQClient.SendBuilderTopic(client.TaskStruct{
			TaskType: taskType,
			TaskBody: taskBody,
			Topic:    client.BuilderTopic,
		})
		if err != nil {
			if reqPluginRel[buildReq.PluginID].PluginType != api_model.PluginTypeSys {
				pluginBuildVersion.Status = "failure"
			}
			errPluginIDs = append(errPluginIDs, reqPluginRel[buildReq.PluginID].PluginID)
			logrus.Errorf("equque mq error, %v", err)
			logger.Error("构建插件任务发送失败 "+err.Error(), map[string]string{"step": "callback", "status": "failure"})
		} else {
			logger.Info("构建插件任务发送成功 ", loggerInfo)
		}
		pluginBuildVersions = append(pluginBuildVersions, pluginBuildVersion)
		event.CloseManager()
	}

	if err := db.GetManager().TenantEnvPluginBuildVersionDao().CreateOrUpdatePluginBuildVersionsInBatch(pluginBuildVersions); err != nil {
		return err
	}
	if len(errPluginIDs) > 0 {
		logrus.Errorf("send mq build task failed pluginIDs is [%v]", errPluginIDs)
	}
	return nil
}

// GetAllPluginBuildVersions GetAllPluginBuildVersions
func (p *PluginAction) GetAllPluginBuildVersions(pluginID string) ([]*dbmodel.TenantEnvPluginBuildVersion, *util.APIHandleError) {
	versions, err := db.GetManager().TenantEnvPluginBuildVersionDao().GetBuildVersionByPluginID(pluginID)
	if err != nil {
		return nil, util.CreateAPIHandleErrorFromDBError("get all plugin build version", err)
	}
	return versions, nil
}

// GetPluginBuildVersion GetPluginBuildVersion
func (p *PluginAction) GetPluginBuildVersion(pluginID, versionID string) (*dbmodel.TenantEnvPluginBuildVersion, *util.APIHandleError) {
	version, err := db.GetManager().TenantEnvPluginBuildVersionDao().GetBuildVersionByVersionID(pluginID, versionID)
	if err != nil {
		return nil, util.CreateAPIHandleErrorFromDBError(fmt.Sprintf("get plugin build version by id %v", versionID), err)
	}
	if version.Status == "building" {
		//check build whether timeout
		if buildTime, err := time.Parse(time.RFC3339, version.BuildTime); err == nil {
			if buildTime.Add(time.Minute * 5).Before(time.Now()) {
				version.Status = "timeout"
			}
		}
	}
	return version, nil
}

// DeletePluginBuildVersion DeletePluginBuildVersion
func (p *PluginAction) DeletePluginBuildVersion(pluginID, versionID string) *util.APIHandleError {

	tx := db.GetManager().Begin()
	defer func() {
		if r := recover(); r != nil {
			logrus.Errorf("Unexpected panic occurred, rollback transaction: %v", r)
			tx.Rollback()
		}
	}()
	err := db.GetManager().TenantEnvPluginBuildVersionDaoTransactions(tx).DeleteBuildVersionByVersionID(versionID)
	if err != nil {
		tx.Rollback()
		return util.CreateAPIHandleErrorFromDBError(fmt.Sprintf("delete plugin build version by id %v", versionID), err)
	}
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return util.CreateAPIHandleErrorFromDBError("commit delete plugin transactions", err)
	}
	return nil
}
