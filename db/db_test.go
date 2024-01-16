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
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"

	dbconfig "github.com/wutong-paas/wutong/db/config"
	"github.com/wutong-paas/wutong/db/model"
	"github.com/wutong-paas/wutong/util"
)

func CreateTestManager() (Manager, error) {
	dbname := "region"
	rootpw := "wutong"

	ctx := context.Background()
	req := testcontainers.ContainerRequest{
		Image:        "mariadb",
		ExposedPorts: []string{"3306/tcp"},
		Env: map[string]string{
			"MYSQL_ROOT_PASSWORD": rootpw,
			"MYSQL_DATABASE":      dbname,
		},
		Cmd: []string{"character-set-server=utf8mb4", "collation-server=utf8mb4_unicode_ci"},
	}
	mariadb, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, err
	}
	defer mariadb.Terminate(ctx)

	host, err := mariadb.Host(ctx)
	if err != nil {
		return nil, err
	}
	port, err := mariadb.MappedPort(ctx, "3306")
	if err != nil {
		return nil, err
	}

	connInfo := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", "root",
		rootpw, host, port.Int(), dbname)
	tryTimes := 3
	for {
		if err := CreateManager(dbconfig.Config{
			DBType:              "mysql",
			MysqlConnectionInfo: connInfo,
		}); err != nil {
			if tryTimes == 0 {
				return nil, fmt.Errorf("Connect info: %s; error creating db manager: %v", connInfo, err)
			}
			tryTimes = tryTimes - 1
			time.Sleep(10 * time.Second)
			continue
		}
		break
	}
	return GetManager(), nil
}

func TestTenantEnvDao(t *testing.T) {
	if err := CreateManager(dbconfig.Config{
		MysqlConnectionInfo: "root:admin@tcp(127.0.0.1:3306)/region",
		DBType:              "mysql",
	}); err != nil {
		t.Fatal(err)
	}
	err := GetManager().TenantEnvDao().AddModel(&model.TenantEnvs{
		Name: "barnett4",
		UUID: util.NewUUID(),
	})
	if err != nil {
		t.Fatal(err)
	}
	tenantEnv, err := GetManager().TenantEnvDao().GetTenantEnvByUUID("27bbdd119b24444696dc51aa2f41eef8")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(tenantEnv)
}

func TestTenantEnvServiceDao(t *testing.T) {
	if err := CreateManager(dbconfig.Config{
		MysqlConnectionInfo: "root:admin@tcp(127.0.0.1:3306)/region",
		DBType:              "mysql",
	}); err != nil {
		t.Fatal(err)
	}
	service, err := GetManager().TenantEnvServiceDao().GetServiceByTenantEnvIDAndServiceAlias("27bbdd119b24444696dc51aa2f41eef8", "wtb58f90")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(service)
	service, err = GetManager().TenantEnvServiceDao().GetServiceByID("2f29882148c19f5f84e3a7cedf6097c7")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(service)
	services, err := GetManager().TenantEnvServiceDao().GetServiceAliasByIDs([]string{"2f29882148c19f5f84e3a7cedf6097c7"})
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range services {
		t.Log(s)
	}
}

