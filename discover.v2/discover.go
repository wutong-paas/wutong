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
	"strings"
	"sync"
	"time"

	"github.com/wutong-paas/wutong/discover/config"
	clientv3 "go.etcd.io/etcd/client/v3"

	"golang.org/x/net/context"

	"github.com/sirupsen/logrus"
	etcdutil "github.com/wutong-paas/wutong/util/etcd"
	"go.etcd.io/etcd/api/v3/mvccpb"
)

// CallbackUpdate 每次返还变化
type CallbackUpdate interface {
	//TODO:
	//weight自动发现更改实现暂时不 Ready
	UpdateEndpoints(operation config.Operation, endpoints ...*config.Endpoint)
	//when watch occurred error,will exec this method
	Error(error)
}

// Callback 每次返回全部节点
type Callback interface {
	UpdateEndpoints(endpoints ...*config.Endpoint)
	//when watch occurred error,will exec this method
	Error(error)
}

// Discover 后端服务自动发现
type Discover interface {
	// Add project to cache if not exists, then watch the endpoints.
	AddProject(name string, callback Callback)
	// Update a project.
	AddUpdateProject(name string, callback CallbackUpdate)
	Stop()
}

// GetDiscover 获取服务发现管理器
func GetDiscover(opt config.DiscoverConfig) (Discover, error) {
	if opt.Ctx == nil {
		opt.Ctx = context.Background()
	}
	ctx, cancel := context.WithCancel(opt.Ctx)
	client, err := etcdutil.NewClient(ctx, opt.EtcdClientArgs)
	if err != nil {
		cancel()
		return nil, err
	}
	etcdD := &etcdDiscover{
		projects: make(map[string]CallbackUpdate),
		ctx:      ctx,
		cancel:   cancel,
		client:   client,
		prefix:   "/wutong/discover",
	}
	return etcdD, nil
}

type etcdDiscover struct {
	projects map[string]CallbackUpdate
	lock     sync.Mutex
	ctx      context.Context
	cancel   context.CancelFunc
	client   *clientv3.Client
	prefix   string
}
type defaultCallBackUpdate struct {
	endpoints map[string]*config.Endpoint
	callback  Callback
	lock      sync.Mutex
}

func (d *defaultCallBackUpdate) UpdateEndpoints(operation config.Operation, endpoints ...*config.Endpoint) {
	d.lock.Lock()
	defer d.lock.Unlock()
	switch operation {
	case config.ADD:
		for _, e := range endpoints {
			if old, ok := d.endpoints[e.Name]; !ok {
				d.endpoints[e.Name] = e
			} else {
				if e.Mode == 0 {
					old.URL = e.URL
				}
				if e.Mode == 1 {
					old.Weight = e.Weight
				}
				if e.Mode == 2 {
					old.URL = e.URL
					old.Weight = e.Weight
				}
			}
		}
	case config.SYNC:
		for _, e := range endpoints {
			if old, ok := d.endpoints[e.Name]; !ok {
				d.endpoints[e.Name] = e
			} else {
				if e.Mode == 0 {
					old.URL = e.URL
				}
				if e.Mode == 1 {
					old.Weight = e.Weight
				}
				if e.Mode == 2 {
					old.URL = e.URL
					old.Weight = e.Weight
				}
			}
		}
	case config.DELETE:
		for _, e := range endpoints {
			if e.Mode == 0 {
				if old, ok := d.endpoints[e.Name]; ok {
					old.URL = ""
				}
			}
			if e.Mode == 1 {
				if old, ok := d.endpoints[e.Name]; ok {
					old.Weight = 0
				}
			}
			if e.Mode == 2 {
				delete(d.endpoints, e.Name)
			}
		}
	case config.UPDATE:
		for _, e := range endpoints {
			if e.Mode == 0 {
				if old, ok := d.endpoints[e.Name]; ok {
					old.URL = e.URL
				}
			}
			if e.Mode == 1 {
				if old, ok := d.endpoints[e.Name]; ok {
					old.Weight = e.Weight
				}
			}
			if e.Mode == 2 {
				if old, ok := d.endpoints[e.Name]; ok {
					old.URL = e.URL
					old.Weight = e.Weight
				}
			}
		}
	}
	var re []*config.Endpoint
	for _, v := range d.endpoints {
		if v.URL != "" {
			re = append(re, v)
		}
	}
	d.callback.UpdateEndpoints(re...)
}

