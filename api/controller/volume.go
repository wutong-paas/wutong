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
	"net/http"
	"strings"

	"github.com/go-chi/chi"
	"github.com/sirupsen/logrus"
	"github.com/wutong-paas/wutong/api/handler"
	api_model "github.com/wutong-paas/wutong/api/model"
	"github.com/wutong-paas/wutong/api/util/bcode"
	ctxutil "github.com/wutong-paas/wutong/api/util/ctx"
	dbmodel "github.com/wutong-paas/wutong/db/model"
	httputil "github.com/wutong-paas/wutong/util/http"
	"gopkg.in/yaml.v2"
)

// VolumeDependency VolumeDependency
func (t *TenantEnvStruct) VolumeDependency(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "DELETE":
		t.DeleteVolumeDependency(w, r)
	case "POST":
		t.AddVolumeDependency(w, r)
	}
}

// AddVolumeDependency add volume dependency
func (t *TenantEnvStruct) AddVolumeDependency(w http.ResponseWriter, r *http.Request) {
	// swagger:operation POST /v2/tenants/{tenant_name}/envs/{tenant_env_name}/services/{service_alias}/volume-dependency v2 addVolumeDependency
	//
	// 增加应用持久化依赖
	//
	// add volume dependency
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

	logrus.Debugf("trans add volumn dependency service ")
	serviceID := r.Context().Value(ctxutil.ContextKey("service_id")).(string)
	tenantEnvID := r.Context().Value(ctxutil.ContextKey("tenant_env_id")).(string)
	var tsr api_model.V2AddVolumeDependencyStruct
	if ok := httputil.ValidatorRequestStructAndErrorResponse(r, w, &tsr.Body, nil); !ok {
		return
	}
	vd := &dbmodel.TenantEnvServiceMountRelation{
		TenantEnvID:     tenantEnvID,
		ServiceID:       serviceID,
		DependServiceID: tsr.Body.DependServiceID,
		HostPath:        tsr.Body.MntDir,
		VolumePath:      tsr.Body.MntName,
	}
	if err := handler.GetServiceManager().VolumeDependency(vd, "add"); err != nil {
		err.Handle(r, w)
		return
	}
	httputil.ReturnSuccess(r, w, nil)
}

// DeleteVolumeDependency delete volume dependency
func (t *TenantEnvStruct) DeleteVolumeDependency(w http.ResponseWriter, r *http.Request) {
	// swagger:operation DELETE /v2/tenants/{tenant_name}/envs/{tenant_env_name}/services/{service_alias}/volume-dependency v2 deleteVolumeDependency
	//
	// 删除应用持久化依赖
	//
	// delete volume dependency
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
	tenantEnvID := r.Context().Value(ctxutil.ContextKey("tenant_env_id")).(string)
	var tsr api_model.V2DelVolumeDependencyStruct
	if ok := httputil.ValidatorRequestStructAndErrorResponse(r, w, &tsr.Body, nil); !ok {
		return
	}
	vd := &dbmodel.TenantEnvServiceMountRelation{
		TenantEnvID:     tenantEnvID,
		ServiceID:       serviceID,
		DependServiceID: tsr.Body.DependServiceID,
	}
	if err := handler.GetServiceManager().VolumeDependency(vd, "delete"); err != nil {
		err.Handle(r, w)
		return
	}
	httputil.ReturnSuccess(r, w, nil)
}

