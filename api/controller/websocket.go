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
	"context"
	"net/http"
	"os"
	"path"

	"github.com/go-chi/chi"
	"github.com/sirupsen/logrus"
	"github.com/wutong-paas/wutong/api/handler"
	"github.com/wutong-paas/wutong/api/proxy"
	ctxutil "github.com/wutong-paas/wutong/api/util/ctx"
	"github.com/wutong-paas/wutong/config/configs"
	"github.com/wutong-paas/wutong/util/constants"
)

// DockerConsole docker console
type DockerConsole struct {
	socketproxy proxy.Proxy
}

// var defaultDockerConsoleEndpoints = []string{"127.0.0.1:7171"}
// var defaultEventLogEndpoints = []string{"local=>wt-eventlog:6363"}

var dockerConsole *DockerConsole

// GetDockerConsole get Docker console
func GetDockerConsole() *DockerConsole {
	if dockerConsole != nil {
		return dockerConsole
	}
	dockerConsole = &DockerConsole{
		// socketproxy: proxy.CreateProxy("dockerconsole", "websocket", defaultDockerConsoleEndpoints),
		socketproxy: proxy.CreateProxy("dockerconsole", "websocket", configs.Default().APIConfig.DockerConsoleServers),
	}
	// discover.GetEndpointDiscover().AddProject("acp_webcli", dockerConsole.socketproxy)
	return dockerConsole
}

// Get get
func (d DockerConsole) Get(w http.ResponseWriter, r *http.Request) {
	d.socketproxy.Proxy(w, r)
}

var dockerLog *DockerLog

// DockerLog docker log
type DockerLog struct {
	socketproxy proxy.Proxy
}

// GetDockerLog get docker log
func GetDockerLog() *DockerLog {
	if dockerLog == nil {
		dockerLog = &DockerLog{
			socketproxy: proxy.CreateProxy("dockerlog", "websocket", configs.Default().APIConfig.EventLogEndpoints),
		}
		// discover.GetEndpointDiscover().AddProject("event_log_event_http", dockerLog.socketproxy)
	}
	return dockerLog
}

// Get get
func (d DockerLog) Get(w http.ResponseWriter, r *http.Request) {
	d.socketproxy.Proxy(w, r)
}

// MonitorMessage monitor message
type MonitorMessage struct {
	socketproxy proxy.Proxy
}

var monitorMessage *MonitorMessage

// GetMonitorMessage get MonitorMessage
func GetMonitorMessage() *MonitorMessage {
	if monitorMessage == nil {
		monitorMessage = &MonitorMessage{
			socketproxy: proxy.CreateProxy("monitormessage", "websocket", configs.Default().APIConfig.EventLogEndpoints),
		}
		// discover.GetEndpointDiscover().AddProject("event_log_event_http", monitorMessage.socketproxy)
	}
	return monitorMessage
}

// Get get
func (d MonitorMessage) Get(w http.ResponseWriter, r *http.Request) {
	d.socketproxy.Proxy(w, r)
}

// EventLog event log
type EventLog struct {
	socketproxy proxy.Proxy
}

var eventLog *EventLog

// GetEventLog get event log
func GetEventLog() *EventLog {
	if eventLog == nil {
		eventLog = &EventLog{
			socketproxy: proxy.CreateProxy("eventlog", "websocket", configs.Default().APIConfig.EventLogEndpoints),
		}
		// discover.GetEndpointDiscover().AddProject("event_log_event_http", eventLog.socketproxy)
	}
	return eventLog
}

// Get get
func (d EventLog) Get(w http.ResponseWriter, r *http.Request) {
	d.socketproxy.Proxy(w, r)
}

// LogFile log file down server
type LogFile struct {
	Root string
}

var logFile *LogFile

// GetLogFile get  log file
func GetLogFile() *LogFile {
	root := os.Getenv("SERVICE_LOG_ROOT")
	if root == "" {
		root = constants.WTDataLogPath
	}
	logrus.Infof("service logs file root path is :%s", root)
	if logFile == nil {
		logFile = &LogFile{
			Root: root,
		}
	}
	return logFile
}

// Get get
func (d LogFile) Get(w http.ResponseWriter, r *http.Request) {
	gid := chi.URLParam(r, "gid")
	filename := chi.URLParam(r, "filename")
	filePath := path.Join(d.Root, gid, filename)
	if isExist(filePath) {
		http.ServeFile(w, r, filePath)
	} else {
		w.WriteHeader(404)
	}
}
func isExist(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil || os.IsExist(err)
}

// GetInstallLog get
func (d LogFile) GetInstallLog(w http.ResponseWriter, r *http.Request) {
	filename := chi.URLParam(r, "filename")
	filePath := d.Root + filename
	if isExist(filePath) {
		http.ServeFile(w, r, filePath)
	} else {
		w.WriteHeader(404)
	}
}

var pubSubControll *PubSubControll

// PubSubControll service pub sub
type PubSubControll struct {
	socketproxy proxy.Proxy
}

// GetPubSubControll get service pub sub controller
func GetPubSubControll() *PubSubControll {
	if pubSubControll == nil {
		pubSubControll = &PubSubControll{
			socketproxy: proxy.CreateProxy("dockerlog", "websocket", configs.Default().APIConfig.EventLogEndpoints),
		}
		// discover.GetEndpointDiscover().AddProject("event_log_event_http", pubSubControll.socketproxy)
	}
	return pubSubControll
}

// Get pubsub controller
func (d PubSubControll) Get(w http.ResponseWriter, r *http.Request) {
	serviceID := chi.URLParam(r, "serviceID")
	name, _ := handler.GetEventHandler().GetLogInstance(serviceID)
	if name != "" {
		// r.URL.Query().Add("host_id", name)
		r = r.WithContext(context.WithValue(r.Context(), proxy.ContextKey("host_id"), name))
	}
	d.socketproxy.Proxy(w, r)
}

// GetHistoryLog get service docker logs
func (d PubSubControll) GetHistoryLog(w http.ResponseWriter, r *http.Request) {
	serviceID := r.Context().Value(ctxutil.ContextKey("service_id")).(string)
	name, _ := handler.GetEventHandler().GetLogInstance(serviceID)
	if name != "" {
		// r.URL.Query().Add("host_id", name)
		r = r.WithContext(context.WithValue(r.Context(), proxy.ContextKey("host_id"), name))
	}
	d.socketproxy.Proxy(w, r)
}
