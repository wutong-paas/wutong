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

package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
	"github.com/wutong-paas/wutong/api/handler"
	"github.com/wutong-paas/wutong/api/util"
	ctxutil "github.com/wutong-paas/wutong/api/util/ctx"
	"github.com/wutong-paas/wutong/db"
	dbmodel "github.com/wutong-paas/wutong/db/model"
	"github.com/wutong-paas/wutong/event"
	"github.com/wutong-paas/wutong/pkg/kube"
	httputil "github.com/wutong-paas/wutong/util/http"
)

var pool []string

func init() {
	pool = []string{
		"services_status",
	}
}

// InitTenantEnv 实现中间件
func InitTenantEnv(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		debugRequestBody(r)

		tenantName := chi.URLParam(r, "tenant_name")
		tenantEnvName := chi.URLParam(r, "tenant_env_name")
		if tenantEnvName == "" {
			httputil.ReturnError(r, w, 400, "cant find tenant env")
			return
		}
		tenantEnv, err := db.GetManager().TenantEnvDao().GetTenantEnvIDByName(tenantName, tenantEnvName)
		if err != nil {
			logrus.Errorf("get tenant env by tenant env name error: %s %v", tenantEnvName, err)
			if err.Error() == gorm.ErrRecordNotFound.Error() {
				httputil.ReturnError(r, w, 404, "cant find tenantEnv")
				return
			}
			httputil.ReturnError(r, w, 500, "get assign tenant env uuid failed")
			return
		}
		ctx := context.WithValue(r.Context(), ctxutil.ContextKey("tenant_env_name"), tenantEnvName)
		ctx = context.WithValue(ctx, ctxutil.ContextKey("tenant_env_id"), tenantEnv.UUID)
		ctx = context.WithValue(ctx, ctxutil.ContextKey("tenant_env"), tenantEnv)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
	return http.HandlerFunc(fn)
}

// InitService 实现serviceinit中间件
func InitService(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		serviceAlias := chi.URLParam(r, "service_alias")
		if serviceAlias == "" {
			httputil.ReturnError(r, w, 400, "cant find service alias")
			return
		}
		tenantEnvID := r.Context().Value(ctxutil.ContextKey("tenant_env_id"))
		service, err := db.GetManager().TenantEnvServiceDao().GetServiceByTenantEnvIDAndServiceAlias(tenantEnvID.(string), serviceAlias)
		if err != nil {
			if err.Error() == gorm.ErrRecordNotFound.Error() {
				httputil.ReturnError(r, w, 404, "cant find service")
				return
			}
			logrus.Errorf("get service by tenant env & service alias error, %v", err)
			httputil.ReturnError(r, w, 500, "get service id error")
			return
		}
		serviceID := service.ServiceID
		ctx := context.WithValue(r.Context(), ctxutil.ContextKey("service_alias"), serviceAlias)
		ctx = context.WithValue(ctx, ctxutil.ContextKey("service_id"), serviceID)
		ctx = context.WithValue(ctx, ctxutil.ContextKey("service"), service)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
	return http.HandlerFunc(fn)
}

// InitApplication -
func InitApplication(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		appID := chi.URLParam(r, "app_id")
		tenantEnvApp, err := handler.GetApplicationHandler().GetAppByID(appID)
		if err != nil {
			httputil.ReturnBcodeError(r, w, err)
			return
		}

		ctx := context.WithValue(r.Context(), ctxutil.ContextKey("app_id"), tenantEnvApp.AppID)
		ctx = context.WithValue(ctx, ctxutil.ContextKey("application"), tenantEnvApp)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
	return http.HandlerFunc(fn)
}

// InitVeleroBackupOrRestore
func InitVeleroBackupOrRestore(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		if ok := kube.IsVeleroInstalled(kube.KubeClient(), kube.APIExtClient()); !ok {
			httputil.ReturnError(r, w, 500, "集群中未检测到 Velero 服务，使用该功能请联系管理员安装 Velero 服务！")
			return
		}
		next.ServeHTTP(w, r.WithContext(r.Context()))
	}
	return http.HandlerFunc(fn)
}

