package handler

import (
	"github.com/jinzhu/gorm"
	api_model "github.com/wutong-paas/wutong/api/model"
	"github.com/wutong-paas/wutong/api/util/bcode"
	"github.com/wutong-paas/wutong/db"
	dbmodel "github.com/wutong-paas/wutong/db/model"
)

// UpdateServiceMonitor update service monitor
func (s *ServiceAction) UpdateServiceMonitor(tenantEnvID, serviceID, name string, update api_model.UpdateServiceMonitorRequestStruct) (*dbmodel.TenantEnvServiceMonitor, error) {
	sm, err := db.GetManager().TenantEnvServiceMonitorDao().GetByName(serviceID, name)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, bcode.ErrServiceMonitorNotFound
		}
		return nil, err
	}
	_, err = db.GetManager().TenantEnvServicesPortDao().GetPort(serviceID, update.Port)
	if err != nil {
		return nil, bcode.ErrPortNotFound
	}
	sm.ServiceShowName = update.ServiceShowName
	sm.Port = update.Port
	sm.Path = update.Path
	sm.Interval = update.Interval
	return sm, db.GetManager().TenantEnvServiceMonitorDao().UpdateModel(sm)
}

// DeleteServiceMonitor delete
func (s *ServiceAction) DeleteServiceMonitor(tenantEnvID, serviceID, name string) (*dbmodel.TenantEnvServiceMonitor, error) {
	sm, err := db.GetManager().TenantEnvServiceMonitorDao().GetByName(serviceID, name)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, bcode.ErrServiceMonitorNotFound
		}
		return nil, err
	}
	return sm, db.GetManager().TenantEnvServiceMonitorDao().DeleteServiceMonitor(sm)
}

// AddServiceMonitor add service monitor
func (s *ServiceAction) AddServiceMonitor(tenantEnvID, serviceID string, add api_model.AddServiceMonitorRequestStruct) (*dbmodel.TenantEnvServiceMonitor, error) {
	_, err := db.GetManager().TenantEnvServicesPortDao().GetPort(serviceID, add.Port)
	if err != nil {
		return nil, bcode.ErrPortNotFound
	}
	sm := dbmodel.TenantEnvServiceMonitor{
		Name:            add.Name,
		TenantEnvID:     tenantEnvID,
		ServiceID:       serviceID,
		ServiceShowName: add.ServiceShowName,
		Port:            add.Port,
		Path:            add.Path,
		Interval:        add.Interval,
	}
	return &sm, db.GetManager().TenantEnvServiceMonitorDao().AddModel(&sm)
}

// SyncComponentMonitors -
func (s *ServiceAction) SyncComponentMonitors(tx *gorm.DB, app *dbmodel.Application, components []*api_model.Component) error {
	var (
		componentIDs []string
		monitors     []*dbmodel.TenantEnvServiceMonitor
	)
	for _, component := range components {
		if component.Monitors == nil {
			continue
		}
		componentIDs = append(componentIDs, component.ComponentBase.ComponentID)
		for _, monitor := range component.Monitors {
			monitors = append(monitors, monitor.DbModel(app.TenantEnvID, component.ComponentBase.ComponentID))
		}
	}
	if err := db.GetManager().TenantEnvServiceMonitorDaoTransactions(tx).DeleteByComponentIDs(componentIDs); err != nil {
		return err
	}
	return db.GetManager().TenantEnvServiceMonitorDaoTransactions(tx).CreateOrUpdateMonitorInBatch(monitors)
}
