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
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	cdb "github.com/wutong-paas/wutong/db"
	"github.com/wutong-paas/wutong/db/model"
	"github.com/wutong-paas/wutong/eventlog/conf"
	"github.com/wutong-paas/wutong/eventlog/db"
	"github.com/wutong-paas/wutong/eventlog/util"
	"golang.org/x/net/context"
)

type handleMessageStore struct {
	barrels                map[string]*EventBarrel // key 为 eventID
	lock                   sync.RWMutex            // 读写锁
	garbageLock            sync.Mutex
	conf                   conf.EventStoreConf
	log                    *logrus.Entry
	garbageMessage         []*db.EventLogMessage
	garbageGC              chan int
	ctx                    context.Context
	barrelEvent            chan []string
	eventfilePlugin        db.Manager // eventfile
	cancel                 func()
	handleEventGoroutines  int
	stopGarbage            chan struct{}
	pool                   *sync.Pool
	manager                *storeManager
	size                   int64
	allLogCount, allBarrel float64
}

func (h *handleMessageStore) Run() {
	go h.handleGarbageMessage()
	for i := 0; i < h.handleEventGoroutines; i++ {
		go h.handleBarrelEvent()
	}
	go h.Gc()
}
func (h *handleMessageStore) Scrape(ch chan<- prometheus.Metric, namespace, exporter, from string) error {
	chanDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, exporter, "event_store_cache_barrel_count"),
		"cache event barrel count.",
		[]string{"from"}, nil,
	)
	ch <- prometheus.MustNewConstMetric(chanDesc, prometheus.GaugeValue, float64(len(h.barrels)), from)
	logDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, exporter, "event_store_log_count"),
		"the handle event log count size.",
		[]string{"from"}, nil,
	)
	ch <- prometheus.MustNewConstMetric(logDesc, prometheus.GaugeValue, h.allLogCount, from)
	barrelDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, exporter, "event_store_barrel_count"),
		"all event barrel count.",
		[]string{"from"}, nil,
	)
	ch <- prometheus.MustNewConstMetric(barrelDesc, prometheus.GaugeValue, h.allBarrel, from)
	return nil
}

func (h *handleMessageStore) GetMonitorData() *db.MonitorData {
	da := &db.MonitorData{
		LogSizePeerM: h.size,
		ServiceSize:  len(h.barrels),
	}
	return da
}

func (h *handleMessageStore) SubChan(eventID, subID string) chan *db.EventLogMessage {
	return nil
}
func (h *handleMessageStore) ReleaseSubChan(eventID, subID string) {}

// GC 操作进行时 消息接收会停止
// TODD 怎么加快gc?
// 使用对象池
func (h *handleMessageStore) Gc() {
	h.log.Debug("Handle message store gc core start.")
	tiker := time.NewTicker(time.Second * 30)
	// defer tiker.Stop()
	for {
		select {
		case <-tiker.C:
			h.size = 0
			h.gcRun()
		case <-h.ctx.Done():
			h.log.Debug("Handle message store gc core stop.")
			return
		}
	}
}
func (h *handleMessageStore) gcRun() {
	h.lock.Lock()
	defer h.lock.Unlock()
	t := time.Now()
	if len(h.barrels) == 0 {
		return
	}
	var gcEvent []string
	for k, v := range h.barrels {
		if v.updateTime.Add(time.Second * 30).Before(time.Now()) {
			h.saveBeforeGc(h.barrels[k])
			gcEvent = append(gcEvent, k)
		}
	}
	if len(gcEvent) > 0 {
		for _, id := range gcEvent {
			barrel := h.barrels[id]
			barrel.empty()
			h.pool.Put(barrel)
			delete(h.barrels, id)
		}
	}
	useTime := time.Now().UnixNano() - t.UnixNano()
	h.log.Debugf("Handle message store complete gc in %d ns", useTime)
}

func (h *handleMessageStore) stop() {
	h.cancel()
	h.lock.Lock()
	defer h.lock.Unlock()
	h.log.Debug("start persistence message before handle Message Store stop.")
	for _, v := range h.barrels {
		h.saveBeforeGc(v)
	}
	close(h.stopGarbage)
	h.log.Debug("handle Message Store stop.")
}

