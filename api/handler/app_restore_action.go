package handler

import (
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
	apimodel "github.com/wutong-paas/wutong/api/model"
	"github.com/wutong-paas/wutong/db"
	"github.com/wutong-paas/wutong/db/errors"
	"github.com/wutong-paas/wutong/db/model"
	"github.com/wutong-paas/wutong/util"
)

// AppRestoreAction is an implementation of AppRestoreHandler
type AppRestoreAction struct {
}

// RestoreEnvs restores environment variables.
func (a *AppRestoreAction) RestoreEnvs(tenantEnvID, serviceID string, req *apimodel.RestoreEnvsReq) error {
	// delete existing env
	tx := db.GetManager().Begin()
	defer func() {
		if r := recover(); r != nil {
			logrus.Errorf("Unexpected panic occurred, rollback transaction: %v", r)
			tx.Rollback()
		}
	}()
	if err := db.GetManager().TenantEnvServiceEnvVarDaoTransactions(tx).DelByServiceIDAndScope(serviceID, req.Scope); err != nil {
		tx.Rollback()
		return err
	}

	// batch create inner envs
	for _, item := range req.Envs {
		env := &model.TenantEnvServiceEnvVar{
			TenantEnvID:   tenantEnvID,
			ServiceID:     serviceID,
			Name:          item.Name,
			AttrName:      item.AttrName,
			AttrValue:     item.AttrValue,
			ContainerPort: item.ContainerPort,
			IsChange:      item.IsChange,
			Scope:         item.Scope,
		}
		if err := db.GetManager().TenantEnvServiceEnvVarDaoTransactions(tx).AddModel(env); err != nil {
			if err == errors.ErrRecordAlreadyExist {
				// ignore record already exist
				logrus.Warningf("Service ID: %s; Attr Name: %s: failed to create env: %v", serviceID, item.AttrName, err)
				continue
			}
			tx.Rollback()
			return err
		}
	}

	return tx.Commit().Error
}

// RestorePorts restores service ports.
func (a *AppRestoreAction) RestorePorts(tenantEnvID, serviceID string, req *apimodel.RestorePortsReq) error {
	// delete existing ports
	tx := db.GetManager().Begin()
	defer func() {
		if r := recover(); r != nil {
			logrus.Errorf("Unexpected panic occurred, rollback transaction: %v", r)
			tx.Rollback()
		}
	}()
	if err := db.GetManager().TenantEnvServicesPortDaoTransactions(tx).DelByServiceID(serviceID); err != nil {
		tx.Rollback()
		return err
	}

	// batch create inner ports
	for _, item := range req.Ports {
		port := &model.TenantEnvServicesPort{}
		port.TenantEnvID = tenantEnvID
		port.ServiceID = serviceID
		port.MappingPort = item.MappingPort
		port.ContainerPort = item.ContainerPort
		port.Protocol = item.Protocol
		port.PortAlias = item.PortAlias
		port.IsInnerService = &item.IsInnerService
		port.IsOuterService = &item.IsOuterService
		if err := db.GetManager().TenantEnvServicesPortDaoTransactions(tx).AddModel(port); err != nil {
			if err == errors.ErrRecordAlreadyExist {
				// ignore record already exist
				logrus.Warningf("Service ID: %s; Container Port: %d: failed to create env: %v", serviceID, item.ContainerPort, err)
				continue
			}
			tx.Rollback()
			return err
		}
	}

	return tx.Commit().Error
}

