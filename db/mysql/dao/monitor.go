package dao

import (
	"github.com/jinzhu/gorm"
	pkgerr "github.com/pkg/errors"
	gormbulkups "github.com/wutong-paas/gorm-bulk-upsert"
	"github.com/wutong-paas/wutong/api/util/bcode"
	"github.com/wutong-paas/wutong/db/model"
)

// TenantEnvServiceMonitorDaoImpl -
type TenantEnvServiceMonitorDaoImpl struct {
	DB *gorm.DB
}

// AddModel create service monitor
func (t *TenantEnvServiceMonitorDaoImpl) AddModel(mo model.Interface) error {
	m := mo.(*model.TenantEnvServiceMonitor)
	var oldTSM model.TenantEnvServiceMonitor
	if ok := t.DB.Where("name = ? and tenant_env_id = ?", m.Name, m.TenantEnvID).Find(&oldTSM).RecordNotFound(); ok {
		if err := t.DB.Create(m).Error; err != nil {
			return err
		}
	} else {
		return bcode.ErrServiceMonitorNameExist
	}
	return nil
}

// UpdateModel update service monitor
func (t *TenantEnvServiceMonitorDaoImpl) UpdateModel(mo model.Interface) error {
	tsm := mo.(*model.TenantEnvServiceMonitor)
	if err := t.DB.Save(tsm).Error; err != nil {
		return err
	}
	return nil
}

// DeleteServiceMonitor delete service monitor
func (t *TenantEnvServiceMonitorDaoImpl) DeleteServiceMonitor(mo *model.TenantEnvServiceMonitor) error {
	if err := t.DB.Delete(mo).Error; err != nil {
		return err
	}
	return nil
}

// DeleteServiceMonitorByServiceID delete service monitor by service id
func (t *TenantEnvServiceMonitorDaoImpl) DeleteServiceMonitorByServiceID(serviceID string) error {
	if err := t.DB.Where("service_id=?", serviceID).Delete(&model.TenantEnvServiceMonitor{}).Error; err != nil {
		return err
	}
	return nil
}

// DeleteByComponentIDs delete service monitor by component ids
func (t *TenantEnvServiceMonitorDaoImpl) DeleteByComponentIDs(componentIDs []string) error {
	return t.DB.Where("service_id in (?)", componentIDs).Delete(&model.TenantEnvServiceMonitor{}).Error
}

// CreateOrUpdateMonitorInBatch -
func (t *TenantEnvServiceMonitorDaoImpl) CreateOrUpdateMonitorInBatch(monitors []*model.TenantEnvServiceMonitor) error {
	var objects []interface{}
	for _, monitor := range monitors {
		objects = append(objects, *monitor)
	}
	if err := gormbulkups.BulkUpsert(t.DB, objects, 2000); err != nil {
		return pkgerr.Wrap(err, "create or update component monitors in batch")
	}
	return nil
}

// GetByServiceID get tsm by service id
func (t *TenantEnvServiceMonitorDaoImpl) GetByServiceID(serviceID string) ([]*model.TenantEnvServiceMonitor, error) {
	var tsm []*model.TenantEnvServiceMonitor
	if err := t.DB.Where("service_id=?", serviceID).Find(&tsm).Error; err != nil {
		return nil, err
	}
	return tsm, nil
}

// GetByName get by name
func (t *TenantEnvServiceMonitorDaoImpl) GetByName(serviceID, name string) (*model.TenantEnvServiceMonitor, error) {
	var tsm model.TenantEnvServiceMonitor
	if err := t.DB.Where("service_id=? and name=?", serviceID, name).Find(&tsm).Error; err != nil {
		return nil, err
	}
	return &tsm, nil
}
