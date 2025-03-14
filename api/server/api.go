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

package server

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"github.com/wutong-paas/wutong/api/api_routers/doc"
	"github.com/wutong-paas/wutong/api/api_routers/license"
	"github.com/wutong-paas/wutong/api/api_routers/version2"
	"github.com/wutong-paas/wutong/api/api_routers/websocket"
	"github.com/wutong-paas/wutong/api/handler"
	"github.com/wutong-paas/wutong/api/metric"
	apimiddleware "github.com/wutong-paas/wutong/api/middleware"
	"github.com/wutong-paas/wutong/cmd/api/option"
	"github.com/wutong-paas/wutong/pkg/interceptors"
	"github.com/wutong-paas/wutong/util"
	clientv3 "go.etcd.io/etcd/client/v3"
)

// Manager apiserver
type Manager struct {
	ctx      context.Context
	cancel   context.CancelFunc
	conf     option.Config
	stopChan chan struct{}
	r        *chi.Mux
	etcdcli  *clientv3.Client
	exporter *metric.Exporter
}

// NewManager newManager
func NewManager(c option.Config, etcdcli *clientv3.Client) *Manager {
	ctx, cancel := context.WithCancel(context.Background())
	manager := &Manager{
		ctx:      ctx,
		cancel:   cancel,
		conf:     c,
		stopChan: make(chan struct{}),
		etcdcli:  etcdcli,
	}
	r := chi.NewRouter()
	manager.r = r
	manager.SetMiddleware()
	return manager
}

// SetMiddleware set api meddleware
func (m *Manager) SetMiddleware() {
	c := m.conf
	r := m.r
	r.Use(m.RequestMetric)
	r.Use(middleware.RequestID)
	//Sets a http.Request's RemoteAddr to either X-Forwarded-For or X-Real-IP
	r.Use(middleware.RealIP)
	//Logs the start and end of each request with the elapsed processing time
	if c.LoggerFile != "" {
		logerFile, err := os.OpenFile(c.LoggerFile, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
		if err != nil {
			logrus.Errorf("open logger file %s error %s", c.LoggerFile, err.Error())
			r.Use(middleware.DefaultLogger)
		} else {
			requestLog := middleware.RequestLogger(&middleware.DefaultLogFormatter{Logger: log.New(logerFile, "", log.LstdFlags)})
			r.Use(requestLog)
		}
	} else {
		r.Use(middleware.DefaultLogger)
	}
	//Gracefully absorb panics and prints the stack trace
	// r.Use(middleware.Recoverer)
	r.Use(interceptors.Recoverer)
	//request time out
	// r.Use(middleware.Timeout(time.Second * 5))
	// set timeout middleware for different paths
	r.Use(func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			timeout := time.Second * 5
			if strings.Contains(r.URL.Path, "/obs") {
				timeout = time.Minute
			}
			if strings.Contains(r.URL.Path, "/console/filebrowser") {
				timeout = time.Minute * 10
			}
			// pod logs(sse)
			if strings.Contains(r.URL.Path, "/instances") && strings.HasSuffix(r.URL.Path, "/logs") {
				timeout = time.Minute * 30
			}
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer func() {
				cancel()
				if ctx.Err() == context.DeadlineExceeded {
					w.WriteHeader(http.StatusGatewayTimeout)
				}
			}()

			r = r.WithContext(ctx)
			next.ServeHTTP(w, r)
		}
		return http.HandlerFunc(fn)
	})
	//simple api version
	r.Use(apimiddleware.APIVersion)
	r.Use(apimiddleware.Proxy)
	// license
	r.Use(apimiddleware.License)
}

// Start manager
func (m *Manager) Start() error {
	go m.Do()
	logrus.Info("start api router success.")
	return nil
}

// Do do
func (m *Manager) Do() {
	for {
		select {
		case <-m.ctx.Done():
			return
		default:
			m.Run()
		}
	}
}

// Stop manager
func (m *Manager) Stop() error {
	logrus.Info("api router is stopped.")
	m.cancel()
	return nil
}

