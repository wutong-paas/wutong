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

package controller

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/go-chi/chi"
	"github.com/jinzhu/gorm"
	"github.com/pquerna/ffjson/ffjson"
	"github.com/sirupsen/logrus"
	"github.com/wutong-paas/wutong/api/handler"
	api_model "github.com/wutong-paas/wutong/api/model"
	ctxutil "github.com/wutong-paas/wutong/api/util/ctx"
	"github.com/wutong-paas/wutong/db"
	dbmodel "github.com/wutong-paas/wutong/db/model"
	"github.com/wutong-paas/wutong/event"
	"github.com/wutong-paas/wutong/util"
	validator "github.com/wutong-paas/wutong/util/govalidator"
	httputil "github.com/wutong-paas/wutong/util/http"
	"github.com/wutong-paas/wutong/worker/discover/model"
)

// StartService StartService
// swagger:operation POST /v2/tenants/{tenant_name}/envs/{tenant_env_name}/services/{service_alias}/start  v2 startService
//
// 启动应用
//
// start service
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
func (t *TenantEnvStruct) StartService(w http.ResponseWriter, r *http.Request) {
	tenantEnvID := r.Context().Value(ctxutil.ContextKey("tenant_env_id")).(string)
	serviceID := r.Context().Value(ctxutil.ContextKey("service_id")).(string)

	tenantEnv := r.Context().Value(ctxutil.ContextKey("tenant_env")).(*dbmodel.TenantEnvs)
	service := r.Context().Value(ctxutil.ContextKey("service")).(*dbmodel.TenantEnvServices)
	sEvent := r.Context().Value(ctxutil.ContextKey("event")).(*dbmodel.ServiceEvent)
	if service.Kind != "third_party" {
		if err := handler.CheckTenantEnvResource(r.Context(), tenantEnv, service.Replicas*service.ContainerMemory); err != nil {
			httputil.ReturnResNotEnough(r, w, sEvent.EventID, err.Error())
			return
		}
	}

	startStopStruct := &api_model.StartStopStruct{
		TenantEnvID: tenantEnvID,
		ServiceID:   serviceID,
		EventID:     sEvent.EventID,
		TaskType:    "start",
	}
	if err := handler.GetServiceManager().StartStopService(startStopStruct); err != nil {
		httputil.ReturnError(r, w, 500, "get service info error.")
		return
	}
	httputil.ReturnSuccess(r, w, sEvent)
}

// StopService StopService
// swagger:operation POST /v2/tenants/{tenant_name}/envs/{tenant_env_name}/services/{service_alias}/stop v2 stopService
//
// 关闭应用
//
// stop service
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
func (t *TenantEnvStruct) StopService(w http.ResponseWriter, r *http.Request) {
	tenantEnvID := r.Context().Value(ctxutil.ContextKey("tenant_env_id")).(string)
	serviceID := r.Context().Value(ctxutil.ContextKey("service_id")).(string)
	sEvent := r.Context().Value(ctxutil.ContextKey("event")).(*dbmodel.ServiceEvent)
	//save event
	// defer event.GetManager().Close()
	defer event.CloseLogger(sEvent.EventID)
	startStopStruct := &api_model.StartStopStruct{
		TenantEnvID: tenantEnvID,
		ServiceID:   serviceID,
		EventID:     sEvent.EventID,
		TaskType:    "stop",
	}
	if err := handler.GetServiceManager().StartStopService(startStopStruct); err != nil {
		httputil.ReturnError(r, w, 500, "get service info error.")
		return
	}
	httputil.ReturnSuccess(r, w, sEvent)
}

// RestartService RestartService
// swagger:operation POST /v2/tenants/{tenant_name}/envs/{tenant_env_name}/services/{service_alias}/restart v2 restartService
//
// 重启应用
//
// restart service
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
func (t *TenantEnvStruct) RestartService(w http.ResponseWriter, r *http.Request) {
	tenantEnvID := r.Context().Value(ctxutil.ContextKey("tenant_env_id")).(string)
	serviceID := r.Context().Value(ctxutil.ContextKey("service_id")).(string)
	sEvent := r.Context().Value(ctxutil.ContextKey("event")).(*dbmodel.ServiceEvent)
	// defer event.GetManager().Close()
	event.CloseLogger(sEvent.EventID)
	startStopStruct := &api_model.StartStopStruct{
		TenantEnvID: tenantEnvID,
		ServiceID:   serviceID,
		EventID:     sEvent.EventID,
		TaskType:    "restart",
	}

	curStatus := t.StatusCli.GetStatus(serviceID)
	if curStatus == "closed" {
		startStopStruct.TaskType = "start"
	}

	tenantEnv := r.Context().Value(ctxutil.ContextKey("tenant_env")).(*dbmodel.TenantEnvs)
	service := r.Context().Value(ctxutil.ContextKey("service")).(*dbmodel.TenantEnvServices)
	if err := handler.CheckTenantEnvResource(r.Context(), tenantEnv, service.Replicas*service.ContainerMemory); err != nil {
		httputil.ReturnResNotEnough(r, w, sEvent.EventID, err.Error())
		return
	}

	if err := handler.GetServiceManager().StartStopService(startStopStruct); err != nil {
		httputil.ReturnError(r, w, 500, "get service info error.")
		return
	}
	httputil.ReturnSuccess(r, w, sEvent)
}

