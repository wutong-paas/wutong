package controller

import (
	"fmt"
	"net/http"
	"strconv"

	k8svalidation "k8s.io/apimachinery/pkg/util/validation"

	"github.com/go-chi/chi"
	"github.com/sirupsen/logrus"
	"github.com/wutong-paas/wutong/api/handler"
	"github.com/wutong-paas/wutong/api/model"
	"github.com/wutong-paas/wutong/api/util/bcode"
	ctxutil "github.com/wutong-paas/wutong/api/util/ctx"
	dbmodel "github.com/wutong-paas/wutong/db/model"
	httputil "github.com/wutong-paas/wutong/util/http"
)

// ApplicationController -
type ApplicationController struct{}

// CreateApp -
func (a *ApplicationController) CreateApp(w http.ResponseWriter, r *http.Request) {
	var tenantEnvReq model.Application
	if !httputil.ValidatorRequestStructAndErrorResponse(r, w, &tenantEnvReq, nil) {
		return
	}
	if tenantEnvReq.AppType == model.AppTypeHelm {
		if tenantEnvReq.AppStoreName == "" {
			httputil.ReturnBcodeError(r, w, bcode.NewBadRequest("the field 'app_tore_name' is required"))
			return
		}
		if tenantEnvReq.AppTemplateName == "" {
			httputil.ReturnBcodeError(r, w, bcode.NewBadRequest("the field 'app_template_name' is required"))
			return
		}
		if tenantEnvReq.AppName == "" {
			httputil.ReturnBcodeError(r, w, bcode.NewBadRequest("the field 'helm_app_name' is required"))
			return
		}
		if tenantEnvReq.Version == "" {
			httputil.ReturnBcodeError(r, w, bcode.NewBadRequest("the field 'version' is required"))
			return
		}
		tenantEnvReq.K8sApp = tenantEnvReq.AppTemplateName
	}
	if tenantEnvReq.K8sApp != "" {
		if len(k8svalidation.IsQualifiedName(tenantEnvReq.K8sApp)) > 0 {
			httputil.ReturnBcodeError(r, w, bcode.ErrInvaildK8sApp)
			return
		}
	}
	// get current tenantEnv
	tenantEnv := r.Context().Value(ctxutil.ContextKey("tenantEnv")).(*dbmodel.TenantEnvs)
	tenantEnvReq.TenantEnvID = tenantEnv.UUID
	// create app
	app, err := handler.GetApplicationHandler().CreateApp(r.Context(), &tenantEnvReq)
	if err != nil {
		logrus.Errorf("create app: %+v", err)
		httputil.ReturnBcodeError(r, w, err)
		return
	}

	httputil.ReturnSuccess(r, w, app)
}

// BatchCreateApp -
func (a *ApplicationController) BatchCreateApp(w http.ResponseWriter, r *http.Request) {
	var apps model.CreateAppRequest
	if !httputil.ValidatorRequestStructAndErrorResponse(r, w, &apps, nil) {
		return
	}

	// get current tenantEnv
	tenantEnv := r.Context().Value(ctxutil.ContextKey("tenantEnv")).(*dbmodel.TenantEnvs)
	respList, err := handler.GetApplicationHandler().BatchCreateApp(r.Context(), &apps, tenantEnv.UUID)
	if err != nil {
		httputil.ReturnBcodeError(r, w, err)
		return
	}
	httputil.ReturnSuccess(r, w, respList)
}

// UpdateApp -
func (a *ApplicationController) UpdateApp(w http.ResponseWriter, r *http.Request) {
	var updateAppReq model.UpdateAppRequest
	if !httputil.ValidatorRequestStructAndErrorResponse(r, w, &updateAppReq, nil) {
		return
	}
	app := r.Context().Value(ctxutil.ContextKey("application")).(*dbmodel.Application)
	if updateAppReq.K8sApp != "" && len(k8svalidation.IsQualifiedName(updateAppReq.K8sApp)) > 0 {
		httputil.ReturnBcodeError(r, w, bcode.ErrInvaildK8sApp)
		return
	}
	if updateAppReq.K8sApp == "" {
		updateAppReq.K8sApp = fmt.Sprintf("app-%s", app.AppID[:8])
		if app.K8sApp != "" {
			updateAppReq.K8sApp = app.K8sApp
		}
	}
	// update app
	app, err := handler.GetApplicationHandler().UpdateApp(r.Context(), app, updateAppReq)
	if err != nil {
		httputil.ReturnBcodeError(r, w, err)
		return
	}

	httputil.ReturnSuccess(r, w, app)
}

