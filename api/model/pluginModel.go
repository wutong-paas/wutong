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

package model

import (
	"time"

	dbmodel "github.com/wutong-paas/wutong/db/model"
)

const (
	PluginTypeSys string = "sys"
)

// Plugin -
type Plugin struct {
	PluginID    string `json:"plugin_id" validate:"plugin_id|required"`
	PluginName  string `json:"plugin_name" validate:"plugin_name|required"`
	PluginInfo  string `json:"plugin_info" validate:"plugin_info"`
	ImageURL    string `json:"image_url" validate:"image_url"`
	GitURL      string `json:"git_url" validate:"git_url"`
	BuildModel  string `json:"build_model" validate:"build_model"`
	PluginModel string `json:"plugin_model" validate:"plugin_model"`
	TenantEnvID string `json:"tenant_env_id" validate:"tenant_env_id"`
	PluginType  string `json:"origin" validate:"origin"`
}

// DbModel return database model
func (p *Plugin) DbModel(tenantEnvID string) *dbmodel.TenantEnvPlugin {
	tp := &dbmodel.TenantEnvPlugin{
		PluginID:    p.PluginID,
		PluginName:  p.PluginName,
		PluginInfo:  p.PluginInfo,
		ImageURL:    p.ImageURL,
		GitURL:      p.GitURL,
		BuildModel:  p.BuildModel,
		PluginModel: p.PluginModel,
		TenantEnvID: tenantEnvID,
		PluginType:  p.PluginType,
	}
	if p.PluginType == PluginTypeSys {
		tp.TenantEnvID = ""
		tp.Domain = ""
	}
	return tp
}

// BatchCreatePlugins -
type BatchCreatePlugins struct {
	Plugins []*Plugin `json:"plugins"`
}

// CreatePluginStruct CreatePluginStruct
//
//swagger:parameters createPlugin
type CreatePluginStruct struct {
	// in: path
	// required: true
	TenantEnvName string `json:"tenant_env_name"`
	// in: body
	Body struct {
		//插件id
		//in: body
		//required: true
		PluginID string `json:"plugin_id" validate:"plugin_id|required"`
		//in: body
		//required: true
		PluginName string `json:"plugin_name" validate:"plugin_name|required"`
		//插件用途描述
		//in: body
		//required: false
		PluginInfo string `json:"plugin_info" validate:"plugin_info"`
		// 插件docker地址
		// in:body
		// required: false
		ImageURL string `json:"image_url" validate:"image_url"`
		//git地址
		//in: body
		//required: false
		GitURL string `json:"git_url" validate:"git_url"`
		//构建模式
		//in: body
		//required: false
		BuildModel string `json:"build_model" validate:"build_model"`
		//插件模式
		//in: body
		//required: false
		PluginModel string `json:"plugin_model" validate:"plugin_model"`
		//租户id
		//in: body
		//required: false
		TenantEnvID string `json:"tenant_env_id" validate:"tenant_env_id"`

		//插件类型
		PluginType string `json:"origin" validate:"origin"`
	}
}

// CreateSysPluginStruct CreateSysPluginStruct
//
//swagger:parameters createSysPlugin
type CreateSysPluginStruct struct {
	// in: body
	Body struct {
		//插件id
		//in: body
		//required: true
		PluginID string `json:"plugin_id" validate:"plugin_id|required"`
		//in: body
		//required: true
		PluginName string `json:"plugin_name" validate:"plugin_name|required"`
		//插件用途描述
		//in: body
		//required: false
		PluginInfo string `json:"plugin_info" validate:"plugin_info"`
		// 插件docker地址
		// in:body
		// required: false
		ImageURL string `json:"image_url" validate:"image_url"`
		//构建模式
		//in: body
		//required: false
		BuildModel string `json:"build_model" validate:"build_model"`
		//插件模式
		//in: body
		//required: false
		PluginModel string `json:"plugin_model" validate:"plugin_model"`

		//插件类型
		PluginType string `json:"origin" validate:"origin"`
	}
}

