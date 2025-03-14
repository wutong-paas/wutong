// WUTONG, Application Management Platform
// Copyright (C) 2014-2017 Wutong Co., Ltd.

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

package monitor

import (
	"context"
	"net/http"

	kingpin "github.com/alecthomas/kingpin/v2"
	"github.com/go-kit/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/node_exporter/collector"
	"github.com/sirupsen/logrus"
	"github.com/wutong-paas/wutong/cmd/node/option"
	"github.com/wutong-paas/wutong/node/api"
	"github.com/wutong-paas/wutong/node/monitormessage"
	"github.com/wutong-paas/wutong/node/statsd"
	innerprometheus "github.com/wutong-paas/wutong/node/statsd/prometheus"
	etcdutil "github.com/wutong-paas/wutong/util/etcd"
)

// Manager Manager
type Manager interface {
	Start(errchan chan error) error
	Stop() error
	SetAPIRoute(apim *api.Manager) error
}

type manager struct {
	statsdExporter     *statsd.Exporter
	statsdRegistry     *innerprometheus.Registry
	nodeExporterRestry *prometheus.Registry
	meserver           *monitormessage.UDPServer
}

func createNodeExporterRestry() (*prometheus.Registry, error) {
	registry := prometheus.NewRegistry()
	filters := []string{"cpu", "diskstats", "filesystem",
		"ipvs", "loadavg", "meminfo", "netdev",
		"netclass", "netdev", "netstat",
		"uname", "mountstats", "nfs"}
	// init kingpin parse
	kingpin.CommandLine.Parse([]string{"--collector.mountstats=true"})
	nc, err := collector.NewNodeCollector(log.NewNopLogger(), filters...)
	if err != nil {
		return nil, err
	}
	for n := range nc.Collectors {
		logrus.Infof("node collector - %s", n)
	}
	err = registry.Register(nc)
	if err != nil {
		return nil, err
	}
	return registry, nil
}

// CreateManager CreateManager
func CreateManager(ctx context.Context, c *option.Conf) (Manager, error) {
	//statsd exporter
	statsdRegistry := innerprometheus.NewRegistry()
	exporter := statsd.CreateExporter(c.StatsdConfig, statsdRegistry)
	etcdClientArgs := &etcdutil.ClientArgs{
		Endpoints: c.EtcdEndpoints,
		CaFile:    c.EtcdCaFile,
		CertFile:  c.EtcdCertFile,
		KeyFile:   c.EtcdKeyFile,
	}
	meserver := monitormessage.CreateUDPServer(ctx, "0.0.0.0", 6666, etcdClientArgs)
	nodeExporterRestry, err := createNodeExporterRestry()
	if err != nil {
		return nil, err
	}
	manage := &manager{
		statsdExporter:     exporter,
		statsdRegistry:     statsdRegistry,
		nodeExporterRestry: nodeExporterRestry,
		meserver:           meserver,
	}
	return manage, nil
}

func (m *manager) Start(errchan chan error) error {
	if err := m.statsdExporter.Start(); err != nil {
		logrus.Errorf("start statsd exporter server error,%s", err.Error())
		return err
	}
	if err := m.meserver.Start(); err != nil {
		return err
	}

	return nil
}

func (m *manager) Stop() error {
	return nil
}

// ReloadStatsdMappConfig ReloadStatsdMappConfig
func (m *manager) ReloadStatsdMappConfig(w http.ResponseWriter, r *http.Request) {
	if err := m.statsdExporter.ReloadConfig(); err != nil {
		w.Write([]byte(err.Error()))
		w.WriteHeader(500)
	} else {
		w.Write([]byte("Success reload"))
		w.WriteHeader(200)
	}
}

// HandleStatsd statsd handle
func (m *manager) HandleStatsd(w http.ResponseWriter, r *http.Request) {
	gatherers := prometheus.Gatherers{
		prometheus.DefaultGatherer,
		m.statsdRegistry,
	}
	// Delegate http serving to Prometheus client library, which will call collector.Collect.
	h := promhttp.HandlerFor(gatherers,
		promhttp.HandlerOpts{
			ErrorLog:      logrus.StandardLogger(),
			ErrorHandling: promhttp.ContinueOnError,
		})
	h.ServeHTTP(w, r)
}

// NodeExporter node exporter
func (m *manager) NodeExporter(w http.ResponseWriter, r *http.Request) {
	gatherers := prometheus.Gatherers{
		prometheus.DefaultGatherer,
		m.nodeExporterRestry,
	}
	// Delegate http serving to Prometheus client library, which will call collector.Collect.
	h := promhttp.HandlerFor(gatherers,
		promhttp.HandlerOpts{
			ErrorLog:      logrus.StandardLogger(),
			ErrorHandling: promhttp.ContinueOnError,
		})
	h.ServeHTTP(w, r)
}

// SetAPIRoute set api route rule
func (m *manager) SetAPIRoute(apim *api.Manager) error {
	apim.GetRouter().Get("/app/metrics", m.HandleStatsd)
	apim.GetRouter().Get("/-/statsdreload", m.ReloadStatsdMappConfig)
	apim.GetRouter().Get("/node/metrics", m.NodeExporter)
	return nil
}
