package controller

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	"github.com/sirupsen/logrus"
	"github.com/wutong-paas/wutong/api/handler"
	"github.com/wutong-paas/wutong/api/model"
	ctxutil "github.com/wutong-paas/wutong/api/util/ctx"
	httputil "github.com/wutong-paas/wutong/util/http"
)

// AppRestoreController is an implementation of AppRestoreInterface
type AppRestoreController struct {
}

// RestoreEnvs restores environment variables. delete the existing environment
// variables first, then create the ones in the request body.
func (a *AppRestoreController) RestoreEnvs(w http.ResponseWriter, r *http.Request) {
	var req model.RestoreEnvsReq
	ok := httputil.ValidatorRequestStructAndErrorResponse(r, w, &req, nil)
	if !ok {
		return
	}

	tenantEnvID := r.Context().Value(ctxutil.ContextKey("tenant_env_id")).(string)
	serviceID := r.Context().Value(ctxutil.ContextKey("service_id")).(string)
	err := handler.GetAppRestoreHandler().RestoreEnvs(tenantEnvID, serviceID, &req)
	if err != nil {
		format := "Service ID: %s; failed to restore envs: %v"
		logrus.Errorf(format, serviceID, err)
		httputil.ReturnError(r, w, 500, fmt.Sprintf(format, serviceID, err))
		return
	}

	httputil.ReturnSuccess(r, w, "ok")
}

// RestorePorts restores service ports. delete the existing ports first,
// then create the ones in the request body.
func (a *AppRestoreController) RestorePorts(w http.ResponseWriter, r *http.Request) {
	var req model.RestorePortsReq
	if !httputil.ValidatorRequestStructAndErrorResponse(r, w, &req, nil) {
		return
	}

	tenantEnvID := r.Context().Value(ctxutil.ContextKey("tenant_env_id")).(string)
	serviceID := r.Context().Value(ctxutil.ContextKey("service_id")).(string)
	err := handler.GetAppRestoreHandler().RestorePorts(tenantEnvID, serviceID, &req)
	if err != nil {
		format := "Service ID: %s; failed to restore ports: %v"
		logrus.Errorf(format, serviceID, err)
		httputil.ReturnError(r, w, 500, fmt.Sprintf(format, serviceID, err))
		return
	}

	httputil.ReturnSuccess(r, w, "ok")
}

// RestoreVolumes restores service volumes. delete the existing volumes first,
// then create the ones in the request body.
func (a *AppRestoreController) RestoreVolumes(w http.ResponseWriter, r *http.Request) {
	var req model.RestoreVolumesReq
	if !httputil.ValidatorRequestStructAndErrorResponse(r, w, &req, nil) {
		return
	}

	tenantEnvID := r.Context().Value(ctxutil.ContextKey("tenant_env_id")).(string)
	serviceID := r.Context().Value(ctxutil.ContextKey("service_id")).(string)
	err := handler.GetAppRestoreHandler().RestoreVolumes(tenantEnvID, serviceID, &req)
	if err != nil {
		format := "Service ID: %s; failed to restore volumes: %v"
		logrus.Errorf(format, serviceID, err)
		httputil.ReturnError(r, w, 500, fmt.Sprintf(format, serviceID, err))
		return
	}

	httputil.ReturnSuccess(r, w, "ok")
}

// RestoreProbe restores service probe. delete the existing probe first,
// then create the one in the request body.
func (a *AppRestoreController) RestoreProbe(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		format := "error reading request body: %v"
		httputil.ReturnError(r, w, 500, fmt.Sprintf(format, err))
	}
	// set a new body, which will simulate the same data we read
	r.Body = io.NopCloser(bytes.NewBuffer(body))

	var probeReq *model.ServiceProbe
	if string(body) != "" {
		var req model.ServiceProbe
		if !httputil.ValidatorRequestStructAndErrorResponse(r, w, &req, nil) {
			return
		}
		probeReq = &req
	} else {
		probeReq = nil
	}

	serviceID := r.Context().Value(ctxutil.ContextKey("service_id")).(string)
	if err := handler.GetAppRestoreHandler().RestoreProbe(serviceID, probeReq); err != nil {
		format := "Service ID: %s; failed to restore volumes: %v"
		logrus.Errorf(format, serviceID, err)
		httputil.ReturnError(r, w, 500, fmt.Sprintf(format, serviceID, err))
		return
	}

	httputil.ReturnSuccess(r, w, "ok")
}

// RestoreDeps restores service dependencies. delete the existing dependencies first,
// then create the ones in the request body.
func (a *AppRestoreController) RestoreDeps(w http.ResponseWriter, r *http.Request) {
	var req model.RestoreDepsReq
	if !httputil.ValidatorRequestStructAndErrorResponse(r, w, &req, nil) {
		return
	}

	serviceID := r.Context().Value(ctxutil.ContextKey("service_id")).(string)
	tenantEnvID := r.Context().Value(ctxutil.ContextKey("tenant_env_id")).(string)
	err := handler.GetAppRestoreHandler().RestoreDeps(tenantEnvID, serviceID, &req)
	if err != nil {
		format := "Service ID: %s; failed to restore service dependencies: %v"
		logrus.Errorf(format, serviceID, err)
		httputil.ReturnError(r, w, 500, fmt.Sprintf(format, serviceID, err))
		return
	}

	httputil.ReturnSuccess(r, w, "ok")
}

// RestoreDepVols restores service dependent volumes. delete the existing
// dependent volumes first, then create the ones in the request body.
func (a *AppRestoreController) RestoreDepVols(w http.ResponseWriter, r *http.Request) {
	var req model.RestoreDepVolsReq
	if ok := httputil.ValidatorRequestStructAndErrorResponse(r, w, &req, nil); !ok {
		return
	}

	serviceID := r.Context().Value(ctxutil.ContextKey("service_id")).(string)
	tenantEnvID := r.Context().Value(ctxutil.ContextKey("tenant_env_id")).(string)
	err := handler.GetAppRestoreHandler().RestoreDepVols(tenantEnvID, serviceID, &req)
	if err != nil {
		format := "Service ID: %s; failed to restore volume dependencies: %v"
		logrus.Errorf(format, serviceID, err)
		httputil.ReturnError(r, w, 500, fmt.Sprintf(format, serviceID, err))
		return
	}

	httputil.ReturnSuccess(r, w, nil)
}

// RestorePlugins restores service plugins. delete the existing
// service plugins first, then create the ones in the request body.
func (a *AppRestoreController) RestorePlugins(w http.ResponseWriter, r *http.Request) {
	var req model.RestorePluginsReq
	if ok := httputil.ValidatorRequestStructAndErrorResponse(r, w, &req, nil); !ok {
		return
	}

	tenantEnvID := r.Context().Value(ctxutil.ContextKey("tenant_env_id")).(string)
	serviceID := r.Context().Value(ctxutil.ContextKey("service_id")).(string)
	if err := handler.GetAppRestoreHandler().RestorePlugins(tenantEnvID, serviceID, &req); err != nil {
		format := "Service ID: %s; failed to restore plugins: %v"
		logrus.Errorf(format, serviceID, err)
		httputil.ReturnError(r, w, 500, fmt.Sprintf(format, serviceID, err))
	}
	httputil.ReturnSuccess(r, w, nil)
}
