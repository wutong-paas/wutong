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
	"testing"

	"github.com/wutong-paas/wutong/discover/config"
	etcdutil "github.com/wutong-paas/wutong/util/etcd"

	"github.com/sirupsen/logrus"
)

func TestAddUpdateProject(t *testing.T) {
	etcdClientArgs := &etcdutil.ClientArgs{Endpoints: []string{"127.0.0.1:2379"}}
	discover, err := GetDiscover(config.DiscoverConfig{
		EtcdClientArgs: etcdClientArgs,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer discover.Stop()
	discover.AddUpdateProject("test", callbackupdate{})
}
func TestAddProject(t *testing.T) {
	etcdClientArgs := &etcdutil.ClientArgs{Endpoints: []string{"127.0.0.1:2379"}}
	discover, err := GetDiscover(config.DiscoverConfig{
		EtcdClientArgs: etcdClientArgs,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer discover.Stop()
	discover.AddProject("test", callback{})
}

type callbackupdate struct {
	callback
}

func (c callbackupdate) UpdateEndpoints(operation config.Operation, endpoints ...*config.Endpoint) {
	logrus.Info(operation, "////", endpoints)
}

type callback struct {
}

func (c callback) UpdateEndpoints(endpoints ...*config.Endpoint) {
	for _, en := range endpoints {
		logrus.Infof("%+v", en)
	}
}

//when watch occurred error,will exec this method
func (c callback) Error(err error) {
	logrus.Error(err.Error())
}