// AddVolume AddVolume
func (t *TenantEnvStruct) AddVolume(w http.ResponseWriter, r *http.Request) {
	// swagger:operation POST /v2/tenants/{tenant_name}/envs/{tenant_env_name}/services/{service_alias}/volume v2 addVolume
	//
	// 增加应用持久化信息
	//
	// add volume
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
	tenantEnvID := r.Context().Value(ctxutil.ContextKey("tenant_env_id")).(string)
	avs := &api_model.V2AddVolumeStruct{}
	if ok := httputil.ValidatorRequestStructAndErrorResponse(r, w, &avs.Body, nil); !ok {
		return
	}
	tsv := &dbmodel.TenantEnvServiceVolume{
		ServiceID:          serviceID,
		VolumePath:         avs.Body.VolumePath,
		HostPath:           avs.Body.HostPath,
		Category:           avs.Body.Category,
		VolumeCapacity:     avs.Body.VolumeCapacity,
		VolumeType:         dbmodel.ShareFileVolumeType.String(),
		VolumeProviderName: avs.Body.VolumeProviderName,
		AccessMode:         avs.Body.AccessMode,
		SharePolicy:        avs.Body.SharePolicy,
		BackupPolicy:       avs.Body.BackupPolicy,
		ReclaimPolicy:      avs.Body.ReclaimPolicy,
	}
	if !strings.HasPrefix(tsv.VolumePath, "/") {
		httputil.ReturnError(r, w, 400, "volume path is invalid,must begin with /")
		return
	}
	if err := handler.GetServiceManager().VolumnVar(tsv, tenantEnvID, "", "add"); err != nil {
		err.Handle(r, w)
		return
	}
	httputil.ReturnSuccess(r, w, nil)
}

// UpdVolume updates service volume.
func (t *TenantEnvStruct) UpdVolume(w http.ResponseWriter, r *http.Request) {
	var req api_model.UpdVolumeReq
	if ok := httputil.ValidatorRequestStructAndErrorResponse(r, w, &req, nil); !ok {
		return
	}

	if req.Mode != nil && (*req.Mode > 777 || *req.Mode < 0) {
		httputil.ReturnBcodeError(r, w, bcode.NewBadRequest("mode be a number between 0 and 777 (octal)"))
		return
	}

	sid := r.Context().Value(ctxutil.ContextKey("service_id")).(string)
	if err := handler.GetServiceManager().UpdVolume(sid, &req); err != nil {
		httputil.ReturnError(r, w, 500, err.Error())
	}
	httputil.ReturnSuccess(r, w, "success")
}

// DeleteVolume DeleteVolume
func (t *TenantEnvStruct) DeleteVolume(w http.ResponseWriter, r *http.Request) {
	// swagger:operation DELETE /v2/tenants/{tenant_name}/envs/{tenant_env_name}/services/{service_alias}/volume v2 deleteVolume
	//
	// 删除应用持久化信息
	//
	// delete volume
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
	tenantEnvID := r.Context().Value(ctxutil.ContextKey("tenant_env_id")).(string)
	avs := &api_model.V2DelVolumeStruct{}
	if ok := httputil.ValidatorRequestStructAndErrorResponse(r, w, &avs.Body, nil); !ok {
		return
	}
	tsv := &dbmodel.TenantEnvServiceVolume{
		ServiceID:  serviceID,
		VolumePath: avs.Body.VolumePath,
		Category:   avs.Body.Category,
	}
	if err := handler.GetServiceManager().VolumnVar(tsv, tenantEnvID, "", "delete"); err != nil {
		err.Handle(r, w)
		return
	}
	httputil.ReturnSuccess(r, w, nil)
}

// DeleteAllVolumes 删除组件所有存储
func (t *TenantEnvStruct) DeleteAllVolumes(w http.ResponseWriter, r *http.Request) {
	serviceID := r.Context().Value(ctxutil.ContextKey("service_id")).(string)
	tenantEnvID := r.Context().Value(ctxutil.ContextKey("tenant_env_id")).(string)
	tsv := &dbmodel.TenantEnvServiceVolume{
		ServiceID: serviceID,
	}
	if err := handler.GetServiceManager().VolumnVar(tsv, tenantEnvID, "", "delete_all"); err != nil {
		err.Handle(r, w)
		return
	}
	httputil.ReturnSuccess(r, w, nil)
}

//以下为V2.1版本持久化API,支持多种持久化模式

