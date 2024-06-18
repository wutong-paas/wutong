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

// import (
// 	"context"
// 	"os"
// 	"os/signal"
// 	"syscall"

// 	"github.com/sirupsen/logrus"
// 	"github.com/wutong-paas/wutong/pkg/kube"
// 	"github.com/wutong-paas/wutong/api/controller"
// 	"github.com/wutong-paas/wutong/api/db"
// 	"github.com/wutong-paas/wutong/api/discover"
// 	"github.com/wutong-paas/wutong/api/handler"
// 	"github.com/wutong-paas/wutong/api/server"
// 	"github.com/wutong-paas/wutong/cmd/api/option"
// 	"github.com/wutong-paas/wutong/event"
// 	etcdutil "github.com/wutong-paas/wutong/util/etcd"
// 	"github.com/wutong-paas/wutong/worker/client"
// )

// // Run start run
// func Run(s *option.APIServer) error {
// 	ctx, cancel := context.WithCancel(context.Background())
// 	defer cancel()

// 	errChan := make(chan error)
// 	etcdClientArgs := &etcdutil.ClientArgs{
// 		Endpoints: s.Config.EtcdEndpoint,
// 		CaFile:    s.Config.EtcdCaFile,
// 		CertFile:  s.Config.EtcdCertFile,
// 		KeyFile:   s.Config.EtcdKeyFile,
// 	}
// 	//启动服务发现
// 	if _, err := discover.CreateEndpointDiscover(etcdClientArgs); err != nil {
// 		return err
// 	}
// 	//创建db manager
// 	if err := db.CreateDBManager(s.Config); err != nil {
// 		logrus.Debugf("create db manager error, %v", err)
// 		return err
// 	}
// 	//创建event manager
// 	if err := db.CreateEventManager(s.Config); err != nil {
// 		logrus.Debugf("create event manager error, %v", err)
// 	}

// 	config := kube.RegionRESTConfig()
// 	clientset := kube.RegionClientset()
// 	wutongClient := kube.RegionWutongClientset()
// 	k8sClient := kube.RegionRuntimeClient()
// 	dynamicClient := kube.RegionDynamicClient()
// 	apiextClient := kube.RegionAPIExtClientset()
// 	veleroClient := kube.RegionVeleroClientset()

// 	if err := event.NewManager(event.EventConfig{
// 		EventLogServers: s.Config.EventLogServers,
// 		DiscoverArgs:    etcdClientArgs,
// 	}); err != nil {
// 		return err
// 	}
// 	defer event.CloseManager()
// 	//create app status client
// 	cli, err := client.NewClient(ctx, client.AppRuntimeSyncClientConf{
// 		EtcdEndpoints: s.Config.EtcdEndpoint,
// 		EtcdCaFile:    s.Config.EtcdCaFile,
// 		EtcdCertFile:  s.Config.EtcdCertFile,
// 		EtcdKeyFile:   s.Config.EtcdKeyFile,
// 		NonBlock:      s.Config.Debug,
// 	})
// 	if err != nil {
// 		logrus.Errorf("create app status client error, %v", err)
// 		return err
// 	}

// 	etcdcli, err := etcdutil.NewClient(ctx, etcdClientArgs)
// 	if err != nil {
// 		logrus.Errorf("create etcd client v3 error, %v", err)
// 		return err
// 	}

// 	//初始化 middleware
// 	handler.InitProxy(s.Config)
// 	//创建handle
// 	if err := handler.InitHandle(s.Config, etcdClientArgs, cli, etcdcli, config, clientset, wutongClient, k8sClient, dynamicClient, apiextClient, veleroClient); err != nil {
// 		logrus.Errorf("init all handle error, %v", err)
// 		return err
// 	}
// 	//创建v2Router manager
// 	if err := controller.CreateV2RouterManager(s.Config, cli); err != nil {
// 		logrus.Errorf("create v2 route manager error, %v", err)
// 	}
// 	// 启动api
// 	apiManager := server.NewManager(s.Config, etcdcli)
// 	if err := apiManager.Start(); err != nil {
// 		return err
// 	}
// 	defer apiManager.Stop()
// 	logrus.Info("api router is running...")

// 	//step finally: listen Signal
// 	term := make(chan os.Signal, 1)
// 	signal.Notify(term, os.Interrupt, syscall.SIGTERM)
// 	select {
// 	case s := <-term:
// 		logrus.Infof("Received a Signal  %s, exiting gracefully...", s.String())
// 	case err := <-errChan:
// 		logrus.Errorf("Received a error %s, exiting gracefully...", err.Error())
// 	}
// 	logrus.Info("See you next time!")
// 	return nil
// }
