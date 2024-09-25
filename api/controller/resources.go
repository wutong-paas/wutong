// Copyright (C) 2014-2018 Wutong Co., Ltd.
// WUTONG, Application Management Platform

// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version. For any non-GPL usage of Wutong
// one or multiple Commercial Licenses authorized by Wutong Co., Ltd.
// must be obtained first.

// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.

// You should have received a copy of the GNU General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package controller

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/go-chi/chi"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
	"github.com/wutong-paas/wutong/api/handler"
	api_model "github.com/wutong-paas/wutong/api/model"
	"github.com/wutong-paas/wutong/api/util/bcode"
	ctxutil "github.com/wutong-paas/wutong/api/util/ctx"
	"github.com/wutong-paas/wutong/cmd"
	"github.com/wutong-paas/wutong/db"
	"github.com/wutong-paas/wutong/db/errors"
	dbmodel "github.com/wutong-paas/wutong/db/model"
	mqclient "github.com/wutong-paas/wutong/mq/client"
	validation "github.com/wutong-paas/wutong/util/endpoint"
	"github.com/wutong-paas/wutong/util/fuzzy"
	validator "github.com/wutong-paas/wutong/util/govalidator"
	httputil "github.com/wutong-paas/wutong/util/http"
	"github.com/wutong-paas/wutong/worker/client"
)

// V2Routes v2Routes
type V2Routes struct {
	ClusterController
	NodeController
	SchedulingController
	TenantEnvStruct
	EventLogStruct
	AppStruct
	GatewayStruct
	ThirdPartyServiceController
	LabelController
	AppRestoreController
	PodController
	ApplicationController
	HelmAppsController
	RegistryAuthSecretStruct
}

// Show test
func (v2 *V2Routes) Show(w http.ResponseWriter, r *http.Request) {
	// swagger:operation GET /v2/show v2 getApiVersion
	//
	// 显示当前的api version 信息
	//
	// show api version
	//
	// ---
	// produces:
	// - application/json
	// - application/xml
	// responses:
	//   default:
	//     schema:
	//       "$ref": "#/responses/commandResponse"
	//     description: 统一返回格式
	w.Write([]byte(cmd.GetVersion()))
}

// Health show health status
func (v2 *V2Routes) Health(w http.ResponseWriter, r *http.Request) {
	httputil.ReturnSuccess(r, w, map[string]string{"status": "health", "info": "api service health"})
}

// AlertManagerWebHook -
func (v2 *V2Routes) AlertManagerWebHook(w http.ResponseWriter, r *http.Request) {
	_, err := io.ReadAll(r.Body)
	if err != nil {
		fmt.Println(err)
		httputil.ReturnError(r, w, 400, "")
		return
	}
	httputil.ReturnSuccess(r, w, "")
}

// Version -
func (v2 *V2Routes) Version(w http.ResponseWriter, r *http.Request) {
	httputil.ReturnSuccess(r, w, map[string]string{"version": cmd.GetVersion()})
}

// TenantEnvStruct tenant env struct
type TenantEnvStruct struct {
	StatusCli *client.AppRuntimeSyncClient
	MQClient  mqclient.MQClient
}

// AllTenantEnvResources GetResources
func (t *TenantEnvStruct) AllTenantEnvResources(w http.ResponseWriter, r *http.Request) {
	tenantEnvs, err := handler.GetTenantEnvManager().GetAllTenantEnvs("")
	if err != nil {
		msg := httputil.ResponseBody{
			Msg: fmt.Sprintf("get tenant env error, %v", err),
		}
		httputil.Return(r, w, 500, msg)
	}
	ts := &api_model.TotalStatsInfo{}
	for _, tenantEnv := range tenantEnvs {
		services, err := handler.GetServiceManager().GetService(tenantEnv.UUID)
		if err != nil {
			msg := httputil.ResponseBody{
				Msg: fmt.Sprintf("get service error, %v", err),
			}
			httputil.Return(r, w, 500, msg)
		}
		statsInfo, _ := handler.GetTenantEnvManager().StatsMemCPU(services)
		statsInfo.UUID = tenantEnv.UUID
		ts.Data = append(ts.Data, statsInfo)
	}
	httputil.ReturnSuccess(r, w, ts.Data)
}

// TenantEnvResources TenantEnvResources
func (t *TenantEnvStruct) TenantEnvResources(w http.ResponseWriter, r *http.Request) {
	// swagger:operation POST /v2/resources/tenants/{tenant_name}/envs v2 tenantEnvResources
	//
	// 租户资源使用情况
	//
	// get tenant env resources
	//
	// ---
	// produces:
	// - application/json
	// - application/xml
	//
	// responses:
	//   default:
	//     schema:
	//       "$ref": "#/responses/commandResponse"
	//     description: 统一返回格式
	var tr api_model.TenantEnvResources
	ok := httputil.ValidatorRequestStructAndErrorResponse(r, w, &tr.Body, nil)
	if !ok {
		return
	}

	rep, err := handler.GetTenantEnvManager().GetTenantEnvsResources(r.Context(), &tr)
	if err != nil {
		httputil.ReturnError(r, w, 500, fmt.Sprintf("get resources error, %v", err))
		return
	}
	var re []map[string]interface{}
	for _, v := range rep {
		if v != nil {
			re = append(re, v)
		}
	}
	httputil.ReturnSuccess(r, w, re)
}

// ServiceResources ServiceResources
func (t *TenantEnvStruct) ServiceResources(w http.ResponseWriter, r *http.Request) {
	// swagger:operation POST /v2/resources/services v2 serviceResources
	//
	// 应用资源使用情况
	//
	// get service resources
	//
	// ---
	// produces:
	// - application/json
	// - application/xml
	//
	// responses:
	//   default:
	//     schema:
	//       "$ref": "#/responses/commandResponse"
	//     description: 统一返回格式
	var tr api_model.ServicesResources
	ok := httputil.ValidatorRequestStructAndErrorResponse(r, w, &tr.Body, nil)
	if !ok {
		return
	}
	rep, err := handler.GetTenantEnvManager().GetServicesResources(&tr)
	if err != nil {
		httputil.ReturnError(r, w, 500, fmt.Sprintf("get resources error, %v", err))
		return
	}
	httputil.ReturnSuccess(r, w, rep)
}

// TenantEnvsQuery TenantEnvsQuery
func (t *TenantEnvStruct) TenantEnvsQuery(w http.ResponseWriter, r *http.Request) {
	// swagger:operation GET /v2/tenants/{tenant_name}/envs/query/{tenant_env_name} v2 tenantEnvs
	//
	// 租户带资源列表
	//
	// get tenant env resources
	//
	// ---
	// produces:
	// - application/json
	// - application/xml
	//
	// parameters:
	// - name: tenant_env_name
	//   in: path
	//   description: '123'
	//   required: true
	//   type: string
	//   format: string
	//
	// responses:
	//   default:
	//     schema:
	//       "$ref": "#/responses/commandResponse"
	//     description: 统一返回格式

	tenantName := strings.TrimSpace(chi.URLParam(r, "tenant_name"))
	tenantEnvName := strings.TrimSpace(chi.URLParam(r, "tenant_env_name"))

	rep, err := handler.GetTenantEnvManager().GetTenantEnvsName(tenantName)
	if err != nil {
		httputil.ReturnError(r, w, 500, fmt.Sprintf("get tenant envs names error, %v", err))
		return
	}

	result := fuzzy.Find(tenantEnvName, rep) // [cartwheel wheel]
	httputil.ReturnSuccess(r, w, result)
}

// TenantEnvsGetByName TenantEnvsGetByName
func (t *TenantEnvStruct) TenantEnvsGetByName(w http.ResponseWriter, r *http.Request) {
	// swagger:operation GET /v2/tenants/{tenant_name}/envs/{tenant_env_name}/res v2 tenantEnvs
	//
	// 租户带资源单个
	//
	// get tenant env resources
	//
	// ---
	// produces:
	// - application/json
	// - application/xml
	//
	// parameters:
	// - name: tenant_env_name
	//   in: path
	//   description: '123'
	//   required: true
	//   type: string
	//   format: string
	//
	// responses:
	//   default:
	//     schema:
	//       "$ref": "#/responses/commandResponse"
	//     description: 统一返回格式

	tenantName := strings.TrimSpace(chi.URLParam(r, "tenant_name"))
	tenantEnvName := strings.TrimSpace(chi.URLParam(r, "tenant_env_name"))

	v, err := handler.GetTenantEnvManager().GetTenantEnvsByName(tenantName, tenantEnvName)
	if err != nil {
		httputil.ReturnError(r, w, 404, fmt.Sprintf("get tenant envs names error, %v", err))
		return
	}
	logrus.Infof("query tenant env from db by name %s ,got %v", tenantEnvName, v)

	tenantEnvServiceRes, err := handler.GetServiceManager().GetTenantEnvRes(v.UUID)
	if err != nil {
		httputil.ReturnError(r, w, 500, fmt.Sprintf("get tenant envs service total resources  error, %v", err))
		return
	}
	tenantEnvServiceRes.UUID = v.UUID
	tenantEnvServiceRes.Name = v.Name

	httputil.ReturnSuccess(r, w, tenantEnvServiceRes)
}

