package handler

import (
	apimodel "github.com/wutong-paas/wutong/api/model"
)

// AppRestoreHandler defines handler methods to restore app.
// app means market service.
type AppRestoreHandler interface {
	RestoreEnvs(tenantEnvID, serviceID string, req *apimodel.RestoreEnvsReq) error
	RestorePorts(tenantEnvID, serviceID string, req *apimodel.RestorePortsReq) error
	RestoreVolumes(tenantEnvID, serviceID string, req *apimodel.RestoreVolumesReq) error
	RestoreProbe(serviceID string, req *apimodel.ServiceProbe) error
	RestoreDeps(tenantEnvID, serviceID string, req *apimodel.RestoreDepsReq) error
	RestoreDepVols(tenantEnvID, serviceID string, req *apimodel.RestoreDepVolsReq) error
	RestorePlugins(tenantEnvID, serviceID string, req *apimodel.RestorePluginsReq) error
}

// NewAppRestoreHandler creates a new AppRestoreHandler.
func NewAppRestoreHandler() AppRestoreHandler {
	return &AppRestoreAction{}
}
