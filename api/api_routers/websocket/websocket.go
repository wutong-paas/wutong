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

package websocket

import (
	"github.com/wutong-paas/wutong/api/controller"

	"github.com/go-chi/chi"
)

// Routes routes
func Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/docker_console", controller.GetDockerConsole().Get)
	// Deprecated: It's not work always, use /container_log instead
	r.Get("/docker_container_log", controller.GetDockerConsole().Get)
	r.Get("/container_log", controller.GetDockerConsole().Get)
	r.Get("/docker_node_console", controller.GetDockerConsole().Get)
	r.Get("/docker_virtctl_console", controller.GetDockerConsole().Get)
	r.Get("/docker_vm_ssh", controller.GetDockerConsole().Get)
	r.Get("/docker_log", controller.GetDockerLog().Get)
	r.Get("/monitor_message", controller.GetMonitorMessage().Get)
	r.Get("/new_monitor_message", controller.GetMonitorMessage().Get)
	r.Get("/event_log", controller.GetEventLog().Get)
	r.Get("/services/{serviceID}/pubsub", controller.GetPubSubControll().Get)
	return r
}

// LogRoutes 日志下载路由
func LogRoutes() chi.Router {
	r := chi.NewRouter()
	r.Get("/{gid}/{filename}", controller.GetLogFile().Get)
	r.Get("/install_log/{filename}", controller.GetLogFile().GetInstallLog)
	return r
}

// AppRoutes 应用导出包下载路由
func AppRoutes() chi.Router {
	r := chi.NewRouter()
	r.Get("/download/{format}/{fileName}", controller.GetManager().Download)
	r.Post("/upload/{eventID}", controller.GetManager().Upload)
	r.Options("/upload/{eventID}", controller.GetManager().Upload)
	return r
}