// TenantEnvsWithResource TenantEnvsWithResource
func (t *TenantEnvStruct) TenantEnvsWithResource(w http.ResponseWriter, r *http.Request) {
	// swagger:operation GET /v2/resources/tenants/{tenant_name}/envs/res/page/{curPage}/size/{pageLen} v2 PagedTenantEnvResList
	//
	// 租户带资源列表
	//
	// get paged tenant env resources
	//
	// ---
	// produces:
	// - application/json
	// - application/xml
	//
	// parameters:
	// - name: curPage
	//   in: path
	//   description: '123'
	//   required: true
	//   type: string
	//   format: string
	// - name: pageLen
	//   in: path
	//   description: '25'
	//   required: true
	//   type: string
	//   format: string
	//
	// responses:
	//   default:
	//     schema:
	//       "$ref": "#/responses/commandResponse"
	//     description: 统一返回格式

	pageLenStr := strings.TrimSpace(chi.URLParam(r, "pageLen"))
	curPageStr := strings.TrimSpace(chi.URLParam(r, "curPage"))

	pageLen, err := strconv.Atoi(pageLenStr)
	if err != nil {
		httputil.ReturnError(r, w, 400, fmt.Sprintf("bad request, %v", err))
		return
	}
	curPage, err := strconv.Atoi(curPageStr)
	if err != nil {
		httputil.ReturnError(r, w, 400, fmt.Sprintf("bad request, %v", err))
		return
	}
	resource, count, err := handler.GetServiceManager().GetPagedTenantEnvRes((curPage-1)*pageLen, pageLen)
	if err != nil {
		httputil.ReturnError(r, w, 500, fmt.Sprintf("get tenant envs  error, %v", err))
		return
	}
	var ret api_model.PagedTenantEnvResList
	ret.List = resource
	ret.Length = count
	httputil.ReturnSuccess(r, w, ret)
}

// SumTenantEnvs 统计租户数量
func (t *TenantEnvStruct) SumTenantEnvs(w http.ResponseWriter, r *http.Request) {
	// swagger:operation GET /v2/resources/tenants/{tenant_name}/envs/sum v2 sumTenantEnvs
	//
	// 获取租户数量
	//
	// get tenant env resources
	//
	// ---
	// produces:
	// - application/json
	// - application/xml
	//
	// responses:
	//   default:
	//     schema:
	//       "$ref": "#/responses/commandResponse"
	//     description: 统一返回格式

	tenantName := strings.TrimSpace(chi.URLParam(r, "tenant_name"))
	s, err := handler.GetTenantEnvManager().TenantEnvsSum(tenantName)
	if err != nil {
		httputil.ReturnError(r, w, 500, fmt.Sprintf("sum tenant envs error, %v", err))
		return
	}
	rc := make(map[string]int)
	rc["num"] = s
	httputil.ReturnSuccess(r, w, rc)
}

// TenantEnv one tenant env controller
func (t *TenantEnvStruct) TenantEnv(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		t.GetTenantEnv(w, r)
	case "DELETE":
		t.DeleteTenantEnv(w, r)
	case "PUT":
		t.UpdateTenantEnv(w, r)
	}
}

// TenantEnvs TenantEnv
// func (t *TenantEnvStruct) TenantEnvs(w http.ResponseWriter, r *http.Request) {
// 	switch r.Method {
// 	case "POST":
// 		t.AddTenantEnv(w, r)
// 	case "GET":
// 		t.GetTenantEnvs(w, r)
// 	}
// }

// AddTenantEnv AddTenantEnv
func (t *TenantEnvStruct) AddTenantEnv(w http.ResponseWriter, r *http.Request) {
	// swagger:operation POST /v2/tenants/{tenant_name}/envs v2 addTenantEnv
	//
	// 添加租户环境信息
	//
	// add tenant env
	//
	// ---
	// consumes:
	// - application/json
	// - application/x-protobuf
	//
	// produces:
	// - application/json
	// - application/xml
	//
	// responses:
	//   default:
	//     schema:
	//       "$ref": "#/responses/commandResponse"
	//     description: 统一返回格式
	var ts api_model.AddTenantEnvStruct
	ok := httputil.ValidatorRequestStructAndErrorResponse(r, w, &ts.Body, nil)
	if !ok {
		httputil.ReturnError(r, w, 400, "bad request")
		return
	}
	var dbts dbmodel.TenantEnvs
	//新接口
	//TODO:生成tenant_env_id and tenant_env_name
	id, name, errN := handler.GetServiceManager().CreateTenantEnvIDAndName()
	if errN != nil {
		httputil.ReturnError(r, w, 500, fmt.Sprintf("create tenant env error, %v", errN))
		return
	}
	if ts.Body.TenantEnvName == "" {
		dbts.Name = name
	} else {
		dbts.Name = ts.Body.TenantEnvName
		name = ts.Body.TenantEnvName
	}
	if ts.Body.TenantEnvID == "" {
		dbts.UUID = id
	} else {
		dbts.UUID = ts.Body.TenantEnvID
		id = ts.Body.TenantEnvID
	}
	dbts.LimitMemory = ts.Body.LimitMemory
	dbts.Namespace = dbts.UUID
	if ts.Body.Namespace != "" {
		dbts.Namespace = ts.Body.Namespace
	}
	dbts.TenantID = ts.Body.TenantID
	dbts.TenantName = ts.Body.TenantName
	if err := handler.GetServiceManager().CreateTenantEnv(&dbts); err != nil {
		if strings.HasSuffix(err.Error(), "is exist") {
			httputil.ReturnError(r, w, 400, err.Error())
			return
		}
		httputil.ReturnError(r, w, 500, fmt.Sprintf("create tenant env error, %v", err))
		return
	}
	rc := make(map[string]string)
	rc["tenant_env_id"] = id
	rc["tenant_env_name"] = name
	rc["namespace"] = dbts.Namespace
	httputil.ReturnSuccess(r, w, rc)
}

// GetAllTenantEnvs GetAllTenantEnvs
func (t *TenantEnvStruct) GetAllTenantEnvs(w http.ResponseWriter, r *http.Request) {
	// swagger:operation GET /v2/tenants/{tenant_name}/envs v2 getTenantEnvs
	//
	// 获取所有租户环境信息
	//
	// get tenant env
	//
	// ---
	// consumes:
	// - application/json
	// - application/x-protobuf
	//
	// produces:
	// - application/json
	// - application/xml
	//
	// responses:
	//   default:
	//     schema:
	//       "$ref": "#/responses/commandResponse"
	//     description: 统一返回格式
	page, _ := strconv.Atoi(r.FormValue("page"))
	if page == 0 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(r.FormValue("pageSize"))
	if pageSize == 0 {
		pageSize = 10
	}
	queryName := r.FormValue("query")
	var tenantEnvs []*dbmodel.TenantEnvs
	var err error
	tenantEnvs, err = handler.GetTenantEnvManager().GetAllTenantEnvs(queryName)
	if err != nil {
		httputil.ReturnError(r, w, 500, "get tenant env error")
		return
	}
	list := handler.GetTenantEnvManager().BindTenantEnvsResource(tenantEnvs)
	re := list.Paging(page, pageSize)
	httputil.ReturnSuccess(r, w, re)
}

// GetTenantEnvs GetTenantEnvs
func (t *TenantEnvStruct) GetTenantEnvs(w http.ResponseWriter, r *http.Request) {
	// swagger:operation GET /v2/tenants/{tenant_name}/envs v2 getTenantEnvs
	//
	// 获取所有租户环境信息
	//
	// get tenant env
	//
	// ---
	// consumes:
	// - application/json
	// - application/x-protobuf
	//
	// produces:
	// - application/json
	// - application/xml
	//
	// responses:
	//   default:
	//     schema:
	//       "$ref": "#/responses/commandResponse"
	//     description: 统一返回格式
	tenantName := strings.TrimSpace(chi.URLParam(r, "tenant_name"))
	page, _ := strconv.Atoi(r.FormValue("page"))
	if page == 0 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(r.FormValue("pageSize"))
	if pageSize == 0 {
		pageSize = 10
	}
	queryName := r.FormValue("query")
	var tenantEnvs []*dbmodel.TenantEnvs

	tenantEnvs, err := handler.GetTenantEnvManager().GetTenantEnvs(tenantName, queryName)
	if err != nil {
		httputil.ReturnError(r, w, 500, "get tenant env error")
		return
	}
	list := handler.GetTenantEnvManager().BindTenantEnvsResource(tenantEnvs)
	re := list.Paging(page, pageSize)
	httputil.ReturnSuccess(r, w, re)
}

// DeleteTenantEnv DeleteTenantEnv
func (t *TenantEnvStruct) DeleteTenantEnv(w http.ResponseWriter, r *http.Request) {
	tenantEnvID := r.Context().Value(ctxutil.ContextKey("tenant_env_id")).(string)

	if err := handler.GetTenantEnvManager().DeleteTenantEnv(r.Context(), tenantEnvID); err != nil {
		if err == handler.ErrTenantEnvStillHasServices || err == handler.ErrTenantEnvStillHasPlugins {
			httputil.ReturnError(r, w, 400, err.Error())
			return
		}
		if err == gorm.ErrRecordNotFound {
			httputil.ReturnError(r, w, 404, err.Error())
			return
		}

		httputil.ReturnError(r, w, 500, fmt.Sprintf("delete tenant env: %v", err))
		return
	}

	httputil.ReturnSuccess(r, w, nil)
}

