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
	"bytes"
	"sync"
	"time"

	"github.com/wutong-paas/wutong/eventlog/conf"
	"github.com/wutong-paas/wutong/eventlog/db"

	"golang.org/x/net/context"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

type dockerLogStore struct {
	conf         conf.EventStoreConf
	log          *logrus.Entry
	barrels      map[string]*dockerLogEventBarrel
	rwLock       sync.RWMutex
	cancel       func()
	ctx          context.Context
	pool         *sync.Pool
	filePlugin   db.Manager
	LogSizePeerM int64
	LogSize      int64
	barrelSize   int
	barrelEvent  chan []string
	allLogCount  float64 //ues to pometheus monitor
}

func (d *dockerLogStore) Scrape(ch chan<- prometheus.Metric, namespace, exporter, from string) error {
	chanDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, exporter, "container_log_store_cache_barrel_count"),
		"the cache container log barrel size.",
		[]string{"from"}, nil,
	)
	ch <- prometheus.MustNewConstMetric(chanDesc, prometheus.GaugeValue, float64(len(d.barrels)), from)
	logDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, exporter, "container_log_store_log_count"),
		"the handle container log count size.",
		[]string{"from"}, nil,
	)
	ch <- prometheus.MustNewConstMetric(logDesc, prometheus.GaugeValue, d.allLogCount, from)

	return nil
}

func (d *dockerLogStore) insertMessage(message *db.EventLogMessage) bool {
	d.rwLock.RLock() //读锁
	defer d.rwLock.RUnlock()
	if ba, ok := d.barrels[message.EventID]; ok {
		ba.insertMessage(message)
		return true
	}
	return false
}

func (d *dockerLogStore) InsertMessage(message *db.EventLogMessage) {
	if message == nil || message.EventID == "" {
		return
	}
	d.LogSize++
	d.allLogCount++
	if ok := d.insertMessage(message); ok {
		return
	}
	d.rwLock.Lock()
	defer d.rwLock.Unlock()
	ba := d.pool.Get().(*dockerLogEventBarrel)
	ba.name = message.EventID
	ba.persistenceTime = time.Now()
	ba.insertMessage(message)
	d.barrels[message.EventID] = ba
	d.barrelSize++
}

func (d *dockerLogStore) subChan(eventID, subID string) chan *db.EventLogMessage {
	d.rwLock.RLock() //读锁
	defer d.rwLock.RUnlock()
	if ba, ok := d.barrels[eventID]; ok {
		ch := ba.addSubChan(subID)
		return ch
	}
	return nil
}

func (d *dockerLogStore) SubChan(eventID, subID string) chan *db.EventLogMessage {
	if ch := d.subChan(eventID, subID); ch != nil {
		return ch
	}
	d.rwLock.Lock()
	defer d.rwLock.Unlock()
	ba := d.pool.Get().(*dockerLogEventBarrel)
	ba.updateTime = time.Now()
	ba.name = eventID
	d.barrels[eventID] = ba
	return ba.addSubChan(subID)
}

func (d *dockerLogStore) ReleaseSubChan(eventID, subID string) {
	d.rwLock.RLock()
	defer d.rwLock.RUnlock()
	if ba, ok := d.barrels[eventID]; ok {
		ba.delSubChan(subID)
	}
}

func (d *dockerLogStore) Run() {
	go d.Gc()
	go d.handleBarrelEvent()
}

func (d *dockerLogStore) GetMonitorData() *db.MonitorData {
	data := &db.MonitorData{
		ServiceSize:  len(d.barrels),
		LogSizePeerM: d.LogSizePeerM,
	}
	if d.LogSizePeerM == 0 {
		data.LogSizePeerM = d.LogSize
	}
	return data
}

func (d *dockerLogStore) Gc() {
	tiker := time.NewTicker(time.Second * 30)
	defer tiker.Stop()
	for {
		select {
		case <-tiker.C:
			d.gcRun()
		case <-d.ctx.Done():
			d.log.Debug("docker log store gc stop.")
			return
		}
	}
}