// VerticalService VerticalService
// swagger:operation PUT /v2/tenants/{tenant_name}/envs/{tenant_env_name}/services/{service_alias}/vertical v2 verticalService
//
// 应用垂直伸缩
//
// service vertical
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
func (t *TenantEnvStruct) VerticalService(w http.ResponseWriter, r *http.Request) {
	rules := validator.MapData{
		"container_cpu":    []string{"required"},
		"container_memory": []string{"required"},
	}
	data, ok := httputil.ValidatorRequestMapAndErrorResponse(r, w, rules, nil)
	if !ok {
		return
	}
	tenantEnvID := r.Context().Value(ctxutil.ContextKey("tenant_env_id")).(string)
	serviceID := r.Context().Value(ctxutil.ContextKey("service_id")).(string)
	sEvent := r.Context().Value(ctxutil.ContextKey("event")).(*dbmodel.ServiceEvent)
	var requestCPU, limitCPU, gpuLimit, requestMemory, limitMemory *int
	var gpuTypeSet *string
	if reqCPU, ok := data["container_request_cpu"].(float64); ok {
		reqCPUInt := int(reqCPU)
		requestCPU = &reqCPUInt
	}
	if cpu, ok := data["container_cpu"].(float64); ok {
		cpuInt := int(cpu)
		limitCPU = &cpuInt
	}
	if limitCPU == nil || limitCPU == util.Ptr(0) {
		limitCPU = util.Ptr(2000)
	}
	if reqMemory, ok := data["container_request_memory"].(float64); ok {
		reqMemoryInt := int(reqMemory)
		requestMemory = &reqMemoryInt
	}
	if memory, ok := data["container_memory"].(float64); ok {
		memoryInt := int(memory)
		limitMemory = &memoryInt
	}
	if limitMemory == nil || *limitMemory == 0 {
		limitMemory = util.Ptr(512)
	}
	if gpuType, ok := data["container_gpu_type"].(string); ok {
		gpuTypeSet = &gpuType
	}
	if gpu, ok := data["container_gpu"].(float64); ok {
		gpuInt := int(gpu)
		gpuLimit = &gpuInt
	}
	tenantEnv := r.Context().Value(ctxutil.ContextKey("tenant_env")).(*dbmodel.TenantEnvs)
	service := r.Context().Value(ctxutil.ContextKey("service")).(*dbmodel.TenantEnvServices)
	if limitMemory != nil {
		if err := handler.CheckTenantEnvResource(r.Context(), tenantEnv, service.Replicas*(*limitMemory)); err != nil {
			httputil.ReturnResNotEnough(r, w, sEvent.EventID, err.Error())
			return
		}
	}
	verticalTask := &model.VerticalScalingTaskBody{
		TenantEnvID:            tenantEnvID,
		ServiceID:              serviceID,
		EventID:                sEvent.EventID,
		ContainerRequestCPU:    requestCPU,
		ContainerCPU:           limitCPU,
		ContainerRequestMemory: requestMemory,
		ContainerMemory:        limitMemory,
		ContainerGPUType:       gpuTypeSet,
		ContainerGPU:           gpuLimit,
	}
	if err := handler.GetServiceManager().ServiceVertical(r.Context(), verticalTask); err != nil {
		httputil.ReturnError(r, w, 500, fmt.Sprintf("service vertical error. %v", err))
		return
	}
	httputil.ReturnSuccess(r, w, sEvent)
}

// HorizontalService HorizontalService
// swagger:operation PUT /v2/tenants/{tenant_name}/envs/{tenant_env_name}/services/{service_alias}/horizontal v2 horizontalService
//
// 应用水平伸缩
//
// service horizontal
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
func (t *TenantEnvStruct) HorizontalService(w http.ResponseWriter, r *http.Request) {
	rules := validator.MapData{
		"node_num": []string{"required"},
	}
	data, ok := httputil.ValidatorRequestMapAndErrorResponse(r, w, rules, nil)
	if !ok {
		return
	}
	tenantEnvID := r.Context().Value(ctxutil.ContextKey("tenant_env_id")).(string)
	serviceID := r.Context().Value(ctxutil.ContextKey("service_id")).(string)
	sEvent := r.Context().Value(ctxutil.ContextKey("event")).(*dbmodel.ServiceEvent)
	replicas := int32(data["node_num"].(float64))

	tenantEnv := r.Context().Value(ctxutil.ContextKey("tenant_env")).(*dbmodel.TenantEnvs)
	service := r.Context().Value(ctxutil.ContextKey("service")).(*dbmodel.TenantEnvServices)
	if err := handler.CheckTenantEnvResource(r.Context(), tenantEnv, service.ContainerMemory*int(replicas)); err != nil {
		httputil.ReturnResNotEnough(r, w, sEvent.EventID, err.Error())
		return
	}

	horizontalTask := &model.HorizontalScalingTaskBody{
		TenantEnvID: tenantEnvID,
		ServiceID:   serviceID,
		EventID:     sEvent.EventID,
		Username:    sEvent.UserName,
		Replicas:    replicas,
	}

	if err := handler.GetServiceManager().ServiceHorizontal(horizontalTask); err != nil {
		httputil.ReturnBcodeError(r, w, err)
		return
	}
	httputil.ReturnSuccess(r, w, sEvent)
}

// BuildService BuildService
// swagger:operation POST /v2/tenants/{tenant_name}/envs/{tenant_env_name}/services/{service_alias}/build v2 serviceBuild
//
// 应用构建
//
// service build
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
func (t *TenantEnvStruct) BuildService(w http.ResponseWriter, r *http.Request) {
	var build api_model.ComponentBuildReq
	ok := httputil.ValidatorRequestStructAndErrorResponse(r, w, &build, nil)
	if !ok {
		return
	}
	serviceID := r.Context().Value(ctxutil.ContextKey("service_id")).(string)
	tenantEnvName := r.Context().Value(ctxutil.ContextKey("tenant_env_name")).(string)
	build.TenantEnvName = tenantEnvName
	build.EventID = r.Context().Value(ctxutil.ContextKey("event_id")).(string)
	if build.ServiceID != serviceID {
		httputil.ReturnError(r, w, 400, "build service id is failure")
		return
	}

	tenantEnv := r.Context().Value(ctxutil.ContextKey("tenant_env")).(*dbmodel.TenantEnvs)
	service := r.Context().Value(ctxutil.ContextKey("service")).(*dbmodel.TenantEnvServices)
	if err := handler.CheckTenantEnvResource(r.Context(), tenantEnv, service.Replicas*service.ContainerMemory); err != nil {
		httputil.ReturnResNotEnough(r, w, build.EventID, err.Error())
		return
	}

	res, err := handler.GetOperationHandler().Build(&build)
	if err != nil {
		httputil.ReturnBcodeError(r, w, err)
		return
	}
	httputil.ReturnSuccess(r, w, res)
}

// BuildList BuildList
func (t *TenantEnvStruct) BuildList(w http.ResponseWriter, r *http.Request) {
	serviceID := r.Context().Value(ctxutil.ContextKey("service_id")).(string)

	resp, err := handler.GetServiceManager().ListVersionInfo(serviceID)

	if err != nil {
		logrus.Error("get version info error", err.Error())
		httputil.ReturnError(r, w, 500, fmt.Sprintf("get version info erro, %v", err))
		return
	}
	httputil.ReturnSuccess(r, w, resp)
}

// BuildVersionIsExist -
func (t *TenantEnvStruct) BuildVersionIsExist(w http.ResponseWriter, r *http.Request) {
	statusMap := make(map[string]bool)
	serviceID := r.Context().Value(ctxutil.ContextKey("service_id")).(string)
	buildVersion := chi.URLParam(r, "build_version")
	_, err := db.GetManager().VersionInfoDao().GetVersionByDeployVersion(buildVersion, serviceID)
	if err != nil && err != gorm.ErrRecordNotFound {
		httputil.ReturnError(r, w, 500, fmt.Sprintf("get build version status erro, %v", err))
		return
	}
	if err == gorm.ErrRecordNotFound {
		statusMap["status"] = false
	} else {
		statusMap["status"] = true
	}
	httputil.ReturnSuccess(r, w, statusMap)

}