// UpdateTenantEnv UpdateTenantEnv
// support update tenant env limit memory
func (t *TenantEnvStruct) UpdateTenantEnv(w http.ResponseWriter, r *http.Request) {
	var ts api_model.UpdateTenantEnvStruct
	ok := httputil.ValidatorRequestStructAndErrorResponse(r, w, &ts.Body, nil)
	if !ok {
		return
	}
	tenantEnv := r.Context().Value(ctxutil.ContextKey("tenant_env")).(*dbmodel.TenantEnvs)
	tenantEnv.LimitMemory = ts.Body.LimitMemory
	if err := handler.GetTenantEnvManager().UpdateTenantEnv(tenantEnv); err != nil {
		httputil.ReturnError(r, w, 500, "update tenant env error")
		return
	}
	httputil.ReturnSuccess(r, w, tenantEnv)
}

// GetTenantEnv get one tenant env
func (t *TenantEnvStruct) GetTenantEnv(w http.ResponseWriter, r *http.Request) {
	tenantEnv := r.Context().Value(ctxutil.ContextKey("tenant_env")).(*dbmodel.TenantEnvs)
	list := handler.GetTenantEnvManager().BindTenantEnvsResource([]*dbmodel.TenantEnvs{tenantEnv})
	httputil.ReturnSuccess(r, w, list[0])
}

// ServicesCount Get all apps and status
func (t *TenantEnvStruct) ServicesCount(w http.ResponseWriter, r *http.Request) {
	allStatus := t.StatusCli.GetAllStatus()
	var closed int
	var running int
	var abnormal int
	for _, v := range allStatus {
		switch v {
		case "closed":
			closed++
		case "running":
			running++
		case "abnormal":
			abnormal++
		}
	}
	serviceCount := map[string]int{"total": len(allStatus), "running": running, "closed": closed, "abnormal": abnormal}
	httputil.ReturnSuccess(r, w, serviceCount)
}

// ServicesInfo GetServiceInfo
func (t *TenantEnvStruct) ServicesInfo(w http.ResponseWriter, r *http.Request) {
	// swagger:operation GET /v2/tenants/{tenant_name}/envs/{tenant_env_name}/services v2 getServiceInfo
	//
	// 获取租户所有应用信息
	//
	// get services info in tenantEnv
	//
	// ---
	// consumes:
	// - application/json
	// - application/x-protobuf
	//
	// produces:
	// - application/json
	// - application/xml
	//
	// responses:
	//   default:
	//     schema:
	//       "$ref": "#/responses/commandResponse"
	//     description: 统一返回格式
	tenantEnvID := r.Context().Value(ctxutil.ContextKey("tenant_env_id")).(string)
	services, err := handler.GetServiceManager().GetService(tenantEnvID)
	if err != nil {
		httputil.ReturnError(r, w, 500, "get tenant env services error")
		return
	}
	httputil.ReturnSuccess(r, w, services)
}

// CreateService create Service
func (t *TenantEnvStruct) CreateService(w http.ResponseWriter, r *http.Request) {
	var ss api_model.ServiceStruct
	if !httputil.ValidatorRequestStructAndErrorResponse(r, w, &ss, nil) {
		return
	}

	// Check if the application ID exists
	if ss.AppID == "" {
		httputil.ReturnBcodeError(r, w, bcode.ErrCreateNeedCorrectAppID)
		return
	}
	_, err := handler.GetApplicationHandler().GetAppByID(ss.AppID)
	if err != nil {
		httputil.ReturnBcodeError(r, w, err)
		return
	}

	// clean etcd data(source check)
	handler.GetEtcdHandler().CleanServiceCheckData(ss.EtcdKey)

	values := url.Values{}
	if ss.Endpoints != nil {
		for _, endpoint := range ss.Endpoints.Static {
			if strings.Contains(endpoint, "127.0.0.1") {
				values["ip"] = []string{"The ip field is can't contains '127.0.0.1'"}
			}
		}
		if len(values) > 0 {
			httputil.ReturnValidationError(r, w, values)
			return
		}
	}

	tenantEnvID := r.Context().Value(ctxutil.ContextKey("tenant_env_id")).(string)
	ss.TenantEnvID = tenantEnvID
	if err := handler.GetServiceManager().ServiceCreate(&ss); err != nil {
		if strings.Contains(err.Error(), "is exist in tenantEnv") {
			httputil.ReturnError(r, w, 400, fmt.Sprintf("create service error, %v", err))
			return
		}
		httputil.ReturnError(r, w, 500, fmt.Sprintf("create service error, %v", err))
		return
	}

	httputil.ReturnSuccess(r, w, nil)
}

// UpdateService create Service
func (t *TenantEnvStruct) UpdateService(w http.ResponseWriter, r *http.Request) {
	// swagger:operation PUT /v2/tenants/{tenant_name}/envs/{tenant_env_name}/services/{service_alias} v2 updateService
	//
	// 应用更新
	//
	// update service
	//
	// ---
	// consumes:
	// - application/json
	// - application/x-protobuf
	//
	// produces:
	// - application/json
	// - application/xml
	//
	// responses:
	//   default:
	//     schema:
	//       "$ref": "#/responses/commandResponse"
	//     description: 统一返回格式
	//目前提供三个元素的修改
	rules := validator.MapData{
		"container_cmd":      []string{},
		"image_name":         []string{},
		"container_memory":   []string{},
		"service_name":       []string{},
		"extend_method":      []string{},
		"app_id":             []string{},
		"k8s_component_name": []string{},
	}
	data, ok := httputil.ValidatorRequestMapAndErrorResponse(r, w, rules, nil)
	if !ok {
		return
	}
	serviceID := r.Context().Value(ctxutil.ContextKey("service_id")).(string)
	data["service_id"] = serviceID

	// Check if the application ID exists
	var appID string
	if data["app_id"] != nil && data["app_id"] != "" {
		appID = data["app_id"].(string)
		_, err := handler.GetApplicationHandler().GetAppByID(appID)
		if err != nil {
			httputil.ReturnBcodeError(r, w, err)
			return
		}
	}

	if err := handler.GetServiceManager().ServiceUpdate(data); err != nil {
		httputil.ReturnBcodeError(r, w, err)
		return
	}
	httputil.ReturnSuccess(r, w, nil)
}

// SetLanguage SetLanguage
func (t *TenantEnvStruct) SetLanguage(w http.ResponseWriter, r *http.Request) {
	// swagger:operation POST  /v2/tenants/{tenant_name}/envs/{tenant_env_name}/services/{service_alias}/language v2 setLanguage
	//
	// 设置应用语言
	//
	// set language
	//
	// ---
	// consumes:
	// - application/json
	// - application/x-protobuf
	//
	// produces:
	// - application/json
	// - application/xml
	//
	// responses:
	//   default:
	//     schema:
	//       "$ref": "#/responses/commandResponse"
	//     description: 统一返回格式
	rules := validator.MapData{
		"language": []string{"required"},
	}
	langS := &api_model.LanguageSet{}
	data, ok := httputil.ValidatorRequestMapAndErrorResponse(r, w, rules, nil)
	if !ok {
		return
	}
	langS.Language = data["language"].(string)
	langS.ServiceID = r.Context().Value(ctxutil.ContextKey("service_id")).(string)
	if err := handler.GetServiceManager().LanguageSet(langS); err != nil {
		httputil.ReturnError(r, w, 500, "set language error.")
		return
	}
	httputil.ReturnSuccess(r, w, nil)
}

// StatusService StatusService
func (t *TenantEnvStruct) StatusService(w http.ResponseWriter, r *http.Request) {
	// swagger:operation GET /v2/tenants/{tenant_name}/envs/{tenant_env_name}/services/{service_alias}/status v2 serviceStatus
	//
	// 获取应用状态
	//
	// get service status
	//
	// ---
	// consumes:
	// - application/json
	// - application/x-protobuf
	//
	// produces:
	// - application/json
	// - application/xml
	//
	// responses:
	//   default:
	//     schema:
	//       "$ref": "#/responses/commandResponse"
	//     description: 统一返回格式
	serviceID := r.Context().Value(ctxutil.ContextKey("service_id")).(string)
	statusList, err := handler.GetServiceManager().GetStatus(serviceID)
	if err != nil {
		httputil.ReturnError(r, w, 500, fmt.Sprintf("get service list error,%v", err))
		return
	}
	httputil.ReturnSuccess(r, w, statusList)
}

// PostStatusService PostStatusService
func (t *TenantEnvStruct) PostStatusService(w http.ResponseWriter, r *http.Request) {
	logrus.Debugf("in status service serviceID")
}

