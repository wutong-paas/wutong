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
	"fmt"

	"github.com/wutong-paas/wutong/api/proxy"
	"github.com/wutong-paas/wutong/cmd/api/option"
	"github.com/wutong-paas/wutong/db"
)

var nodeProxy proxy.Proxy
var builderProxy proxy.Proxy
var prometheusProxy proxy.Proxy

// var monitorProxy proxy.Proxy
var obsProxy proxy.Proxy
var obsmimirProxy proxy.Proxy
var obstempoProxy proxy.Proxy
var obslokiProxy proxy.Proxy

var filebrowserProxies = make(map[string]proxy.Proxy, 20)
var dbgateProxies = make(map[string]proxy.Proxy, 20)

type VirtVNCProxy struct {
	HTTPProxy      proxy.Proxy
	WebSocketProxy proxy.Proxy
}

var virtVNCProxies = make(map[string]*VirtVNCProxy, 20)

// InitProxy 初始化
func InitProxy(conf option.Config) {
	if nodeProxy == nil {
		nodeProxy = proxy.CreateProxy("acp_node", "http", conf.NodeAPI)
		// discover.GetEndpointDiscover().AddProject("acp_node", nodeProxy)
	}
	if builderProxy == nil {
		builderProxy = proxy.CreateProxy("builder", "http", conf.BuilderAPI)
	}
	if prometheusProxy == nil {
		prometheusProxy = proxy.CreateProxy("prometheus", "http", []string{conf.PrometheusEndpoint})
	}
	// if monitorProxy == nil {
	// 	monitorProxy = proxy.CreateProxy("monitor", "http", []string{"127.0.0.1:3329"})
	// 	discover.GetEndpointDiscover().AddProject("monitor", monitorProxy)
	// }
	if obsProxy == nil {
		obsProxy = proxy.CreateProxy("obs", "http", conf.ObsAPI)
	}
	if obsmimirProxy == nil {
		obsmimirProxy = proxy.CreateProxy("obsmimir", "http", conf.ObsMimirAPI)
	}
	if obstempoProxy == nil {
		obstempoProxy = proxy.CreateProxy("obstempo", "http", conf.ObsTempoAPI)
	}
	if obslokiProxy == nil {
		obslokiProxy = proxy.CreateProxy("obsloki", "http", conf.ObsLokiAPI)
	}
}

// GetNodeProxy GetNodeProxy
func GetNodeProxy() proxy.Proxy {
	return nodeProxy
}

// GetBuilderProxy GetBuilderProxy
func GetBuilderProxy() proxy.Proxy {
	return builderProxy
}

// GetPrometheusProxy GetPrometheusProxy
func GetPrometheusProxy() proxy.Proxy {
	return prometheusProxy
}

// GetMonitorProxy GetMonitorProxy
// func GetMonitorProxy() proxy.Proxy {
// 	return monitorProxy
// }

// GetFileBrowserProxy GetFileBrowserProxy
func GetFileBrowserProxy(serviceID string) proxy.Proxy {
	fbProxy, ok := filebrowserProxies[serviceID]
	if !ok {
		// get serviceID and fb plugin
		service, err := db.GetManager().TenantEnvServiceDao().GetServiceByID(serviceID)
		if err != nil {
			return proxy.CreateProxy("filebrowser", "http", nil)
		}
		tenantEnv, err := db.GetManager().TenantEnvDao().GetTenantEnvByUUID(service.TenantEnvID)
		if err != nil {
			return proxy.CreateProxy("filebrowser", "http", nil)
		}
		k8sSvc := fmt.Sprintf("%s-6173.%s:6173", service.ServiceAlias, tenantEnv.Namespace)
		fbProxy = proxy.CreateProxy("filebrowser", "http", []string{k8sSvc})
		filebrowserProxies[serviceID] = fbProxy
	}
	return fbProxy
}

// GetDbgateProxy GetDbgateProxy
func GetDbgateProxy(serviceID string) proxy.Proxy {
	dbgateProxy, ok := dbgateProxies[serviceID]
	if !ok {
		// get serviceID and fb plugin
		service, err := db.GetManager().TenantEnvServiceDao().GetServiceByID(serviceID)
		if err != nil {
			return proxy.CreateProxy("dbgate", "http", nil)
		}
		tenantEnv, err := db.GetManager().TenantEnvDao().GetTenantEnvByUUID(service.TenantEnvID)
		if err != nil {
			return proxy.CreateProxy("dbgate", "http", nil)
		}
		k8sSvc := fmt.Sprintf("%s-3000.%s:3000", service.ServiceAlias, tenantEnv.Namespace)
		dbgateProxy = proxy.CreateProxy("dbgate", "http", []string{k8sSvc})
		dbgateProxies[serviceID] = dbgateProxy
	}
	return dbgateProxy
}

func GetVirtVNCProxy(namespace, vm string) *VirtVNCProxy {
	key := namespace + "/" + vm
	virtVNCProxy, ok := virtVNCProxies[key]
	if !ok || virtVNCProxy == nil {
		// get serviceID and fb plugin
		virtVNCProxy = new(VirtVNCProxy)
		virtVNCProxy.HTTPProxy = proxy.CreateProxy("virt-vnc", "http", []string{fmt.Sprintf("%s-vnc.%s", vm, namespace)})
		virtVNCProxy.WebSocketProxy = proxy.CreateProxy("virt-vnc-websocket", "websocket", []string{fmt.Sprintf("%s-vnc.%s", vm, namespace)})

		virtVNCProxies[key] = virtVNCProxy
	}
	return virtVNCProxy

}

// GetObsProxy GetObsProxy
func GetObsProxy() proxy.Proxy {
	return obsProxy
}

// GetObsMimirProxy GetObsMimirProxy
func GetObsMimirProxy() proxy.Proxy {
	return obsmimirProxy
}

// GetObsTempoProxy GetObsTempoProxy
func GetObsTempoProxy() proxy.Proxy {
	return obstempoProxy
}

// GetObsLokiProxy GetObsLokiProxy
func GetObsLokiProxy() proxy.Proxy {
	return obslokiProxy
}