// DeleteBuildVersion -
func (t *TenantEnvStruct) DeleteBuildVersion(w http.ResponseWriter, r *http.Request) {
	serviceID := r.Context().Value(ctxutil.ContextKey("service_id")).(string)
	buildVersion := chi.URLParam(r, "build_version")
	val, err := db.GetManager().VersionInfoDao().GetVersionByDeployVersion(buildVersion, serviceID)
	if err != nil && err != gorm.ErrRecordNotFound {
		httputil.ReturnError(r, w, 500, fmt.Sprintf("delete build version erro, %v", err))
		return
	}
	if err == gorm.ErrRecordNotFound {

	} else {
		if val.DeliveredType == "slug" && val.FinalStatus == "success" {
			if err := os.Remove(val.DeliveredPath); err != nil {
				httputil.ReturnError(r, w, 500, fmt.Sprintf("delete build version erro, %v", err))
				return

			}
			if err := db.GetManager().VersionInfoDao().DeleteVersionInfo(val); err != nil {
				httputil.ReturnError(r, w, 500, fmt.Sprintf("delete build version erro, %v", err))
				return

			}
		}
		if val.FinalStatus == "failure" {
			if err := db.GetManager().VersionInfoDao().DeleteVersionInfo(val); err != nil {
				httputil.ReturnError(r, w, 500, fmt.Sprintf("delete build version erro, %v", err))
				return
			}
		}
		if val.DeliveredType == "image" {
			if err := db.GetManager().VersionInfoDao().DeleteVersionInfo(val); err != nil {
				httputil.ReturnError(r, w, 500, fmt.Sprintf("delete build version erro, %v", err))
				return
			}
		}
	}
	httputil.ReturnSuccess(r, w, nil)

}

// UpdateBuildVersion -
func (t *TenantEnvStruct) UpdateBuildVersion(w http.ResponseWriter, r *http.Request) {
	var build api_model.UpdateBuildVersionReq
	ok := httputil.ValidatorRequestStructAndErrorResponse(r, w, &build, nil)
	if !ok {
		return
	}
	serviceID := r.Context().Value(ctxutil.ContextKey("service_id")).(string)
	buildVersion := chi.URLParam(r, "build_version")
	versionInfo, err := db.GetManager().VersionInfoDao().GetVersionByDeployVersion(buildVersion, serviceID)
	if err != nil {
		httputil.ReturnError(r, w, 500, fmt.Sprintf("update build version info error, %v", err))
		return
	}
	versionInfo.PlanVersion = build.PlanVersion
	err = db.GetManager().VersionInfoDao().UpdateModel(versionInfo)
	if err != nil {
		httputil.ReturnError(r, w, 500, fmt.Sprintf("update build version info error, %v", err))
		return
	}
	httputil.ReturnSuccess(r, w, nil)
}

// BuildVersionInfo -
func (t *TenantEnvStruct) BuildVersionInfo(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "DELETE":
		t.DeleteBuildVersion(w, r)
	case "GET":
		t.BuildVersionIsExist(w, r)
	case "PUT":
		t.UpdateBuildVersion(w, r)
	}

}

// GetDeployVersion GetDeployVersion by service
func (t *TenantEnvStruct) GetDeployVersion(w http.ResponseWriter, r *http.Request) {
	service := r.Context().Value(ctxutil.ContextKey("service")).(*dbmodel.TenantEnvServices)
	version, err := db.GetManager().VersionInfoDao().GetVersionByDeployVersion(service.DeployVersion, service.ServiceID)
	if err != nil && err != gorm.ErrRecordNotFound {
		httputil.ReturnError(r, w, 500, fmt.Sprintf("get build version status erro, %v", err))
		return
	}
	if err == gorm.ErrRecordNotFound {
		httputil.ReturnError(r, w, 404, "build version do not exist")
		return
	}
	httputil.ReturnSuccess(r, w, version)
}

// GetManyDeployVersion GetDeployVersion by some service id
func (t *TenantEnvStruct) GetManyDeployVersion(w http.ResponseWriter, r *http.Request) {
	rules := validator.MapData{
		"service_ids": []string{"required"},
	}
	data, ok := httputil.ValidatorRequestMapAndErrorResponse(r, w, rules, nil)
	if !ok {
		return
	}
	serviceIDs, ok := data["service_ids"].([]interface{})
	if !ok {
		httputil.ReturnError(r, w, 400, "service ids must be a array")
		return
	}
	var list []string
	for _, s := range serviceIDs {
		list = append(list, s.(string))
	}
	services, err := db.GetManager().TenantEnvServiceDao().GetServiceByIDs(list)
	if err != nil {
		httputil.ReturnError(r, w, 500, err.Error())
		return
	}
	var versionList []*dbmodel.VersionInfo
	for _, service := range services {
		version, err := db.GetManager().VersionInfoDao().GetVersionByDeployVersion(service.DeployVersion, service.ServiceID)
		if err != nil && err != gorm.ErrRecordNotFound {
			httputil.ReturnError(r, w, 500, fmt.Sprintf("get build version status erro, %v", err))
			return
		}
		versionList = append(versionList, version)
	}
	httputil.ReturnSuccess(r, w, versionList)
}

// DeployService DeployService
func (t *TenantEnvStruct) DeployService(w http.ResponseWriter, r *http.Request) {
	logrus.Debugf("trans deploy service")
	w.Write([]byte("deploy service"))
}

// UpgradeService UpgradeService
// swagger:operation POST /v2/tenants/{tenant_name}/envs/{tenant_env_name}/services/{service_alias}/upgrade v2 upgradeService
//
// 升级应用
//
// upgrade service
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
func (t *TenantEnvStruct) UpgradeService(w http.ResponseWriter, r *http.Request) {
	var upgradeRequest api_model.ComponentUpgradeReq
	ok := httputil.ValidatorRequestStructAndErrorResponse(r, w, &upgradeRequest, nil)
	if !ok {
		logrus.Errorf("start operation validate request body failure")
		return
	}
	upgradeRequest.EventID = r.Context().Value(ctxutil.ContextKey("event_id")).(string)
	serviceID := r.Context().Value(ctxutil.ContextKey("service_id")).(string)
	if upgradeRequest.ServiceID != serviceID {
		httputil.ReturnError(r, w, 400, "upgrade service id failure")
		return
	}

	tenantEnv := r.Context().Value(ctxutil.ContextKey("tenant_env")).(*dbmodel.TenantEnvs)
	service := r.Context().Value(ctxutil.ContextKey("service")).(*dbmodel.TenantEnvServices)
	if service.Kind != "third_party" {
		if err := handler.CheckTenantEnvResource(r.Context(), tenantEnv, service.Replicas*service.ContainerMemory); err != nil {
			httputil.ReturnResNotEnough(r, w, upgradeRequest.EventID, err.Error())
			return
		}
	}

	res, err := handler.GetOperationHandler().Upgrade(&upgradeRequest)
	if err != nil {
		httputil.ReturnBcodeError(r, w, err)
		return
	}
	httputil.ReturnSuccess(r, w, res)
}