func (d *defaultCallBackUpdate) Error(err error) {
	d.callback.Error(err)
}

func (e *etcdDiscover) AddProject(name string, callback Callback) {
	e.lock.Lock()
	defer e.lock.Unlock()
	if _, ok := e.projects[name]; !ok {
		cal := &defaultCallBackUpdate{
			callback:  callback,
			endpoints: make(map[string]*config.Endpoint),
		}
		e.projects[name] = cal
		go e.discover(name, cal)
	}
}

func (e *etcdDiscover) AddUpdateProject(name string, callback CallbackUpdate) {
	e.lock.Lock()
	defer e.lock.Unlock()
	if _, ok := e.projects[name]; !ok {
		e.projects[name] = callback
		go e.discover(name, callback)
	}
}

func (e *etcdDiscover) Stop() {
	e.cancel()
}

func (e *etcdDiscover) removeProject(name string) {
	e.lock.Lock()
	defer e.lock.Unlock()
	delete(e.projects, name)
}

func (e *etcdDiscover) discover(name string, callback CallbackUpdate) {
	ctx, cancel := context.WithCancel(e.ctx)
	defer cancel()
	endpoints := e.list(name)
	if len(endpoints) > 0 {
		callback.UpdateEndpoints(config.SYNC, endpoints...)
	}
	watch := e.client.Watch(ctx, fmt.Sprintf("%s/%s", e.prefix, name), clientv3.WithPrefix())
	timer := time.NewTimer(time.Second * 20)
	defer timer.Stop()
	for {
		select {
		case <-e.ctx.Done():
			return
		case <-timer.C:
			go e.discover(name, callback)
			return
		case res := <-watch:
			if err := res.Err(); err != nil {
				callback.Error(err)
				logrus.Debugf("monitor discover get watch error: %s, remove this watch target first, and then sleep 10 sec, we will re-watch it", err.Error())
				e.removeProject(name)
				time.Sleep(10 * time.Second)
				e.AddUpdateProject(name, callback)
				return
			}
			for _, event := range res.Events {
				if event.Kv != nil {
					var end *config.Endpoint
					kstep := strings.Split(string(event.Kv.Key), "/")
					if len(kstep) > 2 {
						serverName := kstep[len(kstep)-1]
						serverURL := string(event.Kv.Value)
						end = &config.Endpoint{Name: serverName, URL: serverURL, Mode: 0}
					}
					if end != nil { //获取服务地址
						switch event.Type {
						case mvccpb.DELETE:
							callback.UpdateEndpoints(config.DELETE, end)
						case mvccpb.PUT:
							if event.Kv.Version == 1 {
								callback.UpdateEndpoints(config.ADD, end)
							} else {
								callback.UpdateEndpoints(config.UPDATE, end)
							}
						}
					}
				}
			}
			timer.Reset(time.Second * 20)
		}
	}
}
func (e *etcdDiscover) list(name string) []*config.Endpoint {
	ctx, cancel := context.WithTimeout(e.ctx, time.Second*10)
	defer cancel()
	res, err := e.client.Get(ctx, fmt.Sprintf("%s/%s", e.prefix, name), clientv3.WithPrefix())
	if err != nil {
		logrus.Errorf("list all servers of %s error.%s", name, err.Error())
		return nil
	}
	if res.Count == 0 {
		return nil
	}
	return makeEndpointForKvs(res.Kvs)
}

func makeEndpointForKvs(kvs []*mvccpb.KeyValue) (res []*config.Endpoint) {
	var ends = make(map[string]*config.Endpoint)
	for _, kv := range kvs {
		//获取服务地址
		kstep := strings.Split(string(kv.Key), "/")
		if len(kstep) > 2 {
			serverName := kstep[len(kstep)-1]
			serverURL := string(kv.Value)
			if en, ok := ends[serverName]; ok {
				en.URL = serverURL
			} else {
				ends[serverName] = &config.Endpoint{Name: serverName, URL: serverURL}
			}
		}
	}
	for _, v := range ends {
		res = append(res, v)
	}
	return
}