// UpdatePluginStruct UpdatePluginStruct
//
//swagger:parameters updatePlugin
type UpdatePluginStruct struct {
	// 租户名称
	// in: path
	// required: true
	TenantEnvName string `json:"tenant_env_name" validate:"tenant_env_name|required"`
	// 插件id
	// in: path
	// required: true
	PluginID string `json:"plugin_id" validate:"tenant_env_name|required"`
	// in: body
	Body struct {
		//插件名称
		//in: body
		//required: false
		PluginName string `json:"plugin_name" validate:"plugin_name"`
		//插件用途描述
		//in: body
		//required: false
		PluginInfo string `json:"plugin_info" validate:"plugin_info"`
		//插件docker地址
		//in: body
		//required: false
		ImageURL string `json:"image_url" validate:"image_url"`
		//git地址
		//in: body
		//required: false
		GitURL string `json:"git_url" validate:"git_url"`
		//构建模式
		//in: body
		//required: false
		BuildModel string `json:"build_model" validate:"build_model"`
		//插件模式
		//in: body
		//required: false
		PluginModel string `json:"plugin_model" validate:"plugin_model"`
	}
}

// UpdateSysPluginStruct UpdateSysPluginStruct
//
//swagger:parameters updateSysPlugin
type UpdateSysPluginStruct struct {
	// 插件id
	// in: path
	// required: true
	PluginID string `json:"plugin_id" validate:"tenant_env_name|required"`
	// in: body
	Body struct {
		//插件名称
		//in: body
		//required: false
		PluginName string `json:"plugin_name" validate:"plugin_name"`
		//插件用途描述
		//in: body
		//required: false
		PluginInfo string `json:"plugin_info" validate:"plugin_info"`
		//插件docker地址
		//in: body
		//required: false
		ImageURL string `json:"image_url" validate:"image_url"`
		//git地址
		//in: body
		//required: false
		GitURL string `json:"git_url" validate:"git_url"`
		//构建模式
		//in: body
		//required: false
		BuildModel string `json:"build_model" validate:"build_model"`
		//插件模式
		//in: body
		//required: false
		PluginModel string `json:"plugin_model" validate:"plugin_model"`
	}
}

// DeletePluginStruct deletePluginStruct
//
//swagger:parameters deletePlugin
type DeletePluginStruct struct {
	// in: path
	// required: true
	TenantEnvName string `json:"tenant_env_name" validate:"tenant_env_name|required"`
	// in: path
	// required: true
	PluginID string `json:"plugin_id" validate:"plugin_id|required"`
}

// DeleteSysPluginStruct DeleteSysPluginStruct
//
//swagger:parameters deleteSysPlugin
type DeleteSysPluginStruct struct {
	// in: path
	// required: true
	PluginID string `json:"plugin_id" validate:"plugin_id|required"`
}

// ENVStruct ENVStruct
//
//swagger:parameters adddefaultenv updatedefaultenv
type ENVStruct struct {
	// 租户名称
	// in: path
	// required: true
	TenantEnvName string `json:"tenant_env_name" validate:"tenant_env_name"`
	// 插件id
	// in: path
	// required: true
	PluginID string `json:"plugin_id" validate:"plugin_id"`
	// 构建版本
	// in: path
	// required; true
	VersionID string `json:"version_id" validate:"version_id"`
	//in : body
	Body struct {
		//in: body
		//required: true
		EVNInfo []*PluginDefaultENV
	}
}

// DeleteENVstruct DeleteENVstruct
//
//swagger:parameters deletedefaultenv
type DeleteENVstruct struct {
	// 租户名称
	// in: path
	// required: true
	TenantEnvName string `json:"tenant_env_name" validate:"tenant_env_name|required"`
	// 插件id
	// in: path
	// required: true
	PluginID string `json:"plugin_id" validate:"plugin_id|required"`
	// 构建版本
	// in: path
	// required; true
	VersionID string `json:"version_id" validate:"version_id|required"`
	//配置项名称
	//in: path
	//required: true
	ENVName string `json:"env_name" validate:"env_name|required"`
}