// CheckCode CheckCode
// swagger:operation POST /v2/tenants/{tenant_name}/envs/{tenant_env_name}/code-check v2 checkCode
//
// 应用代码检测
//
// check  code
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
func (t *TenantEnvStruct) CheckCode(w http.ResponseWriter, r *http.Request) {

	var ccs api_model.CheckCodeStruct
	ok := httputil.ValidatorRequestStructAndErrorResponse(r, w, &ccs.Body, nil)
	if !ok {
		return
	}
	if ccs.Body.TenantEnvID == "" {
		tenantEnvID := r.Context().Value(ctxutil.ContextKey("tenant_env_id")).(string)
		ccs.Body.TenantEnvID = tenantEnvID
	}
	ccs.Body.Action = "code_check"
	if err := handler.GetServiceManager().CodeCheck(&ccs); err != nil {
		httputil.ReturnError(r, w, 500, fmt.Sprintf("task code check error,%v", err))
		return
	}
	httputil.ReturnSuccess(r, w, nil)
}

// RollBack RollBack
// swagger:operation Post /v2/tenants/{tenant_name}/envs/{tenant_env_name}/services/{service_alias}/rollback v2 rollback
//
// 应用版本回滚
//
// service rollback
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
func (t *TenantEnvStruct) RollBack(w http.ResponseWriter, r *http.Request) {
	var rollbackRequest api_model.RollbackInfoRequestStruct
	ok := httputil.ValidatorRequestStructAndErrorResponse(r, w, &rollbackRequest, nil)
	if !ok {
		logrus.Errorf("start operation validate request body failure")
		return
	}
	serviceID := r.Context().Value(ctxutil.ContextKey("service_id")).(string)
	if rollbackRequest.ServiceID != serviceID {
		httputil.ReturnError(r, w, 400, "rollback service id failure")
		return
	}
	rollbackRequest.EventID = r.Context().Value(ctxutil.ContextKey("event_id")).(string)

	tenantEnv := r.Context().Value(ctxutil.ContextKey("tenant_env")).(*dbmodel.TenantEnvs)
	service := r.Context().Value(ctxutil.ContextKey("service")).(*dbmodel.TenantEnvServices)
	if err := handler.CheckTenantEnvResource(r.Context(), tenantEnv, service.Replicas*service.ContainerMemory); err != nil {
		httputil.ReturnResNotEnough(r, w, rollbackRequest.EventID, err.Error())
		return
	}

	re := handler.GetOperationHandler().RollBack(rollbackRequest)
	httputil.ReturnSuccess(r, w, re)
}

type limitMemory struct {
	LimitMemory int `json:"limit_memory"`
}

// LimitTenantEnvMemory -
func (t *TenantEnvStruct) LimitTenantEnvMemory(w http.ResponseWriter, r *http.Request) {
	var lm limitMemory
	body, err := io.ReadAll(r.Body)

	if err != nil {
		httputil.ReturnError(r, w, 500, err.Error())
		return
	}
	err = ffjson.Unmarshal(body, &lm)
	if err != nil {
		httputil.ReturnError(r, w, 500, err.Error())
		return
	}

	tenantEnvID := r.Context().Value(ctxutil.ContextKey("tenant_env_id")).(string)
	tenantEnv, err := db.GetManager().TenantEnvDao().GetTenantEnvByUUID(tenantEnvID)
	if err != nil {
		httputil.ReturnError(r, w, 400, err.Error())
		return
	}
	tenantEnv.LimitMemory = lm.LimitMemory
	if err := db.GetManager().TenantEnvDao().UpdateModel(tenantEnv); err != nil {
		httputil.ReturnError(r, w, 500, err.Error())
	}
	httputil.ReturnSuccess(r, w, "success!")

}

// SourcesInfo -
type SourcesInfo struct {
	TenantEnvID     string `json:"tenant_env_id"`
	AvailableMemory int    `json:"available_memory"`
	Status          bool   `json:"status"`
	MemTotal        int    `json:"mem_total"`
	MemUsed         int    `json:"mem_used"`
	CPUTotal        int    `json:"cpu_total"`
	CPUUsed         int    `json:"cpu_used"`
}

// TenantEnvResourcesStatus tenant env resources status
func (t *TenantEnvStruct) TenantEnvResourcesStatus(w http.ResponseWriter, r *http.Request) {

	tenantEnvID := r.Context().Value(ctxutil.ContextKey("tenant_env_id")).(string)
	tenantEnv, err := db.GetManager().TenantEnvDao().GetTenantEnvByUUID(tenantEnvID)
	if err != nil {
		httputil.ReturnError(r, w, 400, err.Error())
		return
	}
	//11ms
	services, err := handler.GetServiceManager().GetService(tenantEnv.UUID)
	if err != nil {
		msg := httputil.ResponseBody{
			Msg: fmt.Sprintf("get service error, %v", err),
		}
		httputil.Return(r, w, 500, msg)
		return
	}

	statsInfo, _ := handler.GetTenantEnvManager().StatsMemCPU(services)

	if tenantEnv.LimitMemory == 0 {
		sourcesInfo := SourcesInfo{
			TenantEnvID:     tenantEnvID,
			AvailableMemory: 0,
			Status:          true,
			MemTotal:        tenantEnv.LimitMemory,
			MemUsed:         statsInfo.MEM,
			CPUTotal:        0,
			CPUUsed:         statsInfo.CPU,
		}
		httputil.ReturnSuccess(r, w, sourcesInfo)
		return
	}
	if statsInfo.MEM >= tenantEnv.LimitMemory {
		sourcesInfo := SourcesInfo{
			TenantEnvID:     tenantEnvID,
			AvailableMemory: tenantEnv.LimitMemory - statsInfo.MEM,
			Status:          false,
			MemTotal:        tenantEnv.LimitMemory,
			MemUsed:         statsInfo.MEM,
			CPUTotal:        tenantEnv.LimitMemory / 4,
			CPUUsed:         statsInfo.CPU,
		}
		httputil.ReturnSuccess(r, w, sourcesInfo)
	} else {
		sourcesInfo := SourcesInfo{
			TenantEnvID:     tenantEnvID,
			AvailableMemory: tenantEnv.LimitMemory - statsInfo.MEM,
			Status:          true,
			MemTotal:        tenantEnv.LimitMemory,
			MemUsed:         statsInfo.MEM,
			CPUTotal:        tenantEnv.LimitMemory / 4,
			CPUUsed:         statsInfo.CPU,
		}
		httputil.ReturnSuccess(r, w, sourcesInfo)
	}
}

