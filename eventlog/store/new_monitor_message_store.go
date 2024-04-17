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
	"context"
	"encoding/json"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"

	"github.com/wutong-paas/wutong/eventlog/conf"
	"github.com/wutong-paas/wutong/eventlog/db"
)

type newMonitorMessageStore struct {
	conf        conf.EventStoreConf
	log         *logrus.Entry
	barrels     map[string]*CacheMonitorMessageList
	lock        sync.RWMutex
	cancel      func()
	ctx         context.Context
	size        int64
	allLogCount float64
}

func (m *newMonitorMessageStore) Scrape(ch chan<- prometheus.Metric, namespace, exporter, from string) error {
	chanDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, exporter, "new_monitor_store_barrel_count"),
		"the handle container log count size.",
		[]string{"from"}, nil,
	)
	ch <- prometheus.MustNewConstMetric(chanDesc, prometheus.GaugeValue, float64(len(m.barrels)), from)
	logDesc := prometheus.NewDesc(
		prometheus.BuildFQName(namespace, exporter, "new_monitor_store_log_count"),
		"the handle monitor log count size.",
		[]string{"from"}, nil,
	)
	ch <- prometheus.MustNewConstMetric(logDesc, prometheus.GaugeValue, m.allLogCount, from)

	return nil
}
func (m *newMonitorMessageStore) insertMessage(message *db.EventLogMessage) ([]MonitorMessage, bool) {
	m.lock.RLock()
	defer m.lock.RUnlock()
	mm := fromByte(message.MonitorData)
	if len(mm) < 1 {
		return mm, true
	}
	if mm[0].ServiceID == "" {
		return mm, true
	}
	if ba, ok := m.barrels[mm[0].ServiceID]; ok {
		ba.Insert(mm...)
		return mm, true
	}
	return mm, false
}

func (m *newMonitorMessageStore) InsertMessage(message *db.EventLogMessage) {
	if message == nil {
		return
	}
	m.size++
	m.allLogCount++
	mm, ok := m.insertMessage(message)
	if ok {
		return
	}
	m.lock.Lock()
	defer m.lock.Unlock()
	ba := CreateCacheMonitorMessageList(mm[0].ServiceID)
	ba.Insert(mm...)
	m.barrels[mm[0].ServiceID] = ba
}
func (m *newMonitorMessageStore) GetMonitorData() *db.MonitorData {
	data := &db.MonitorData{
		ServiceSize:  len(m.barrels),
		LogSizePeerM: m.size,
	}
	return data
}

func (m *newMonitorMessageStore) SubChan(eventID, subID string) chan *db.EventLogMessage {
	m.lock.Lock()
	defer m.lock.Unlock()
	if ba, ok := m.barrels[eventID]; ok {
		return ba.addSubChan(subID)
	}
	ba := CreateCacheMonitorMessageList(eventID)
	m.barrels[eventID] = ba
	return ba.addSubChan(subID)
}
func (m *newMonitorMessageStore) ReleaseSubChan(eventID, subID string) {
	m.lock.RLock()
	defer m.lock.RUnlock()
	if ba, ok := m.barrels[eventID]; ok {
		ba.delSubChan(subID)
	}
}
func (m *newMonitorMessageStore) Run() {
	go m.Gc()
}
func (m *newMonitorMessageStore) Gc() {
	tiker := time.NewTicker(time.Second * 30)
	defer tiker.Stop()
	for {
		select {
		case <-tiker.C:
		case <-m.ctx.Done():
			m.log.Debug("read message store gc stop.")
			return
		}
		m.size = 0
		if len(m.barrels) == 0 {
			continue
		}
		var gcEvent []string
		for k, v := range m.barrels {
			if len(v.subSocketChan) == 0 {
				if v.UpdateTime.Add(time.Minute * 3).Before(time.Now()) { // barrel 超时未收到消息
					gcEvent = append(gcEvent, k)
				}
			}
		}
		if len(gcEvent) > 0 {
			for _, id := range gcEvent {
				m.log.Infof("monitor message barrel %s will be gc", id)
				barrel := m.barrels[id]
				barrel.empty()
				delete(m.barrels, id)
			}
		}
	}
}
func (m *newMonitorMessageStore) stop() {
	m.cancel()
}
func (m *newMonitorMessageStore) InsertGarbageMessage(message ...*db.EventLogMessage) {}
func (m *newMonitorMessageStore) GetHistoryMessage(eventID string, length int) (re []string) {
	return nil
}

// MonitorMessage 性能监控消息系统模型
type MonitorMessage struct {
	ServiceID   string
	Port        string
	HostName    string
	MessageType string //mysql，http ...
	Key         string
	//总时间
	CumulativeTime float64
	AverageTime    float64
	MaxTime        float64
	Count          uint64
	//异常请求次数
	AbnormalCount uint64
}

// cacheMonitorMessage 每个实例的数据缓存
type cacheMonitorMessage struct {
	updateTime time.Time
	hostName   string
	mms        MonitorMessageList
}

// CacheMonitorMessageList 某个应用性能分析数据
type CacheMonitorMessageList struct {
	list          []*cacheMonitorMessage
	subSocketChan map[string]chan *db.EventLogMessage
	subLock       sync.Mutex
	message       db.EventLogMessage
	UpdateTime    time.Time
}