// PluginDefaultENV 插件默认环境变量
type PluginDefaultENV struct {
	//对应插件id
	//in: body
	//required: true
	PluginID string `json:"plugin_id" validate:"plugin_id"`
	//构建版本id
	//in: body
	//required: true
	VersionID string `json:"version_id" validate:"version_id"`
	//配置项名称
	//in: body
	//required: true
	ENVName string `json:"env_name" validate:"env_name"`
	//配置项值
	//in: body
	//required: true
	ENVValue string `json:"env_value" validate:"env_value"`
	//是否可以被使用者修改
	//in :body
	//required: false
	IsChange bool `json:"is_change" validate:"is_change|bool"`
}

type BuildPluginBody struct {
	// the event id
	// in: body
	// required: false
	EventID string `json:"event_id" validate:"event_id"`
	// 插件CPU权重, 默认125
	// in: body
	// required: true
	PluginCPU int `json:"plugin_cpu" validate:"plugin_cpu|required"`
	// 插件最大内存, 默认50
	// in: body
	// required: true
	PluginMemory int `json:"plugin_memory" validate:"plugin_memory|required"`
	// 插件cmd, 默认50
	// in: body
	// required: false
	PluginCMD string `json:"plugin_cmd" validate:"plugin_cmd"`
	// 插件的版本号
	// in: body
	// required: true
	BuildVersion string `json:"build_version" validate:"build_version|required"`
	// 插件构建版本号
	// in: body
	// required: true
	DeployVersion string `json:"deploy_version" validate:"deploy_version"`
	// git地址分支信息，默认为master
	// in: body
	// required: false
	RepoURL string `json:"repo_url" validate:"repo_url"`
	// git username
	// in: body
	// required: false
	Username string `json:"username"`
	// git password
	// in: body
	// required: false
	Password string `json:"password"`
	// 版本信息, 协助选择插件版本
	// in:body
	// required: true
	Info string `json:"info" validate:"info"`
	// 操作人
	// in: body
	// required: false
	Operator string `json:"operator" validate:"operator"`
	//租户id
	// in: body
	// required: true
	TenantEnvID string `json:"tenant_env_id" validate:"tenant_env_id"`
	// 镜像地址
	// in: body
	// required: false
	BuildImage string `json:"build_image" validate:"build_image"`
	//ImageInfo
	ImageInfo struct {
		HubURL      string `json:"hub_url"`
		HubUser     string `json:"hub_user"`
		HubPassword string `json:"hub_password"`
		Namespace   string `json:"namespace"`
		IsTrust     bool   `json:"is_trust,omitempty"`
	} `json:"ImageInfo" validate:"ImageInfo"`
}

// BuildPluginStruct BuildPluginStruct
//
//swagger:parameters buildPlugin
type BuildPluginStruct struct {
	// in: path
	// required: true
	TenantEnvName string `json:"tenant_env_name" validate:"tenant_env_name"`
	// in: path
	// required: true
	PluginID string `json:"plugin_id" validate:"plugin_id"`
	//in: body
	Body BuildPluginBody
}

// BuildSysPluginStruct BuildSysPluginStruct
//
//swagger:parameters buildSysPlugin
type BuildSysPluginStruct struct {
	// in: path
	// required: true
	PluginID string `json:"plugin_id" validate:"plugin_id"`
	//in: body
	Body BuildPluginBody
}

