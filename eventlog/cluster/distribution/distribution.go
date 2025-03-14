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

package distribution

import (
	"sync"
	"time"

	"github.com/wutong-paas/wutong/eventlog/cluster/discover"
	"github.com/wutong-paas/wutong/eventlog/conf"
	"github.com/wutong-paas/wutong/eventlog/db"
	clientv3 "go.etcd.io/etcd/client/v3"

	"golang.org/x/net/context"

	"sort"

	"github.com/sirupsen/logrus"
)

// Distribution 数据分区
type Distribution struct {
	monitorDatas map[string]*db.MonitorData
	updateTime   map[string]time.Time
	abnormalNode map[string]int
	lock         sync.Mutex
	cancel       func()
	context      context.Context
	discover     discover.Manager
	log          *logrus.Entry
	etcdClient   *clientv3.Client
	conf         conf.DiscoverConf
}

func NewDistribution(etcdClient *clientv3.Client, conf conf.DiscoverConf, dis discover.Manager, log *logrus.Entry) *Distribution {
	ctx, cancel := context.WithCancel(context.Background())
	d := &Distribution{
		cancel:       cancel,
		context:      ctx,
		discover:     dis,
		monitorDatas: make(map[string]*db.MonitorData),
		updateTime:   make(map[string]time.Time),
		abnormalNode: make(map[string]int),
		log:          log,
		etcdClient:   etcdClient,
		conf:         conf,
	}
	return d
}

// Start 开始健康监测
func (d *Distribution) Start() error {
	go d.checkHealth()
	return nil
}

// Stop 停止
func (d *Distribution) Stop() {
	d.cancel()
}

// Update 更新监控数据
func (d *Distribution) Update(m db.MonitorData) {
	d.lock.Lock()
	defer d.lock.Unlock()
	if m.InstanceID == "" {
		d.log.Warning("update monitor data but instance id is empty.")
		return
	}
	if md, ok := d.monitorDatas[m.InstanceID]; ok {
		md.LogSizePeerM = m.LogSizePeerM
		md.ServiceSize = m.ServiceSize
		// try delete abnormal node if exist
		delete(d.abnormalNode, m.InstanceID)
	} else {
		d.monitorDatas[m.InstanceID] = &m
	}
	d.updateTime[m.InstanceID] = time.Now()
}

func (d *Distribution) checkHealth() {
	tick := time.NewTicker(time.Second * 5)
	defer tick.Stop()
	for {
		select {
		case <-tick.C:
		case <-d.context.Done():
			return
		}
		d.lock.Lock()
		for k, v := range d.updateTime {
			if v.Add(time.Second * 10).Before(time.Now()) { // 如果是 10s 之前更新的节点，那么开始检查节点状态
				status := d.discover.InstanceCheckHealth(k)
				if status == "delete" { // 如果节点状态是删除，那么删除节点
					delete(d.monitorDatas, k)
					delete(d.updateTime, k)
					d.log.Errorf("instance (%s) health is delete.", k)
				}
				if status == "abnormal" { // 如果是异常节点，那么标记为异常
					d.abnormalNode[k] = 1
					d.log.Errorf("instance (%s) health is abnormal.", k)
				}
			}
		}
		d.lock.Unlock()
	}
}

// GetSuitableInstance 获取推荐节点
func (d *Distribution) GetSuitableInstance(serviceID string) *discover.Instance {
	d.lock.Lock()
	defer d.lock.Unlock()
	var suitableInstance *discover.Instance

	instanceID, err := discover.GetDokerLogInInstance(d.etcdClient, d.conf, serviceID)
	if err != nil {
		d.log.Error("Get docker log in instance id error ", err.Error())
	}
	if instanceID != "" {
		if _, ok := d.abnormalNode[instanceID]; !ok {
			if _, ok := d.monitorDatas[instanceID]; ok {
				suitableInstance = d.discover.GetInstance(instanceID)
				if suitableInstance != nil {
					return suitableInstance
				}
			}
		}
	}
	if len(d.monitorDatas) < 1 {
		ins := d.discover.GetCurrentInstance()
		d.log.Debug("monitor data length <1 return self")
		return &ins
	}
	d.log.Debug("start select suitable Instance")
	var flags []int
	var instances = make(map[int]*discover.Instance)
	for k, v := range d.monitorDatas {
		if _, ok := d.abnormalNode[k]; !ok {
			if ins := d.discover.GetInstance(k); ins != nil {
				flag := int(v.LogSizePeerM) + 20*v.ServiceSize
				flags = append(flags, flag)
				instances[flag] = ins
			} else {
				d.log.Debugf("instance %s stat is delete", k)
			}
		} else {
			d.log.Debugf("instance %s stat is abnormal", k)
		}
	}

	if len(flags) > 0 {
		sort.Ints(flags)
		suitableInstance = instances[flags[0]]
	}
	if suitableInstance == nil {
		d.log.Debug("suitableInstance is nil return self")
		ins := d.discover.GetCurrentInstance()
		return &ins
	}
	return suitableInstance
}