// StatusServiceList service list status
func (t *TenantEnvStruct) StatusServiceList(w http.ResponseWriter, r *http.Request) {
	// swagger:operation POST /v2/tenants/{tenant_name}/envs/{tenant_env_name}/services_status v2 serviceStatuslist
	//
	// 获取应用状态
	//
	// get service statuslist
	//
	// ---
	// consumes:
	// - application/json
	// - application/x-protobuf
	//
	// produces:
	// - application/json
	// - application/xml
	//
	// responses:
	//   default:
	//     schema:
	//       "$ref": "#/responses/commandResponse"
	//     description: 统一返回格式
	var services api_model.StatusServiceListStruct
	ok := httputil.ValidatorRequestStructAndErrorResponse(r, w, &services.Body, nil)
	if !ok {
		return
	}
	//logrus.Info(services.Body.ServiceIDs)
	serviceList := services.Body.ServiceIDs
	tenantEnvID := r.Context().Value(ctxutil.ContextKey("tenant_env_id")).(string)
	info := handler.GetServiceManager().GetServicesStatus(tenantEnvID, serviceList)

	httputil.ReturnSuccess(r, w, info)
}

// Label -
func (t *TenantEnvStruct) Label(w http.ResponseWriter, r *http.Request) {
	var req api_model.LabelsStruct
	ok := httputil.ValidatorRequestStructAndErrorResponse(r, w, &req, nil)
	if !ok {
		return
	}
	reqJSON, _ := json.Marshal(req)
	logrus.Debugf("Request is : %s", string(reqJSON))

	// verify request
	values := url.Values{}
	if len(req.Labels) == 0 {
		values["labels"] = []string{"The labels field should have someting"}
	}
	for _, label := range req.Labels {
		if label.LabelKey == "" {
			values["label_key"] = []string{"The label_key field is required"}
		}
		if label.LabelValue == "" {
			values["label_value"] = []string{"The label_value field is required"}
		}
	}
	if len(values) != 0 {
		httputil.ReturnValidationError(r, w, values)
		return
	}

	switch r.Method {
	case "DELETE":
		t.DeleteLabel(w, r, &req)
	case "POST":
		t.AddLabel(w, r, &req)
	case "PUT":
		t.UpdateLabel(w, r, &req)
	}
}

// AddLabel adds label
func (t *TenantEnvStruct) AddLabel(w http.ResponseWriter, r *http.Request, labels *api_model.LabelsStruct) {
	logrus.Debugf("add label")
	serviceID := r.Context().Value(ctxutil.ContextKey("service_id")).(string)
	if err := handler.GetServiceManager().AddLabel(labels, serviceID); err != nil {
		httputil.ReturnError(r, w, 500, fmt.Sprintf("add label error, %v", err))
		return
	}
	httputil.ReturnSuccess(r, w, nil)
}

// DeleteLabel deletes labels
func (t *TenantEnvStruct) DeleteLabel(w http.ResponseWriter, r *http.Request, labels *api_model.LabelsStruct) {
	serviceID := r.Context().Value(ctxutil.ContextKey("service_id")).(string)
	if err := handler.GetServiceManager().DeleteLabel(labels, serviceID); err != nil {
		httputil.ReturnError(r, w, 500, fmt.Sprintf("delete node label failure, %v", err))
		return
	}
	httputil.ReturnSuccess(r, w, nil)
}

// UpdateLabel Update updates labels
func (t *TenantEnvStruct) UpdateLabel(w http.ResponseWriter, r *http.Request, labels *api_model.LabelsStruct) {
	serviceID := r.Context().Value(ctxutil.ContextKey("service_id")).(string)
	if err := handler.GetServiceManager().UpdateLabel(labels, serviceID); err != nil {
		httputil.ReturnError(r, w, 500, fmt.Sprintf("error updating label: %v", err))
		return
	}
	httputil.ReturnSuccess(r, w, nil)
}

// StatusContainerID StatusContainerID
func (t *TenantEnvStruct) StatusContainerID(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("status container IDs list"))
}

// SingleServiceInfo SingleServiceInfo
func (t *TenantEnvStruct) SingleServiceInfo(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "DELETE":
		t.DeleteSingleServiceInfo(w, r)
	case "GET":
		t.GetSingleServiceInfo(w, r)
	}
}

// GetSingleServiceInfo GetSingleServiceInfo
func (t *TenantEnvStruct) GetSingleServiceInfo(w http.ResponseWriter, r *http.Request) {
	// swagger:operation GET /v2/tenants/{tenant_name}/envs/{tenant_env_name}/services/{service_alias} v2 getService
	//
	// 获取应用信息
	//
	// get service info
	//
	// ---
	// consumes:
	// - application/json
	// - application/x-protobuf
	//
	// produces:
	// - application/json
	// - application/xml
	//
	// responses:
	//   default:
	//     schema:
	//       "$ref": "#/responses/commandResponse"
	//     description: 统一返回格式

	tenantEnvID := r.Context().Value(ctxutil.ContextKey("tenant_env_id")).(string)
	serviceID := r.Context().Value(ctxutil.ContextKey("service_id")).(string)
	tenantEnvName := r.Context().Value(ctxutil.ContextKey("tenant_env_name")).(string)
	serviceName := r.Context().Value(ctxutil.ContextKey("service_alias")).(string)
	result := make(map[string]string)
	result["tenantEnvName"] = tenantEnvName
	result["serviceAlias"] = serviceName
	result["tenantEnvId"] = tenantEnvID
	result["serviceId"] = serviceID
	httputil.ReturnSuccess(r, w, result)
}

// DeleteSingleServiceInfo DeleteService
// swagger:operation DELETE /v2/tenants/{tenant_name}/envs/{tenant_env_name}/services/{service_alias} v2 deleteService
//
// 删除应用
//
// delete service
//
// ---
// consumes:
// - application/json
// - application/x-protobuf
//
// produces:
// - application/json
// - application/xml
//
// responses:
//
//	default:
//	  schema:
//	    "$ref": "#/responses/commandResponse"
//	  description: 统一返回格式
func (t *TenantEnvStruct) DeleteSingleServiceInfo(w http.ResponseWriter, r *http.Request) {
	serviceID := r.Context().Value(ctxutil.ContextKey("service_id")).(string)
	tenantEnvID := r.Context().Value(ctxutil.ContextKey("tenant_env_id")).(string)
	var req api_model.EtcdCleanReq
	if httputil.ValidatorRequestStructAndErrorResponse(r, w, &req, nil) {
		logrus.Debugf("delete service etcd keys : %+v", req.Keys)
		handler.GetEtcdHandler().CleanAllServiceData(req.Keys)
	}

	if err := handler.GetServiceManager().TransServieToDelete(r.Context(), tenantEnvID, serviceID); err != nil {
		if err == handler.ErrServiceNotClosed {
			httputil.ReturnError(r, w, 400, "Service must be closed")
			return
		}
		httputil.ReturnError(r, w, 500, fmt.Sprintf("delete service error, %v", err))
		return
	}
	httputil.ReturnSuccess(r, w, nil)
}

// Dependency Dependency
func (t *TenantEnvStruct) Dependency(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "DELETE":
		t.DeleteDependency(w, r)
	case "POST":
		t.AddDependency(w, r)
	}
}

// DeleteDependencies 删除组件依赖
func (t *TenantEnvStruct) DeleteDependencies(w http.ResponseWriter, r *http.Request) {
	logrus.Debugf("trans delete depend service")
	ds := &api_model.DependService{
		TenantEnvID: r.Context().Value(ctxutil.ContextKey("tenant_env_id")).(string),
		ServiceID:   r.Context().Value(ctxutil.ContextKey("service_id")).(string),
	}
	if err := handler.GetServiceManager().ServiceDepend("delete_all", ds); err != nil {
		httputil.ReturnError(r, w, 500, fmt.Sprintf("delete dependency error, %v", err))
		return
	}
	httputil.ReturnSuccess(r, w, nil)
}

// AddDependency AddDependency
// swagger:operation POST /v2/tenants/{tenant_name}/envs/{tenant_env_name}/services/{service_alias}/dependency v2 addDependency
//
// 增加应用依赖关系
//
// add dependency
//
// ---
// consumes:
// - application/json
// - application/x-protobuf
//
// produces:
// - application/json
// - application/xml
//
// responses:
//
//	default:
//	  schema:
//	    "$ref": "#/responses/commandResponse"
//	  description: 统一返回格式
func (t *TenantEnvStruct) AddDependency(w http.ResponseWriter, r *http.Request) {
	rules := validator.MapData{
		"dep_service_id":   []string{"required"},
		"dep_service_type": []string{"required"},
		"dep_order":        []string{},
	}
	data, ok := httputil.ValidatorRequestMapAndErrorResponse(r, w, rules, nil)
	if !ok {
		return
	}
	ds := &api_model.DependService{
		TenantEnvID:    r.Context().Value(ctxutil.ContextKey("tenant_env_id")).(string),
		ServiceID:      r.Context().Value(ctxutil.ContextKey("service_id")).(string),
		DepServiceID:   data["dep_service_id"].(string),
		DepServiceType: data["dep_service_type"].(string),
	}
	if err := handler.GetServiceManager().ServiceDepend("add", ds); err != nil {
		httputil.ReturnError(r, w, 500, fmt.Sprintf("add dependency error, %v", err))
		return
	}
	httputil.ReturnSuccess(r, w, nil)
}

