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

package discover

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/wutong-paas/wutong/cmd/worker/option"
	"github.com/wutong-paas/wutong/mq/api/grpc/pb"
	"github.com/wutong-paas/wutong/mq/client"
	etcdutil "github.com/wutong-paas/wutong/util/etcd"
	"github.com/wutong-paas/wutong/worker/appm/controller"
	"github.com/wutong-paas/wutong/worker/appm/store"
	"github.com/wutong-paas/wutong/worker/discover/model"
	"github.com/wutong-paas/wutong/worker/gc"
	"github.com/wutong-paas/wutong/worker/handle"
	grpc1 "google.golang.org/grpc"
)

var healthStatus = make(map[string]string, 1)

//TaskNum exec task number
var TaskNum float64

//TaskError exec error task number
var TaskError float64

//TaskManager task
type TaskManager struct {
	ctx           context.Context
	cancel        context.CancelFunc
	config        option.Config
	handleManager *handle.Manager
	client        client.MQClient
}

//NewTaskManager return *TaskManager
func NewTaskManager(cfg option.Config,
	store store.Storer,
	controllermanager *controller.Manager,
	garbageCollector *gc.GarbageCollector) *TaskManager {

	ctx, cancel := context.WithCancel(context.Background())
	handleManager := handle.NewManager(ctx, cfg, store, controllermanager, garbageCollector)
	healthStatus["status"] = "health"
	healthStatus["info"] = "worker service health"
	return &TaskManager{
		ctx:           ctx,
		cancel:        cancel,
		config:        cfg,
		handleManager: handleManager,
	}
}

//Start 启动
func (t *TaskManager) Start() error {
	etcdClientArgs := &etcdutil.ClientArgs{
		Endpoints: t.config.EtcdEndPoints,
		CaFile:    t.config.EtcdCaFile,
		CertFile:  t.config.EtcdCertFile,
		KeyFile:   t.config.EtcdKeyFile,
	}
	client, err := client.NewMqClient(etcdClientArgs, t.config.MQAPI)
	if err != nil {
		logrus.Errorf("new Mq client error, %v", err)
		healthStatus["status"] = "unusual"
		healthStatus["info"] = fmt.Sprintf("new Mq client error, %v", err)
		return err
	}
	t.client = client
	go t.Do()
	logrus.Info("start discover success.")
	return nil
}

//Do do
func (t *TaskManager) Do() {
	logrus.Info("start receive task from mq")
	hostname, _ := os.Hostname()
	for {
		select {
		case <-t.ctx.Done():
			return
		default:
			data, err := t.client.Dequeue(t.ctx, &pb.DequeueRequest{Topic: client.WorkerTopic, ClientHost: hostname + "-worker"})
			if err != nil {
				if grpc1.ErrorDesc(err) == context.DeadlineExceeded.Error() {
					continue
				}
				if grpc1.ErrorDesc(err) == "context canceled" {
					logrus.Info("receive task core context canceled")
					healthStatus["status"] = "unusual"
					healthStatus["info"] = "receive task core context canceled"
					return
				}
				if grpc1.ErrorDesc(err) == "context timeout" {
					continue
				}
				logrus.Error("receive task error.", err.Error())
				time.Sleep(time.Second * 2)
				continue
			}
			logrus.Debugf("receive a task: %v", data)
			transData, err := model.TransTask(data)
			if err != nil {
				logrus.Error("trans mq msg data error ", err.Error())
				continue
			}
			rc := t.handleManager.AnalystToExec(transData)
			if rc != nil && rc != handle.ErrCallback {
				logrus.Warningf("execute task: %v", rc)
				TaskError++
			} else if rc != nil && rc == handle.ErrCallback {
				logrus.Errorf("err callback; analyst to exet: %v", rc)
				ctx, cancel := context.WithCancel(t.ctx)
				reply, err := t.client.Enqueue(ctx, &pb.EnqueueRequest{
					Topic:   client.WorkerTopic,
					Message: data,
				})
				cancel()
				logrus.Debugf("retry send task to mq ,reply is %v", reply)
				if err != nil {
					logrus.Errorf("enqueue task %v to mq topic %v Error", data, client.WorkerTopic)
					continue
				}
				//if handle is waiting, sleep 3 second
				time.Sleep(time.Second * 3)
			} else {
				TaskNum++
			}
		}
	}
}

//Stop 停止
func (t *TaskManager) Stop() error {
	logrus.Info("discover manager is stoping.")
	t.cancel()
	if t.client != nil {
		t.client.Close()
	}
	return nil
}

//HealthCheck health check
func HealthCheck() map[string]string {
	return healthStatus
}