// InitVM -
func InitVM(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		if ok := kube.IsKubevirtInstalled(kube.KubeClient(), kube.APIExtClient()); !ok {
			httputil.ReturnError(r, w, 500, "集群中未检测到 Kubevirt 服务，使用该功能请联系管理员安装 Kubevirt 服务！")
			return
		}
		next.ServeHTTP(w, r.WithContext(r.Context()))
	}
	return http.HandlerFunc(fn)
}

// InitVMID -
func InitVMID(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		if ok := kube.IsKubevirtInstalled(kube.KubeClient(), kube.APIExtClient()); !ok {
			httputil.ReturnError(r, w, 500, "集群中未检测到 Kubevirt 服务，使用该功能请联系管理员安装 Kubevirt 服务！")
			return
		}

		vmID := chi.URLParam(r, "vm_id")
		if vmID == "" {
			httputil.ReturnError(r, w, 400, "need vm id")
			return
		}
		ctx := context.WithValue(r.Context(), ctxutil.ContextKey("vm_id"), vmID)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
	return http.HandlerFunc(fn)
}

// InitPlugin 实现plugin init中间件
func InitPlugin(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		debugRequestBody(r)

		pluginID := chi.URLParam(r, "plugin_id")
		tenantEnvID := r.Context().Value(ctxutil.ContextKey("tenant_env_id")).(string)
		if pluginID == "" {
			httputil.ReturnError(r, w, 400, "need plugin id")
			return
		}
		_, err := db.GetManager().TenantEnvPluginDao().GetPluginByID(pluginID, tenantEnvID)
		if err != nil {
			if err.Error() == gorm.ErrRecordNotFound.Error() {
				httputil.ReturnError(r, w, 404, "cant find plugin")
				return
			}
			logrus.Errorf("get plugin error, %v", err)
			httputil.ReturnError(r, w, 500, "get plugin error")
			return
		}
		ctx := context.WithValue(r.Context(), ctxutil.ContextKey("plugin_id"), pluginID)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
	return http.HandlerFunc(fn)
}

// InitSysPlugin 实现 sys plugin init中间件
func InitSysPlugin(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		debugRequestBody(r)

		pluginID := chi.URLParam(r, "plugin_id")
		if pluginID == "" {
			httputil.ReturnError(r, w, 400, "need plugin id")
			return
		}
		_, err := db.GetManager().TenantEnvPluginDao().GetPluginByID(pluginID, "")
		if err != nil {
			if err.Error() == gorm.ErrRecordNotFound.Error() {
				httputil.ReturnError(r, w, 404, "cant find sys plugin")
				return
			}
			logrus.Errorf("get plugin error, %v", err)
			httputil.ReturnError(r, w, 500, "get sys plugin error")
			return
		}
		ctx := context.WithValue(r.Context(), ctxutil.ContextKey("plugin_id"), pluginID)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
	return http.HandlerFunc(fn)
}

// SetLog SetLog
func SetLog(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		eventID := chi.URLParam(r, "event_id")
		if eventID != "" {
			logger := event.GetLogger(eventID)
			ctx := context.WithValue(r.Context(), ctxutil.ContextKey("logger"), logger)
			next.ServeHTTP(w, r.WithContext(ctx))
		}
	}
	return http.HandlerFunc(fn)
}