// DeleteDependency DeleteDependency
// swagger:operation DELETE /v2/tenants/{tenant_name}/envs/{tenant_env_name}/services/{service_alias}/dependency v2 deleteDependency
//
// 删除应用依赖关系
//
// delete dependency
//
// ---
// consumes:
// - application/json
// - application/x-protobuf
//
// produces:
// - application/json
// - application/xml
//
// responses:
//
//	default:
//	  schema:
//	    "$ref": "#/responses/commandResponse"
//	  description: 统一返回格式
func (t *TenantEnvStruct) DeleteDependency(w http.ResponseWriter, r *http.Request) {
	logrus.Debugf("trans delete depend service ")
	rules := validator.MapData{
		"dep_service_id":   []string{"required"},
		"dep_service_type": []string{},
		"dep_order":        []string{},
	}
	data, ok := httputil.ValidatorRequestMapAndErrorResponse(r, w, rules, nil)
	if !ok {
		return
	}
	ds := &api_model.DependService{
		TenantEnvID:  r.Context().Value(ctxutil.ContextKey("tenant_env_id")).(string),
		ServiceID:    r.Context().Value(ctxutil.ContextKey("service_id")).(string),
		DepServiceID: data["dep_service_id"].(string),
	}
	if err := handler.GetServiceManager().ServiceDepend("delete", ds); err != nil {
		httputil.ReturnError(r, w, 500, fmt.Sprintf("delete dependency error, %v", err))
		return
	}
	httputil.ReturnSuccess(r, w, nil)
}

// Env Env
func (t *TenantEnvStruct) Env(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "DELETE":
		t.DeleteEnv(w, r)
	case "POST":
		t.AddEnv(w, r)
	case "PUT":
		t.UpdateEnv(w, r)
	}
}

// DeleteAllEnvs 删除组件所有环境变量
func (t *TenantEnvStruct) DeleteAllEnvs(w http.ResponseWriter, r *http.Request) {
	serviceID := r.Context().Value(ctxutil.ContextKey("service_id")).(string)
	tenantEnvID := r.Context().Value(ctxutil.ContextKey("tenant_env_id")).(string)
	if err := handler.GetServiceManager().EnvAttr("delete_all", &dbmodel.TenantEnvServiceEnvVar{
		ServiceID:   serviceID,
		TenantEnvID: tenantEnvID,
	}); err != nil && err.Error() != gorm.ErrRecordNotFound.Error() {
		logrus.Errorf("delete all envs error, %v", err)
		httputil.ReturnError(r, w, 500, fmt.Sprintf("Delete env error, %v", err))
		return
	}
	httputil.ReturnSuccess(r, w, nil)
}

// DeleteAllInnerEnvs 删除组件所有内部环境变量
func (t *TenantEnvStruct) DeleteAllInnerEnvs(w http.ResponseWriter, r *http.Request) {
	serviceID := r.Context().Value(ctxutil.ContextKey("service_id")).(string)
	tenantEnvID := r.Context().Value(ctxutil.ContextKey("tenant_env_id")).(string)
	if err := handler.GetServiceManager().EnvAttr("delete_all_inner", &dbmodel.TenantEnvServiceEnvVar{
		ServiceID:   serviceID,
		TenantEnvID: tenantEnvID,
		Scope:       "inner",
	}); err != nil && err.Error() != gorm.ErrRecordNotFound.Error() {
		logrus.Errorf("delete all inner envs error, %v", err)
		httputil.ReturnError(r, w, 500, fmt.Sprintf("Delete env error, %v", err))
		return
	}
	httputil.ReturnSuccess(r, w, nil)
}

// AddEnv AddEnv
// swagger:operation POST /v2/tenants/{tenant_name}/envs/{tenant_env_name}/services/{service_alias}/env v2 addEnv
//
// 增加环境变量
//
// add env var
//
// ---
// consumes:
// - application/json
// - application/x-protobuf
//
// produces:
// - application/json
// - application/xml
//
// responses:
//
//	default:
//	  schema:
//	    "$ref": "#/responses/commandResponse"
//	  description: 统一返回格式
func (t *TenantEnvStruct) AddEnv(w http.ResponseWriter, r *http.Request) {
	var envM api_model.AddTenantEnvServiceEnvVar
	if !httputil.ValidatorRequestStructAndErrorResponse(r, w, &envM, nil) {
		return
	}
	tenantEnvID := r.Context().Value(ctxutil.ContextKey("tenant_env_id")).(string)
	serviceID := r.Context().Value(ctxutil.ContextKey("service_id")).(string)
	var envD dbmodel.TenantEnvServiceEnvVar
	envD.AttrName = envM.AttrName
	envD.AttrValue = envM.AttrValue
	envD.TenantEnvID = tenantEnvID
	envD.ServiceID = serviceID
	envD.ContainerPort = envM.ContainerPort
	envD.IsChange = envM.IsChange
	envD.Name = envM.Name
	envD.Scope = envM.Scope
	if err := handler.GetServiceManager().EnvAttr("add", &envD); err != nil {
		if err == errors.ErrRecordAlreadyExist {
			httputil.ReturnError(r, w, 400, fmt.Sprintf("%v", err))
			return
		}
		logrus.Errorf("Add env error, %v", err)
		httputil.ReturnError(r, w, 500, fmt.Sprintf("Add env error, %v", err))
		return
	}
	httputil.ReturnSuccess(r, w, nil)
}

// UpdateEnv UpdateEnv
// swagger:operation PUT /v2/tenants/{tenant_name}/envs/{tenant_env_name}/services/{service_alias}/env v2 update Env
//
// 修改环境变量
//
// update env var
//
// ---
// consumes:
// - application/json
// - application/x-protobuf
//
// produces:
// - application/json
// - application/xml
//
// responses:
//
//	default:
//	  schema:
//	    "$ref": "#/responses/commandResponse"
//	  description: 统一返回格式
func (t *TenantEnvStruct) UpdateEnv(w http.ResponseWriter, r *http.Request) {
	var envM api_model.AddTenantEnvServiceEnvVar
	if !httputil.ValidatorRequestStructAndErrorResponse(r, w, &envM, nil) {
		return
	}
	tenantEnvID := r.Context().Value(ctxutil.ContextKey("tenant_env_id")).(string)
	serviceID := r.Context().Value(ctxutil.ContextKey("service_id")).(string)
	var envD dbmodel.TenantEnvServiceEnvVar
	envD.AttrName = envM.AttrName
	envD.AttrValue = envM.AttrValue
	envD.TenantEnvID = tenantEnvID
	envD.ServiceID = serviceID
	envD.ContainerPort = envM.ContainerPort
	envD.IsChange = envM.IsChange
	envD.Name = envM.Name
	envD.Scope = envM.Scope
	if err := handler.GetServiceManager().EnvAttr("update", &envD); err != nil {
		logrus.Errorf("update env error, %v", err)
		httputil.ReturnError(r, w, 500, fmt.Sprintf("update env error, %v", err))
		return
	}
	httputil.ReturnSuccess(r, w, nil)
}

// DeleteEnv DeleteEnv
// swagger:operation DELETE /v2/tenants/{tenant_name}/envs/{tenant_env_name}/services/{service_alias}/env v2 deleteEnv
//
// 删除环境变量
//
// delete env var
//
// ---
// consumes:
// - application/json
// - application/x-protobuf
//
// produces:
// - application/json
// - application/xml
//
// responses:
//
//	default:
//	  schema:
//	    "$ref": "#/responses/commandResponse"
//	  description: 统一返回格式
func (t *TenantEnvStruct) DeleteEnv(w http.ResponseWriter, r *http.Request) {
	var envM api_model.DelTenantEnvServiceEnvVar
	if !httputil.ValidatorRequestStructAndErrorResponse(r, w, &envM, nil) {
		return
	}
	tenantEnvID := r.Context().Value(ctxutil.ContextKey("tenant_env_id")).(string)
	serviceID := r.Context().Value(ctxutil.ContextKey("service_id")).(string)
	envM.TenantEnvID = tenantEnvID
	envM.ServiceID = serviceID
	var envD dbmodel.TenantEnvServiceEnvVar
	envD.AttrName = envM.AttrName
	envD.AttrValue = envM.AttrValue
	envD.TenantEnvID = tenantEnvID
	envD.ServiceID = serviceID
	envD.ContainerPort = envM.ContainerPort
	envD.IsChange = envM.IsChange
	envD.Name = envM.Name
	envD.Scope = envM.Scope
	if err := handler.GetServiceManager().EnvAttr("delete", &envD); err != nil {
		logrus.Errorf("delete env error, %v", err)
		if err.Error() == gorm.ErrRecordNotFound.Error() {
			httputil.ReturnError(r, w, 404, "service port "+err.Error())
			return
		}
		httputil.ReturnError(r, w, 500, fmt.Sprintf("Delete env error, %v", err))
		return
	}
	httputil.ReturnSuccess(r, w, nil)
}

// Ports 应用端口控制器
func (t *TenantEnvStruct) Ports(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "DELETE":
		t.deletePortController(w, r)
	case "POST":
		t.addPortController(w, r)
	case "PUT":
		t.updatePortController(w, r)
	}
}