// CreateCacheMonitorMessageList 创建应用监控信息缓存器
func CreateCacheMonitorMessageList(eventID string) *CacheMonitorMessageList {
	return &CacheMonitorMessageList{
		subSocketChan: make(map[string]chan *db.EventLogMessage),
		message: db.EventLogMessage{
			EventID: eventID,
		},
	}
}

// Insert 认为mms的hostname一致
// 每次收到消息进行gc
func (c *CacheMonitorMessageList) Insert(mms ...MonitorMessage) {
	if mms == nil || len(mms) < 1 {
		return
	}
	c.UpdateTime = time.Now()
	hostname := mms[0].HostName
	if len(c.list) == 0 {
		c.list = []*cacheMonitorMessage{
			{
				updateTime: time.Now(),
				hostName:   hostname,
				mms:        mms,
			}}
	}
	var update bool
	for i := range c.list {
		cm := c.list[i]
		if cm.hostName == hostname {
			cm.updateTime = time.Now()
			cm.mms = mms
			update = true
			break
		}
	}
	if !update {
		c.list = append(c.list, &cacheMonitorMessage{
			updateTime: time.Now(),
			hostName:   hostname,
			mms:        mms,
		})
	}
	c.Gc()
	c.pushMessage()
}

// Gc 清理数据
func (c *CacheMonitorMessageList) Gc() {
	var list []*cacheMonitorMessage
	for i := range c.list {
		cmm := c.list[i]
		if !cmm.updateTime.Add(time.Second * 30).Before(time.Now()) {
			list = append(list, cmm)
		}
	}
	c.list = list
}

func (c *CacheMonitorMessageList) pushMessage() {
	if len(c.list) == 0 {
		return
	}
	source := c.list[0].mms
	for i := 1; i < len(c.list); i++ {
		addSource := c.list[i].mms
		source = merge(source, addSource)
	}
	//降序排序
	sort.Sort(sort.Reverse(&source))
	mdata := getByte(*source.Pop(20))
	c.message.MonitorData = mdata
	for _, ch := range c.subSocketChan {
		select {
		case ch <- &c.message:
		default:
		}
	}
}

// 增加socket订阅
func (c *CacheMonitorMessageList) addSubChan(subID string) chan *db.EventLogMessage {
	c.subLock.Lock()
	defer c.subLock.Unlock()
	if sub, ok := c.subSocketChan[subID]; ok {
		return sub
	}
	ch := make(chan *db.EventLogMessage, 10)
	c.subSocketChan[subID] = ch
	c.pushMessage()
	return ch
}

// delSubChan delete socket sub chan
func (c *CacheMonitorMessageList) delSubChan(subID string) {
	c.subLock.Lock()
	defer c.subLock.Unlock()
	if ch, ok := c.subSocketChan[subID]; ok {
		close(ch)
		delete(c.subSocketChan, subID)
	}
}
func (c *CacheMonitorMessageList) empty() {
	c.subLock.Lock()
	defer c.subLock.Unlock()
	for _, v := range c.subSocketChan {
		close(v)
	}
}
func getByte(source []MonitorMessage) []byte {
	b, _ := json.Marshal(source)
	return b
}
func fromByte(source []byte) []MonitorMessage {
	var mm []MonitorMessage
	json.Unmarshal(source, &mm)
	return mm
}

func merge(source, addsource MonitorMessageList) (result MonitorMessageList) {
	var cache = make(map[string]MonitorMessage)
	for _, mm := range source {
		cache[mm.Key] = mm
	}
	for _, mm := range addsource {
		if oldmm, ok := cache[mm.Key]; ok {
			oldmm.Count += mm.Count
			oldmm.AbnormalCount += mm.AbnormalCount
			oldmm.AverageTime = Round((oldmm.AverageTime+mm.AverageTime)/2, 2)
			oldmm.CumulativeTime = Round(oldmm.CumulativeTime+mm.CumulativeTime, 2)
			if mm.MaxTime > oldmm.MaxTime {
				oldmm.MaxTime = mm.MaxTime
			}
			cache[mm.Key] = oldmm
			continue
		}
		cache[mm.Key] = mm
	}
	for _, c := range cache {
		result.Add(&c)
	}
	return
}

// Round Round
func Round(f float64, n int) float64 {
	pow10n := math.Pow10(n)
	return math.Trunc((f+0.5/pow10n)*pow10n) / pow10n
}

// MonitorMessageList 消息列表
type MonitorMessageList []MonitorMessage

// Add 添加
func (m *MonitorMessageList) Add(mm *MonitorMessage) {
	*m = append(*m, *mm)
}

// Len 为集合内元素的总数
func (m *MonitorMessageList) Len() int {
	return len(*m)
}

// Less 如果index为i的元素小于index为j的元素，则返回true，否则返回false
func (m *MonitorMessageList) Less(i, j int) bool {
	return (*m)[i].CumulativeTime < (*m)[j].CumulativeTime
}

// Swap 交换索引为 i 和 j 的元素
func (m *MonitorMessageList) Swap(i, j int) {
	tmp := (*m)[i]
	(*m)[i] = (*m)[j]
	(*m)[j] = tmp
}

// Pop Pop
func (m *MonitorMessageList) Pop(i int) *MonitorMessageList {
	if len(*m) <= i {
		return m
	}
	cache := (*m)[:i]
	return &cache
}

// String json string
func (m *MonitorMessageList) String() string {
	body, _ := json.Marshal(m)
	return string(body)
}