// RestoreVolumes restores service volumes.
func (a *AppRestoreAction) RestoreVolumes(tenantEnvID, serviceID string, req *apimodel.RestoreVolumesReq) error {
	// delete existing volumes
	tx := db.GetManager().Begin()
	defer func() {
		if r := recover(); r != nil {
			logrus.Errorf("Unexpected panic occurred, rollback transaction: %v", r)
			tx.Rollback()
		}
	}()
	if err := db.GetManager().TenantEnvServiceVolumeDaoTransactions(tx).DelShareableBySID(serviceID); err != nil {
		tx.Rollback()
		return err
	}
	if err := db.GetManager().TenantEnvServiceConfigFileDaoTransactions(tx).DelByServiceID(serviceID); err != nil {
		tx.Rollback()
		return err
	}

	sharePath := os.Getenv("SHARE_DATA_PATH")
	if sharePath == "" {
		sharePath = "/wtdata"
	}
	for k := range req.Volumes {
		item := req.Volumes[k]
		v := &model.TenantEnvServiceVolume{}
		if item.HostPath == "" && item.VolumeType == model.ShareFileVolumeType.String() {
			v.HostPath = fmt.Sprintf("%s/tenantEnv/%s/service/%s%s", sharePath, tenantEnvID, serviceID, item.VolumePath)
		}
		if item.VolumeName == "" {
			item.VolumeName = util.NewUUID()
		}
		if err := db.GetManager().TenantEnvServiceVolumeDaoTransactions(tx).AddModel(v); err != nil {
			tx.Rollback()
			return err
		}
		if item.FileContent == "" {
			continue
		}
		cfg := &model.TenantEnvServiceConfigFile{
			ServiceID:   serviceID,
			VolumeName:  item.VolumeName,
			FileContent: item.FileContent,
		}
		if err := db.GetManager().TenantEnvServiceConfigFileDaoTransactions(tx).AddModel(cfg); err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit().Error
}

// RestoreProbe restores service probe.
func (a *AppRestoreAction) RestoreProbe(serviceID string, req *apimodel.ServiceProbe) error {
	tx := db.GetManager().Begin()
	defer func() {
		if r := recover(); r != nil {
			logrus.Errorf("Unexpected panic occurred, rollback transaction: %v", r)
			tx.Rollback()
		}
	}()
	if err := db.GetManager().ServiceProbeDaoTransactions(tx).DelByServiceID(serviceID); err != nil {
		tx.Rollback()
		return err
	}

	if req != nil {
		probe := &model.TenantEnvServiceProbe{}
		probe.ServiceID = serviceID
		probe.Cmd = req.Cmd
		probe.FailureThreshold = req.FailureThreshold
		probe.HTTPHeader = req.HTTPHeader
		probe.InitialDelaySecond = req.InitialDelaySecond
		probe.IsUsed = &req.IsUsed
		probe.Mode = req.Mode
		probe.Path = req.Path
		probe.PeriodSecond = req.PeriodSecond
		probe.Port = req.Port
		probe.ProbeID = req.ProbeID
		probe.Scheme = req.Scheme
		probe.SuccessThreshold = req.SuccessThreshold
		probe.TimeoutSecond = req.TimeoutSecond
		if err := db.GetManager().ServiceProbeDaoTransactions(tx).AddModel(probe); err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit().Error
}

// RestoreDeps restores service dependencies.
func (a *AppRestoreAction) RestoreDeps(tenantEnvID, serviceID string, req *apimodel.RestoreDepsReq) error {
	tx := db.GetManager().Begin()
	if err := db.GetManager().TenantEnvServiceRelationDaoTransactions(tx).DELRelationsByServiceID(serviceID); err != nil {
		tx.Rollback()
		return err
	}

	for idx := range req.Deps {
		item := req.Deps[idx]
		tsr := &model.TenantEnvServiceRelation{
			TenantEnvID:       tenantEnvID,
			ServiceID:         serviceID,
			DependServiceID:   item.DepServiceID,
			DependServiceType: item.DepServiceType,
			DependOrder:       1,
		}
		if err := db.GetManager().TenantEnvServiceRelationDaoTransactions(tx).AddModel(tsr); err != nil {
			if err == errors.ErrRecordAlreadyExist {
				logrus.Warningf("Service id: %s; Dep service id: %s; failed to create service dependecy: %s",
					serviceID, item.DepServiceID, err)
				continue
			}
			tx.Rollback()
			return err
		}
	}

	return tx.Commit().Error
}

// RestoreDepVols restores service dependent volumes.
func (a *AppRestoreAction) RestoreDepVols(tenantEnvID, serviceID string, req *apimodel.RestoreDepVolsReq) error {
	tx := db.GetManager().Begin()
	defer func() {
		if r := recover(); r != nil {
			logrus.Errorf("Unexpected panic occurred, rollback transaction: %v", r)
			tx.Rollback()
		}
	}()
	if err := db.GetManager().TenantEnvServiceMountRelationDaoTransactions(tx).DELTenantEnvServiceMountRelationByServiceID(serviceID); err != nil {
		tx.Rollback()
		return err
	}

	for idx := range req.DepVols {
		item := req.DepVols[idx]
		dv, err := db.GetManager().TenantEnvServiceVolumeDaoTransactions(tx).GetVolumeByServiceIDAndName(item.DepServiceID, item.VolumeName)
		if err != nil {
			// err contains gorm.ErrRecordNotFound
			tx.Rollback()
			return fmt.Errorf("dep service id: %s; error getting dep volume: %s", item.DepServiceID, err)
		}

		mr := &model.TenantEnvServiceMountRelation{
			TenantEnvID:     tenantEnvID,
			ServiceID:       serviceID,
			DependServiceID: item.DepServiceID,
			VolumePath:      item.VolumePath,
			HostPath:        dv.HostPath,
			VolumeName:      item.VolumeName,
			VolumeType:      dv.VolumeType,
		}
		if err := db.GetManager().TenantEnvServiceMountRelationDaoTransactions(tx).AddModel(mr); err != nil {
			if err == errors.ErrRecordAlreadyExist {
				logrus.Warningf("Service id: %s; Dep service id: %s; failed to create dep volume: %s",
					serviceID, item.DepServiceID, err)
				continue
			}
			tx.Rollback()
			return err
		}
	}

	return tx.Commit().Error
}

// RestorePlugins restores service plugins.
func (a *AppRestoreAction) RestorePlugins(tenantEnvID, serviceID string, req *apimodel.RestorePluginsReq) error {
	tx := db.GetManager().Begin()
	if err := db.GetManager().TenantEnvServicePluginRelationDaoTransactions(tx).DeleteALLRelationByServiceID(serviceID); err != nil {
		tx.Rollback()
		return err
	}

	for idx := range req.Plugins {
		item := req.Plugins[idx]
		plugin, err := db.GetManager().TenantEnvPluginDaoTransactions(tx).GetPluginByID(item.PluginID, tenantEnvID)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("plugin id: %s; failed to get plugin: %v", item.PluginID, err)
		}
		pluginversion, err := db.GetManager().TenantEnvPluginBuildVersionDaoTransactions(tx).GetBuildVersionByVersionID(item.PluginID, item.VersionID)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("plugin id: %s; version id: %s; failed to get plugin version: %v", item.PluginID, item.VersionID, err)
		}

		relation := &model.TenantEnvServicePluginRelation{
			VersionID:       item.VersionID,
			ServiceID:       serviceID,
			PluginID:        item.PluginID,
			Switch:          item.Switch,
			PluginModel:     plugin.PluginModel,
			ContainerCPU:    pluginversion.ContainerCPU,
			ContainerMemory: pluginversion.ContainerMemory,
		}
		if err := db.GetManager().TenantEnvServicePluginRelationDaoTransactions(tx).AddModel(relation); err != nil {
			if err == errors.ErrRecordAlreadyExist {
				logrus.Warningf("failed to create plugin relation: %v", err)
				continue
			}
			tx.Rollback()
			return fmt.Errorf("failed to create plugin relation: %v", err)
		}
	}

	return tx.Commit().Error
}
