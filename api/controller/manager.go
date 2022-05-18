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
	"net/http"

	"github.com/wutong-paas/wutong/api/api"
	"github.com/wutong-paas/wutong/api/discover"
	"github.com/wutong-paas/wutong/api/proxy"
	"github.com/wutong-paas/wutong/cmd/api/option"
	mqclient "github.com/wutong-paas/wutong/mq/client"
	etcdutil "github.com/wutong-paas/wutong/util/etcd"
	"github.com/wutong-paas/wutong/worker/client"
)

//V2Manager v2 manager
type V2Manager interface {
	Show(w http.ResponseWriter, r *http.Request)
	Health(w http.ResponseWriter, r *http.Request)
	AlertManagerWebHook(w http.ResponseWriter, r *http.Request)
	Version(w http.ResponseWriter, r *http.Request)
	api.ClusterInterface
	api.TenantInterface
	api.ServiceInterface
	api.LogInterface
	api.PluginInterface
	api.RulesInterface
	api.AppInterface
	api.Gatewayer
	api.ThirdPartyServicer
	api.Labeler
	api.AppRestoreInterface
	api.PodInterface
	api.ApplicationInterface
	api.HelmAppsInterface
}

var defaultV2Manager V2Manager

//CreateV2RouterManager 创建manager
func CreateV2RouterManager(conf option.Config, statusCli *client.AppRuntimeSyncClient) (err error) {
	defaultV2Manager, err = NewManager(conf, statusCli)
	return err
}

//GetManager 获取管理器
func GetManager() V2Manager {
	return defaultV2Manager
}

//NewManager new manager
func NewManager(conf option.Config, statusCli *client.AppRuntimeSyncClient) (*V2Routes, error) {
	etcdClientArgs := &etcdutil.ClientArgs{
		Endpoints: conf.EtcdEndpoint,
		CaFile:    conf.EtcdCaFile,
		CertFile:  conf.EtcdCertFile,
		KeyFile:   conf.EtcdKeyFile,
	}
	mqClient, err := mqclient.NewMqClient(etcdClientArgs, conf.MQAPI)
	if err != nil {
		return nil, err
	}
	var v2r V2Routes
	v2r.TenantStruct.StatusCli = statusCli
	v2r.TenantStruct.MQClient = mqClient
	v2r.GatewayStruct.MQClient = mqClient
	v2r.GatewayStruct.cfg = &conf
	v2r.LabelController.optconfig = &conf
	eventServerProxy := proxy.CreateProxy("eventlog", "http", []string{"local=>wt-eventlog:6363"})
	discover.GetEndpointDiscover().AddProject("event_log_event_http", eventServerProxy)
	v2r.EventLogStruct.EventlogServerProxy = eventServerProxy
	return &v2r, nil
}
