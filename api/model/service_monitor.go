package model

import dbmodel "github.com/wutong-paas/wutong/db/model"

// AddServiceMonitorRequestStruct add service monitor request
type AddServiceMonitorRequestStruct struct {
	// name
	// in: body
	// required: true
	Name string `json:"name" validate:"name|required"`
	// service_show_name
	// in: body
	// required: true
	ServiceShowName string `json:"service_show_name" validate:"service_show_name|required"`
	// port
	// in: body
	// required: true
	Port int `json:"port" validate:"port|required"`
	// path
	// in: body
	// required: true
	Path string `json:"path" validate:"path|required"`
	// interval
	// in: body
	// required: true
	Interval string `json:"interval" validate:"interval|required"`
}

// DbModel return database model
func (a *AddServiceMonitorRequestStruct) DbModel(tenantEnvID, serviceID string) *dbmodel.TenantEnvServiceMonitor {
	return &dbmodel.TenantEnvServiceMonitor{
		Name:            a.Name,
		TenantEnvID:     tenantEnvID,
		ServiceID:       serviceID,
		ServiceShowName: a.ServiceShowName,
		Port:            a.Port,
		Path:            a.Path,
		Interval:        a.Interval,
	}
}

// UpdateServiceMonitorRequestStruct update service monitor request
type UpdateServiceMonitorRequestStruct struct {
	// service_show_name
	// in: body
	// required: true
	ServiceShowName string `json:"service_show_name" validate:"service_show_name|required"`
	// port
	// in: body
	// required: true
	Port int `json:"port" validate:"port|required"`
	// path
	// in: body
	// required: true
	Path string `json:"path" validate:"path|required"`
	// interval
	// in: body
	// required: true
	Interval string `json:"interval" validate:"interval|required"`
}
