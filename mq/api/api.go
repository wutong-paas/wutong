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

package api

import (
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/wutong-paas/wutong/cmd/mq/option"
	"github.com/wutong-paas/wutong/mq/api/controller"
	"github.com/wutong-paas/wutong/mq/api/mq"
	"github.com/wutong-paas/wutong/mq/monitor"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"golang.org/x/net/context"

	_ "net/http/pprof"

	restful "github.com/emicklei/go-restful"
	swagger "github.com/emicklei/go-restful-swagger12"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/version"
	"github.com/sirupsen/logrus"
	grpcserver "github.com/wutong-paas/wutong/mq/api/grpc/server"
	httputil "github.com/wutong-paas/wutong/util/http"
)

type Manager struct {
	container *restful.Container
	ctx       context.Context
	cancel    context.CancelFunc
	conf      option.Config
	server    Server
	actionMQ  mq.ActionMQ
}
type Server interface {
	Server() error
	Close() error
}

type httpServer struct {
	server *http.Server
}

func (h *httpServer) Server() error {
	if err := h.server.ListenAndServe(); err != nil {
		logrus.Error("mq api http listen error.", err.Error())
		return err
	}
	return nil
}
func (h *httpServer) Close() error {
	if h.server != nil {
		ctx, _ := context.WithTimeout(context.Background(), time.Second*5)
		return h.server.Shutdown(ctx)
	}
	return nil
}

type grpcServer struct {
	server *grpc.Server
	lis    net.Listener
}

func (h *grpcServer) Server() error {
	if err := h.server.Serve(h.lis); err != nil {
		logrus.Error("mq api grpc listen error.", err.Error())
		return err
	}
	return nil
}
func (h *grpcServer) Close() error {
	return h.lis.Close()
}

//NewManager api manager
func NewManager(c option.Config) (*Manager, error) {
	ctx, cancel := context.WithCancel(context.Background())
	actionMQ := mq.NewActionMQ(ctx, c)
	manager := &Manager{
		ctx:      ctx,
		cancel:   cancel,
		conf:     c,
		actionMQ: actionMQ,
	}
	go func() {
		manager.Prometheus()
		health()
		if err := http.ListenAndServe(":6301", nil); err != nil {
			logrus.Error("mq pprof listen error.", err.Error())
		}
	}()
	if c.RunMode == "http" {
		wsContainer := restful.NewContainer()
		server := &http.Server{Addr: fmt.Sprintf(":%d", c.APIPort), Handler: wsContainer}
		controller.Register(wsContainer, actionMQ)
		manager.container = wsContainer
		manager.server = &httpServer{server}
		manager.doc()
		logrus.Info("mq server api run with http")
	} else {
		lis, err := net.Listen("tcp", fmt.Sprintf(":%d", c.APIPort))
		if err != nil {
			logrus.Errorf("failed to listen: %v", err)
			return nil, err
		}
		s := grpc.NewServer()
		grpcserver.RegisterServer(s, actionMQ)
		// Register reflection service on gRPC server.
		reflection.Register(s)
		manager.server = &grpcServer{
			server: s,
			lis:    lis,
		}
		logrus.Info("mq server api run with gRPC")
	}

	return manager, nil
}

//Start 启动
func (m *Manager) Start(errChan chan error) {
	logrus.Infof("api server start listening on 0.0.0.0:%d", m.conf.APIPort)
	err := m.actionMQ.Start()
	if err != nil {
		errChan <- err
	}
	go func() {
		if err := m.server.Server(); err != nil {
			logrus.Error("mq api listen error.", err.Error())
			errChan <- err
		}
	}()
}

func (m *Manager) doc() {
	// Optionally, you can install the Swagger Service which provides a nice Web UI on your REST API
	// You need to download the Swagger HTML5 assets and change the FilePath location in the config below.
	// Open http://localhost:8080/apidocs and enter http://localhost:8080/swagger.json in the api input field.
	config := swagger.Config{
		WebServices: m.container.RegisteredWebServices(), // you control what services are visible
		ApiPath:     "/swagger.json",

		// Optionally, specify where the UI is located
		SwaggerPath: "/apidocs/",
		Info: swagger.Info{
			Title: "wutong mq api doc.",
		},
		ApiVersion:      "1.0",
		SwaggerFilePath: "./dist"}
	swagger.RegisterSwaggerService(config, m.container)

}

//Stop 停止
func (m *Manager) Stop() error {
	logrus.Info("api server is stoping.")
	m.cancel()
	//m.server.Close()
	return m.actionMQ.Stop()
}

//Prometheus prometheus init
func (m *Manager) Prometheus() {
	prometheus.MustRegister(version.NewCollector("acp_mq"))
	exporter := monitor.NewExporter(m.actionMQ)
	prometheus.MustRegister(exporter)
	http.Handle("/metrics", promhttp.Handler())
}

func health() {
	http.HandleFunc("/health", checkHalth)
}

func checkHalth(w http.ResponseWriter, r *http.Request) {
	httputil.ReturnSuccess(r, w, map[string]string{"status": "health", "info": "mq service health"})
}
