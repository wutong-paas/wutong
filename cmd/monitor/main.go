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

package main

import (
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/wutong-paas/wutong/cmd"

	"github.com/wutong-paas/wutong/monitor/custom"

	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"github.com/wutong-paas/wutong/cmd/monitor/option"
	"github.com/wutong-paas/wutong/monitor"
	"github.com/wutong-paas/wutong/monitor/api"
	"github.com/wutong-paas/wutong/monitor/api/controller"
	"github.com/wutong-paas/wutong/monitor/prometheus"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "version" {
		cmd.ShowVersion("monitor")
	}
	c := option.NewConfig()
	c.AddFlag(pflag.CommandLine)
	c.AddPrometheusFlag(pflag.CommandLine)
	pflag.Parse()

	c.CompleteConfig()

	// start prometheus daemon and watching tis status in all time, exit monitor process if start failed
	a := prometheus.NewRulesManager(c)
	p := prometheus.NewManager(c, a)
	controllerManager := controller.NewControllerManager(a, p)

	monitorMysql(c, p)
	monitorKSM(c, p)

	errChan := make(chan error, 1)
	defer close(errChan)
	p.StartDaemon(errChan)
	defer p.StopDaemon()

	// register prometheus address to etcd cluster
	p.Registry.Start()
	defer p.Registry.Stop()

	// start watching components from etcd, and update modify to prometheus config
	m := monitor.NewMonitor(c, p)
	m.Start()
	defer m.Stop()

	r := api.Server(controllerManager)
	logrus.Info("monitor api listen port 3329")
	go http.ListenAndServe(":3329", r)

	//step finally: listen Signal
	term := make(chan os.Signal)
	defer close(term)
	signal.Notify(term, os.Interrupt, syscall.SIGTERM)

	select {
	case <-term:
		logrus.Warn("Received SIGTERM, exiting monitor gracefully...")
	case err := <-errChan:
		if err != nil {
			logrus.Errorf("Received a error %s from prometheus, exiting monitor gracefully...", err.Error())
		}
	}
	logrus.Info("See you next time!")
}

func monitorMysql(c *option.Config, p *prometheus.Manager) {
	if strings.TrimSpace(c.MysqldExporter) != "" {
		metrics := strings.TrimSpace(c.MysqldExporter)
		logrus.Infof("add mysql metrics[%s] into prometheus", metrics)
		custom.AddMetrics(p, custom.Metrics{Name: "mysql", Path: "/metrics", Metrics: []string{metrics}, Interval: 30 * time.Second, Timeout: 15 * time.Second})
	}
}

func monitorKSM(c *option.Config, p *prometheus.Manager) {
	if strings.TrimSpace(c.KSMExporter) != "" {
		metrics := strings.TrimSpace(c.KSMExporter)
		logrus.Infof("add kube-state-metrics[%s] into prometheus", metrics)
		custom.AddMetrics(p, custom.Metrics{
			Name: "kubernetes",
			Path: "/metrics",
			Scheme: func() string {
				if strings.HasSuffix(metrics, "443") {
					return "https"
				}
				return "http"
			}(),
			Metrics: []string{metrics}, Interval: 30 * time.Second, Timeout: 10 * time.Second},
		)
	}
}
