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

	pkgerr "github.com/pkg/errors"
	gormbulkups "github.com/wutong-paas/gorm-bulk-upsert"

	"github.com/wutong-paas/wutong/db/model"

	"github.com/jinzhu/gorm"
)

// ServiceProbeDaoImpl probe dao impl
type ServiceProbeDaoImpl struct {
	DB *gorm.DB
}

// AddModel 添加应用Probe
func (t *ServiceProbeDaoImpl) AddModel(mo model.Interface) error {
	probe := mo.(*model.TenantEnvServiceProbe)
	var oldProbe model.TenantEnvServiceProbe
	if ok := t.DB.Where("service_id=? and mode=?", probe.ServiceID, probe.Mode).Find(&oldProbe).RecordNotFound(); ok {
		if err := t.DB.Create(probe).Error; err != nil {
			return err
		}
	} else {
		return fmt.Errorf("probe mode %s of service %s is exist", probe.Mode, probe.ServiceID)
	}
	return nil
}

// UpdateModel 更新应用Probe
func (t *ServiceProbeDaoImpl) UpdateModel(mo model.Interface) error {
	probe := mo.(*model.TenantEnvServiceProbe)
	if probe.ID == 0 {
		var oldProbe model.TenantEnvServiceProbe
		if err := t.DB.Where("service_id = ? and probe_id=?", probe.ServiceID,
			probe.ProbeID).Find(&oldProbe).Error; err != nil {
			return err
		}
		if oldProbe.ID == 0 {
			return gorm.ErrRecordNotFound
		}
		probe.ID = oldProbe.ID
		probe.CreatedAt = oldProbe.CreatedAt
	}
	return t.DB.Save(probe).Error
}

// DeleteModel 删除应用探针
func (t *ServiceProbeDaoImpl) DeleteModel(serviceID string, args ...interface{}) error {
	probeID := args[0].(string)
	relation := &model.TenantEnvServiceProbe{
		ServiceID: serviceID,
		ProbeID:   probeID,
	}
	if err := t.DB.Where("service_id=? and probe_id=?", serviceID, probeID).Delete(relation).Error; err != nil {
		return err
	}
	return nil
}

// DelByServiceID deletes TenantEnvServiceProbe based on sid(service_id)
func (t *ServiceProbeDaoImpl) DelByServiceID(sid string) error {
	return t.DB.Where("service_id=?", sid).Delete(&model.TenantEnvServiceProbe{}).Error
}

// GetServiceProbes 获取应用探针
func (t *ServiceProbeDaoImpl) GetServiceProbes(serviceID string) ([]*model.TenantEnvServiceProbe, error) {
	var probes []*model.TenantEnvServiceProbe
	if err := t.DB.Where("service_id=?", serviceID).Find(&probes).Error; err != nil {
		return nil, err
	}
	return probes, nil
}

// GetServiceUsedProbe 获取指定模式的可用探针定义
func (t *ServiceProbeDaoImpl) GetServiceUsedProbe(serviceID, mode string) (*model.TenantEnvServiceProbe, error) {
	var probe model.TenantEnvServiceProbe
	if err := t.DB.Where("service_id=? and mode=? and is_used=?", serviceID, mode, 1).Find(&probe).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &probe, nil
}

// DELServiceProbesByServiceID DELServiceProbesByServiceID
func (t *ServiceProbeDaoImpl) DELServiceProbesByServiceID(serviceID string) error {
	probes := &model.TenantEnvServiceProbe{
		ServiceID: serviceID,
	}
	if err := t.DB.Where("service_id=?", serviceID).Delete(probes).Error; err != nil {
		return err
	}
	return nil
}

// DeleteByComponentIDs deletes TenantEnvServiceProbe based on componentIDs
func (t *ServiceProbeDaoImpl) DeleteByComponentIDs(componentIDs []string) error {
	return t.DB.Where("service_id in (?)", componentIDs).Delete(&model.TenantEnvServiceProbe{}).Error
}

// CreateOrUpdateProbesInBatch -
func (t *ServiceProbeDaoImpl) CreateOrUpdateProbesInBatch(probes []*model.TenantEnvServiceProbe) error {
	var objects []interface{}
	for _, probe := range probes {
		objects = append(objects, *probe)
	}
	if err := gormbulkups.BulkUpsert(t.DB, objects, 2000); err != nil {
		return pkgerr.Wrap(err, "create or update probe in batch")
	}
	return nil
}

// LocalSchedulerDaoImpl 本地调度存储mysql实现
type LocalSchedulerDaoImpl struct {
	DB *gorm.DB
}

// AddModel 添加本地调度信息
func (t *LocalSchedulerDaoImpl) AddModel(mo model.Interface) error {
	ls := mo.(*model.LocalScheduler)
	var oldLs model.LocalScheduler
	if ok := t.DB.Where("service_id=? and pod_name=?", ls.ServiceID, ls.PodName).Find(&oldLs).RecordNotFound(); ok {
		if err := t.DB.Create(ls).Error; err != nil {
			return err
		}
	} else {
		return fmt.Errorf("service %s local scheduler of pod  %s is exist", ls.ServiceID, ls.PodName)
	}
	return nil
}

// UpdateModel 更新调度信息
func (t *LocalSchedulerDaoImpl) UpdateModel(mo model.Interface) error {
	ls := mo.(*model.LocalScheduler)
	if ls.ID == 0 {
		return fmt.Errorf("LocalScheduler id can not be empty when update ")
	}
	if err := t.DB.Save(ls).Error; err != nil {
		return err
	}
	return nil
}

// GetLocalScheduler 获取应用本地调度信息
func (t *LocalSchedulerDaoImpl) GetLocalScheduler(serviceID string) ([]*model.LocalScheduler, error) {
	var ls []*model.LocalScheduler
	if err := t.DB.Where("service_id=?", serviceID).Find(&ls).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return ls, nil
}

// ServiceSourceImpl service source
type ServiceSourceImpl struct {
	DB *gorm.DB
}

// AddModel add service source
func (t *ServiceSourceImpl) AddModel(mo model.Interface) error {
	ls := mo.(*model.ServiceSourceConfig)
	var oldLs model.ServiceSourceConfig
	if ok := t.DB.Where("service_id=? and source_type=?", ls.ServiceID, ls.SourceType).Find(&oldLs).RecordNotFound(); ok {
		if err := t.DB.Create(ls).Error; err != nil {
			return err
		}
	} else {
		oldLs.SourceBody = ls.SourceBody
		t.DB.Save(oldLs)
	}
	return nil
}

// UpdateModel update service source
func (t *ServiceSourceImpl) UpdateModel(mo model.Interface) error {
	ls := mo.(*model.LocalScheduler)
	if ls.ID == 0 {
		return fmt.Errorf("ServiceSourceImpl id can not be empty when update ")
	}
	if err := t.DB.Save(ls).Error; err != nil {
		return err
	}
	return nil
}

// GetServiceSource get services source
func (t *ServiceSourceImpl) GetServiceSource(serviceID string) ([]*model.ServiceSourceConfig, error) {
	var serviceSources []*model.ServiceSourceConfig
	if err := t.DB.Where("service_id=?", serviceID).Find(&serviceSources).Error; err != nil {
		return nil, err
	}
	return serviceSources, nil
}