// Proxy 反向代理中间件
func Proxy(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.RequestURI, "/v2/nodes") {
			handler.GetNodeProxy().Proxy(w, r)
			return
		}
		if strings.HasPrefix(r.RequestURI, "/v2/cluster/service-health") {
			handler.GetNodeProxy().Proxy(w, r)
			return
		}
		if strings.HasPrefix(r.RequestURI, "/v2/builder") {
			handler.GetBuilderProxy().Proxy(w, r)
			return
		}
		if strings.HasPrefix(r.RequestURI, "/v2/tasks") {
			handler.GetNodeProxy().Proxy(w, r)
			return
		}
		if strings.HasPrefix(r.RequestURI, "/v2/tasktemps") {
			handler.GetNodeProxy().Proxy(w, r)
			return
		}
		if strings.HasPrefix(r.RequestURI, "/v2/taskgroups") {
			handler.GetNodeProxy().Proxy(w, r)
			return
		}
		if strings.HasPrefix(r.RequestURI, "/v2/configs") {
			handler.GetNodeProxy().Proxy(w, r)
			return
		}
		// if strings.HasPrefix(r.RequestURI, "/v2/rules") {
		// 	handler.GetMonitorProxy().Proxy(w, r)
		// 	return
		// }
		if strings.HasPrefix(r.RequestURI, "/console/filebrowser") {
			paths := strings.Split(r.URL.Path, "/")
			if len(paths) > 3 && len(paths[3]) == 32 {
				serviceID := paths[3]
				proxy := handler.GetFileBrowserProxy(serviceID)
				r.URL.Path = strings.Replace(r.URL.Path, "/console/filebrowser/"+serviceID, "", 1)
				proxy.Proxy(w, r)
				return
			}
		}
		if strings.HasPrefix(r.RequestURI, "/console/dbgate") {
			paths := strings.Split(r.URL.Path, "/")
			if len(paths) > 3 && len(paths[3]) == 32 {
				serviceID := paths[3]
				proxy := handler.GetDbgateProxy(serviceID)
				proxy.Proxy(w, r)
				return
			}
		}
		if strings.HasPrefix(r.RequestURI, "/obsmimir") {
			r.URL.Path = strings.Replace(r.URL.Path, "/obsmimir", "", 1)
			handler.GetObsMimirProxy().Proxy(w, r)
			return
		}
		if strings.HasPrefix(r.RequestURI, "/obstempo") {
			r.URL.Path = strings.Replace(r.URL.Path, "/obstempo", "", 1)
			handler.GetObsTempoProxy().Proxy(w, r)
			return
		}
		if strings.HasPrefix(r.RequestURI, "/obsloki") {
			r.URL.Path = strings.Replace(r.URL.Path, "/obsloki", "", 1)
			handler.GetObsLokiProxy().Proxy(w, r)
			return
		}
		if strings.HasPrefix(r.RequestURI, "/obs") {
			r.URL.Path = strings.Replace(r.URL.Path, "/obs", "", 1)
			handler.GetObsProxy().Proxy(w, r)
			return
		}
		// virt-vnc proxy
		if strings.HasPrefix(r.RequestURI, "/console/virt-vnc") {
			paths := strings.Split(r.URL.Path, "/")
			var namespace, vm string
			if len(paths) > 4 {
				namespace = paths[3]
				vm = paths[4]
			}
			if namespace == "" {
				httputil.ReturnError(r, w, 400, "virt-vnc: namespace is required")
				return
			}
			if vm == "" {
				httputil.ReturnError(r, w, 400, "virt-vnc: vm is required")
				return
			}
			if vm == "package.json" {
				return
			}
			if strings.HasPrefix(r.RequestURI, fmt.Sprintf("/console/virt-vnc/%s/%s/k8s", namespace, vm)) {
				handler.GetVirtVNCProxy(namespace, vm).WebSocketProxy.Proxy(w, r)
				return
			}
			r.URL.Path = strings.Replace(r.URL.Path, fmt.Sprintf("/console/virt-vnc/%s/%s", namespace, vm), "", 1)
			handler.GetVirtVNCProxy(namespace, vm).HTTPProxy.Proxy(w, r)
			return
		}
		next.ServeHTTP(w, r)
	}
	return http.HandlerFunc(fn)
}

func apiExclude(r *http.Request) bool {
	if r.Method == "GET" {
		return true
	}
	for _, item := range pool {
		if strings.Contains(r.RequestURI, item) {
			return true
		}
	}
	return false
}

