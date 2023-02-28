// WUTONG, Application Management Platform
// Copyright (C) 2014-2017 Wutong Co., Ltd.

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

package exector

import (
	"testing"

	"github.com/pquerna/ffjson/ffjson"
	"github.com/sirupsen/logrus"

	dbmodel "github.com/wutong-paas/wutong/db/model"
	"github.com/wutong-paas/wutong/event"
	"github.com/wutong-paas/wutong/util"
)

func TestModify(t *testing.T) {
	var b = BackupAPPRestore{
		BackupID:      "test",
		EventID:       "test",
		serviceChange: make(map[string]*Info, 0),
		Logger:        event.GetTestLogger(),
	}
	var appSnapshots = []*RegionServiceSnapshot{
		&RegionServiceSnapshot{
			ServiceID: "1234",
			Service: &dbmodel.TenantEnvServices{
				ServiceID: "1234",
			},
			ServiceMntRelation: []*dbmodel.TenantEnvServiceMountRelation{
				&dbmodel.TenantEnvServiceMountRelation{
					ServiceID:       "1234",
					DependServiceID: "123456",
				},
			},
		},
		&RegionServiceSnapshot{
			ServiceID: "123456",
			Service: &dbmodel.TenantEnvServices{
				ServiceID: "1234",
			},
			ServiceEnv: []*dbmodel.TenantEnvServiceEnvVar{
				&dbmodel.TenantEnvServiceEnvVar{
					ServiceID: "123456",
					Name:      "testenv",
				},
				&dbmodel.TenantEnvServiceEnvVar{
					ServiceID: "123456",
					Name:      "testenv2",
				},
			},
		},
	}
	appSnapshot := AppSnapshot{
		Services: appSnapshots,
	}
	b.modify(&appSnapshot)
	re, _ := ffjson.Marshal(appSnapshot)
	t.Log(string(re))
}

func TestUnzipAllDataFile(t *testing.T) {
	allDataFilePath := "/tmp/__all_data.zip"
	allTmpDir := "/tmp/4f25c53e864744ec95d037528acaa708"
	if err := util.Unzip(allDataFilePath, allTmpDir); err != nil {
		logrus.Errorf("unzip all data file failure %s", err.Error())
	}
}