// ListApps -
func (a *ApplicationController) ListApps(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	appName := query.Get("app_name")
	pageQuery := query.Get("page")
	pageSizeQuery := query.Get("pageSize")

	page, _ := strconv.Atoi(pageQuery)
	if page == 0 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(pageSizeQuery)
	if pageSize == 0 {
		pageSize = 10
	}

	// get current tenantEnvID
	tenantEnvID := r.Context().Value(ctxutil.ContextKey("tenant_env_id")).(string)

	// List apps
	resp, err := handler.GetApplicationHandler().ListApps(tenantEnvID, appName, page, pageSize)
	if err != nil {
		httputil.ReturnBcodeError(r, w, err)
		return
	}

	httputil.ReturnSuccess(r, w, resp)
}

// ListComponents -
func (a *ApplicationController) ListComponents(w http.ResponseWriter, r *http.Request) {
	appID := chi.URLParam(r, "app_id")
	query := r.URL.Query()
	pageQuery := query.Get("page")
	pageSizeQuery := query.Get("pageSize")

	page, _ := strconv.Atoi(pageQuery)
	if page == 0 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(pageSizeQuery)
	if pageSize == 0 {
		pageSize = 10
	}

	// List services
	resp, err := handler.GetServiceManager().GetServicesByAppID(appID, page, pageSize)
	if err != nil {
		httputil.ReturnBcodeError(r, w, err)
		return
	}

	httputil.ReturnSuccess(r, w, resp)
}

// DeleteApp -
func (a *ApplicationController) DeleteApp(w http.ResponseWriter, r *http.Request) {
	app := r.Context().Value(ctxutil.ContextKey("application")).(*dbmodel.Application)

	var req model.EtcdCleanReq
	if httputil.ValidatorRequestStructAndErrorResponse(r, w, &req, nil) {
		logrus.Debugf("delete app etcd keys : %+v", req.Keys)
		handler.GetEtcdHandler().CleanAllServiceData(req.Keys)
	}
	// Delete application
	err := handler.GetApplicationHandler().DeleteApp(r.Context(), app)
	if err != nil {
		httputil.ReturnBcodeError(r, w, err)
		return
	}
	httputil.ReturnSuccess(r, w, nil)
}

// BatchUpdateComponentPorts update component ports in batch.
func (a *ApplicationController) BatchUpdateComponentPorts(w http.ResponseWriter, r *http.Request) {
	var appPorts []*model.AppPort
	if err := httputil.ReadEntity(r, &appPorts); err != nil {
		httputil.ReturnBcodeError(r, w, err)
		return
	}
	for _, port := range appPorts {
		if err := httputil.ValidateStruct(port); err != nil {
			httputil.ReturnBcodeError(r, w, err)
			return
		}
	}

	appID := r.Context().Value(ctxutil.ContextKey("app_id")).(string)

	if err := handler.GetApplicationHandler().BatchUpdateComponentPorts(appID, appPorts); err != nil {
		httputil.ReturnBcodeError(r, w, err)
		return
	}

	httputil.ReturnSuccess(r, w, nil)
}

// GetAppStatus returns the status of the application.
func (a *ApplicationController) GetAppStatus(w http.ResponseWriter, r *http.Request) {
	app := r.Context().Value(ctxutil.ContextKey("application")).(*dbmodel.Application)

	res, err := handler.GetApplicationHandler().GetStatus(r.Context(), app)
	if err != nil {
		httputil.ReturnBcodeError(r, w, err)
		return
	}

	httputil.ReturnSuccess(r, w, res)
}

