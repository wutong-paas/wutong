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
	"os"
	"os/signal"
	"syscall"

	"github.com/eapache/channels"
	"github.com/sirupsen/logrus"
	"github.com/wutong-paas/wutong/cmd/worker/option"
	"github.com/wutong-paas/wutong/db"
	"github.com/wutong-paas/wutong/db/config"
	"github.com/wutong-paas/wutong/event"
	"github.com/wutong-paas/wutong/pkg/common"
	"github.com/wutong-paas/wutong/pkg/generated/clientset/versioned"

	// etcdutil "github.com/wutong-paas/wutong/util/etcd"
	k8sutil "github.com/wutong-paas/wutong/util/k8s"
	"github.com/wutong-paas/wutong/worker/appm/componentdefinition"
	"github.com/wutong-paas/wutong/worker/appm/controller"
	"github.com/wutong-paas/wutong/worker/appm/store"
	"github.com/wutong-paas/wutong/worker/discover"
	"github.com/wutong-paas/wutong/worker/gc"
	"github.com/wutong-paas/wutong/worker/master"
	"github.com/wutong-paas/wutong/worker/monitor"
	"github.com/wutong-paas/wutong/worker/server"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/flowcontrol"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Run start run
func Run(s *option.Worker) error {
	errChan := make(chan error, 2)
	dbconfig := config.Config{
		DBType:              s.Config.DBType,
		MysqlConnectionInfo: s.Config.MysqlConnectionInfo,
		EtcdEndPoints:       s.Config.EtcdEndPoints,
		EtcdTimeout:         s.Config.EtcdTimeout,
	}
	//step 1:db manager init ,event log client init
	if err := db.CreateManager(dbconfig); err != nil {
		return err
	}
	defer db.CloseManager()
	loggerManager, err := event.NewLoggerManager()
	if err != nil {
		return err
	}
	defer loggerManager.Close()

	//step 2 : create kube client and etcd client
	restConfig, err := k8sutil.NewRestConfig(s.Config.KubeConfig)
	if err != nil {
		logrus.Errorf("create kube rest config error: %s", err.Error())
		return err
	}
	restConfig.RateLimiter = flowcontrol.NewTokenBucketRateLimiter(float32(s.Config.KubeAPIQPS), s.Config.KubeAPIBurst)
	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		logrus.Errorf("create kube client error: %s", err.Error())
		return err
	}
	s.Config.KubeClient = clientset
	runtimeClient, err := client.New(restConfig, client.Options{Scheme: common.Scheme})
	if err != nil {
		logrus.Errorf("create kube runtime client error: %s", err.Error())
		return err
	}
	wutongClient := versioned.NewForConfigOrDie(restConfig)
	//step 3: create componentdefinition builder factory
	componentdefinition.NewComponentDefinitionBuilder(s.Config.WTNamespace)

	//step 4: create component resource store
	updateCh := channels.NewRingChannel(1024)
	cachestore := store.NewStore(restConfig, clientset, wutongClient, db.GetManager(), s.Config)
	if err := cachestore.Start(); err != nil {
		logrus.Error("start kube cache store error", err)
		return err
	}

	//step 5: create controller manager
	controllerManager := controller.NewManager(cachestore, clientset, runtimeClient)
	defer controllerManager.Stop()

	//step 6 : start runtime master
	masterCon, err := master.NewMasterController(s.Config, cachestore, clientset, wutongClient, restConfig)
	if err != nil {
		return err
	}
	if err := masterCon.Start(); err != nil {
		return err
	}
	defer masterCon.Stop()

	//step 7 : create discover module
	garbageCollector := gc.NewGarbageCollector(clientset)
	taskManager := discover.NewTaskManager(s.Config, cachestore, controllerManager, garbageCollector)
	if err := taskManager.Start(); err != nil {
		return err
	}
	defer taskManager.Stop()

	//step 8: start app runtimer server
	runtimeServer := server.CreaterRuntimeServer(s.Config, cachestore, clientset, updateCh)
	runtimeServer.Start(errChan)

	//step 9: create application use resource exporter.
	exporterManager := monitor.NewManager(s.Config, masterCon, controllerManager)
	if err := exporterManager.Start(); err != nil {
		return err
	}
	defer exporterManager.Stop()

	logrus.Info("worker begin running...")

	//step finally: listen Signal
	term := make(chan os.Signal, 1)
	signal.Notify(term, os.Interrupt, syscall.SIGTERM)
	select {
	case <-term:
		logrus.Warn("Received SIGTERM, exiting gracefully...")
	case err := <-errChan:
		if err != nil {
			logrus.Errorf("Received a error %s, exiting gracefully...", err.Error())
		}
	}
	logrus.Info("See you next time!")
	return nil
}
