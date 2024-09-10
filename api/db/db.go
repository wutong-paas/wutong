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

package db

import (
	"context"

	tsdbClient "github.com/bluebreezecf/opentsdb-goclient/client"
	tsdbConfig "github.com/bluebreezecf/opentsdb-goclient/config"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
	"github.com/wutong-paas/wutong/config/configs"
	"github.com/wutong-paas/wutong/db"
	"github.com/wutong-paas/wutong/db/config"
	"github.com/wutong-paas/wutong/worker/discover/model"
)

// ConDB struct
type ConDB struct {
	ConnectionInfo string
	DBType         string
}

func Database() *ConDB {
	return &ConDB{}
}

// Start -
func (condb *ConDB) Start(ctx context.Context, cfg *configs.Config) error {
	logrus.Info("start db client...")
	dbCfg := config.Config{
		MysqlConnectionInfo: cfg.APIConfig.DBConnectionInfo,
		DBType:              cfg.APIConfig.DBType,
		ShowSQL:             cfg.APIConfig.ShowSQL,
	}
	if err := db.CreateManager(dbCfg); err != nil {
		logrus.Errorf("get db manager failed,%s", err.Error())
		return err
	}
	return nil
}

// CloseHandle -
func (condb *ConDB) CloseHandle() {
	err := db.CloseManager()

	if err != nil {
		logrus.Errorf("close db manager failed,%s", err.Error())
	}
}

// TaskStruct task struct
type TaskStruct struct {
	TaskType string
	TaskBody model.TaskBody
	User     string
}

// OpentsdbManager OpentsdbManager
type OpentsdbManager struct {
	Endpoint string
}

// NewOpentsdbManager NewOpentsdbManager
func (o *OpentsdbManager) NewOpentsdbManager() (tsdbClient.Client, error) {
	opentsdbCfg := tsdbConfig.OpenTSDBConfig{
		OpentsdbHost: o.Endpoint,
	}
	tc, err := tsdbClient.NewClient(opentsdbCfg)
	if err != nil {
		return nil, err
	}
	return tc, nil
}

// GetBegin get db transaction
func GetBegin() *gorm.DB {
	return db.GetManager().Begin()
}