// DeletePorts PortVar
func (t *TenantEnvStruct) DeleteAllPorts(w http.ResponseWriter, r *http.Request) {
	tenantEnvID := r.Context().Value(ctxutil.ContextKey("tenant_env_id")).(string)
	serviceID := r.Context().Value(ctxutil.ContextKey("service_id")).(string)
	if err := handler.GetServiceManager().PortVar("delete_all", tenantEnvID, serviceID, nil, 0); err != nil {
		httputil.ReturnBcodeError(r, w, err)
		return
	}
	httputil.ReturnSuccess(r, w, nil)
}

// PutPorts PortVar
// swagger:operation PUT /v2/tenants/{tenant_name}/envs/{tenant_env_name}/services/{service_alias}/ports v2 updatePort
//
// 更新应用端口信息(旧)
//
// update port
//
// ---
// consumes:
// - application/json
// - application/x-protobuf
//
// produces:
// - application/json
// - application/xml
//
// responses:
//
//	default:
//	  schema:
//	    "$ref": "#/responses/commandResponse"
//	  description: 统一返回格式
func (t *TenantEnvStruct) PutPorts(w http.ResponseWriter, r *http.Request) {
	tenantEnvID := r.Context().Value(ctxutil.ContextKey("tenant_env_id")).(string)
	serviceID := r.Context().Value(ctxutil.ContextKey("service_id")).(string)
	var ports api_model.ServicePorts
	if ok := httputil.ValidatorRequestStructAndErrorResponse(r, w, &ports, nil); !ok {
		return
	}
	if err := handler.GetServiceManager().PortVar("update", tenantEnvID, serviceID, &ports, 0); err != nil {
		httputil.ReturnBcodeError(r, w, err)
		return
	}
	httputil.ReturnSuccess(r, w, nil)
}

// AddPortVar PortVar
// swagger:operation POST /v2/tenants/{tenant_name}/envs/{tenant_env_name}/services/{service_alias}/ports v2 addPort
//
// 增加应用端口,默认关闭对内和对外选项，需要开启使用相应接口
//
// add port
//
// ---
// consumes:
// - application/json
// - application/x-protobuf
//
// produces:
// - application/json
// - application/xml
//
// responses:
//
//	default:
//	  schema:
//	    "$ref": "#/responses/commandResponse"
//	  description: 统一返回格式
func (t *TenantEnvStruct) addPortController(w http.ResponseWriter, r *http.Request) {
	tenantEnvID := r.Context().Value(ctxutil.ContextKey("tenant_env_id")).(string)
	serviceID := r.Context().Value(ctxutil.ContextKey("service_id")).(string)
	var ports api_model.ServicePorts
	if ok := httputil.ValidatorRequestStructAndErrorResponse(r, w, &ports, nil); !ok {
		return
	}
	if err := handler.GetServiceManager().CreatePorts(tenantEnvID, serviceID, &ports); err != nil {
		logrus.Errorf("add port error. %v", err)
		httputil.ReturnError(r, w, 500, err.Error())
		return
	}
	httputil.ReturnSuccess(r, w, ports.Port)
}

// UpdatePortVar PortVar
// swagger:operation PUT /v2/tenants/{tenant_name}/envs/{tenant_env_name}/services/{service_alias}/ports/{port} v2 updatePort
//
// 更新应用端口信息
//
// update port
//
// ---
// consumes:
// - application/json
// - application/x-protobuf
//
// produces:
// - application/json
// - application/xml
//
// responses:
//
//	default:
//	  schema:
//	    "$ref": "#/responses/commandResponse"
//	  description: 统一返回格式
func (t *TenantEnvStruct) updatePortController(w http.ResponseWriter, r *http.Request) {
	tenantEnvID := r.Context().Value(ctxutil.ContextKey("tenant_env_id")).(string)
	serviceID := r.Context().Value(ctxutil.ContextKey("service_id")).(string)
	portStr := chi.URLParam(r, "port")
	oldPort, err := strconv.Atoi(portStr)
	if err != nil {
		httputil.ReturnError(r, w, 400, "port must be a number")
		return
	}
	var ports api_model.ServicePorts
	if ok := httputil.ValidatorRequestStructAndErrorResponse(r, w, &ports, nil); !ok {
		return
	}
	if err := handler.GetServiceManager().PortVar("update", tenantEnvID, serviceID, &ports, oldPort); err != nil {
		logrus.Errorf("update port error. %v", err)
		httputil.ReturnError(r, w, 500, err.Error())
		return
	}
	httputil.ReturnSuccess(r, w, nil)
}

// DeletePortVar PortVar
// swagger:operation DELETE /v2/tenants/{tenant_name}/envs/{tenant_env_name}/services/{service_alias}/ports/{port} v2 deletePort
//
// 删除端口变量
//
// delete port
//
// ---
// Consumes:
// - application/json
// - application/x-protobuf
//
// produces:
// - application/json
// - application/xml
//
// responses:
//
//	default:
//	  schema:
//	    "$ref": "#/responses/commandResponse"
//	  description: 统一返回格式
func (t *TenantEnvStruct) deletePortController(w http.ResponseWriter, r *http.Request) {
	tenantEnvID := r.Context().Value(ctxutil.ContextKey("tenant_env_id")).(string)
	serviceID := r.Context().Value(ctxutil.ContextKey("service_id")).(string)
	portStr := chi.URLParam(r, "port")
	oldPort, err := strconv.Atoi(portStr)
	if err != nil {
		httputil.ReturnError(r, w, 400, "port must be a number")
		return
	}
	var port = &api_model.TenantEnvServicesPort{
		TenantEnvID:   tenantEnvID,
		ServiceID:     serviceID,
		ContainerPort: oldPort,
	}
	var ports api_model.ServicePorts
	ports.Port = append(ports.Port, port)
	if err := handler.GetServiceManager().PortVar("delete", tenantEnvID, serviceID, &ports, oldPort); err != nil {
		if err.Error() == gorm.ErrRecordNotFound.Error() {
			httputil.ReturnError(r, w, 404, "port can not found")
			return
		}
		httputil.ReturnError(r, w, 500, err.Error())
		return
	}
	httputil.ReturnSuccess(r, w, nil)
}

// PortOuterController 开关端口对外服务
// swagger:operation PUT /v2/tenants/{tenant_name}/envs/{tenant_env_name}/services/{service_alias}/ports/{port}/outer v2 PortOuterController
//
// 开关端口对外服务，应用无需重启自动生效
//
// add port
//
// ---
// Consumes:
// - application/json
// - application/x-protobuf
//
// produces:
// - application/json
// - application/xml
//
// responses:
//
//	default:
//	  schema:
//	    "$ref": "#/responses/commandResponse"
//	  description: 统一返回格式
func (t *TenantEnvStruct) PortOuterController(w http.ResponseWriter, r *http.Request) {
	var data api_model.ServicePortInnerOrOuter
	if !httputil.ValidatorRequestStructAndErrorResponse(r, w, &(data.Body), nil) {
		return
	}

	serviceID := r.Context().Value(ctxutil.ContextKey("service_id")).(string)
	service := r.Context().Value(ctxutil.ContextKey("service")).(*dbmodel.TenantEnvServices)
	if dbmodel.ServiceKind(service.Kind) == dbmodel.ServiceKindThirdParty {
		endpoints, err := db.GetManager().EndpointsDao().List(serviceID)
		if err != nil {
			logrus.Errorf("find endpoints by sid[%s], error: %s", serviceID, err.Error())
			httputil.ReturnError(r, w, 500, "fund endpoints failure")
			return
		}
		for _, ep := range endpoints {
			address := validation.SplitEndpointAddress(ep.IP)
			if validation.IsDomainNotIP(address) {
				httputil.ReturnError(r, w, 400, "do not allow operate outer port for thirdpart domain endpoints")
				return
			}
		}
	}

	tenantEnvName := r.Context().Value(ctxutil.ContextKey("tenant_env_name")).(string)
	portStr := chi.URLParam(r, "port")
	containerPort, err := strconv.Atoi(portStr)
	if err != nil {
		httputil.ReturnError(r, w, 400, "port must be a number")
		return
	}
	vsPort, protocol, errV := handler.GetServiceManager().PortOuter(tenantEnvName, serviceID, containerPort, &data)
	if errV != nil {
		if strings.HasSuffix(errV.Error(), gorm.ErrRecordNotFound.Error()) {
			httputil.ReturnError(r, w, 404, errV.Error())
			return
		}
		httputil.ReturnError(r, w, 500, errV.Error())
		return
	}
	rc := make(map[string]string)
	domain := os.Getenv("EX_DOMAIN")
	if domain == "" {
		httputil.ReturnError(r, w, 500, "have no EX_DOMAIN")
		return
	}
	mm := strings.Split(domain, ":")
	if protocol == "http" || protocol == "https" {
		rc["domain"] = mm[0]
		if len(mm) == 2 {
			rc["port"] = mm[1]
		} else {
			rc["port"] = "10080"
		}
	} else if vsPort != nil {
		rc["domain"] = mm[0]
		rc["port"] = fmt.Sprintf("%v", vsPort.Port)
	}

	if err := handler.GetGatewayHandler().SendTaskDeprecated(map[string]interface{}{
		"service_id": serviceID,
		"action":     "port-" + data.Body.Operation,
		"port":       containerPort,
		"is_inner":   false,
	}); err != nil {
		logrus.Errorf("send runtime message about gateway failure %s", err.Error())
	}

	httputil.ReturnSuccess(r, w, rc)
}