// BuildPluginReq -
type BuildPluginReq struct {
	PluginID      string `json:"plugin_id" validate:"plugin_id"`
	EventID       string `json:"event_id" validate:"event_id"`
	PluginCPU     int    `json:"plugin_cpu" validate:"plugin_cpu|required"`
	PluginMemory  int    `json:"plugin_memory" validate:"plugin_memory|required"`
	PluginCMD     string `json:"plugin_cmd" validate:"plugin_cmd"`
	BuildVersion  string `json:"build_version" validate:"build_version|required"`
	DeployVersion string `json:"deploy_version" validate:"deploy_version"`
	RepoURL       string `json:"repo_url" validate:"repo_url"`
	Username      string `json:"username"`
	Password      string `json:"password"`
	Info          string `json:"info" validate:"info"`
	Operator      string `json:"operator" validate:"operator"`
	TenantEnvID   string `json:"tenant_env_id" validate:"tenant_env_id"`
	BuildImage    string `json:"build_image" validate:"build_image"`
	ImageInfo     struct {
		HubURL      string `json:"hub_url"`
		HubUser     string `json:"hub_user"`
		HubPassword string `json:"hub_password"`
		Namespace   string `json:"namespace"`
		IsTrust     bool   `json:"is_trust,omitempty"`
	} `json:"ImageInfo" validate:"ImageInfo"`
}

// DbModel return database model
func (b BuildPluginReq) DbModel(plugin *dbmodel.TenantEnvPlugin) *dbmodel.TenantEnvPluginBuildVersion {
	buildVersion := &dbmodel.TenantEnvPluginBuildVersion{
		VersionID:       b.BuildVersion,
		DeployVersion:   b.DeployVersion,
		PluginID:        b.PluginID,
		Kind:            plugin.BuildModel,
		Repo:            b.RepoURL,
		GitURL:          plugin.GitURL,
		BaseImage:       plugin.ImageURL,
		ContainerCPU:    b.PluginCPU,
		ContainerMemory: b.PluginMemory,
		ContainerCMD:    b.PluginCMD,
		BuildTime:       time.Now().Format(time.RFC3339),
		Info:            b.Info,
		Status:          "building",
	}
	if b.PluginCPU == 0 {
		buildVersion.ContainerCPU = 125
	}
	if b.PluginMemory == 0 {
		buildVersion.ContainerMemory = 50
	}
	if plugin.PluginType == PluginTypeSys {
		buildVersion.BuildLocalImage = plugin.ImageURL
		buildVersion.Status = "complete"
	}
	return buildVersion
}

// BatchBuildPlugins -
type BatchBuildPlugins struct {
	Plugins []*BuildPluginReq `json:"plugins"`
}

// PluginBuildVersionStruct PluginBuildVersionStruct
//
//swagger:parameters deletePluginVersion pluginVersion
type PluginBuildVersionStruct struct {
	//in: path
	//required: true
	TenantEnvName string `json:"tenant_env_name" validate:"tenant_env_name"`
	//in: path
	//required: true
	PluginID string `json:"plugin_id" validate:"plugin_id"`
	//in: path
	//required: true
	VersionID string `json:"version_id" validate:"version_id"`
}

// AllPluginBuildVersionStruct AllPluginBuildVersionStruct
//
//swagger:parameters allPluginVersions
type AllPluginBuildVersionStruct struct {
	//in: path
	//required: true
	TenantEnvName string `json:"tenant_env_name" validate:"tenant_env_name"`
	//in: path
	//required: true
	PluginID string `json:"plugin_id" validate:"plugin_id"`
}

// PluginSetStruct PluginSetStruct
//
//swagger:parameters updatePluginSet addPluginSet
type PluginSetStruct struct {
	//in: path
	//required: true
	TenantEnvName string `json:"tenant_env_name"`
	//in: path
	//required: true
	ServiceAlias string `json:"service_alias"`
	// in: body
	Body struct {
		//plugin id
		//in: body
		//required: true
		PluginID string `json:"plugin_id" validate:"plugin_id"`
		// plugin version
		//in: body
		//required: true
		VersionID string `json:"version_id" validate:"version_id"`
		// plugin is uesd
		//in: body
		//required: false
		Switch bool `json:"switch" validate:"switch|bool"`
		// plugin cpu size default 125
		// in: body
		// required: false
		PluginCPU int `json:"plugin_cpu" validate:"plugin_cpu"`
		// plugin memory size default 64
		// in: body
		// required: false
		PluginMemory int `json:"plugin_memory" validate:"plugin_memory"`
		// app plugin config
		// in: body
		// required: true
		ConfigEnvs ConfigEnvs `json:"config_envs" validate:"config_envs"`
	}
}