type resWriter struct {
	origWriter http.ResponseWriter
	statusCode int
}

func (w *resWriter) Header() http.Header {
	return w.origWriter.Header()
}
func (w *resWriter) Write(p []byte) (int, error) {
	return w.origWriter.Write(p)
}
func (w *resWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.origWriter.WriteHeader(statusCode)
}

var targetKeyMap = map[string]string{
	dbmodel.TargetTypeService: "service_id",
	dbmodel.TargetTypeVM:      "vm_id",
}

// WrapEL wrap eventlog, handle event log before and after process
func WrapEL(f http.HandlerFunc, target, optType string, synType int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				logrus.Warningf("error reading request body: %v", err)
			} else {
				logrus.Debugf("method: %s; uri: %s; body: %s", r.Method, r.RequestURI, string(body))
			}
			// set a new body, which will simulate the same data we read
			r.Body = io.NopCloser(bytes.NewBuffer(body))
			var targetID string
			var ok bool

			targetKey := targetKeyMap[target]
			if targetKey == "" {
				httputil.ReturnError(r, w, 400, "操作对象未指定")
				return
			}

			// tenantEnvID can not null
			tenantEnvID := r.Context().Value(ctxutil.ContextKey("tenant_env_id")).(string)
			if targetID, ok = r.Context().Value(ctxutil.ContextKey(targetKey)).(string); !ok {
				var reqDataMap map[string]interface{}
				if err = json.Unmarshal(body, &reqDataMap); err != nil {
					httputil.ReturnError(r, w, 400, "操作对象未指定")
					return
				}

				if target == dbmodel.TargetTypeVM && strings.HasSuffix(r.URL.Path, "/vms") {
					targetKey = "name"
				}

				if targetID, ok = reqDataMap[targetKey].(string); !ok {
					httputil.ReturnError(r, w, 400, "操作对象未指定")
					return
				}
			}
			if target == dbmodel.TargetTypeVM {
				// 使用tenantEnvID作为targetID的前缀，避免不同环境下的targetID冲突
				targetID = fmt.Sprintf("%s/%s", tenantEnvID, targetID)
			}

			// eventLog check the latest event
			// if !util.CanDoEvent(optType, synType, target, targetID, serviceKind) {
			// 	logrus.Errorf("operation too frequently. uri: %s; target: %s; target id: %s", r.RequestURI, target, targetID)
			// 	httputil.ReturnError(r, w, 409, "操作过于频繁，请稍后再试") // status code 409 conflict
			// 	return
			// }

			// handle operator
			var operator string
			var reqData map[string]interface{}
			if err = json.Unmarshal(body, &reqData); err == nil {
				if operatorI := reqData["operator"]; operatorI != nil {
					operator = operatorI.(string)
				}
			}

			event, err := util.CreateEvent(target, optType, targetID, tenantEnvID, string(body), operator, synType)
			if err != nil {
				logrus.Error("create event error : ", err)
				httputil.ReturnError(r, w, 500, "操作失败")
				return
			}

			var ctx = context.WithValue(r.Context(), ctxutil.ContextKey("event"), event)
			ctx = context.WithValue(ctx, ctxutil.ContextKey("event_id"), event.EventID)
			rw := &resWriter{origWriter: w}
			f(rw, r.WithContext(ctx))
			if synType == dbmodel.SyncEventType || (synType == dbmodel.AsyncEventType && rw.statusCode >= 400) { // status code 2XX/3XX all equal to success
				util.UpdateEvent(event.EventID, rw.statusCode)
			}
		}
	}
}

func debugRequestBody(r *http.Request) {
	if !apiExclude(r) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			logrus.Warningf("error reading request body: %v", err)
		}
		logrus.Debugf("method: %s; uri: %s; body: %s", r.Method, r.RequestURI, string(body))

		// set a new body, which will simulate the same data we read
		r.Body = io.NopCloser(bytes.NewBuffer(body))
	}
}