// PortInnerController 开关端口对内服务
// swagger:operation PUT /v2/tenants/{tenant_name}/envs/{tenant_env_name}/services/{service_alias}/ports/{port}/inner v2 PortInnerController
//
// 开关对内服务，应用无需重启，自动生效
//
// add port
//
// ---
// Consumes:
// - application/json
// - application/x-protobuf
//
// produces:
// - application/json
// - application/xml
//
// responses:
//
//	default:
//	  schema:
//	    "$ref": "#/responses/commandResponse"
//	  description: 统一返回格式
func (t *TenantEnvStruct) PortInnerController(w http.ResponseWriter, r *http.Request) {
	var data api_model.ServicePortInnerOrOuter
	if !httputil.ValidatorRequestStructAndErrorResponse(r, w, &(data.Body), nil) {
		return
	}
	serviceID := r.Context().Value(ctxutil.ContextKey("service_id")).(string)
	tenantEnvName := r.Context().Value(ctxutil.ContextKey("tenant_env_name")).(string)
	portStr := chi.URLParam(r, "port")
	containerPort, err := strconv.Atoi(portStr)
	if err != nil {
		httputil.ReturnError(r, w, 400, "port must be a number")
		return
	}
	if err := handler.GetServiceManager().PortInner(tenantEnvName, serviceID, data.Body.Operation, containerPort); err != nil {
		if err.Error() == gorm.ErrRecordNotFound.Error() {
			httputil.ReturnError(r, w, 404, "service port "+err.Error())
			return
		} else if err.Error() == "already open" || err.Error() == "already close" {
			httputil.Return(r, w, 200, httputil.ResponseBody{Msg: err.Error()})
			return
		} else {
			httputil.ReturnError(r, w, 500, err.Error())
			return
		}
	}

	if err := handler.GetGatewayHandler().SendTaskDeprecated(map[string]interface{}{
		"service_id": serviceID,
		"action":     "port-" + data.Body.Operation,
		"port":       containerPort,
		"is_inner":   true,
	}); err != nil {
		logrus.Errorf("send runtime message about gateway failure %s", err.Error())
	}

	httputil.ReturnSuccess(r, w, nil)
}

// Pods pods
// swagger:operation GET  /v2/tenants/{tenant_name}/envs/{tenant_env_name}/services/{service_alias}/pods v2 getPodsInfo
//
// 获取pods信息
//
// get pods info
//
// ---
// consumes:
// - application/json
// - application/x-protobuf
//
// produces:
// - application/json
// - application/xml
//
// responses:
//
//	default:
//	  schema:
//	    "$ref": "#/responses/commandResponse"
//	  description: 统一返回格式
func (t *TenantEnvStruct) Pods(w http.ResponseWriter, r *http.Request) {
	serviceID := r.Context().Value(ctxutil.ContextKey("service_id")).(string)
	pods, err := handler.GetServiceManager().GetPods(serviceID)
	if err != nil {
		if err.Error() == gorm.ErrRecordNotFound.Error() {
			logrus.Error("record not found:", err)
			httputil.ReturnError(r, w, 404, fmt.Sprintf("get pods error, %v", err))
			return
		}
		logrus.Error("get pods error:", err)
		httputil.ReturnError(r, w, 500, fmt.Sprintf("get pods error, %v", err))
		return
	}
	httputil.ReturnSuccess(r, w, pods)
}

// ListServiceInstances 获取组件实例列表
// swagger:operation GET  /v2/tenants/{tenant_name}/envs/{tenant_env_name}/services/{service_alias}/instances v2 ListServiceInstances
//
// 获取组件实例列表
//
// list instances
//
// ---
// consumes:
// - application/json
// - application/x-protobuf
//
// produces:
// - application/json
// - application/xml
//
// responses:
//
//	default:
//	  schema:
//	    "$ref": "#/responses/commandResponse"
//	  description: 统一返回格式
func (t *TenantEnvStruct) ListServiceInstances(w http.ResponseWriter, r *http.Request) {
	tenantEnv := r.Context().Value(ctxutil.ContextKey("tenant_env")).(*dbmodel.TenantEnvs)
	serviceID := r.Context().Value(ctxutil.ContextKey("service_id")).(string)
	pods, err := handler.GetServiceManager().ListServiceInstances(tenantEnv.Namespace, serviceID)
	if err != nil {
		if err.Error() == gorm.ErrRecordNotFound.Error() {
			logrus.Error("record not found:", err)
			httputil.ReturnError(r, w, 404, fmt.Sprintf("list service instances error, %v", err))
			return
		}
		logrus.Error("list service instances error:", err)
		httputil.ReturnError(r, w, 500, fmt.Sprintf("list service instances error, %v", err))
		return
	}
	httputil.ReturnSuccess(r, w, pods)
}

// ListServiceInstanceContainers 获取组件实例容器列表
func (t *TenantEnvStruct) ListServiceInstanceContainers(w http.ResponseWriter, r *http.Request) {
	tenantEnv := r.Context().Value(ctxutil.ContextKey("tenant_env")).(*dbmodel.TenantEnvs)
	service := r.Context().Value(ctxutil.ContextKey("service")).(*dbmodel.TenantEnvServices)
	instance := chi.URLParam(r, "instance_id")
	pods, err := handler.GetServiceManager().ListServiceInstanceContainers(service, tenantEnv.Namespace, instance)
	if err != nil {
		if err.Error() == gorm.ErrRecordNotFound.Error() {
			logrus.Error("record not found:", err)
			httputil.ReturnError(r, w, 404, fmt.Sprintf("list service instance containers error, %v", err))
			return
		}
		logrus.Error("list service instance containers error:", err)
		httputil.ReturnError(r, w, 500, fmt.Sprintf("list service instance containers error, %v", err))
		return
	}
	httputil.ReturnSuccess(r, w, pods)
}

// ListServiceInstanceContainerOptions 获取组件实例容器选项列表
func (t *TenantEnvStruct) ListServiceInstanceContainerOptions(w http.ResponseWriter, r *http.Request) {
	tenantEnv := r.Context().Value(ctxutil.ContextKey("tenant_env")).(*dbmodel.TenantEnvs)
	service := r.Context().Value(ctxutil.ContextKey("service")).(*dbmodel.TenantEnvServices)
	pods, err := handler.GetServiceManager().ListServiceInstanceContainerOptions(service, tenantEnv.Namespace)
	if err != nil {
		if err.Error() == gorm.ErrRecordNotFound.Error() {
			logrus.Error("record not found:", err)
			httputil.ReturnError(r, w, 404, fmt.Sprintf("get service instance containers tree error, %v", err))
			return
		}
		logrus.Error("get service instance contianers tree error:", err)
		httputil.ReturnError(r, w, 500, fmt.Sprintf("get service instance containers tree error, %v", err))
		return
	}
	httputil.ReturnSuccess(r, w, pods)
}

// ListServiceInstanceEvents 获取组件实例事件列表
func (t *TenantEnvStruct) ListServiceInstanceEvents(w http.ResponseWriter, r *http.Request) {
	tenantEnv := r.Context().Value(ctxutil.ContextKey("tenant_env")).(*dbmodel.TenantEnvs)
	instance := chi.URLParam(r, "instance_id")
	pods, err := handler.GetServiceManager().ListServiceInstanceEvents(tenantEnv.Namespace, instance)
	if err != nil {
		if err.Error() == gorm.ErrRecordNotFound.Error() {
			logrus.Error("record not found:", err)
			httputil.ReturnError(r, w, 404, fmt.Sprintf("get pods error, %v", err))
			return
		}
		logrus.Error("get pods error:", err)
		httputil.ReturnError(r, w, 500, fmt.Sprintf("get pods error, %v", err))
		return
	}
	httputil.ReturnSuccess(r, w, pods)
}

// Probe probe
func (t *TenantEnvStruct) Probe(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "PUT":
		t.UpdateProbe(w, r)
	case "DELETE":
		t.DeleteProbe(w, r)
	case "POST":
		t.AddProbe(w, r)
	}
}

// AddProbe add probe
// swagger:operation POST /v2/tenants/{tenant_name}/envs/{tenant_env_name}/services/{service_alias}/probe v2 addProbe
//
// 增加应用探针
//
// add probe
//
// ---
// consumes:
// - application/json
// - application/x-protobuf
//
// produces:
// - application/json
// - application/xml
//
// responses:
//
//	default:
//	  schema:
//	    "$ref": "#/responses/commandResponse"
//	  description: 统一返回格式
func (t *TenantEnvStruct) AddProbe(w http.ResponseWriter, r *http.Request) {
	serviceID := r.Context().Value(ctxutil.ContextKey("service_id")).(string)
	var tsp api_model.ServiceProbe
	if ok := httputil.ValidatorRequestStructAndErrorResponse(r, w, &tsp, nil); !ok {
		return
	}
	var tspD dbmodel.TenantEnvServiceProbe
	tspD.ServiceID = serviceID
	tspD.Cmd = tsp.Cmd
	tspD.FailureThreshold = tsp.FailureThreshold
	tspD.HTTPHeader = tsp.HTTPHeader
	tspD.InitialDelaySecond = tsp.InitialDelaySecond
	tspD.IsUsed = &tsp.IsUsed
	tspD.Mode = tsp.Mode
	tspD.Path = tsp.Path
	tspD.PeriodSecond = tsp.PeriodSecond
	tspD.Port = tsp.Port
	tspD.ProbeID = tsp.ProbeID
	tspD.Scheme = tsp.Scheme
	tspD.SuccessThreshold = tsp.SuccessThreshold
	tspD.TimeoutSecond = tsp.TimeoutSecond
	tspD.FailureAction = tsp.FailureAction
	//注意端口问题
	if err := handler.GetServiceManager().ServiceProbe(&tspD, "add"); err != nil {
		httputil.ReturnError(r, w, 500, fmt.Sprintf("add service probe error, %v", err))
		return
	}
	httputil.ReturnSuccess(r, w, nil)
}

