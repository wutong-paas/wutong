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
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/wutong-paas/wutong/util"
	etcdutil "github.com/wutong-paas/wutong/util/etcd"
	grpcutil "github.com/wutong-paas/wutong/util/grpc"
	clientv3 "go.etcd.io/etcd/client/v3"
	naming "go.etcd.io/etcd/client/v3/naming/endpoints"
)

// KeepAlive 服务注册
type KeepAlive struct {
	cancel         context.CancelFunc
	EtcdClientArgs *etcdutil.ClientArgs
	ServerName     string
	HostName       string
	Endpoint       naming.Endpoint
	TTL            int64
	LID            clientv3.LeaseID
	Done           chan struct{}
	etcdClient     *clientv3.Client
	gRPCResolver   *grpcutil.GRPCResolver
	once           sync.Once
}

// CreateKeepAlive create keepalive for server
func CreateKeepAlive(etcdClientArgs *etcdutil.ClientArgs, ServerName string, Protocol string, HostIP string, Port int) (*KeepAlive, error) {
	if ServerName == "" || Port == 0 {
		return nil, fmt.Errorf("servername or serverport can not be empty")
	}
	if HostIP == "" {
		ip, err := util.LocalIP()
		if err != nil {
			logrus.Errorf("get ip failed,details %s", err.Error())
			return nil, err
		}
		HostIP = ip.String()
	}

	ctx, cancel := context.WithCancel(context.Background())
	etcdclient, err := etcdutil.NewClient(ctx, etcdClientArgs)
	if err != nil {
		cancel()
		return nil, err
	}

	k := &KeepAlive{
		EtcdClientArgs: etcdClientArgs,
		ServerName:     ServerName,
		Endpoint: naming.Endpoint{
			Addr: fmt.Sprintf("%s:%d", HostIP, Port),
		},
		TTL:        5,
		Done:       make(chan struct{}),
		etcdClient: etcdclient,
		cancel:     cancel,
	}
	if Protocol == "" {
		k.Endpoint = naming.Endpoint{
			Addr: fmt.Sprintf("%s:%d", HostIP, Port),
		}
	} else {
		k.Endpoint = naming.Endpoint{
			Addr: fmt.Sprintf("%s://%s:%d", Protocol, HostIP, Port),
		}
	}
	return k, nil
}

// Start 开始
func (k *KeepAlive) Start() error {
	// duration := time.Duration(k.TTL) * time.Second
	// timer := time.NewTimer(duration)
	// defer timer.Stop()

	go func() {
		duration := time.Duration(k.TTL) * time.Second
		timer := time.NewTimer(duration)
		defer timer.Stop()
		for {
			select {
			case <-k.Done:
				return
			case <-timer.C:
				if k.LID > 0 {
					func() {
						ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
						defer cancel()
						defer timer.Reset(duration)
						_, err := k.etcdClient.KeepAliveOnce(ctx, k.LID)
						if err == nil {
							return
						}
						logrus.Warnf("%s lid[%x] keepAlive err: %s, try to reset...", k.Endpoint, k.LID, err.Error())
						k.LID = 0
					}()
				} else {
					if err := k.reg(); err != nil {
						logrus.Warnf("%s set lid err: %s, try to reset after %d seconds...", k.Endpoint, err.Error(), k.TTL)
					} else {
						logrus.Infof("%s set lid[%x] success", k.Endpoint, k.LID)
					}
					timer.Reset(duration)
				}
			}
		}
	}()
	return nil
}

func (k *KeepAlive) etcdKey() string {
	return fmt.Sprintf("/wutong/discover/%s", k.ServerName)
}

func (k *KeepAlive) reg() error {
	k.gRPCResolver = &grpcutil.GRPCResolver{Client: k.etcdClient}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	resp, err := k.etcdClient.Grant(ctx, k.TTL+3)
	if err != nil {
		return err
	}
	if err := k.gRPCResolver.Update(ctx, k.etcdKey(), naming.Update{Op: naming.Add, Endpoint: k.Endpoint}, clientv3.WithLease(resp.ID)); err != nil {
		return err
	}
	logrus.Infof("Register a %s server endpoint %s to cluster", k.ServerName, k.Endpoint)
	k.LID = resp.ID
	return nil
}

// Stop 结束
func (k *KeepAlive) Stop() {
	k.once.Do(func() {
		close(k.Done)
		k.cancel()

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()
		if k.gRPCResolver != nil {
			if err := k.gRPCResolver.Update(ctx, k.etcdKey(), naming.Update{Op: naming.Delete, Endpoint: k.Endpoint}); err != nil {
				logrus.Errorf("cancel %s server endpoint %s from etcd error %s", k.ServerName, k.Endpoint, err.Error())
			} else {
				logrus.Infof("cancel %s server endpoint %s from etcd", k.ServerName, k.Endpoint)
			}
		}
	})
}