// Install installs the application.
func (a *ApplicationController) Install(w http.ResponseWriter, r *http.Request) {
	app := r.Context().Value(ctxutil.ContextKey("application")).(*dbmodel.Application)

	var installAppReq model.InstallAppReq
	if !httputil.ValidatorRequestStructAndErrorResponse(r, w, &installAppReq, nil) {
		return
	}

	if err := handler.GetApplicationHandler().Install(r.Context(), app, installAppReq.Overrides); err != nil {
		httputil.ReturnBcodeError(r, w, err)
		return
	}
}

// ListServices returns the list fo the application.
func (a *ApplicationController) ListServices(w http.ResponseWriter, r *http.Request) {
	app := r.Context().Value(ctxutil.ContextKey("application")).(*dbmodel.Application)

	services, err := handler.GetApplicationHandler().ListServices(r.Context(), app)
	if err != nil {
		httputil.ReturnBcodeError(r, w, err)
		return
	}

	httputil.ReturnSuccess(r, w, services)
}

// BatchBindService -
func (a *ApplicationController) BatchBindService(w http.ResponseWriter, r *http.Request) {
	appID := chi.URLParam(r, "app_id")
	var bindServiceReq model.BindServiceRequest
	if !httputil.ValidatorRequestStructAndErrorResponse(r, w, &bindServiceReq, nil) {
		return
	}

	// bind service
	err := handler.GetApplicationHandler().BatchBindService(appID, bindServiceReq)
	if err != nil {
		httputil.ReturnBcodeError(r, w, err)
		return
	}
	httputil.ReturnSuccess(r, w, nil)
}

// ListHelmAppReleases returns the list of helm releases.
func (a *ApplicationController) ListHelmAppReleases(w http.ResponseWriter, r *http.Request) {
	app := r.Context().Value(ctxutil.ContextKey("application")).(*dbmodel.Application)

	releases, err := handler.GetApplicationHandler().ListHelmAppReleases(r.Context(), app)
	if err != nil {
		httputil.ReturnBcodeError(r, w, err)
		return
	}

	httputil.ReturnSuccess(r, w, releases)
}

// ListAppStatuses returns the status of the applications.
func (a *ApplicationController) ListAppStatuses(w http.ResponseWriter, r *http.Request) {
	var req model.AppStatusesReq
	if !httputil.ValidatorRequestStructAndErrorResponse(r, w, &req, nil) {
		return
	}
	res, err := handler.GetApplicationHandler().ListAppStatuses(r.Context(), req.AppIDs)
	if err != nil {
		httputil.ReturnBcodeError(r, w, err)
		return
	}
	httputil.ReturnSuccess(r, w, res)
}

// CheckGovernanceMode check governance mode.
func (a *ApplicationController) CheckGovernanceMode(w http.ResponseWriter, r *http.Request) {
	governanceMode := r.URL.Query().Get("governance_mode")
	err := handler.GetApplicationHandler().CheckGovernanceMode(r.Context(), governanceMode)
	if err != nil {
		httputil.ReturnBcodeError(r, w, err)
		return
	}
	httputil.ReturnSuccess(r, w, nil)
}

// ChangeVolumes Since the component name supports modification, the storage directory of stateful components will change.
// This interface is used to modify the original directory name to the storage directory that will actually be used.
func (a *ApplicationController) ChangeVolumes(w http.ResponseWriter, r *http.Request) {
	app := r.Context().Value(ctxutil.ContextKey("application")).(*dbmodel.Application)
	err := handler.GetApplicationHandler().ChangeVolumes(app)
	if err != nil {
		httputil.ReturnBcodeError(r, w, err)
		return
	}
	httputil.ReturnSuccess(r, w, nil)
}

// GetApplicationKubeResources get kube resources for application
func (t *TenantEnvStruct) GetApplicationKubeResources(w http.ResponseWriter, r *http.Request) {
	var customSetting model.KubeResourceCustomSetting
	customSetting.Namespace = r.URL.Query().Get("namespace")
	serviceAliases := r.URL.Query()["service_aliases"]
	tenantEnv := r.Context().Value(ctxutil.ContextKey("tenantEnv")).(*dbmodel.TenantEnvs)
	resources := handler.GetApplicationHandler().GetKubeResources(tenantEnv.Namespace, serviceAliases, customSetting)
	httputil.ReturnSuccess(r, w, resources)
}