// UpdateProbe update probe
// swagger:operation PUT /v2/tenants/{tenant_name}/envs/{tenant_env_name}/services/{service_alias}/probe v2 updateProbe
//
// 更新应用探针信息, *注意此处为全量更新
//
// update probe
//
// ---
// consumes:
// - application/json
// - application/x-protobuf
//
// produces:
// - application/json
// - application/xml
//
// responses:
//
//	default:
//	  schema:
//	    "$ref": "#/responses/commandResponse"
//	  description: 统一返回格式
func (t *TenantEnvStruct) UpdateProbe(w http.ResponseWriter, r *http.Request) {
	serviceID := r.Context().Value(ctxutil.ContextKey("service_id")).(string)
	var tsp api_model.ServiceProbe
	if ok := httputil.ValidatorRequestStructAndErrorResponse(r, w, &tsp, nil); !ok {
		return
	}
	var tspD dbmodel.TenantEnvServiceProbe
	tspD.ServiceID = serviceID
	tspD.Cmd = tsp.Cmd
	tspD.FailureThreshold = tsp.FailureThreshold
	tspD.HTTPHeader = tsp.HTTPHeader
	tspD.InitialDelaySecond = tsp.InitialDelaySecond
	tspD.IsUsed = &tsp.IsUsed
	tspD.Mode = tsp.Mode
	tspD.Path = tsp.Path
	tspD.PeriodSecond = tsp.PeriodSecond
	tspD.Port = tsp.Port
	tspD.ProbeID = tsp.ProbeID
	tspD.Scheme = tsp.Scheme
	tspD.SuccessThreshold = tsp.SuccessThreshold
	tspD.TimeoutSecond = tsp.TimeoutSecond
	//注意端口问题
	if err := handler.GetServiceManager().ServiceProbe(&tspD, "update"); err != nil {
		if err.Error() == gorm.ErrRecordNotFound.Error() {
			httputil.ReturnError(r, w, 404, fmt.Sprintf("update prob error, %v", err))
			return
		}
		httputil.ReturnError(r, w, 500, fmt.Sprintf("update service probe error, %v", err))
		return
	}
	httputil.ReturnSuccess(r, w, nil)
}

// DeleteProbe delete probe
// swagger:operation DELETE /v2/tenants/{tenant_name}/envs/{tenant_env_name}/services/{service_alias}/probe v2 deleteProbe
//
// 删除应用探针
//
// delete probe
//
// ---
// consumes:
// - application/json
// - application/x-protobuf
//
// produces:
// - application/json
// - application/xml
//
// responses:
//
//	default:
//	  schema:
//	    "$ref": "#/responses/commandResponse"
//	  description: 统一返回格式
func (t *TenantEnvStruct) DeleteProbe(w http.ResponseWriter, r *http.Request) {
	serviceID := r.Context().Value(ctxutil.ContextKey("service_id")).(string)
	var tsp api_model.ServiceProbe
	if ok := httputil.ValidatorRequestStructAndErrorResponse(r, w, &tsp, nil); !ok {
		return
	}
	var tspD dbmodel.TenantEnvServiceProbe
	tspD.ServiceID = serviceID
	tspD.ProbeID = tsp.ProbeID
	//注意端口问题
	if err := handler.GetServiceManager().ServiceProbe(&tspD, "delete"); err != nil {
		if err.Error() == gorm.ErrRecordNotFound.Error() {
			httputil.ReturnError(r, w, 404, fmt.Sprintf("delete prob error, %v", err))
			return
		}
		httputil.ReturnError(r, w, 500, fmt.Sprintf("delete service probe error, %v", err))
		return
	}
	httputil.ReturnSuccess(r, w, nil)
}

// Port Port
func (t *TenantEnvStruct) Port(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "PUT":
		t.UpdatePort(w, r)
	case "DELETE":
		t.DeletePort(w, r)
	case "POST":
		t.AddPort(w, r)
	}
}

// AddPort add port
func (t *TenantEnvStruct) AddPort(w http.ResponseWriter, r *http.Request) {
}

// DeletePort delete port
func (t *TenantEnvStruct) DeletePort(w http.ResponseWriter, r *http.Request) {
}

// UpdatePort Update port
func (t *TenantEnvStruct) UpdatePort(w http.ResponseWriter, r *http.Request) {
}

// SingleTenantEnvResources SingleTenantEnvResources
// swagger:operation GET /v2/tenants/{tenant_name}/envs/{tenant_env_name}/resources v2 singletenantEnvResources
//
// 指定租户资源使用情况
//
// get tenant env resources
//
// ---
// produces:
// - application/json
// - application/xml
// parameters:
//   - name: tenant_env_name
//     in: path
//     description: tenant env name
//     required: true
//     type: string
//
// responses:
//
//	default:
//	  schema:
//	    "$ref": "#/responses/commandResponse"
//	  description: 统一返回格式
func (t *TenantEnvStruct) SingleTenantEnvResources(w http.ResponseWriter, r *http.Request) {
	tenantEnvID := r.Context().Value(ctxutil.ContextKey("tenant_env_id")).(string)
	//11ms
	services, err := handler.GetServiceManager().GetService(tenantEnvID)
	if err != nil {
		msg := httputil.ResponseBody{
			Msg: fmt.Sprintf("get service error, %v", err),
		}
		httputil.Return(r, w, 500, msg)
	}
	//19ms
	statsInfo, _ := handler.GetTenantEnvManager().StatsMemCPU(services)
	//900ms
	statsInfo.UUID = tenantEnvID
	httputil.ReturnSuccess(r, w, statsInfo)
}

// GetSupportProtocols GetSupportProtocols
// swagger:operation GET /v2/tenants/{tenant_name}/envs/{tenant_env_name}/protocols v2 getSupportProtocols
//
// 获取当前数据中心支持的protocols
//
// get region protocols
//
// ---
// produces:
// - application/json
// - application/xml
// parameters:
//   - name: tenant_env_name
//     in: path
//     description: tenant env name
//     required: true
//     type: string
//
// responses:
//
//	default:
//	  schema:
//	    "$ref": "#/responses/commandResponse"
//	  description: 统一返回格式
func (t *TenantEnvStruct) GetSupportProtocols(w http.ResponseWriter, r *http.Request) {
	rps, err := handler.GetTenantEnvManager().GetProtocols()
	if err != nil {
		err.Handle(r, w)
		return
	}
	httputil.ReturnSuccess(r, w, rps)
}

// TransPlugins transPlugins
// swagger:operation POST /v2/tenants/{tenant_name}/envs/{tenant_env_name}/transplugins v2 transPlugins
//
// 安装云帮默认plugins
//
// trans plugins
//
// ---
// produces:
// - application/json
// - application/xml
// parameters:
//   - name: tenant_env_name
//     in: path
//     description: tenant env name
//     required: true
//     type: string
//
// responses:
//
//	default:
//	  schema:
//	    "$ref": "#/responses/commandResponse"
//	  description: 统一返回格式
func (t *TenantEnvStruct) TransPlugins(w http.ResponseWriter, r *http.Request) {
	var tps api_model.TransPlugins
	ok := httputil.ValidatorRequestStructAndErrorResponse(r, w, &tps.Body, nil)
	if !ok {
		return
	}
	tenantEnvID := r.Context().Value(ctxutil.ContextKey("tenant_env_id")).(string)
	tenantEnvName := r.Context().Value(ctxutil.ContextKey("tenant_env_name")).(string)
	rc := make(map[string]string)
	err := handler.GetTenantEnvManager().TransPlugins(tenantEnvID, tenantEnvName, tps.Body.FromTenantEnvName, tps.Body.PluginsID)
	if err != nil {
		err.Handle(r, w)
		return
	}
	rc["result"] = "success"
	httputil.ReturnSuccess(r, w, rc)
}

// CheckResourceName checks the resource name.
func (t *TenantEnvStruct) CheckResourceName(w http.ResponseWriter, r *http.Request) {
	var req api_model.CheckResourceNameReq
	if !httputil.ValidatorRequestStructAndErrorResponse(r, w, &req, nil) {
		return
	}

	tenantEnv := r.Context().Value(ctxutil.ContextKey("tenant_env")).(*dbmodel.TenantEnvs)

	res, err := handler.GetTenantEnvManager().CheckResourceName(r.Context(), tenantEnv.UUID, &req)
	if err != nil {
		httputil.ReturnBcodeError(r, w, err)
		return
	}

	httputil.ReturnSuccess(r, w, res)
}
