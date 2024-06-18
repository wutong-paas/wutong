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

// Wutong datacenter api binary
package main

import (
	"context"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/wutong-paas/wutong/cmd"
	"github.com/wutong-paas/wutong/cmd/api/option"
	"github.com/wutong-paas/wutong/config/configs"
	"github.com/wutong-paas/wutong/pkg/component"
	"github.com/wutong-paas/wutong/pkg/wutong"

	"github.com/spf13/pflag"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "version" {
		cmd.ShowVersion("api")
	}
	s := option.NewAPIServer()
	s.AddFlags(pflag.CommandLine)
	pflag.Parse()
	s.SetLog()

	configs.SetDefault(&configs.Config{
		AppName:   "wt-api",
		APIConfig: s.Config,
	})
	// 启动 wt-api
	err := wutong.New(context.Background(), configs.Default()).Registry(component.Database()).
		Registry(component.Grpc()).
		Registry(component.Event()).
		Registry(component.K8sClient()).
		Registry(component.HubRegistry()).
		Registry(component.Proxy()).
		Registry(component.Etcd()).
		Registry(component.MQ()).
		Registry(component.Prometheus()).
		Registry(component.Handler()).
		Registry(component.Router()).
		Start()
	if err != nil {
		logrus.Errorf("start wt-api error %s", err.Error())
	}
}