// GetServiceDeployInfo get service deploy info
func GetServiceDeployInfo(w http.ResponseWriter, r *http.Request) {
	tenantEnvID := r.Context().Value(ctxutil.ContextKey("tenant_env_id")).(string)
	serviceID := r.Context().Value(ctxutil.ContextKey("service_id")).(string)
	info, err := handler.GetServiceManager().GetServiceDeployInfo(tenantEnvID, serviceID)
	if err != nil {
		err.Handle(r, w)
		return
	}
	httputil.ReturnSuccess(r, w, info)
}

// Log -
func (t *TenantEnvStruct) Log(w http.ResponseWriter, r *http.Request) {
	component := r.Context().Value(ctxutil.ContextKey("service")).(*dbmodel.TenantEnvServices)
	podName := r.URL.Query().Get("podName")
	containerName := r.URL.Query().Get("containerName")
	follow, _ := strconv.ParseBool(r.URL.Query().Get("follow"))

	err := handler.GetServiceManager().Log(w, r, component, podName, containerName, follow)
	if err != nil {
		httputil.ReturnBcodeError(r, w, err)
		return
	}
}

// GetKubeConfig get kube config for developer
func (t *TenantEnvStruct) GetKubeConfig(w http.ResponseWriter, r *http.Request) {
	tenantEnvID := r.Context().Value(ctxutil.ContextKey("tenant_env_id")).(string)
	tenantEnv, err := db.GetManager().TenantEnvDao().GetTenantEnvByUUID(tenantEnvID)
	if err != nil {
		httputil.ReturnError(r, w, 400, err.Error())
		return
	}
	kubeConfig, err := handler.GetTenantEnvManager().GetKubeConfig(tenantEnv.Namespace)
	if err != nil {
		httputil.ReturnError(r, w, 400, err.Error())
		return
	}
	httputil.ReturnSuccess(r, w, kubeConfig)
}

// GetTenantEnvKubeResources get kube resources for tenantEnv
func (t *TenantEnvStruct) GetTenantEnvKubeResources(w http.ResponseWriter, r *http.Request) {
	var customSetting api_model.KubeResourceCustomSetting
	customSetting.Namespace = strings.Trim(r.URL.Query().Get("namespace"), " ")
	if customSetting.Namespace == "" {
		customSetting.Namespace = "default"
	}
	tenantEnv := r.Context().Value(ctxutil.ContextKey("tenant_env")).(*dbmodel.TenantEnvs)
	resources, err := handler.GetTenantEnvManager().GetKubeResources(tenantEnv.Namespace, tenantEnv.UUID, customSetting)
	if err != nil {
		httputil.ReturnError(r, w, 400, err.Error())
		return
	}
	httputil.ReturnSuccess(r, w, resources)
}

// GetServiceKubeResources get kube resources for component
func (t *TenantEnvStruct) GetServiceKubeResources(w http.ResponseWriter, r *http.Request) {
	var customSetting api_model.KubeResourceCustomSetting
	customSetting.Namespace = strings.Trim(r.URL.Query().Get("namespace"), " ")
	if customSetting.Namespace == "" {
		customSetting.Namespace = "default"
	}
	tenantEnv := r.Context().Value(ctxutil.ContextKey("tenant_env")).(*dbmodel.TenantEnvs)
	serviceID := r.Context().Value(ctxutil.ContextKey("service_id")).(string)
	resources, err := handler.GetServiceManager().GetKubeResources(tenantEnv.Namespace, serviceID, customSetting)
	if err != nil {
		httputil.ReturnError(r, w, 400, err.Error())
		return
	}
	httputil.ReturnSuccess(r, w, resources)
}

// CreateBackup create backup for service resource and data
func (t *TenantEnvStruct) CreateBackup(w http.ResponseWriter, r *http.Request) {
	tenantEnvID := r.Context().Value(ctxutil.ContextKey("tenant_env_id")).(string)
	serviceID := r.Context().Value(ctxutil.ContextKey("service_id")).(string)
	var req api_model.CreateBackupRequest
	ok := httputil.ValidatorRequestStructAndErrorResponse(r, w, &req, nil)
	if !ok {
		logrus.Errorf("start operation validate request body failure")
		return
	}
	err := handler.GetServiceManager().CreateBackup(tenantEnvID, serviceID, req)
	if err != nil {
		httputil.ReturnError(r, w, 500, err.Error())
		return
	}
	httputil.ReturnSuccess(r, w, nil)
}

// CreateBackupSchedule create backup schedule for service resource and data
func (t *TenantEnvStruct) CreateBackupSchedule(w http.ResponseWriter, r *http.Request) {
	tenantEnvID := r.Context().Value(ctxutil.ContextKey("tenant_env_id")).(string)
	serviceID := r.Context().Value(ctxutil.ContextKey("service_id")).(string)
	var req api_model.CreateBackupScheduleRequest
	ok := httputil.ValidatorRequestStructAndErrorResponse(r, w, &req, nil)
	if !ok {
		logrus.Errorf("start operation validate request body failure")
		return
	}
	err := handler.GetServiceManager().CreateBackupSchedule(tenantEnvID, serviceID, req)
	if err != nil {
		httputil.ReturnError(r, w, 500, err.Error())
		return
	}
	httputil.ReturnSuccess(r, w, nil)
}

// UpdateBackupSchedule update backup schedule for service resource and data
func (t *TenantEnvStruct) UpdateBackupSchedule(w http.ResponseWriter, r *http.Request) {
	tenantEnvID := r.Context().Value(ctxutil.ContextKey("tenant_env_id")).(string)
	serviceID := r.Context().Value(ctxutil.ContextKey("service_id")).(string)
	var req api_model.UpdateBackupScheduleRequest
	ok := httputil.ValidatorRequestStructAndErrorResponse(r, w, &req, nil)
	if !ok {
		logrus.Errorf("start operation validate request body failure")
		return
	}
	err := handler.GetServiceManager().UpdateBackupSchedule(tenantEnvID, serviceID, req)
	if err != nil {
		httputil.ReturnError(r, w, 500, err.Error())
		return
	}
	httputil.ReturnSuccess(r, w, nil)
}

// DeleteBackupSchedule delete backup schedule for service resource and data
func (t *TenantEnvStruct) DeleteBackupSchedule(w http.ResponseWriter, r *http.Request) {
	serviceID := r.Context().Value(ctxutil.ContextKey("service_id")).(string)
	err := handler.GetServiceManager().DeleteBackupSchedule(serviceID)
	if err != nil {
		httputil.ReturnError(r, w, 500, err.Error())
		return
	}
	httputil.ReturnSuccess(r, w, nil)
}

