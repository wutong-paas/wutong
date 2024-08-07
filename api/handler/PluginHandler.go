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
	api_model "github.com/wutong-paas/wutong/api/model"
	"github.com/wutong-paas/wutong/api/util"
	dbmodel "github.com/wutong-paas/wutong/db/model"
)

// PluginHandler plugin handler
type PluginHandler interface {
	CreatePluginAct(cps *api_model.CreatePluginStruct) *util.APIHandleError
	CreateSysPluginAct(cps *api_model.CreateSysPluginStruct) *util.APIHandleError
	UpdatePluginAct(pluginID, tenantEnvID string, cps *api_model.UpdatePluginStruct) *util.APIHandleError
	UpdateSysPluginAct(pluginID string, cps *api_model.UpdateSysPluginStruct) *util.APIHandleError
	DeletePluginAct(pluginID, tenantEnvID string) *util.APIHandleError
	DeleteSysPluginAct(pluginID string) *util.APIHandleError
	GetPlugins(tenantEnvID string) ([]*dbmodel.TenantEnvPlugin, *util.APIHandleError)
	AddDefaultEnv(est *api_model.ENVStruct) *util.APIHandleError
	UpdateDefaultEnv(est *api_model.ENVStruct) *util.APIHandleError
	DeleteDefaultEnv(pluginID, versionID, envName string) *util.APIHandleError
	BuildPluginManual(bps *api_model.BuildPluginStruct) (*dbmodel.TenantEnvPluginBuildVersion, *util.APIHandleError)
	BuildSysPluginManual(bps *api_model.BuildSysPluginStruct) (*dbmodel.TenantEnvPluginBuildVersion, *util.APIHandleError)
	GetAllPluginBuildVersions(pluginID string) ([]*dbmodel.TenantEnvPluginBuildVersion, *util.APIHandleError)
	GetPluginBuildVersion(pluginID, versionID string) (*dbmodel.TenantEnvPluginBuildVersion, *util.APIHandleError)
	DeletePluginBuildVersion(pluginID, versionID string) *util.APIHandleError
	GetDefaultEnv(pluginID, versionID string) ([]*dbmodel.TenantEnvPluginDefaultENV, *util.APIHandleError)
	GetEnvsWhichCanBeSet(serviceID, pluginID string) (interface{}, *util.APIHandleError)
	BatchCreatePlugins(tenantEnvID string, plugins []*api_model.Plugin) *util.APIHandleError
	BatchBuildPlugins(req *api_model.BatchBuildPlugins, tenantEnvID string) *util.APIHandleError
}
