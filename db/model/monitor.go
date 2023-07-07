package model

// TenantEnvServiceMonitor custom service monitor
type TenantEnvServiceMonitor struct {
	Model
	TenantEnvID     string `gorm:"column:tenant_env_id;size:40;unique_index:unique_tenant_env_id_name" json:"tenant_env_id"`
	ServiceID       string `gorm:"column:service_id;size:40" json:"service_id"`
	Name            string `gorm:"column:name;size:40;unique_index:unique_tenant_env_id_name" json:"name"`
	ServiceShowName string `gorm:"column:service_show_name" json:"service_show_name"`
	Port            int    `gorm:"column:port;size:5" json:"port"`
	Path            string `gorm:"column:path;size:255" json:"path"`
	Interval        string `gorm:"column:interval;size:20" json:"interval"`
}

// TableName returns table name of TenantEnvServiceMonitor
func (TenantEnvServiceMonitor) TableName() string {
	return "tenant_env_services_monitor"
}