// DownloadBackup download backup for service resource and data
// func (t *TenantEnvStruct) DownloadBackup(w http.ResponseWriter, r *http.Request) {
// 	serviceID := r.Context().Value(ctxutil.ContextKey("service_id")).(string)
// 	backupID := chi.URLParam(r, "backup_id")
// 	bytes, err := handler.GetServiceManager().DownloadBackup(serviceID, backupID)
// 	if err != nil {
// 		httputil.ReturnError(r, w, 500, err.Error())
// 		return
// 	}

// 	w.Header().Set("Content-Type", "application/gzip")
// 	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.tar.gz", backupID))
// 	w.Header().Set("Content-Length", strconv.Itoa(len(bytes)))
// 	w.Write(bytes)
// }

func (t *TenantEnvStruct) DownloadBackup(w http.ResponseWriter, r *http.Request) {
	serviceID := r.Context().Value(ctxutil.ContextKey("service_id")).(string)
	backupID := chi.URLParam(r, "backup_id")
	result, err := handler.GetServiceManager().DownloadBackup(serviceID, backupID)
	if err != nil {
		httputil.ReturnError(r, w, 500, err.Error())
		return
	}

	buf := bytes.NewBuffer(result)

	_, err = io.Copy(w, buf)
	if err != nil {
		httputil.ReturnError(r, w, 500, err.Error())
		return
	}
}

// DeleteBackup delete backup for service resource and data
func (t *TenantEnvStruct) DeleteBackup(w http.ResponseWriter, r *http.Request) {
	serviceID := r.Context().Value(ctxutil.ContextKey("service_id")).(string)
	backupID := chi.URLParam(r, "backup_id")
	err := handler.GetServiceManager().DeleteBackup(serviceID, backupID)
	if err != nil {
		httputil.ReturnError(r, w, 500, err.Error())
		return
	}
	httputil.ReturnSuccess(r, w, nil)
}

// CreateRestore create restore for service resource and data
func (t *TenantEnvStruct) CreateRestore(w http.ResponseWriter, r *http.Request) {
	tenantEnvID := r.Context().Value(ctxutil.ContextKey("tenant_env_id")).(string)
	serviceID := r.Context().Value(ctxutil.ContextKey("service_id")).(string)

	var req api_model.CreateRestoreRequest
	ok := httputil.ValidatorRequestStructAndErrorResponse(r, w, &req, nil)
	if !ok {
		logrus.Errorf("start operation validate request body failure")
		return
	}
	err := handler.GetServiceManager().CreateRestore(tenantEnvID, serviceID, req)
	if err != nil {
		httputil.ReturnError(r, w, 500, err.Error())
		return
	}
	httputil.ReturnSuccess(r, w, nil)
}

// DeleteRestore delete restore for service resource and data
func (t *TenantEnvStruct) DeleteRestore(w http.ResponseWriter, r *http.Request) {
	serviceID := r.Context().Value(ctxutil.ContextKey("service_id")).(string)
	restoreID := chi.URLParam(r, "restore_id")
	err := handler.GetServiceManager().DeleteRestore(serviceID, restoreID)
	if err != nil {
		httputil.ReturnError(r, w, 500, err.Error())
		return
	}
	httputil.ReturnSuccess(r, w, nil)
}

// BackupRecords get backup histories
func (t *TenantEnvStruct) BackupRecords(w http.ResponseWriter, r *http.Request) {
	tenantEnvID := r.Context().Value(ctxutil.ContextKey("tenant_env_id")).(string)
	serviceID := r.Context().Value(ctxutil.ContextKey("service_id")).(string)

	records, err := handler.GetServiceManager().BackupRecords(tenantEnvID, serviceID)
	if err != nil {
		httputil.ReturnError(r, w, 500, err.Error())
		return
	}
	httputil.ReturnSuccess(r, w, records)
}

// GetBackupSchedule get backup schedule
func (t *TenantEnvStruct) GetBackupSchedule(w http.ResponseWriter, r *http.Request) {
	tenantEnvID := r.Context().Value(ctxutil.ContextKey("tenant_env_id")).(string)
	serviceID := r.Context().Value(ctxutil.ContextKey("service_id")).(string)

	schedule, _ := handler.GetServiceManager().GetBackupSchedule(tenantEnvID, serviceID)
	httputil.ReturnSuccess(r, w, schedule)
}

// RestoreRecords get restore histories
func (t *TenantEnvStruct) RestoreRecords(w http.ResponseWriter, r *http.Request) {
	tenantEnvID := r.Context().Value(ctxutil.ContextKey("tenant_env_id")).(string)
	serviceID := r.Context().Value(ctxutil.ContextKey("service_id")).(string)

	records, err := handler.GetServiceManager().RestoreRecords(tenantEnvID, serviceID)
	if err != nil {
		httputil.ReturnError(r, w, 500, err.Error())
		return
	}
	httputil.ReturnSuccess(r, w, records)
}

func (t *TenantEnvStruct) CreateVM(w http.ResponseWriter, r *http.Request) {
	tenantEnv := r.Context().Value(ctxutil.ContextKey("tenant_env")).(*dbmodel.TenantEnvs)
	var req api_model.CreateVMRequest
	ok := httputil.ValidatorRequestStructAndErrorResponse(r, w, &req, nil)
	if !ok {
		logrus.Errorf("start operation validate request body failure")
		return
	}
	resp, err := handler.GetServiceManager().CreateVM(tenantEnv, &req)
	if err != nil {
		httputil.ReturnError(r, w, 500, err.Error())
		return
	}
	httputil.ReturnSuccess(r, w, resp)
}

func (t *TenantEnvStruct) GetVM(w http.ResponseWriter, r *http.Request) {
	tenantEnv := r.Context().Value(ctxutil.ContextKey("tenant_env")).(*dbmodel.TenantEnvs)
	vmID := r.Context().Value(ctxutil.ContextKey("vm_id")).(string)

	resp, err := handler.GetServiceManager().GetVM(tenantEnv, vmID)
	if err != nil {
		httputil.ReturnError(r, w, 500, err.Error())
		return
	}
	httputil.ReturnSuccess(r, w, resp)
}

func (t *TenantEnvStruct) GetVMConditions(w http.ResponseWriter, r *http.Request) {
	tenantEnv := r.Context().Value(ctxutil.ContextKey("tenant_env")).(*dbmodel.TenantEnvs)
	vmID := r.Context().Value(ctxutil.ContextKey("vm_id")).(string)

	resp, err := handler.GetServiceManager().GetVMConditions(tenantEnv, vmID)
	if err != nil {
		httputil.ReturnError(r, w, 500, err.Error())
		return
	}
	httputil.ReturnSuccess(r, w, resp)
}

