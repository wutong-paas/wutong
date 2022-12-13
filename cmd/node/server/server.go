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
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/sirupsen/logrus"
	"github.com/wutong-paas/wutong/cmd/node/option"
	"github.com/wutong-paas/wutong/discover.v2"
	eventLog "github.com/wutong-paas/wutong/event"
	"github.com/wutong-paas/wutong/node/api"
	"github.com/wutong-paas/wutong/node/api/controller"
	"github.com/wutong-paas/wutong/node/core/store"
	"github.com/wutong-paas/wutong/node/initiate"
	"github.com/wutong-paas/wutong/node/kubecache"
	"github.com/wutong-paas/wutong/node/masterserver"
	"github.com/wutong-paas/wutong/node/nodem"
	"github.com/wutong-paas/wutong/node/nodem/docker"
	"github.com/wutong-paas/wutong/node/nodem/envoy"
	"github.com/wutong-paas/wutong/util/constants"
	etcdutil "github.com/wutong-paas/wutong/util/etcd"
	k8sutil "github.com/wutong-paas/wutong/util/k8s"
	"k8s.io/client-go/kubernetes"
)

// Run start run
func Run(cfg *option.Conf) error {
	var stoped = make(chan struct{})
	stopfunc := func() error {
		close(stoped)
		return nil
	}
	startfunc := func() error {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		etcdClientArgs := &etcdutil.ClientArgs{
			Endpoints:   cfg.EtcdEndpoints,
			CaFile:      cfg.EtcdCaFile,
			CertFile:    cfg.EtcdCertFile,
			KeyFile:     cfg.EtcdKeyFile,
			DialTimeout: cfg.EtcdDialTimeout,
		}
		if err := cfg.ParseClient(ctx, etcdClientArgs); err != nil {
			return fmt.Errorf("config parse error:%s", err.Error())
		}

		config, err := k8sutil.NewRestConfig(cfg.K8SConfPath)
		if err != nil {
			return err
		}
		clientset, err := kubernetes.NewForConfig(config)
		if err != nil {
			return err
		}

		k8sDiscover := discover.NewK8sDiscover(ctx, clientset, cfg)
		defer k8sDiscover.Stop()

		nodemanager, err := nodem.NewNodeManager(ctx, cfg)
		if err != nil {
			return fmt.Errorf("create node manager failed: %s", err)
		}
		if err := nodemanager.InitStart(); err != nil {
			return err
		}

		err = eventLog.NewManager(eventLog.EventConfig{
			EventLogServers: cfg.EventLogServer,
			DiscoverArgs:    etcdClientArgs,
		})
		if err != nil {
			logrus.Errorf("error creating eventlog manager")
			return nil
		}
		defer eventLog.CloseManager()
		logrus.Debug("create and start event log client success")

		kubecli, err := kubecache.NewKubeClient(cfg, clientset)
		if err != nil {
			return err
		}
		defer kubecli.Stop()

		if cfg.ImageRepositoryHost == constants.DefImageRepository {
			hostManager, err := initiate.NewHostManager(cfg, k8sDiscover)
			if err != nil {
				return fmt.Errorf("create new host manager: %v", err)
			}
			hostManager.Start()
		}

		logrus.Debugf("wt-namespace=%s; wt-docker-secret=%s", os.Getenv("WT_NAMESPACE"), os.Getenv("WT_DOCKER_SECRET"))
		// sync docker inscure registries cert info into all wutong node
		if err = docker.SyncDockerCertFromSecret(clientset, os.Getenv("WT_NAMESPACE"), os.Getenv("WT_DOCKER_SECRET")); err != nil { // TODO fanyangyang namespace secretname
			return fmt.Errorf("sync docker cert from secret error: %s", err.Error())
		}

		// init etcd client
		if err = store.NewClient(ctx, cfg, etcdClientArgs); err != nil {
			return fmt.Errorf("Connect to ETCD %s failed: %s", cfg.EtcdEndpoints, err)
		}
		errChan := make(chan error, 3)
		if err := nodemanager.Start(errChan); err != nil {
			return fmt.Errorf("start node manager failed: %s", err)
		}
		defer nodemanager.Stop()
		logrus.Debug("create and start node manager moudle success")

		//master服务在node服务之后启动
		var ms *masterserver.MasterServer
		if cfg.RunMode == "master" {
			ms, err = masterserver.NewMasterServer(nodemanager.GetCurrentNode(), kubecli)
			if err != nil {
				logrus.Errorf(err.Error())
				return err
			}
			ms.Cluster.UpdateNode(nodemanager.GetCurrentNode())
			if err := ms.Start(errChan); err != nil {
				logrus.Errorf(err.Error())
				return err
			}
			defer ms.Stop(nil)
			logrus.Debug("create and start master server moudle success")
		}
		//create api manager
		apiManager := api.NewManager(*cfg, nodemanager.GetCurrentNode(), ms, kubecli)
		if err := apiManager.Start(errChan); err != nil {
			return err
		}
		if err := nodemanager.AddAPIManager(apiManager); err != nil {
			return err
		}
		defer apiManager.Stop()

		//create service mesh controller
		grpcserver, err := envoy.CreateDiscoverServerManager(clientset, *cfg)
		if err != nil {
			return err
		}
		if err := grpcserver.Start(errChan); err != nil {
			return err
		}
		defer grpcserver.Stop()

		logrus.Debug("create and start api server moudle success")

		defer controller.Exist(nil)
		//step finally: listen Signal
		term := make(chan os.Signal, 1)
		signal.Notify(term, os.Interrupt, syscall.SIGTERM)
		select {
		case <-stoped:
			logrus.Infof("windows service stoped..")
		case <-term:
			logrus.Warn("Received SIGTERM, exiting gracefully...")
		case err := <-errChan:
			logrus.Errorf("Received a error %s, exiting gracefully...", err.Error())
		}
		logrus.Info("See you next time!")
		return nil
	}
	err := initService(cfg, startfunc, stopfunc)
	if err != nil {
		return err
	}
	return nil
}