// AddVolumeDependency add volume dependency
func AddVolumeDependency(w http.ResponseWriter, r *http.Request) {
	// swagger:operation POST /v2/tenants/{tenant_name}/envs/{tenant_env_name}/services/{service_alias}/depvolumes v2 addDepVolume
	//
	// 增加应用持久化依赖(V2.1支持多种类型存储)
	//
	// add volume dependency
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

	logrus.Debugf("trans add volumn dependency service ")
	serviceID := r.Context().Value(ctxutil.ContextKey("service_id")).(string)
	tenantEnvID := r.Context().Value(ctxutil.ContextKey("tenant_env_id")).(string)
	var tsr api_model.AddVolumeDependencyStruct
	if ok := httputil.ValidatorRequestStructAndErrorResponse(r, w, &tsr.Body, nil); !ok {
		return
	}

	vd := &dbmodel.TenantEnvServiceMountRelation{
		TenantEnvID:     tenantEnvID,
		ServiceID:       serviceID,
		DependServiceID: tsr.Body.DependServiceID,
		VolumeName:      tsr.Body.VolumeName,
		VolumePath:      tsr.Body.VolumePath,
		VolumeType:      tsr.Body.VolumeType,
	}
	if err := handler.GetServiceManager().VolumeDependency(vd, "add"); err != nil {
		err.Handle(r, w)
		return
	}
	httputil.ReturnSuccess(r, w, nil)
}

// DeleteVolumeDependency delete volume dependency
func DeleteVolumeDependency(w http.ResponseWriter, r *http.Request) {
	// swagger:operation DELETE /v2/tenants/{tenant_name}/envs/{tenant_env_name}/services/{service_alias}/depvolumes v2 delDepVolume
	//
	// 删除应用持久化依赖(V2.1支持多种类型存储)
	//
	// delete volume dependency
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
	tenantEnvID := r.Context().Value(ctxutil.ContextKey("tenant_env_id")).(string)
	var tsr api_model.DeleteVolumeDependencyStruct
	if ok := httputil.ValidatorRequestStructAndErrorResponse(r, w, &tsr.Body, nil); !ok {
		return
	}
	vd := &dbmodel.TenantEnvServiceMountRelation{
		TenantEnvID:     tenantEnvID,
		ServiceID:       serviceID,
		DependServiceID: tsr.Body.DependServiceID,
		VolumeName:      tsr.Body.VolumeName,
	}
	if err := handler.GetServiceManager().VolumeDependency(vd, "delete"); err != nil {
		err.Handle(r, w)
		return
	}
	httputil.ReturnSuccess(r, w, nil)
}

// AddVolume AddVolume
func AddVolume(w http.ResponseWriter, r *http.Request) {
	// swagger:operation POST /v2/tenants/{tenant_name}/envs/{tenant_env_name}/services/{service_alias}/volumes v2 addVolumes
	//
	// 增加应用持久化信息(V2.1支持多种类型存储)
	//
	// add volume
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
	tenantEnvID := r.Context().Value(ctxutil.ContextKey("tenant_env_id")).(string)
	avs := &api_model.AddVolumeStruct{}
	if ok := httputil.ValidatorRequestStructAndErrorResponse(r, w, &avs.Body, nil); !ok {
		return
	}

	if avs.Body.Mode != nil && (*avs.Body.Mode > 777 || *avs.Body.Mode < 0) {
		httputil.ReturnBcodeError(r, w, bcode.NewBadRequest("mode be a number between 0 and 777 (octal)"))
		return
	}

	// TODO fanyangyang validate VolumeCapacity  AccessMode SharePolicy BackupPolicy ReclaimPolicy AllowExpansion

	if !strings.HasPrefix(avs.Body.VolumePath, "/") {
		httputil.ReturnError(r, w, 400, "volume path is invalid, must begin with /")
		return
	}

	if strings.HasSuffix(avs.Body.VolumePath, "/") {
		valid, content := correctConfigMapContent(avs.Body.FileContent)
		if !valid {
			httputil.ReturnError(r, w, 400, "content is invalid for multi file volume, must be yaml format")
			return
		}
		avs.Body.FileContent = content
	}

	tsv := &dbmodel.TenantEnvServiceVolume{
		ServiceID:          serviceID,
		VolumeName:         avs.Body.VolumeName,
		VolumePath:         avs.Body.VolumePath,
		VolumeType:         avs.Body.VolumeType,
		Category:           avs.Body.Category,
		VolumeProviderName: avs.Body.VolumeProviderName,
		IsReadOnly:         avs.Body.IsReadOnly,
		VolumeCapacity:     avs.Body.VolumeCapacity,
		AccessMode:         avs.Body.AccessMode,
		SharePolicy:        avs.Body.SharePolicy,
		BackupPolicy:       avs.Body.BackupPolicy,
		ReclaimPolicy:      avs.Body.ReclaimPolicy,
		AllowExpansion:     avs.Body.AllowExpansion,
		Mode:               avs.Body.Mode,
	}

	if err := handler.GetServiceManager().VolumnVar(tsv, tenantEnvID, avs.Body.FileContent, "add"); err != nil {
		err.Handle(r, w)
		return
	}
	httputil.ReturnSuccess(r, w, nil)
}