// GetPluginsStruct GetPluginsStruct
//
//swagger:parameters getPlugins
type GetPluginsStruct struct {
	//in: path
	//required: true
	TenantEnvName string `json:"tenant_env_name"`
}

// GetPluginSetStruct GetPluginSetStruct
//
//swagger:parameters getPluginSet
type GetPluginSetStruct struct {
	//in: path
	//required: true
	TenantEnvName string `json:"tenant_env_name"`
	//in: path
	//required: true
	ServiceAlias string `json:"service_alias"`
}

// DeletePluginSetStruct DeletePluginSetStruct
//
//swagger:parameters deletePluginRelation
type DeletePluginSetStruct struct {
	//in: path
	//required: true
	TenantEnvName string `json:"tenant_env_name"`
	//in: path
	//required: true
	ServiceAlias string `json:"service_alias"`
	//插件id
	//in: path
	//required: true
	PluginID string `json:"plugin_id"`
}

// GetPluginEnvStruct GetPluginEnvStruct
//
//swagger:parameters getPluginEnv getPluginDefaultEnv
type GetPluginEnvStruct struct {
	//租户名称
	//in: path
	//required: true
	TenantEnvName string `json:"tenant_env_name"`
	// 插件id
	// in: path
	// required: true
	PluginID string `json:"plugin_id"`
	// 构建版本id
	// in: path
	// required: true
	VersionID string `json:"version_id"`
}

// GetVersionEnvStruct GetVersionEnvStruct
//
//swagger:parameters getVersionEnvs
type GetVersionEnvStruct struct {
	// in: path
	// required: true
	TenantEnvName string `json:"tenant_env_name"`
	// in: path
	// required: true
	ServiceAlias string `json:"service_alias"`
	// 插件id
	// in: path
	// required: true
	PluginID string `json:"plugin_id"`
}

// SetVersionEnv SetVersionEnv
//
//swagger:parameters setVersionEnv updateVersionEnv
type SetVersionEnv struct {
	// in: path
	// required: true
	TenantEnvName string `json:"tenant_env_name"`
	// in: path
	// required: true
	ServiceAlias string `json:"service_alias"`
	// 插件id
	// in: path
	// required: true
	PluginID string `json:"plugin_id"`
	//in: body
	Body struct {
		TenantEnvID string `json:"tenant_env_id"`
		ServiceID   string `json:"service_id"`
		// 环境变量
		// in: body
		// required: true
		ConfigEnvs ConfigEnvs `json:"config_envs" validate:"config_envs"`
	}
}

// UpdateComponentPluginConfigRequest 更新组件插件配置
type UpdateComponentPluginConfigRequest struct {
	// in: path
	// required: true
	TenantEnvName string `json:"tenant_env_name"`
	// in: path
	// required: true
	ServiceAlias string `json:"service_alias"`
	//in: body
	Body struct {
		TenantEnvID string `json:"tenant_env_id"`
		// plugin cpu size default 125
		// in: body
		// required: true
		PluginID  string `json:"plugin_id"`
		ServiceID string `json:"service_id"`
		// plugin cpu size default 125
		// in: body
		// required: false
		PluginCPU int `json:"plugin_cpu" validate:"plugin_cpu"`
		// plugin memory size default 64
		// in: body
		// required: false
		PluginMemory int `json:"plugin_memory" validate:"plugin_memory"`
		// 环境变量
		// in: body
		// required: true
		ConfigEnvs ConfigEnvs `json:"config_envs" validate:"config_envs"`
	}
}

