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

package masterserver

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/wutong-paas/wutong/node/kubecache"
	"github.com/wutong-paas/wutong/node/nodem/client"

	"github.com/wutong-paas/wutong/node/core/config"
	"github.com/wutong-paas/wutong/node/core/store"
	"github.com/wutong-paas/wutong/node/masterserver/node"
)

//MasterServer 主节点服务
type MasterServer struct {
	*store.Client
	*client.HostNode
	Cluster          *node.Cluster
	ctx              context.Context
	cancel           context.CancelFunc
	datacenterConfig *config.DataCenterConfig
}

//NewMasterServer 创建master节点
func NewMasterServer(modelnode *client.HostNode, kubecli kubecache.KubeClient) (*MasterServer, error) {
	datacenterConfig := config.GetDataCenterConfig()
	ctx, cancel := context.WithCancel(context.Background())
	nodecluster := node.CreateCluster(kubecli, modelnode, datacenterConfig)
	ms := &MasterServer{
		Client:           store.DefalutClient,
		HostNode:         modelnode,
		Cluster:          nodecluster,
		ctx:              ctx,
		cancel:           cancel,
		datacenterConfig: datacenterConfig,
	}
	return ms, nil
}

//Start master node start
func (m *MasterServer) Start(errchan chan error) error {
	m.datacenterConfig.Start()
	if err := m.Cluster.Start(errchan); err != nil {
		logrus.Error("node cluster start error,", err.Error())
		return err
	}
	return nil
}

//Stop 停止
func (m *MasterServer) Stop(i interface{}) {
	if m.Cluster != nil {
		m.Cluster.Stop(i)
	}
	m.cancel()
}
