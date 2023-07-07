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
	"testing"
	"time"

	"github.com/wutong-paas/wutong/eventlog/cluster/discover"
	"github.com/wutong-paas/wutong/eventlog/conf"
	"github.com/wutong-paas/wutong/eventlog/db"

	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
)

func TestGetSuitableInstance(t *testing.T) {
	dis := discover.New(nil, conf.DiscoverConf{}, logrus.WithField("Module", "Test"))
	ctx, cancel := context.WithCancel(context.Background())
	d := &Distribution{
		cancel:       cancel,
		context:      ctx,
		discover:     dis,
		updateTime:   make(map[string]time.Time),
		abnormalNode: make(map[string]int),
		log:          logrus.WithField("Module", "Test"),
	}
	d.monitorDatas = map[string]*db.MonitorData{
		"a": {
			InstanceID: "a", LogSizePeerM: 200,
		},
		"b": {
			InstanceID: "b", LogSizePeerM: 150,
		},
	}
	d.GetSuitableInstance("todo service id")
}