func (d *dockerLogStore) handle() []string {
	d.rwLock.RLock()
	defer d.rwLock.RUnlock()
	if len(d.barrels) == 0 {
		return nil
	}
	var gcEvent []string
	for k := range d.barrels {
		if d.barrels[k].updateTime.Add(time.Minute*1).Before(time.Now()) && d.barrels[k].GetSubChanLength() == 0 {
			d.saveBeforeGc(k, d.barrels[k])
			gcEvent = append(gcEvent, k)
			d.log.Debugf("barrel %s need be gc", k)
		} else if d.barrels[k].persistenceTime.Add(time.Minute * 1).Before(time.Now()) {
			//The interval not persisted for more than 1 minute should be more than 30 seconds
			if len(d.barrels[k].barrel) > 0 {
				d.log.Debugf("barrel %s need persistence", k)
				d.barrels[k].persistence()
			}
		}
	}
	return gcEvent
}

func (d *dockerLogStore) gcRun() {
	t := time.Now()
	// 每分钟进行数据重置，获得每分钟日志量数据
	d.LogSizePeerM = d.LogSize
	d.LogSize = 0
	gcEvent := d.handle()
	if len(gcEvent) > 0 {
		d.rwLock.Lock()
		defer d.rwLock.Unlock()
		for _, id := range gcEvent {
			barrel := d.barrels[id]
			barrel.empty()
			d.pool.Put(barrel)
			delete(d.barrels, id)
			d.barrelSize--
			d.log.Debugf("docker log barrel(%s) gc complete", id)
		}
	}
	useTime := time.Since(t).Nanoseconds()
	d.log.Debugf("Docker log message store complete gc in %d ns", useTime)
}

func (d *dockerLogStore) stop() {
	d.cancel()
	d.rwLock.RLock()
	defer d.rwLock.RUnlock()
	for k, v := range d.barrels {
		d.saveBeforeGc(k, v)
	}
}

// gc删除前持久化数据
func (d *dockerLogStore) saveBeforeGc(eventID string, v *dockerLogEventBarrel) {
	v.persistencelock.Lock()
	v.gcPersistence()
	if len(v.persistenceBarrel) > 0 {
		if err := d.filePlugin.SaveMessage(v.persistenceBarrel); err != nil {
			d.log.Error("persistence barrel message error.", err.Error())
			d.InsertGarbageMessage(v.persistenceBarrel...)
		}
		d.log.Debugf("dockerLogStore.saveBeforeGc: persistence barrel(%s) %d log message to file.", eventID, len(v.persistenceBarrel))
	}
	v.persistenceBarrel = nil
	v.persistencelock.Unlock()
}

func (d *dockerLogStore) InsertGarbageMessage(message ...*db.EventLogMessage) {}

// TODO
func (d *dockerLogStore) handleBarrelEvent() {
	for {
		select {
		case event := <-d.barrelEvent:
			if len(event) < 1 {
				return
			}
			d.log.Debug("Handle message store do event.", event)
			if event[0] == "persistence" { //持久化命令
				d.persistence(event)
			}
		case <-d.ctx.Done():
			return
		}
	}
}

func (d *dockerLogStore) persistence(event []string) {
	if len(event) == 2 {
		eventID := event[1]
		d.rwLock.RLock()
		defer d.rwLock.RUnlock()
		if ba, ok := d.barrels[eventID]; ok {
			if ba.needPersistence { // 取消异步持久化
				if err := d.filePlugin.SaveMessage(ba.persistenceBarrel); err != nil {
					d.log.Error("persistence barrel message error.", err.Error())
					d.InsertGarbageMessage(ba.persistenceBarrel...)
				}
				d.log.Debugf("dockerLogStore.persistence: persistence barrel(%s) %d log message to file.", eventID, len(ba.persistenceBarrel))
				ba.persistenceBarrel = ba.persistenceBarrel[:0]
				ba.needPersistence = false
			}
		}
	}
}

func (d *dockerLogStore) GetHistoryMessage(eventID string, length int) (re []string) {
	d.rwLock.RLock()
	defer d.rwLock.RUnlock()
	if ba, ok := d.barrels[eventID]; ok {
		for _, m := range ba.barrel {
			if len(m.Content) > 0 {
				// v2 log archivement
				var content = m.Content
				if bytes.HasPrefix(content, []byte("v2:")) && len(content) > 23 {
					content = content[23:]
				}
				re = append(re, string(content))
			}
		}
	}
	logrus.Debugf("want length: %d; the length of re: %d;", length, len(re))
	if len(re) >= length && length > 0 {
		return re[:length-1]
	}
	filelength := func() int {
		if length-len(re) > 0 {
			return length - len(re)
		}
		return 0
	}()
	result, err := d.filePlugin.GetMessages(eventID, "", filelength)
	if result == nil || err != nil {
		return re
	}
	re = append(result.([]string), re...)
	return re
}