// gc删除前持久化数据
func (h *handleMessageStore) saveBeforeGc(v *EventBarrel) {
	v.persistencelock.Lock()
	v.gcPersistence()
	if len(v.persistenceBarrel) > 0 {
		if err := h.eventfilePlugin.SaveMessage(v.persistenceBarrel); err != nil {
			h.log.Error("persistence barrel message error.", err.Error())
			h.InsertGarbageMessage(v.persistenceBarrel...)
		}
	}
	v.persistenceBarrel = nil
	v.persistencelock.Unlock()
	h.log.Debugf("Handle message store complete gc barrel(%s)", v.eventID)
}
func (h *handleMessageStore) insertMessage(message *db.EventLogMessage) bool {
	h.lock.RLock()
	defer h.lock.RUnlock()
	if barrel, ok := h.barrels[message.EventID]; ok {
		err := barrel.insert(message)
		if err != nil {
			h.log.Warn("insert message to barrel error.", err.Error())
			h.InsertGarbageMessage(message)
		}
		return true
	}
	return false
}
func (h *handleMessageStore) InsertMessage(message *db.EventLogMessage) {
	if message == nil || message.EventID == "" {
		return
	}
	h.size++
	h.allLogCount++
	eventID := message.EventID
	if h.insertMessage(message) {
		return
	}
	h.lock.Lock()
	defer h.lock.Unlock()
	barrel := h.pool.Get().(*EventBarrel)
	barrel.eventID = eventID
	err := barrel.insert(message)
	if err != nil {
		h.log.Warn("insert message to barrel error.", err.Error())
		h.InsertGarbageMessage(message)
	}
	h.barrels[eventID] = barrel
	h.allBarrel++
}

func (h *handleMessageStore) InsertGarbageMessage(message ...*db.EventLogMessage) {
	h.garbageLock.Lock()
	defer h.garbageLock.Unlock()
	h.garbageMessage = append(h.garbageMessage, message...)
}

func (h *handleMessageStore) handleGarbageMessage() {
	tick := time.NewTicker(10 * time.Second)
	defer tick.Stop()
	for {
		select {
		case <-tick.C:
			if len(h.garbageMessage) > 0 {
				h.saveGarbageMessage()
			}
		case <-h.garbageGC:
			h.saveGarbageMessage()
		case <-h.stopGarbage:
			h.saveGarbageMessage()
			h.log.Debug("handle message store garbage message handle-core stop.")
			return
		}
	}
}

// 保存垃圾消息到文件
func (h *handleMessageStore) saveGarbageMessage() {
	h.garbageLock.Lock()
	defer h.garbageLock.Unlock()
	var content string
	for _, m := range h.garbageMessage {
		if m != nil && m.Content != nil {
			content += fmt.Sprintf("(%s-%s) %s: %s\n", m.Step, m.Level, m.Time, m.Message)
		}
	}
	err := util.AppendToFile(h.conf.GarbageMessageFile, content)
	if err != nil {
		h.log.Error("Save garbage message to file error.context", err.Error())
	} else {
		h.log.Info("Save the garbage message to file.")
	}
	h.garbageMessage = h.garbageMessage[:0]
}

// 持久化事件，将事件日志写到 /wtdata/logs/eventlog/{eventID}.log 文件中
func (h *handleMessageStore) persistence(eventID string) {
	h.lock.RLock()
	defer h.lock.RUnlock()
	if ba, ok := h.barrels[eventID]; ok {
		ba.persistencelock.Lock()
		if ba.needPersistence {
			if err := h.eventfilePlugin.SaveMessage(ba.persistenceBarrel); err != nil {
				h.log.Error("persistence barrel message error.", err.Error())
				// 持久化失败，将数据放入垃圾消息中
				h.InsertGarbageMessage(ba.persistenceBarrel...)
			}
			h.log.Debugf("handleMessageStore.persistence: persistence barrel(%s) %d message  to db.", eventID, len(ba.persistenceBarrel))
			ba.persistenceBarrel = ba.persistenceBarrel[:0]
			ba.needPersistence = false
		}
		ba.persistencelock.Unlock()
	}
}

func (h *handleMessageStore) handleBarrelEvent() {
	for {
		select {
		case event := <-h.barrelEvent:
			if len(event) < 1 {
				continue
			}

			h.log.Debug("Handle message store do event.", event)
			switch event[0] {
			case "persistence": // 持久化事件
				if len(event) == 2 {
					h.persistence(event[1]) // event[1] 为 eventID
				}
			case "callback": // 回调事件，更新数据库中 event 的数据
				if len(event) == 4 {
					eventID := event[1]
					status := event[2]
					message := event[3]
					event, err := cdb.GetManager().ServiceEventDao().GetEventByEventID(eventID)
					if err != nil {
						logrus.Errorf("get event by event id %s failure %s", eventID, err.Error())
						continue
					}
					event.Status = status
					if strings.Contains(event.FinalStatus, model.EventFinalStatusEmpty.String()) {
						event.FinalStatus = model.EventFinalStatusEmptyComplete.String()
					} else {
						event.FinalStatus = model.EventFinalStatusComplete.String()
					}
					event.Message = message
					event.EndTime = time.Now().Format(time.RFC3339)
					logrus.Infof("updating event %s's status: %s", eventID, status)
					if err := cdb.GetManager().ServiceEventDao().UpdateModel(event); err != nil {
						logrus.Errorf("update event status failure %s", err.Error())
					}
				}
			}
		case <-h.ctx.Done():
			return
		}
	}
}
func (h *handleMessageStore) GetHistoryMessage(eventID string, length int) (re []string) {
	h.lock.RLock()
	defer h.lock.RUnlock()
	for _, m := range h.barrels[eventID].barrel {
		re = append(re, string(m.Content))
	}
	if len(re) > length && length != 0 {
		return re[:length-1]
	}
	return re
}
