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

package handler

import (
	"context"

	api_model "github.com/wutong-paas/wutong/api/model"
	"github.com/wutong-paas/wutong/api/util"
	dbmodel "github.com/wutong-paas/wutong/db/model"
)

// TenantEnvHandler tenant env handler
type TenantEnvHandler interface {
	GetAllTenantEnvs(query string) ([]*dbmodel.TenantEnvs, error)
	GetTenantEnvs(tenantName, query string) ([]*dbmodel.TenantEnvs, error)
	GetTenantEnvsByName(tenantName, tenantEnvName string) (*dbmodel.TenantEnvs, error)
	GetTenantEnvsByUUID(uuid string) (*dbmodel.TenantEnvs, error)
	GetTenantEnvsName(tenantName string) ([]string, error)
	StatsMemCPU(services []*dbmodel.TenantEnvServices) (*api_model.StatsInfo, error)
	TotalMemCPU(services []*dbmodel.TenantEnvServices) (*api_model.StatsInfo, error)
	GetTenantEnvsResources(ctx context.Context, tr *api_model.TenantEnvResources) (map[string]map[string]interface{}, error)
	GetTenantEnvResource(tenantEnvID string) (TenantEnvResourceStats, error)
	GetAllocatableResources(ctx context.Context) (*ClusterResourceStats, error)
	GetServicesResources(tr *api_model.ServicesResources) (map[string]map[string]interface{}, error)
	TenantEnvsSum(tenantName string) (int, error)
	GetProtocols() ([]*dbmodel.RegionProcotols, *util.APIHandleError)
	TransPlugins(tenantEnvID, tenantEnvName, fromTenantEnv string, pluginList []string) *util.APIHandleError
	GetServicesStatus(ids string) map[string]string
	IsClosedStatus(status string) bool
	BindTenantEnvsResource(source []*dbmodel.TenantEnvs) api_model.TenantEnvList
	UpdateTenantEnv(*dbmodel.TenantEnvs) error
	DeleteTenantEnv(ctx context.Context, tenantEnvID string) error
	GetClusterResource(ctx context.Context) *ClusterResourceStats
	CheckResourceName(ctx context.Context, namespace string, req *api_model.CheckResourceNameReq) (*api_model.CheckResourceNameResp, error)
	GetKubeConfig(namespace string) (string, error)
	GetKubeResources(namespace, tenantEnvID string, customSetting api_model.KubeResourceCustomSetting) string
}