func TestGetServiceEnvs(t *testing.T) {
	if err := CreateManager(dbconfig.Config{
		MysqlConnectionInfo: "root:admin@tcp(127.0.0.1:3306)/region",
		DBType:              "mysql",
	}); err != nil {
		t.Fatal(err)
	}
	envs, err := GetManager().TenantEnvServiceEnvVarDao().GetServiceEnvs("2f29882148c19f5f84e3a7cedf6097c7", nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range envs {
		t.Log(e)
	}
}

func TestSetServiceLabel(t *testing.T) {
	if err := CreateManager(dbconfig.Config{
		MysqlConnectionInfo: "root:admin@tcp(127.0.0.1:3306)/region",
		DBType:              "mysql",
	}); err != nil {
		t.Fatal(err)
	}
	label := model.TenantEnvServiceLabel{
		LabelKey:   "labelkey",
		LabelValue: "labelvalue",
		ServiceID:  "889bb1f028f655bebd545f24aa184a0b",
	}
	label.CreatedAt = time.Now()
	label.ID = 1
	err := GetManager().TenantEnvServiceLabelDao().UpdateModel(&label)
	if err != nil {
		t.Fatal(err)
	}
}

func TestCreateTenantEnvServiceLBMappingPort(t *testing.T) {
	if err := CreateManager(dbconfig.Config{
		MysqlConnectionInfo: "root:admin@tcp(127.0.0.1:3306)/region",
		DBType:              "mysql",
	}); err != nil {
		t.Fatal(err)
	}
	mapPort, err := GetManager().TenantEnvServiceLBMappingPortDao().CreateTenantEnvServiceLBMappingPort("889bb1f028f655bebd545f24aa184a0b", 8080)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(mapPort)
}

func TestCreateTenantEnvServiceLBMappingPortTran(t *testing.T) {
	if err := CreateManager(dbconfig.Config{
		MysqlConnectionInfo: "root:admin@tcp(127.0.0.1:3306)/region",
		DBType:              "mysql",
	}); err != nil {
		t.Fatal(err)
	}
	tx := GetManager().Begin()
	mapPort, err := GetManager().TenantEnvServiceLBMappingPortDaoTransactions(tx).CreateTenantEnvServiceLBMappingPort("889bb1f028f655bebd545f24aa184a0b", 8082)
	if err != nil {
		tx.Rollback()
		t.Fatal(err)
		return
	}
	tx.Commit()
	t.Log(mapPort)
}

func TestGetMem(t *testing.T) {
	if err := CreateManager(dbconfig.Config{
		MysqlConnectionInfo: "admin:admin@tcp(127.0.0.1:3306)/region",
		DBType:              "mysql",
	}); err != nil {
		t.Fatal(err)
	}
	// err := GetManager().TenantEnvDao().AddModel(&model.TenantEnvs{
	// 	Name: "barnett3",
	// 	UUID: util.NewUUID(),
	// })
	// if err != nil {
	// 	t.Fatal(err)
	// }
}

func TestCockroachDBCreateTable(t *testing.T) {
	if err := CreateManager(dbconfig.Config{
		MysqlConnectionInfo: "postgresql://root@localhost:5432/region?sslmode=disable",
		DBType:              "cockroachdb",
	}); err != nil {
		t.Fatal(err)
	}
}
func TestCockroachDBCreateService(t *testing.T) {
	if err := CreateManager(dbconfig.Config{
		MysqlConnectionInfo: "postgresql://root@localhost:5432/region?sslmode=disable",
		DBType:              "cockroachdb",
	}); err != nil {
		t.Fatal(err)
	}
	err := GetManager().TenantEnvServiceDao().AddModel(&model.TenantEnvServices{
		TenantEnvID:  "asdasd",
		ServiceID:    "asdasdasdasd",
		ServiceAlias: "wtasdasdasdads",
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestCockroachDBDeleteService(t *testing.T) {
	if err := CreateManager(dbconfig.Config{
		MysqlConnectionInfo: "postgresql://root@localhost:5432/region?sslmode=disable",
		DBType:              "cockroachdb",
	}); err != nil {
		t.Fatal(err)
	}
	err := GetManager().TenantEnvServiceDao().DeleteServiceByServiceID("asdasdasdasd")
	if err != nil {
		t.Fatal(err)
	}
}

//func TestCockroachDBSaveDeployInfo(t *testing.T) {
//	if err := CreateManager(dbconfig.Config{
//		MysqlConnectionInfo: "postgresql://root@localhost:5432/region?sslmode=disable",
//		DBType:              "cockroachdb",
//	}); err != nil {
//		t.Fatal(err)
//	}
//	err := GetManager().K8sDeployReplicationDao().AddModel(&model.K8sDeployReplication{
//		TenantEnvID:        "asdasd",
//		ServiceID:       "asdasdasdasd",
//		ReplicationID:   "asdasdadsasdasdasd",
//		ReplicationType: model.TypeReplicationController,
//	})
//	if err != nil {
//		t.Fatal(err)
//	}
//}

//func TestCockroachDBDeleteDeployInfo(t *testing.T) {
//	if err := CreateManager(dbconfig.Config{
//		MysqlConnectionInfo: "postgresql://root@localhost:5432/region?sslmode=disable",
//		DBType:              "cockroachdb",
//	}); err != nil {
//		t.Fatal(err)
//	}
//err := GetManager().K8sDeployReplicationDao().DeleteK8sDeployReplication("asdasdadsasdasdasd")
//if err != nil {
//	t.Fatal(err)
//}
//}

func TestGetCertificateByID(t *testing.T) {
	if err := CreateManager(dbconfig.Config{
		MysqlConnectionInfo: "admin:admin@tcp(127.0.0.1:3306)/region",
		DBType:              "mysql",
	}); err != nil {
		t.Fatal(err)
	}

	cert, err := GetManager().TenantEnvServiceLabelDao().GetTenantEnvNodeAffinityLabel("105bb7d4b94774f922edb3051bdf8ce1")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(cert)
}

var shareFileVolumeType = &model.TenantEnvServiceVolumeType{
	VolumeType:  "share-file",
	NameShow:    "共享存储（文件）",
	Description: "分布式文件存储，可租户内共享挂载，适用于所有类型应用",
	// CapacityValidation:   string(bs),
	ReclaimPolicy: "Retain",
	BackupPolicy:  "exclusive",
	SharePolicy:   "exclusive",
	AccessMode:    "RWO,ROX,RWX",
}

var localVolumeType = &model.TenantEnvServiceVolumeType{
	VolumeType:  "local",
	NameShow:    "本地存储",
	Description: "本地存储设备，适用于有状态数据库服务",
	// CapacityValidation:   string(bs),
	ReclaimPolicy: "Retain",
	BackupPolicy:  "exclusive",
	SharePolicy:   "exclusive",
	AccessMode:    "RWO,ROX,RWX",
}

var memoryFSVolumeType = &model.TenantEnvServiceVolumeType{
	VolumeType:  "memoryfs",
	NameShow:    "内存文件存储",
	Description: "基于内存的存储设备，容量由内存量限制。应用重启数据即丢失，适用于高速暂存数据",
	// CapacityValidation:   string(bs),
	ReclaimPolicy: "Retain",
	BackupPolicy:  "exclusive",
	SharePolicy:   "exclusive",
	AccessMode:    "RWO,ROX,RWX",
}

var alicloudDiskAvailableVolumeType = &model.TenantEnvServiceVolumeType{
	VolumeType:  "alicloud-disk-available",
	NameShow:    "阿里云盘（智能选择）",
	Description: "阿里云智能选择云盘。会通过高效云盘、SSD、基础云盘的顺序依次尝试创建当前阿里云区域支持的云盘类型",
	// CapacityValidation:   string(bs),
	ReclaimPolicy: "Delete",
	BackupPolicy:  "exclusive",
	SharePolicy:   "exclusive",
	AccessMode:    "RWO",
	Sort:          10,
}

var alicloudDiskcommonVolumeType = &model.TenantEnvServiceVolumeType{
	VolumeType:  "alicloud-disk-common",
	NameShow:    "阿里云盘（基础）",
	Description: "阿里云普通基础云盘。最小限额5G",
	// CapacityValidation:   string(bs),
	ReclaimPolicy: "Delete",
	BackupPolicy:  "exclusive",
	SharePolicy:   "exclusive",
	AccessMode:    "RWO",
	Sort:          13,
}
var alicloudDiskEfficiencyVolumeType = &model.TenantEnvServiceVolumeType{
	VolumeType:  "alicloud-disk-efficiency",
	NameShow:    "阿里云盘（高效）",
	Description: "阿里云高效云盘。最小限额20G",
	// CapacityValidation:   string(bs),
	ReclaimPolicy: "Delete",
	BackupPolicy:  "exclusive",
	SharePolicy:   "exclusive",
	AccessMode:    "RWO",
	Sort:          11,
}
var alicloudDiskeSSDVolumeType = &model.TenantEnvServiceVolumeType{
	VolumeType:  "alicloud-disk-ssd",
	NameShow:    "阿里云盘（SSD）",
	Description: "阿里云SSD类型云盘。最小限额20G",
	// CapacityValidation:   string(bs),
	ReclaimPolicy: "Delete",
	BackupPolicy:  "exclusive",
	SharePolicy:   "exclusive",
	AccessMode:    "RWO",
	Sort:          12,
}

func TestVolumeType(t *testing.T) {

	capacityValidation := make(map[string]interface{})
	capacityValidation["required"] = true
	capacityValidation["default"] = 20
	capacityValidation["min"] = 20
	capacityValidation["max"] = 32768 // [ali-cloud-disk usage limit](https://help.aliyun.com/document_detail/25412.html?spm=5176.2020520101.0.0.41d84df5faliP4)
	bs, _ := json.Marshal(capacityValidation)
	shareFileVolumeType.CapacityValidation = string(bs)
	localVolumeType.CapacityValidation = string(bs)
	memoryFSVolumeType.CapacityValidation = string(bs)
	alicloudDiskeSSDVolumeType.CapacityValidation = string(bs)
	if err := CreateManager(dbconfig.Config{
		MysqlConnectionInfo: "ieZoo9:Maigoed0@tcp(192.168.2.108:3306)/region",
		DBType:              "mysql",
	}); err != nil {
		t.Fatal(err)
	}
	if err := GetManager().VolumeTypeDao().AddModel(alicloudDiskeSSDVolumeType); err != nil {
		t.Error(err)
	} else {
		t.Log("yes")
	}
}

func initDBManager(t *testing.T) {
	if err := CreateManager(dbconfig.Config{
		MysqlConnectionInfo: "ieZoo9:Maigoed0@tcp(192.168.2.108:3306)/region",
		DBType:              "mysql",
	}); err != nil {
		t.Fatal(err)
	}
}
func TestGetVolumeType(t *testing.T) {
	initDBManager(t)
	vts, err := GetManager().VolumeTypeDao().GetAllVolumeTypes()
	if err != nil {
		t.Fatal(err)
	}
	for _, vt := range vts {
		t.Logf("%+v", vt)
	}
	t.Logf("volume type len is : %v", len(vts))
}

func TestGetVolumeTypeByType(t *testing.T) {
	initDBManager(t)
	vt, err := GetManager().VolumeTypeDao().GetVolumeTypeByType("ceph-wt")
	if err != nil {
		t.Fatal("get volumeType by type error: ", err.Error())
	}
	t.Logf("%+v", vt)
}