// Run run
func (m *Manager) Run() {
	v2R := &version2.V2{
		Cfg: &m.conf,
	}
	m.Metric()
	if m.conf.Debug {
		util.ProfilerSetup(m.r)
	}
	m.r.Get("/monitor", func(res http.ResponseWriter, req *http.Request) {
		res.Write([]byte("ok"))
	})
	m.r.Mount("/v2", v2R.Routes())
	m.r.Mount("/", doc.Routes())
	m.r.Mount("/license", license.Routes())
	//兼容老版docker
	m.r.Get("/v1/etcd/event-log/instances", m.EventLogInstance)

	//prometheus单节点代理
	m.r.Get("/api/v1/query", m.PrometheusAPI)
	m.r.Get("/api/v1/query_range", m.PrometheusAPI)
	//开启对浏览器的websocket服务和文件服务
	go func() {
		websocketRouter := chi.NewRouter()
		websocketRouter.Mount("/", websocket.Routes())
		websocketRouter.Mount("/logs", websocket.LogRoutes())
		websocketRouter.Mount("/app", websocket.AppRoutes())
		if m.conf.WebsocketSSL {
			logrus.Infof("websocket listen on (HTTPs) %s", m.conf.WebsocketAddr)
			logrus.Fatal(http.ListenAndServeTLS(m.conf.WebsocketAddr, m.conf.WebsocketCertFile, m.conf.WebsocketKeyFile, websocketRouter))
		} else {
			logrus.Infof("websocket listen on (HTTP) %s", m.conf.WebsocketAddr)
			logrus.Fatal(http.ListenAndServe(m.conf.WebsocketAddr, websocketRouter))
		}
	}()
	if m.conf.APISSL {
		go func() {
			pool := x509.NewCertPool()
			caCrt, err := os.ReadFile(m.conf.APICaFile)
			if err != nil {
				logrus.Fatal("ReadFile ca err:", err)
				return
			}
			pool.AppendCertsFromPEM(caCrt)
			s := &http.Server{
				Addr:    m.conf.APIAddrSSL,
				Handler: m.r,
				TLSConfig: &tls.Config{
					ClientCAs:  pool,
					ClientAuth: tls.RequireAndVerifyClientCert,
					CipherSuites: []uint16{
						tls.TLS_AES_128_GCM_SHA256,
						tls.TLS_CHACHA20_POLY1305_SHA256,
						tls.TLS_AES_256_GCM_SHA384,
						tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
						tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
						tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
						tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
						tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
						tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
						tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256,
						tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
						tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256,
						tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
						tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
						tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
					},
				},
			}
			logrus.Infof("api listen on (HTTPs) %s", m.conf.APIAddrSSL)
			logrus.Fatal(s.ListenAndServeTLS(m.conf.APICertFile, m.conf.APIKeyFile))
		}()
	}
	// health check
	go func() {
		healthzRouter := chi.NewRouter()
		healthzRouter.Get("/healthz", func(res http.ResponseWriter, req *http.Request) {
			res.Write([]byte("ok"))
			res.WriteHeader(http.StatusOK)
		})
		logrus.Infof("health check listen on (HTTP) %s", m.conf.APIHealthzAddr)
		logrus.Fatal(http.ListenAndServe(m.conf.APIHealthzAddr, healthzRouter))
	}()
	logrus.Infof("api listen on (HTTP) %s", m.conf.APIAddr)
	logrus.Fatal(http.ListenAndServe(m.conf.APIAddr, m.r))
}

// EventLogInstance 查询event server instance
func (m *Manager) EventLogInstance(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithCancel(m.ctx)
	defer cancel()

	res, err := m.etcdcli.Get(ctx, "/event/instance", clientv3.WithPrefix())
	if err != nil {
		w.WriteHeader(500)
		return
	}
	if len(res.Kvs) > 0 {
		result := `{"data":{"instance":[`
		for _, kv := range res.Kvs {
			result += string(kv.Value) + ","
		}
		result = result[:len(result)-1] + `]},"ok":true}`
		w.Write([]byte(result))
		w.WriteHeader(200)
		return
	}
	w.WriteHeader(404)
}

// PrometheusAPI prometheus api 代理
func (m *Manager) PrometheusAPI(w http.ResponseWriter, r *http.Request) {
	handler.GetPrometheusProxy().Proxy(w, r)
}

// Metric prometheus metric
func (m *Manager) Metric() {
	exporter := metric.NewExporter()
	m.exporter = exporter
	prometheus.MustRegister(exporter)
	m.r.Handle("/metrics", promhttp.Handler())
}

// RequestMetric request metric midd
func (m *Manager) RequestMetric(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		defer func() {
			path := r.RequestURI
			if strings.Contains(r.RequestURI, "?") {
				path = r.RequestURI[:strings.Index(r.RequestURI, "?")]
			}
			m.exporter.RequestInc(ww.Status(), path)
		}()
		next.ServeHTTP(ww, r)
	}
	return http.HandlerFunc(fn)
}