// ToggleComponentPluginRequest 启用/禁用组件插件
type ToggleComponentPluginRequest struct {
	//in: path
	//required: true
	TenantEnvName string `json:"tenant_env_name"`
	//in: path
	//required: true
	ServiceAlias string `json:"service_alias"`
	// in: body
	Body struct {
		ServiceID string `json:"service_id"`
		//plugin id
		//in: body
		//required: true
		PluginID string `json:"plugin_id" validate:"plugin_id"`
		// plugin is uesd
		//in: body
		//required: false
		Switch bool `json:"switch" validate:"switch|bool"`
	}
}

// ConfigEnvs Config
type ConfigEnvs struct {
	NormalEnvs  []*VersionEnv `json:"normal_envs" validate:"normal_envs"`
	ComplexEnvs *ResourceSpec `json:"complex_envs" validate:"complex_envs"`
}

// VersionEnv VersionEnv
type VersionEnv struct {
	//变量名
	//in:body
	//required: true
	EnvName string `json:"env_name" validate:"env_name"`
	//变量值
	//in:body
	//required: true
	EnvValue string `json:"env_value" validate:"env_value"`
}

// DbModel return database model
func (v *VersionEnv) DbModel(componentID, pluginID string) *dbmodel.TenantEnvPluginVersionEnv {
	return &dbmodel.TenantEnvPluginVersionEnv{
		ServiceID: componentID,
		PluginID:  pluginID,
		EnvName:   v.EnvName,
		EnvValue:  v.EnvValue,
	}
}

// TransPlugins TransPlugins
type TransPlugins struct {
	// in: path
	// required: true
	TenantEnvName string `json:"tenant_env_name"`
	//in: body
	Body struct {
		// 从该租户安装
		// in: body
		// required: true
		FromTenantEnvName string `json:"from_tenant_env_name" validate:"from_tenant_env_name"`
		// 插件id
		// in: body
		// required: true
		PluginsID []string `json:"plugins_id" validate:"plugins_id"`
	}
}

// PluginVersionEnv -
type PluginVersionEnv struct {
	EnvName  string `json:"env_name" validate:"env_name"`
	EnvValue string `json:"env_value" validate:"env_value"`
}

// DbModel return database model
func (p *PluginVersionEnv) DbModel(componentID, pluginID string) *dbmodel.TenantEnvPluginVersionEnv {
	return &dbmodel.TenantEnvPluginVersionEnv{
		ServiceID: componentID,
		PluginID:  pluginID,
		EnvName:   p.EnvName,
		EnvValue:  p.EnvValue,
	}
}

// TenantEnvPluginVersionConfig -
type TenantEnvPluginVersionConfig struct {
	ConfigStr string `json:"config_str" validate:"config_str"`
}

// DbModel return database model
func (p *TenantEnvPluginVersionConfig) DbModel(componentID, pluginID string) *dbmodel.TenantEnvPluginVersionDiscoverConfig {
	return &dbmodel.TenantEnvPluginVersionDiscoverConfig{
		ServiceID: componentID,
		PluginID:  pluginID,
		ConfigStr: p.ConfigStr,
	}
}

// ComponentPlugin -
type ComponentPlugin struct {
	PluginID        string     `json:"plugin_id"`
	VersionID       string     `json:"version_id"`
	PluginModel     string     `json:"plugin_model"`
	ContainerCPU    int        `json:"container_cpu"`
	ContainerMemory int        `json:"container_memory"`
	Switch          bool       `json:"switch"`
	ConfigEnvs      ConfigEnvs `json:"config_envs" validate:"config_envs"`
}

// DbModel return database model
func (p *ComponentPlugin) DbModel(componentID string) *dbmodel.TenantEnvServicePluginRelation {
	return &dbmodel.TenantEnvServicePluginRelation{
		VersionID:       p.VersionID,
		ServiceID:       componentID,
		PluginID:        p.PluginID,
		Switch:          p.Switch,
		PluginModel:     p.PluginModel,
		ContainerCPU:    p.ContainerCPU,
		ContainerMemory: p.ContainerMemory,
	}
}