func correctConfigMapContent(content string) (bool, string) {
	content = trimSpaceAndNewLine(content)
	var m map[string]interface{}
	// ignore json
	if strings.HasPrefix(content, "{") {
		return false, content
	}
	err := yaml.UnmarshalStrict([]byte(content), &m)
	return err == nil && len(m) > 0, content
}

func trimSpaceAndNewLine(content string) string {
	for strings.HasPrefix(content, " ") ||
		strings.HasPrefix(content, "\n") ||
		strings.HasSuffix(content, " ") ||
		strings.HasSuffix(content, "\n") {
		content = strings.Trim(content, " ")
		content = strings.Trim(content, "\n")
	}
	return content
}

// DeleteVolume DeleteVolume
func DeleteVolume(w http.ResponseWriter, r *http.Request) {
	// swagger:operation DELETE /v2/tenants/{tenant_name}/envs/{tenant_env_name}/services/{service_alias}/volumes/{volume_name} v2 deleteVolumes
	//
	// 删除应用持久化信息(V2.1支持多种类型存储)
	//
	// delete volume
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
	tenantEnvID := r.Context().Value(ctxutil.ContextKey("tenant_env_id")).(string)
	tsv := &dbmodel.TenantEnvServiceVolume{}
	tsv.ServiceID = serviceID
	tsv.VolumeName = chi.URLParam(r, "volume_name")
	if err := handler.GetServiceManager().VolumnVar(tsv, tenantEnvID, "", "delete"); err != nil {
		err.Handle(r, w)
		return
	}
	httputil.ReturnSuccess(r, w, nil)
}

// GetVolume 获取应用全部存储，包括依赖的存储
func GetVolume(w http.ResponseWriter, r *http.Request) {
	// swagger:operation GET /v2/tenants/{tenant_name}/envs/{tenant_env_name}/services/{service_alias}/volumes v2 getVolumes
	//
	// 获取应用全部存储，包括依赖的存储(V2.1支持多种类型存储)
	//
	// get volumes
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
	volumes, err := handler.GetServiceManager().GetVolumes(serviceID)
	if err != nil {
		err.Handle(r, w)
		return
	}
	httputil.ReturnSuccess(r, w, volumes)
}

// GetDepVolume 获取应用所有依赖的存储
func GetDepVolume(w http.ResponseWriter, r *http.Request) {
	// swagger:operation GET /v2/tenants/{tenant_name}/envs/{tenant_env_name}/services/{service_alias}/depvolumes v2 getDepVolumes
	//
	// 获取应用依赖的存储(V2.1支持多种类型存储)
	//
	// get depvolumes
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
	volumes, err := handler.GetServiceManager().GetDepVolumes(serviceID)
	if err != nil {
		err.Handle(r, w)
		return
	}
	httputil.ReturnSuccess(r, w, volumes)
}