func (t *TenantEnvStruct) StartVM(w http.ResponseWriter, r *http.Request) {
	tenantEnv := r.Context().Value(ctxutil.ContextKey("tenant_env")).(*dbmodel.TenantEnvs)
	vmID := r.Context().Value(ctxutil.ContextKey("vm_id")).(string)

	resp, err := handler.GetServiceManager().StartVM(tenantEnv, vmID)
	if err != nil {
		httputil.ReturnError(r, w, 500, err.Error())
		return
	}
	httputil.ReturnSuccess(r, w, resp)
}

func (t *TenantEnvStruct) StopVM(w http.ResponseWriter, r *http.Request) {
	tenantEnv := r.Context().Value(ctxutil.ContextKey("tenant_env")).(*dbmodel.TenantEnvs)
	vmID := r.Context().Value(ctxutil.ContextKey("vm_id")).(string)

	resp, err := handler.GetServiceManager().StopVM(tenantEnv, vmID)
	if err != nil {
		httputil.ReturnError(r, w, 500, err.Error())
		return
	}
	httputil.ReturnSuccess(r, w, resp)
}

func (t *TenantEnvStruct) RestartVM(w http.ResponseWriter, r *http.Request) {
	tenantEnv := r.Context().Value(ctxutil.ContextKey("tenant_env")).(*dbmodel.TenantEnvs)
	vmID := r.Context().Value(ctxutil.ContextKey("vm_id")).(string)

	resp, err := handler.GetServiceManager().RestartVM(tenantEnv, vmID)
	if err != nil {
		httputil.ReturnError(r, w, 500, err.Error())
		return
	}
	httputil.ReturnSuccess(r, w, resp)
}

func (t *TenantEnvStruct) UpdateVM(w http.ResponseWriter, r *http.Request) {
	tenantEnv := r.Context().Value(ctxutil.ContextKey("tenant_env")).(*dbmodel.TenantEnvs)
	vmID := r.Context().Value(ctxutil.ContextKey("vm_id")).(string)
	var req api_model.UpdateVMRequest
	ok := httputil.ValidatorRequestStructAndErrorResponse(r, w, &req, nil)
	if !ok {
		logrus.Errorf("start operation validate request body failure")
		return
	}
	resp, err := handler.GetServiceManager().UpdateVM(tenantEnv, vmID, &req)
	if err != nil {
		httputil.ReturnError(r, w, 500, err.Error())
		return
	}
	httputil.ReturnSuccess(r, w, resp)
}

func (t *TenantEnvStruct) AddVMPort(w http.ResponseWriter, r *http.Request) {
	tenantEnv := r.Context().Value(ctxutil.ContextKey("tenant_env")).(*dbmodel.TenantEnvs)
	vmID := r.Context().Value(ctxutil.ContextKey("vm_id")).(string)
	var req api_model.AddVMPortRequest
	ok := httputil.ValidatorRequestStructAndErrorResponse(r, w, &req, nil)
	if !ok {
		logrus.Errorf("start operation validate request body failure")
		return
	}
	err := handler.GetServiceManager().AddVMPort(tenantEnv, vmID, &req)
	if err != nil {
		httputil.ReturnError(r, w, 500, err.Error())
		return
	}
	httputil.ReturnSuccess(r, w, nil)
}

func (t *TenantEnvStruct) GetVMPorts(w http.ResponseWriter, r *http.Request) {
	tenantEnv := r.Context().Value(ctxutil.ContextKey("tenant_env")).(*dbmodel.TenantEnvs)
	vmID := r.Context().Value(ctxutil.ContextKey("vm_id")).(string)

	resp, err := handler.GetServiceManager().GetVMPorts(tenantEnv, vmID)
	if err != nil {
		httputil.ReturnError(r, w, 500, err.Error())
		return
	}
	httputil.ReturnSuccess(r, w, resp)
}

func (t *TenantEnvStruct) EnableVMPort(w http.ResponseWriter, r *http.Request) {
	tenantEnv := r.Context().Value(ctxutil.ContextKey("tenant_env")).(*dbmodel.TenantEnvs)

	vmID := r.Context().Value(ctxutil.ContextKey("vm_id")).(string)
	var req api_model.EnableVMPortRequest
	ok := httputil.ValidatorRequestStructAndErrorResponse(r, w, &req, nil)
	if !ok {
		logrus.Errorf("start operation validate request body failure")
		return
	}
	err := handler.GetServiceManager().EnableVMPort(tenantEnv, vmID, &req)
	if err != nil {
		httputil.ReturnError(r, w, 500, err.Error())
		return
	}
	httputil.ReturnSuccess(r, w, nil)
}

func (t *TenantEnvStruct) DisableVMPort(w http.ResponseWriter, r *http.Request) {
	tenantEnv := r.Context().Value(ctxutil.ContextKey("tenant_env")).(*dbmodel.TenantEnvs)

	vmID := r.Context().Value(ctxutil.ContextKey("vm_id")).(string)
	var req api_model.DisableVMPortRequest
	ok := httputil.ValidatorRequestStructAndErrorResponse(r, w, &req, nil)
	if !ok {
		logrus.Errorf("start operation validate request body failure")
		return
	}
	err := handler.GetServiceManager().DisableVMPort(tenantEnv, vmID, &req)
	if err != nil {
		httputil.ReturnError(r, w, 500, err.Error())
		return
	}
	httputil.ReturnSuccess(r, w, nil)
}

func (t *TenantEnvStruct) CreateVMPortGateway(w http.ResponseWriter, r *http.Request) {
	tenantEnv := r.Context().Value(ctxutil.ContextKey("tenant_env")).(*dbmodel.TenantEnvs)
	vmID := r.Context().Value(ctxutil.ContextKey("vm_id")).(string)
	var req api_model.CreateVMPortGatewayRequest
	ok := httputil.ValidatorRequestStructAndErrorResponse(r, w, &req, nil)
	if !ok {
		logrus.Errorf("start operation validate request body failure")
		return
	}
	err := handler.GetServiceManager().CreateVMPortGateway(tenantEnv, vmID, &req)
	if err != nil {
		httputil.ReturnError(r, w, 500, err.Error())
		return
	}
	httputil.ReturnSuccess(r, w, nil)
}

