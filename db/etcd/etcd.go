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

package etcd

import (
	"context"
	"sync"

	"github.com/wutong-paas/wutong/db/config"
	"github.com/wutong-paas/wutong/db/model"

	etcdutil "github.com/wutong-paas/wutong/util/etcd"

	"github.com/coreos/etcd/clientv3"
	"github.com/sirupsen/logrus"
)

//Manager db manager
type Manager struct {
	client  *clientv3.Client
	config  config.Config
	initOne sync.Once
	models  []model.Interface
}

//CreateManager 创建manager
func CreateManager(config config.Config) (*Manager, error) {
	etcdClientArgs := &etcdutil.ClientArgs{
		Endpoints: config.EtcdEndPoints,
		CaFile:    config.EtcdCaFile,
		CertFile:  config.EtcdCertFile,
		KeyFile:   config.EtcdKeyFile,
	}
	cli, err := etcdutil.NewClient(context.Background(), etcdClientArgs)
	if err != nil {
		etcdutil.HandleEtcdError(err)
		return nil, err
	}
	manager := &Manager{
		client:  cli,
		config:  config,
		initOne: sync.Once{},
	}
	logrus.Debug("etcd db driver create")
	return manager, nil
}

//CloseManager 关闭管理器
func (m *Manager) CloseManager() error {
	return m.client.Close()
}
