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
	"time"

	tsdbClient "github.com/bluebreezecf/opentsdb-goclient/client"
	tsdbConfig "github.com/bluebreezecf/opentsdb-goclient/config"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
	"github.com/wutong-paas/wutong/config/configs"
	"github.com/wutong-paas/wutong/db"
	"github.com/wutong-paas/wutong/db/config"
	dbModel "github.com/wutong-paas/wutong/db/model"
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

// CreateDBManager get db manager
// TODO: need to try when happened error, try 4 times
// func CreateDBManager(conf option.Config) error {
// 	dbCfg := config.Config{
// 		MysqlConnectionInfo: conf.DBConnectionInfo,
// 		DBType:              conf.DBType,
// 		ShowSQL:             conf.ShowSQL,
// 	}
// 	if err := db.CreateManager(dbCfg); err != nil {
// 		logrus.Errorf("get db manager failed,%s", err.Error())
// 		return err
// 	}
// 	// api database initialization
// 	go dataInitialization()

// 	return nil
// }

// CreateEventManager create event manager
// func CreateEventManager(conf option.Config) error {
// 	var tryTime time.Duration
// 	var err error
// 	etcdClientArgs := &etcdutil.ClientArgs{
// 		Endpoints: conf.EtcdEndpoint,
// 		CaFile:    conf.EtcdCaFile,
// 		CertFile:  conf.EtcdCertFile,
// 		KeyFile:   conf.EtcdKeyFile,
// 	}
// 	for tryTime < 4 {
// 		tryTime++
// 		if err = event.NewManager(event.EventConfig{
// 			EventLogServers: conf.EventLogServers,
// 			DiscoverArgs:    etcdClientArgs,
// 		}); err != nil {
// 			logrus.Errorf("get event manager failed, try time is %v,%s", tryTime, err.Error())
// 			time.Sleep((5 + tryTime*10) * time.Second)
// 		} else {
// 			break
// 		}
// 	}
// 	if err != nil {
// 		logrus.Errorf("get event manager failed. %v", err.Error())
// 		return err
// 	}
// 	logrus.Debugf("init event manager success")
// 	return nil
// }

// MQManager mq manager
// type MQManager struct {
// 	EtcdClientArgs *etcdutil.ClientArgs
// 	DefaultServer  string
// }

// NewMQManager new mq manager
// func (m *MQManager) NewMQManager() (client.MQClient, error) {
// 	client, err := client.NewMqClient(m.EtcdClientArgs, m.DefaultServer)
// 	if err != nil {
// 		logrus.Errorf("new mq manager error, %v", err)
// 		return client, err
// 	}
// 	return client, nil
// }

// Start -
func (_ *ConDB) Start(ctx context.Context, cfg *configs.Config) error {
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
func (_ *ConDB) CloseHandle() {
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

// BuildTask build task
// func BuildTask(t *TaskStruct) (*pb.EnqueueRequest, error) {
// 	var er pb.EnqueueRequest
// 	taskJSON, err := json.Marshal(t.TaskBody)
// 	if err != nil {
// 		logrus.Errorf("tran task json error")
// 		return &er, err
// 	}
// 	er.Topic = "worker"
// 	er.Message = &pb.TaskMessage{
// 		TaskType:   t.TaskType,
// 		CreateTime: time.Now().Format(time.RFC3339),
// 		TaskBody:   taskJSON,
// 		User:       t.User,
// 	}
// 	return &er, nil
// }

// GetBegin get db transaction
func GetBegin() *gorm.DB {
	return db.GetManager().Begin()
}

func dbInit() error {
	logrus.Info("api database initialization starting...")
	begin := GetBegin()
	// Permissions set
	var rac dbModel.RegionAPIClass
	if err := begin.Where("class_level=? and prefix=?", "server_source", "/v2/show").Find(&rac).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			data := map[string]string{
				"/v2/show":                       "server_source",
				"/v2/cluster":                    "server_source",
				"/v2/resources":                  "server_source",
				"/v2/builder":                    "server_source",
				"/v2/tenants/{tenant_name}/envs": "server_source",
				"/v2/app":                        "server_source",
				"/v2/port":                       "server_source",
				"/v2/volume-options":             "server_source",
				"/api/v1":                        "server_source",
				"/v2/events":                     "server_source",
				"/v2/gateway/ips":                "server_source",
				"/v2/gateway/ports":              "server_source",
				"/v2/nodes":                      "node_manager",
				"/v2/job":                        "node_manager",
				"/v2/configs":                    "node_manager",
			}
			tx := begin
			var rollback bool
			for k, v := range data {
				if err := db.GetManager().RegionAPIClassDaoTransactions(tx).AddModel(&dbModel.RegionAPIClass{
					ClassLevel: v,
					Prefix:     k,
				}); err != nil {
					tx.Rollback()
					rollback = true
					break
				}
			}
			if !rollback {
				tx.Commit()
			}
		} else {
			return err
		}
	}

	return nil
}

func dataInitialization() {
	timer := time.NewTimer(time.Second * 2)
	defer timer.Stop()
	for {
		err := dbInit()
		if err != nil {
			logrus.Error("Initializing database failed, ", err)
		} else {
			logrus.Info("api database initialization success!")
			return
		}
		<-timer.C
		timer.Reset(time.Second * 2)
	}
}