func (t *TenantEnvStruct) UpdateVMPortGateway(w http.ResponseWriter, r *http.Request) {
	tenantEnv := r.Context().Value(ctxutil.ContextKey("tenant_env")).(*dbmodel.TenantEnvs)
	vmID := r.Context().Value(ctxutil.ContextKey("vm_id")).(string)
	gatewayID := chi.URLParam(r, "gateway_id")
	if gatewayID == "" {
		httputil.ReturnError(r, w, 400, "gateway id is required")
		return
	}
	var req api_model.UpdateVMPortGatewayRequest
	ok := httputil.ValidatorRequestStructAndErrorResponse(r, w, &req, nil)
	if !ok {
		logrus.Errorf("start operation validate request body failure")
		return
	}

	err := handler.GetServiceManager().UpdateVMPortGateway(tenantEnv, vmID, gatewayID, &req)
	if err != nil {
		httputil.ReturnError(r, w, 500, err.Error())
		return
	}
	httputil.ReturnSuccess(r, w, nil)
}

func (t *TenantEnvStruct) DeleteVMPortGateway(w http.ResponseWriter, r *http.Request) {
	tenantEnv := r.Context().Value(ctxutil.ContextKey("tenant_env")).(*dbmodel.TenantEnvs)
	vmID := r.Context().Value(ctxutil.ContextKey("vm_id")).(string)
	gatewayID := chi.URLParam(r, "gateway_id")
	if gatewayID == "" {
		httputil.ReturnError(r, w, 400, "gateway id is required")
		return
	}

	err := handler.GetServiceManager().DeleteVMPortGateway(tenantEnv, vmID, gatewayID)
	if err != nil {
		httputil.ReturnError(r, w, 500, err.Error())
		return
	}
	httputil.ReturnSuccess(r, w, nil)
}

func (t *TenantEnvStruct) DeleteVMPort(w http.ResponseWriter, r *http.Request) {
	tenantEnv := r.Context().Value(ctxutil.ContextKey("tenant_env")).(*dbmodel.TenantEnvs)
	vmID := r.Context().Value(ctxutil.ContextKey("vm_id")).(string)
	var req api_model.DeleteVMPortRequest
	ok := httputil.ValidatorRequestStructAndErrorResponse(r, w, &req, nil)
	if !ok {
		logrus.Errorf("start operation validate request body failure")
		return
	}
	err := handler.GetServiceManager().DeleteVMPort(tenantEnv, vmID, &req)
	if err != nil {
		httputil.ReturnError(r, w, 500, err.Error())
		return
	}
	httputil.ReturnSuccess(r, w, nil)
}

func (t *TenantEnvStruct) DeleteVM(w http.ResponseWriter, r *http.Request) {
	tenantEnv := r.Context().Value(ctxutil.ContextKey("tenant_env")).(*dbmodel.TenantEnvs)
	vmID := r.Context().Value(ctxutil.ContextKey("vm_id")).(string)

	err := handler.GetServiceManager().DeleteVM(tenantEnv, vmID)
	if err != nil {
		httputil.ReturnError(r, w, 500, err.Error())
		return
	}
	httputil.ReturnSuccess(r, w, nil)
}

func (t *TenantEnvStruct) ListVMs(w http.ResponseWriter, r *http.Request) {
	tenantEnv := r.Context().Value(ctxutil.ContextKey("tenant_env")).(*dbmodel.TenantEnvs)

	resp, err := handler.GetServiceManager().ListVMs(tenantEnv)
	if err != nil {
		httputil.ReturnError(r, w, 500, err.Error())
		return
	}
	httputil.ReturnSuccess(r, w, resp)
}

func (t *TenantEnvStruct) ListVMVolumes(w http.ResponseWriter, r *http.Request) {
	tenantEnv := r.Context().Value(ctxutil.ContextKey("tenant_env")).(*dbmodel.TenantEnvs)
	vmID := r.Context().Value(ctxutil.ContextKey("vm_id")).(string)

	resp, err := handler.GetServiceManager().ListVMVolumes(tenantEnv, vmID)
	if err != nil {
		httputil.ReturnError(r, w, 500, err.Error())
		return
	}
	httputil.ReturnSuccess(r, w, resp)
}

func (t *TenantEnvStruct) AddVMVolume(w http.ResponseWriter, r *http.Request) {
	tenantEnv := r.Context().Value(ctxutil.ContextKey("tenant_env")).(*dbmodel.TenantEnvs)
	vmID := r.Context().Value(ctxutil.ContextKey("vm_id")).(string)
	var req api_model.AddVMVolumeRequest
	ok := httputil.ValidatorRequestStructAndErrorResponse(r, w, &req, nil)
	if !ok {
		logrus.Errorf("start operation validate request body failure")
		return
	}
	err := handler.GetServiceManager().AddVMVolume(tenantEnv, vmID, &req)
	if err != nil {
		httputil.ReturnError(r, w, 500, err.Error())
		return
	}
	httputil.ReturnSuccess(r, w, nil)
}

func (t *TenantEnvStruct) DeleteVMVolume(w http.ResponseWriter, r *http.Request) {
	tenantEnv := r.Context().Value(ctxutil.ContextKey("tenant_env")).(*dbmodel.TenantEnvs)
	vmID := r.Context().Value(ctxutil.ContextKey("vm_id")).(string)
	volumeName := chi.URLParam(r, "volume_name")
	if volumeName == "" {
		httputil.ReturnError(r, w, 400, "volume name is required")
		return
	}
	err := handler.GetServiceManager().DeleteVMVolume(tenantEnv, vmID, volumeName)
	if err != nil {
		httputil.ReturnError(r, w, 500, err.Error())
		return
	}
	httputil.ReturnSuccess(r, w, nil)
}

func (t *TenantEnvStruct) RemoveBootDisk(w http.ResponseWriter, r *http.Request) {
	tenantEnv := r.Context().Value(ctxutil.ContextKey("tenant_env")).(*dbmodel.TenantEnvs)
	vmID := r.Context().Value(ctxutil.ContextKey("vm_id")).(string)
	err := handler.GetServiceManager().RemoveBootDisk(tenantEnv, vmID)
	if err != nil {
		httputil.ReturnError(r, w, 500, err.Error())
		return
	}
	httputil.ReturnSuccess(r, w, nil)
}

func (t *TenantEnvStruct) ChangeServiceApp(w http.ResponseWriter, r *http.Request) {
	serviceID := r.Context().Value(ctxutil.ContextKey("service_id")).(string)
	var req api_model.ChangeServiceAppRequest
	ok := httputil.ValidatorRequestStructAndErrorResponse(r, w, &req, nil)
	if !ok {
		logrus.Errorf("start operation validate request body failure")
		return
	}
	err := handler.GetServiceManager().ChangeServiceApp(serviceID, &req)
	if err != nil {
		httputil.ReturnError(r, w, 500, err.Error())
		return
	}
	httputil.ReturnSuccess(r, w, nil)
}
