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

package statistical

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/wutong-paas/wutong/db"
	"github.com/wutong-paas/wutong/db/model"
	"github.com/wutong-paas/wutong/util"
	"golang.org/x/net/context"
)

// DiskCache 磁盘异步统计
type DiskCache struct {
	cache []struct {
		Key   string
		Value float64
	}
	dbmanager db.Manager
	ctx       context.Context
	cancel    context.CancelFunc
}

// CreatDiskCache 创建
func CreatDiskCache(ctx context.Context) *DiskCache {
	cctx, cancel := context.WithCancel(ctx)
	return &DiskCache{
		dbmanager: db.GetManager(),
		ctx:       cctx,
		cancel:    cancel,
	}
}

// Start 开始启动统计
func (d *DiskCache) Start() {
	d.setcache()
	timer := time.NewTimer(time.Minute * 5)
	defer timer.Stop()
	for {
		select {
		case <-d.ctx.Done():
			return
		case <-timer.C:
			d.setcache()
			timer.Reset(time.Minute * 5)
		}
	}
}

// Stop stop
func (d *DiskCache) Stop() {
	logrus.Info("stop disk cache statistics")
	d.cancel()
}
func (d *DiskCache) setcache() {
	logrus.Info("start get all service disk size")
	start := time.Now()
	var diskcache []struct {
		Key   string
		Value float64
	}
	services, err := d.dbmanager.TenantEnvServiceDao().GetAllServicesID()
	if err != nil {
		logrus.Errorln("Error get tenant env service when select db :", err)
		return
	}
	_, err = d.dbmanager.TenantEnvServiceVolumeDao().GetAllVolumes()
	if err != nil {
		logrus.Errorln("Error get tenant env service volume when select db :", err)
		return
	}
	sharePath := os.Getenv("SHARE_DATA_PATH")
	if sharePath == "" {
		sharePath = "/wtdata"
	}
	var cache = make(map[string]*model.TenantEnvServices)
	for _, service := range services {
		//service nfs volume
		size := util.GetDirSize(fmt.Sprintf("%s/tenantEnv/%s/service/%s", sharePath, service.TenantEnvID, service.ServiceID))
		if size != 0 {
			diskcache = append(diskcache, struct {
				Key   string
				Value float64
			}{
				Key:   service.ServiceID + "_" + service.AppID + "_" + service.TenantEnvID,
				Value: size,
			})
		}
		cache[service.ServiceID] = service
	}
	d.cache = diskcache
	logrus.Infof("end get all service disk size,time consum %2.f s", time.Since(start).Seconds())
}

// Get 获取磁盘统计结果
func (d *DiskCache) Get() map[string]float64 {
	newcache := make(map[string]float64)
	for _, v := range d.cache {
		newcache[v.Key] += v.Value
	}
	return newcache
}

// GetTenantEnvDisk GetTenantEnvDisk
func (d *DiskCache) GetTenantEnvDisk(tenantEnvID string) float64 {
	var value float64
	for _, v := range d.cache {
		if strings.HasSuffix(v.Key, "_"+tenantEnvID) {
			value += v.Value
		}
	}
	return value
}

// GetServiceDisk GetServiceDisk
func (d *DiskCache) GetServiceDisk(serviceID string) float64 {
	var value float64
	for _, v := range d.cache {
		if strings.HasPrefix(v.Key, serviceID+"_") {
			value += v.Value
		}
	}
	return value
}
