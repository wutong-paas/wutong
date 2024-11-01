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
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/wutong-paas/wutong/eventlog/conf"
	clientv3 "go.etcd.io/etcd/client/v3"

	"golang.org/x/net/context"
)

// SaveDockerLogInInstance 存储service和node 的对应关系
func SaveDockerLogInInstance(etcdClient *clientv3.Client, conf conf.DiscoverConf, serviceID, instanceID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	_, err := etcdClient.Put(ctx, conf.HomePath+"/dockerloginstacne/"+serviceID, instanceID)
	if err != nil {
		logrus.Errorf("Failed to put dockerlog instance %v", err)
		return err
	}
	return nil
}

// GetDokerLogInInstance 获取应用日志接收节点
func GetDokerLogInInstance(etcdClient *clientv3.Client, conf conf.DiscoverConf, serviceID string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	res, err := etcdClient.Get(ctx, conf.HomePath+"/dockerloginstacne/"+serviceID)
	if err != nil {
		return "", err
	}
	if len(res.Kvs) == 0 {
		return "", fmt.Errorf("get docker log instance failed")
	}
	return string(res.Kvs[0].Value), nil
}
