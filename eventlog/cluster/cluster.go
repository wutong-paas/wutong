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

package cluster

import (
	"fmt"
	"time"

	"github.com/wutong-paas/wutong/eventlog/cluster/connect"
	"github.com/wutong-paas/wutong/eventlog/cluster/discover"
	"github.com/wutong-paas/wutong/eventlog/conf"
	"github.com/wutong-paas/wutong/eventlog/db"
	clientv3 "go.etcd.io/etcd/client/v3"

	"golang.org/x/net/context"

	"github.com/wutong-paas/wutong/eventlog/store"

	"github.com/wutong-paas/wutong/eventlog/cluster/distribution"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

// Cluster 集群模块对外服务
type Cluster interface {
	//获取一个承接日志的节点
	GetSuitableInstance(serviceID string) *discover.Instance
	//集群消息广播
	MessageRadio(...db.ClusterMessage)
	Start() error
	Stop()
	GetInstanceID() string
	GetInstanceHost() string
	Scrape(ch chan<- prometheus.Metric, namespace, exporter string) error
}

// ClusterManager 控制器
type ClusterManager struct {
	discover     discover.Manager
	zmqPub       *connect.Pub
	zmqSub       *connect.Sub
	distribution *distribution.Distribution
	Conf         conf.ClusterConf
	log          *logrus.Entry
	storeManager store.Manager
	cancel       func()
	context      context.Context
	etcdClient   *clientv3.Client
}

// NewCluster 创建集群控制器
func NewCluster(etcdClient *clientv3.Client, conf conf.ClusterConf, log *logrus.Entry, storeManager store.Manager) Cluster {
	ctx, cancel := context.WithCancel(context.Background())
	discover := discover.New(etcdClient, conf.Discover, log.WithField("module", "Discover"))
	distribution := distribution.NewDistribution(etcdClient, conf.Discover, discover, log.WithField("Module", "Distribution"))
	sub := connect.NewSub(conf.PubSub, log.WithField("module", "MessageSubManager"), storeManager, discover, distribution)
	pub := connect.NewPub(conf.PubSub, log.WithField("module", "MessagePubServer"), storeManager, discover)

	return &ClusterManager{
		discover:     discover,
		zmqSub:       sub,
		zmqPub:       pub,
		distribution: distribution,
		Conf:         conf,
		log:          log,
		storeManager: storeManager,
		cancel:       cancel,
		context:      ctx,
		etcdClient:   etcdClient,
	}
}

// Start 启动
func (s *ClusterManager) Start() error {
	if err := s.discover.Run(); err != nil {
		return err
	}
	if err := s.zmqPub.Run(); err != nil {
		return err
	}
	if err := s.zmqSub.Run(); err != nil {
		return err
	}
	if err := s.distribution.Start(); err != nil {
		return err
	}
	go s.monitor()
	return nil
}

// Stop 停止
func (s *ClusterManager) Stop() {
	s.cancel()
	s.distribution.Stop()
	s.zmqPub.Stop()
	s.zmqSub.Stop()
	s.discover.Stop()
}

// GetSuitableInstance 获取适合的日志接收节点
func (s *ClusterManager) GetSuitableInstance(serviceID string) *discover.Instance {
	return s.distribution.GetSuitableInstance(serviceID)
}

// MessageRadio 消息广播
func (s *ClusterManager) MessageRadio(mes ...db.ClusterMessage) {
	for _, m := range mes {
		s.zmqPub.RadioChan <- m
	}
}

func (s *ClusterManager) GetInstanceID() string {
	return s.discover.GetCurrentInstance().HostID
}

func (s *ClusterManager) GetInstanceHost() string {
	return s.discover.GetCurrentInstance().HostIP.String()
}

func (s *ClusterManager) monitor() {
	ticker := time.NewTicker(time.Second * 5)
	defer ticker.Stop()
	for {
		messages := s.storeManager.Monitor()
		for _, m := range messages {
			me := db.ClusterMessage{Mode: db.MonitorMessage,
				Data: []byte(fmt.Sprintf("%s,%d,%d", s.GetInstanceID(), m.ServiceSize, m.LogSizePeerM))}
			s.MessageRadio(me)
			m.InstanceID = s.GetInstanceID()
			s.distribution.Update(m)
		}
		select {
		case <-s.context.Done():
			return
		case <-ticker.C:
		}
	}
}

// Scrape prometheus monitor metrics
func (s *ClusterManager) Scrape(ch chan<- prometheus.Metric, namespace, exporter string) error {
	s.discover.Scrape(ch, namespace, exporter)
	return nil
}
