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

package store

import (
	"sync"
	"time"

	"github.com/wutong-paas/wutong/eventlog/conf"
	"github.com/wutong-paas/wutong/eventlog/db"

	"golang.org/x/net/context"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

type readMessageStore struct {
	conf    conf.EventStoreConf
	log     *logrus.Entry
	barrels map[string]*readEventBarrel
	lock    sync.Mutex
	cancel  func()
	ctx     context.Context
	pool    *sync.Pool
}

func (r *readMessageStore) Scrape(ch chan<- prometheus.Metric, namespace, exporter, from string) error {
	return nil
}
func (r *readMessageStore) InsertMessage(message *db.EventLogMessage) {
	if message == nil || message.EventID == "" {
		return
	}
	r.lock.Lock()
	defer r.lock.Unlock()
	if ba, ok := r.barrels[message.EventID]; ok {
		ba.insertMessage(message)
	} else {
		ba := r.pool.Get().(*readEventBarrel)
		ba.insertMessage(message)
		r.barrels[message.EventID] = ba
	}
}

func (r *readMessageStore) GetMonitorData() *db.MonitorData {
	return nil
}

func (r *readMessageStore) SubChan(eventID, subID string) chan *db.EventLogMessage {
	r.lock.Lock()
	defer r.lock.Unlock()
	if ba, ok := r.barrels[eventID]; ok {
		return ba.addSubChan(subID)
	}
	ba := r.pool.Get().(*readEventBarrel)
	ba.updateTime = time.Now()
	r.barrels[eventID] = ba
	return ba.addSubChan(subID)
}

func (r *readMessageStore) ReleaseSubChan(eventID, subID string) {
	r.lock.Lock()
	defer r.lock.Unlock()
	if ba, ok := r.barrels[eventID]; ok {
		ba.delSubChan(subID)
	}
}

func (r *readMessageStore) Run() {
	go r.Gc()
}

func (r *readMessageStore) Gc() {
	tiker := time.NewTicker(time.Second * 30) // 30s 读取一次
	defer tiker.Stop()
	for {
		select {
		case <-tiker.C:
		case <-r.ctx.Done():
			r.log.Debug("read message store gc stop.")
			return
		}
		if len(r.barrels) == 0 {
			continue
		}
		for eventID, v := range r.barrels {
			if v.updateTime.Add(time.Minute * 2).Before(time.Now()) { // barrel 超时未收到消息
				barrel := r.barrels[eventID]
				barrel.empty()     // 清空
				r.pool.Put(barrel) //放回对象池
				delete(r.barrels, eventID)
			}
		}
	}
}

func (r *readMessageStore) stop() {
	r.cancel()
}

func (r *readMessageStore) InsertGarbageMessage(message ...*db.EventLogMessage) {}

func (r *readMessageStore) GetHistoryMessage(eventID string, length int) []string {
	return nil
}
