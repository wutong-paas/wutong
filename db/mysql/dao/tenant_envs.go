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

package dao

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"time"

	"github.com/jinzhu/gorm"
	pkgerr "github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	gormbulkups "github.com/wutong-paas/gorm-bulk-upsert"
	"github.com/wutong-paas/wutong/api/util/bcode"
	"github.com/wutong-paas/wutong/db/dao"
	"github.com/wutong-paas/wutong/db/errors"
	"github.com/wutong-paas/wutong/db/model"
)

// TenantEnvDaoImpl 租户环境信息管理
type TenantEnvDaoImpl struct {
	DB *gorm.DB
}

// AddModel 添加租户
func (t *TenantEnvDaoImpl) AddModel(mo model.Interface) error {
	tenantEnv := mo.(*model.TenantEnvs)
	var oldTenantEnv model.TenantEnvs
	if ok := t.DB.Where("uuid = ? or name=?", tenantEnv.UUID, tenantEnv.Name).Find(&oldTenantEnv).RecordNotFound(); ok {
		if err := t.DB.Create(tenantEnv).Error; err != nil {
			return err
		}
	} else {
		return fmt.Errorf("tenant env uuid  %s or name %s is exist", tenantEnv.UUID, tenantEnv.Name)
	}
	return nil
}

// UpdateModel 更新租户
func (t *TenantEnvDaoImpl) UpdateModel(mo model.Interface) error {
	tenantEnv := mo.(*model.TenantEnvs)
	if err := t.DB.Save(tenantEnv).Error; err != nil {
		return err
	}
	return nil
}

// GetTenantEnvByUUID 获取租户
func (t *TenantEnvDaoImpl) GetTenantEnvByUUID(uuid string) (*model.TenantEnvs, error) {
	var tenantEnv model.TenantEnvs
	if err := t.DB.Where("uuid = ?", uuid).Find(&tenantEnv).Error; err != nil {
		return nil, err
	}
	return &tenantEnv, nil
}

// GetTenantEnvByUUIDIsExist 获取租户
func (t *TenantEnvDaoImpl) GetTenantEnvByUUIDIsExist(uuid string) bool {
	var tenantEnv model.TenantEnvs
	isExist := t.DB.Where("uuid = ?", uuid).First(&tenantEnv).RecordNotFound()
	return isExist

}

// GetTenantEnvIDByName 获取租户
func (t *TenantEnvDaoImpl) GetTenantEnvIDByName(tenantName, tenantEnvName string) (*model.TenantEnvs, error) {
	var tenantEnv model.TenantEnvs
	if err := t.DB.Where("tenant_name = ? and name = ?", tenantName, tenantEnvName).Find(&tenantEnv).Error; err != nil {
		return nil, err
	}
	return &tenantEnv, nil
}

// GetAllTenantEnvs GetAllTenantEnvs
func (t *TenantEnvDaoImpl) GetAllTenantEnvs(query string) ([]*model.TenantEnvs, error) {
	var tenantEnvs []*model.TenantEnvs
	if query != "" {
		if err := t.DB.Where("name like ?", "%"+query+"%").Find(&tenantEnvs).Error; err != nil {
			return nil, err
		}
	} else {
		if err := t.DB.Find(&tenantEnvs).Error; err != nil {
			return nil, err
		}
	}
	return tenantEnvs, nil
}

// GetTenantEnvs GetTenantEnvs
func (t *TenantEnvDaoImpl) GetTenantEnvs(tenantName string, query string) ([]*model.TenantEnvs, error) {
	var tenantEnvs []*model.TenantEnvs
	if query != "" {
		if err := t.DB.Where("tenantName = ? and name like ?", tenantName, "%"+query+"%").Find(&tenantEnvs).Error; err != nil {
			return nil, err
		}
	} else {
		if err := t.DB.Find(&tenantEnvs).Error; err != nil {
			return nil, err
		}
	}
	return tenantEnvs, nil
}

// GetTenantEnvByEid get tenantEnvs by eid
func (t *TenantEnvDaoImpl) GetTenantEnvByEid(eid, query string) ([]*model.TenantEnvs, error) {
	var tenantEnvs []*model.TenantEnvs
	if query != "" {
		if err := t.DB.Where("eid = ? and name like '%?%'", eid, query).Find(&tenantEnvs).Error; err != nil {
			return nil, err
		}
	} else {
		if err := t.DB.Where("eid = ?", eid).Find(&tenantEnvs).Error; err != nil {
			return nil, err
		}
	}
	return tenantEnvs, nil
}

// GetTenantEnvIDsByNames get tenant env ids by names
func (t *TenantEnvDaoImpl) GetTenantEnvIDsByNames(tenantName string, tenantEnvNames []string) (re []string, err error) {
	rows, err := t.DB.Raw("select uuid from tenantEnvs where tenant_name = ? and name in (?)", tenantName, tenantEnvNames).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var uuid string
		rows.Scan(&uuid)
		re = append(re, uuid)
	}
	return
}

// GetTenantEnvLimitsByNames get tenantEnvs memory limit
func (t *TenantEnvDaoImpl) GetTenantEnvLimitsByNames(tenantName string, tenantEnvNames []string) (limit map[string]int, err error) {
	limit = make(map[string]int)
	rows, err := t.DB.Raw("select uuid,limit_memory from tenantEnvs where tenant_name = ? and name in (?)", tenantName, tenantEnvNames).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var limitmemory int
		var uuid string
		rows.Scan(&uuid, &limitmemory)
		limit[uuid] = limitmemory
	}
	return
}

// GetPagedTenantEnvs -
func (t *TenantEnvDaoImpl) GetPagedTenantEnvs(offset, len int) ([]*model.TenantEnvs, error) {
	var tenantEnvs []*model.TenantEnvs
	if err := t.DB.Find(&tenantEnvs).Group("").Error; err != nil {
		return nil, err
	}
	return tenantEnvs, nil
}

// DelByTenantEnvID -
func (t *TenantEnvDaoImpl) DelByTenantEnvID(tenantEnvID string) error {
	if err := t.DB.Where("uuid=?", tenantEnvID).Delete(&model.TenantEnvs{}).Error; err != nil {
		return err
	}

	return nil
}

// TenantEnvServicesDaoImpl 租户应用dao
type TenantEnvServicesDaoImpl struct {
	DB *gorm.DB
}

// GetServiceTypeByID  get service type by service id
func (t *TenantEnvServicesDaoImpl) GetServiceTypeByID(serviceID string) (*model.TenantEnvServices, error) {
	var service model.TenantEnvServices
	if err := t.DB.Select("tenant_env_id, service_id, service_alias, extend_method").Where("service_id=?", serviceID).Find(&service).Error; err != nil {
		return nil, err
	}
	if service.ExtendMethod == "" {
		// for before V5.2 version
		logrus.Infof("get low version service[%s] type", serviceID)
		rows, err := t.DB.Raw("select label_value from tenant_env_services_label where service_id=? and label_key=?", serviceID, "service-type").Rows()
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		for rows.Next() {
			rows.Scan(&service.ExtendMethod)
		}
	}
	return &service, nil
}

// GetAllServicesID get all service sample info
func (t *TenantEnvServicesDaoImpl) GetAllServicesID() ([]*model.TenantEnvServices, error) {
	var services []*model.TenantEnvServices
	if err := t.DB.Select("service_id,service_alias,tenant_env_id,app_id").Find(&services).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return services, nil
		}
		return nil, err
	}
	return services, nil
}

// ListServicesByTenantEnvID -
func (t *TenantEnvServicesDaoImpl) ListServicesByTenantEnvID(tenantEnvID string) ([]*model.TenantEnvServices, error) {
	var services []*model.TenantEnvServices
	if err := t.DB.Where("tenant_env_id=?", tenantEnvID).Find(&services).Error; err != nil {
		return nil, err
	}

	return services, nil
}

// UpdateDeployVersion update service current deploy version
func (t *TenantEnvServicesDaoImpl) UpdateDeployVersion(serviceID, deployversion string) error {
	if err := t.DB.Exec("update tenant_env_services set deploy_version=? where service_id=?", deployversion, serviceID).Error; err != nil {
		return err
	}
	return nil
}

// AddModel 添加租户应用
func (t *TenantEnvServicesDaoImpl) AddModel(mo model.Interface) error {
	service := mo.(*model.TenantEnvServices)
	var oldService model.TenantEnvServices
	if ok := t.DB.Where("service_alias = ? and tenant_env_id=?", service.ServiceAlias, service.TenantEnvID).Find(&oldService).RecordNotFound(); ok {
		if err := t.DB.Create(service).Error; err != nil {
			return err
		}
	} else {
		return fmt.Errorf("service name  %s and  is exist in tenant env %s", service.ServiceAlias, service.TenantEnvID)
	}
	return nil
}

// UpdateModel 更新租户应用
func (t *TenantEnvServicesDaoImpl) UpdateModel(mo model.Interface) error {
	service := mo.(*model.TenantEnvServices)
	if err := t.DB.Save(service).Error; err != nil {
		return err
	}
	return nil
}

// GetServiceByID 获取服务通过服务id
func (t *TenantEnvServicesDaoImpl) GetServiceByID(serviceID string) (*model.TenantEnvServices, error) {
	var service model.TenantEnvServices
	if err := t.DB.Where("service_id=?", serviceID).Find(&service).Error; err != nil {
		return nil, err
	}
	return &service, nil
}

// GetServiceByServiceAlias 获取服务通过服务别名
func (t *TenantEnvServicesDaoImpl) GetServiceByServiceAlias(serviceAlias string) (*model.TenantEnvServices, error) {
	var service model.TenantEnvServices
	if err := t.DB.Where("service_alias=?", serviceAlias).Find(&service).Error; err != nil {
		return nil, err
	}
	return &service, nil
}

// GetServiceMemoryByTenantEnvIDs get service memory by tenant env ids
func (t *TenantEnvServicesDaoImpl) GetServiceMemoryByTenantEnvIDs(tenantEnvIDs []string, runningServiceIDs []string) (map[string]map[string]interface{}, error) {
	rows, err := t.DB.Raw("select tenant_env_id, sum(container_cpu) as cpu,sum(container_memory * replicas) as memory from tenant_env_services where tenant_env_id in (?) and service_id in (?) group by tenant_env_id", tenantEnvIDs, runningServiceIDs).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var rc = make(map[string]map[string]interface{})
	for rows.Next() {
		var cpu, mem int
		var tenantEnvID string
		rows.Scan(&tenantEnvID, &cpu, &mem)
		res := make(map[string]interface{})
		res["cpu"] = cpu
		res["memory"] = mem
		rc[tenantEnvID] = res
	}
	for _, sid := range tenantEnvIDs {
		if _, ok := rc[sid]; !ok {
			rc[sid] = make(map[string]interface{})
			rc[sid]["cpu"] = 0
			rc[sid]["memory"] = 0
		}
	}
	return rc, nil
}

// GetServiceMemoryByServiceIDs get service memory by service ids
func (t *TenantEnvServicesDaoImpl) GetServiceMemoryByServiceIDs(serviceIDs []string) (map[string]map[string]interface{}, error) {
	rows, err := t.DB.Raw("select service_id, container_cpu as cpu, container_memory as memory from tenant_env_services where service_id in (?)", serviceIDs).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var rc = make(map[string]map[string]interface{})
	for rows.Next() {
		var cpu, mem int
		var serviceID string
		rows.Scan(&serviceID, &cpu, &mem)
		res := make(map[string]interface{})
		res["cpu"] = cpu
		res["memory"] = mem
		rc[serviceID] = res
	}
	for _, sid := range serviceIDs {
		if _, ok := rc[sid]; !ok {
			rc[sid] = make(map[string]interface{})
			rc[sid]["cpu"] = 0
			rc[sid]["memory"] = 0
		}
	}
	return rc, nil
}

// GetPagedTenantEnvService GetPagedTenantEnvResource
func (t *TenantEnvServicesDaoImpl) GetPagedTenantEnvService(offset, length int, serviceIDs []string) ([]map[string]interface{}, int, error) {
	var count int
	var service model.TenantEnvServices
	var result []map[string]interface{}
	if len(serviceIDs) == 0 {
		return result, count, nil
	}
	var re []*model.TenantEnvServices
	if err := t.DB.Table(service.TableName()).Select("tenant_env_id").Where("service_id in (?)", serviceIDs).Group("tenant_env_id").Find(&re).Error; err != nil {
		return nil, count, err
	}
	count = len(re)
	rows, err := t.DB.Raw("SELECT tenant_env_id, SUM(container_cpu * replicas) AS use_cpu, SUM(container_memory * replicas) AS use_memory FROM tenant_env_services where service_id in (?) GROUP BY tenant_env_id ORDER BY use_memory DESC LIMIT ?,?", serviceIDs, offset, length).Rows()
	if err != nil {
		return nil, count, err
	}
	defer rows.Close()
	var rc = make(map[string]*map[string]interface{}, length)
	var tenantEnvIDs []string
	for rows.Next() {
		var tenantEnvID string
		var useCPU int
		var useMem int
		rows.Scan(&tenantEnvID, &useCPU, &useMem)
		res := make(map[string]interface{})
		res["usecpu"] = useCPU
		res["usemem"] = useMem
		res["tenantEnv"] = tenantEnvID
		rc[tenantEnvID] = &res
		result = append(result, res)
		tenantEnvIDs = append(tenantEnvIDs, tenantEnvID)
	}
	newrows, err := t.DB.Raw("SELECT tenant_env_id, SUM(container_cpu * replicas) AS cap_cpu, SUM(container_memory * replicas) AS cap_memory FROM tenant_env_services where tenant_env_id in (?) GROUP BY tenant_env_id", tenantEnvIDs).Rows()
	if err != nil {
		return nil, count, err
	}
	defer newrows.Close()
	for newrows.Next() {
		var tenantEnvID string
		var capCPU int
		var capMem int
		newrows.Scan(&tenantEnvID, &capCPU, &capMem)
		if _, ok := rc[tenantEnvID]; ok {
			s := (*rc[tenantEnvID])
			s["capcpu"] = capCPU
			s["capmem"] = capMem
			*rc[tenantEnvID] = s
		}
	}
	tenantEnvs, err := t.DB.Raw("SELECT uuid,name,eid from tenantEnvs where uuid in (?)", tenantEnvIDs).Rows()
	if err != nil {
		return nil, 0, pkgerr.Wrap(err, "list tenantEnvs")
	}
	defer tenantEnvs.Close()
	for tenantEnvs.Next() {
		var tenantEnvID string
		var name string
		var eid string
		tenantEnvs.Scan(&tenantEnvID, &name, &eid)
		if _, ok := rc[tenantEnvID]; ok {
			s := (*rc[tenantEnvID])
			s["eid"] = eid
			s["tenant_env_name"] = name
			*rc[tenantEnvID] = s
		}
	}
	return result, count, nil
}

// GetServiceAliasByIDs 获取应用别名
func (t *TenantEnvServicesDaoImpl) GetServiceAliasByIDs(uids []string) ([]*model.TenantEnvServices, error) {
	var services []*model.TenantEnvServices
	if err := t.DB.Where("service_id in (?)", uids).Select("service_alias,service_id").Find(&services).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return services, nil
		}
		return nil, err
	}
	return services, nil
}

// GetServiceByIDs get some service by service ids
func (t *TenantEnvServicesDaoImpl) GetServiceByIDs(uids []string) ([]*model.TenantEnvServices, error) {
	var services []*model.TenantEnvServices
	if err := t.DB.Where("service_id in (?)", uids).Find(&services).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return services, nil
		}
		return nil, err
	}
	return services, nil
}

// GetServiceByTenantEnvIDAndServiceAlias 根据租户名和服务名
func (t *TenantEnvServicesDaoImpl) GetServiceByTenantEnvIDAndServiceAlias(tenantEnvID, serviceName string) (*model.TenantEnvServices, error) {
	var service model.TenantEnvServices
	if err := t.DB.Where("service_alias = ? and tenant_env_id=?", serviceName, tenantEnvID).Find(&service).Error; err != nil {
		return nil, err
	}
	return &service, nil
}

// GetServicesByTenantEnvID GetServicesByTenantEnvID
func (t *TenantEnvServicesDaoImpl) GetServicesByTenantEnvID(tenantEnvID string) ([]*model.TenantEnvServices, error) {
	var services []*model.TenantEnvServices
	if err := t.DB.Where("tenant_env_id=?", tenantEnvID).Find(&services).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return services, nil
		}
		return nil, err
	}
	return services, nil
}

// GetServicesByTenantEnvIDs GetServicesByTenantEnvIDs
func (t *TenantEnvServicesDaoImpl) GetServicesByTenantEnvIDs(tenantEnvIDs []string) ([]*model.TenantEnvServices, error) {
	var services []*model.TenantEnvServices
	if err := t.DB.Where("tenant_env_id in (?)", tenantEnvIDs).Find(&services).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return services, nil
		}
		return nil, err
	}
	return services, nil
}

// GetServicesAllInfoByTenantEnvID GetServicesAllInfoByTenantEnvID
func (t *TenantEnvServicesDaoImpl) GetServicesAllInfoByTenantEnvID(tenantEnvID string) ([]*model.TenantEnvServices, error) {
	var services []*model.TenantEnvServices
	if err := t.DB.Where("tenant_env_id= ?", tenantEnvID).Find(&services).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return services, nil
		}
		return nil, err
	}
	return services, nil
}

// GetServicesInfoByAppID Get Services Info By ApplicationID
func (t *TenantEnvServicesDaoImpl) GetServicesInfoByAppID(appID string, page, pageSize int) ([]*model.TenantEnvServices, int64, error) {
	var (
		total    int64
		services []*model.TenantEnvServices
	)
	offset := (page - 1) * pageSize
	db := t.DB.Where("app_id=?", appID).Order("create_time desc")

	if err := db.Model(&model.TenantEnvServices{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := db.Limit(pageSize).Offset(offset).Find(&services).Error; err != nil {
		return nil, 0, err
	}
	return services, total, nil
}

// CountServiceByAppID get Service number by AppID
func (t *TenantEnvServicesDaoImpl) CountServiceByAppID(appID string) (int64, error) {
	var total int64

	if err := t.DB.Model(&model.TenantEnvServices{}).Where("app_id=?", appID).Count(&total).Error; err != nil {
		return 0, err
	}
	return total, nil
}

// GetServiceIDsByAppID get ServiceIDs by AppID
func (t *TenantEnvServicesDaoImpl) GetServiceIDsByAppID(appID string) (re []model.ServiceID) {
	if err := t.DB.Raw("SELECT service_id FROM tenant_env_services WHERE app_id=?", appID).
		Scan(&re).Error; err != nil {
		logrus.Errorf("select service_id failure %s", err.Error())
		return
	}
	return
}

// GetServicesByServiceIDs Get Services By ServiceIDs
func (t *TenantEnvServicesDaoImpl) GetServicesByServiceIDs(serviceIDs []string) ([]*model.TenantEnvServices, error) {
	var services []*model.TenantEnvServices
	if err := t.DB.Where("service_id in (?)", serviceIDs).Find(&services).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return services, nil
		}
		return nil, err
	}
	return services, nil
}

// SetTenantEnvServiceStatus SetTenantEnvServiceStatus
func (t *TenantEnvServicesDaoImpl) SetTenantEnvServiceStatus(serviceID, status string) error {
	var service model.TenantEnvServices
	if status == "closed" || status == "undeploy" {
		if err := t.DB.Model(&service).Where("service_id = ?", serviceID).Update(map[string]interface{}{"cur_status": status, "status": 0}).Error; err != nil {
			return err
		}
	} else {
		if err := t.DB.Model(&service).Where("service_id = ?", serviceID).Update(map[string]interface{}{"cur_status": status, "status": 1}).Error; err != nil {
			return err
		}
	}
	return nil
}

// DeleteServiceByServiceID DeleteServiceByServiceID
func (t *TenantEnvServicesDaoImpl) DeleteServiceByServiceID(serviceID string) error {
	ts := &model.TenantEnvServices{
		ServiceID: serviceID,
	}
	if err := t.DB.Where("service_id = ?", serviceID).Delete(ts).Error; err != nil {
		return err
	}
	return nil
}

// ListThirdPartyServices lists all third party services
func (t *TenantEnvServicesDaoImpl) ListThirdPartyServices() ([]*model.TenantEnvServices, error) {
	var res []*model.TenantEnvServices
	if err := t.DB.Where("kind=?", model.ServiceKindThirdParty.String()).Find(&res).Error; err != nil {
		return nil, err
	}
	return res, nil
}

// BindAppByServiceIDs binding application by serviceIDs
func (t *TenantEnvServicesDaoImpl) BindAppByServiceIDs(appID string, serviceIDs []string) error {
	var service model.TenantEnvServices
	if err := t.DB.Model(&service).Where("service_id in (?)", serviceIDs).Update("app_id", appID).Error; err != nil {
		return err
	}
	return nil
}

// CreateOrUpdateComponentsInBatch Batch insert or update component
func (t *TenantEnvServicesDaoImpl) CreateOrUpdateComponentsInBatch(components []*model.TenantEnvServices) error {
	var objects []interface{}
	for _, component := range components {
		objects = append(objects, *component)
	}
	if err := gormbulkups.BulkUpsert(t.DB, objects, 2000); err != nil {
		return pkgerr.Wrap(err, "create or update component in batch")
	}
	return nil
}

// DeleteByComponentIDs deletes components based on the given componentIDs.
func (t *TenantEnvServicesDaoImpl) DeleteByComponentIDs(tenantEnvID, appID string, componentIDs []string) error {
	if err := t.DB.Where("tenant_env_id=? and app_id=? and service_id in (?)", tenantEnvID, appID, componentIDs).Delete(&model.TenantEnvServices{}).Error; err != nil {
		return pkgerr.Wrap(err, "delete component failed")
	}
	return nil
}

// IsK8sComponentNameDuplicate -
func (t *TenantEnvServicesDaoImpl) IsK8sComponentNameDuplicate(appID, serviceID, k8sComponentName string) bool {
	var count int64
	if err := t.DB.Model(&model.TenantEnvServices{}).Where("app_id=? and service_id<>? and k8s_component_name=?", appID, serviceID, k8sComponentName).Count(&count).Error; err != nil {
		logrus.Errorf("judge K8s Component Name Duplicate failed %v", err)
		return true
	}
	return count > 0
}

// TenantEnvServicesDeleteImpl TenantEnvServiceDeleteImpl
type TenantEnvServicesDeleteImpl struct {
	DB *gorm.DB
}

// AddModel 添加已删除的应用
func (t *TenantEnvServicesDeleteImpl) AddModel(mo model.Interface) error {
	service := mo.(*model.TenantEnvServicesDelete)
	var oldService model.TenantEnvServicesDelete
	if ok := t.DB.Where("service_alias = ? and tenant_env_id=?", service.ServiceAlias, service.TenantEnvID).Find(&oldService).RecordNotFound(); ok {
		if err := t.DB.Create(service).Error; err != nil {
			return err
		}
	}
	return nil
}

// UpdateModel 更新租户应用
func (t *TenantEnvServicesDeleteImpl) UpdateModel(mo model.Interface) error {
	service := mo.(*model.TenantEnvServicesDelete)
	if err := t.DB.Save(service).Error; err != nil {
		return err
	}
	return nil
}

// GetTenantEnvServicesDeleteByCreateTime -
func (t *TenantEnvServicesDeleteImpl) GetTenantEnvServicesDeleteByCreateTime(createTime time.Time) ([]*model.TenantEnvServicesDelete, error) {
	var ServiceDel []*model.TenantEnvServicesDelete
	if err := t.DB.Where("create_time < ?", createTime).Find(&ServiceDel).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return ServiceDel, nil
		}
		return nil, err
	}
	return ServiceDel, nil
}

// DeleteTenantEnvServicesDelete -
func (t *TenantEnvServicesDeleteImpl) DeleteTenantEnvServicesDelete(record *model.TenantEnvServicesDelete) error {
	if err := t.DB.Delete(record).Error; err != nil {
		return err
	}
	return nil
}

// List returns a list of TenantEnvServicesDeletes.
func (t *TenantEnvServicesDeleteImpl) List() ([]*model.TenantEnvServicesDelete, error) {
	var components []*model.TenantEnvServicesDelete
	if err := t.DB.Find(&components).Error; err != nil {
		return nil, pkgerr.Wrap(err, "list deleted components")
	}
	return components, nil
}

// TenantEnvServicesPortDaoImpl 租户应用端口操作
type TenantEnvServicesPortDaoImpl struct {
	DB *gorm.DB
}

// AddModel 添加应用端口
func (t *TenantEnvServicesPortDaoImpl) AddModel(mo model.Interface) error {
	port := mo.(*model.TenantEnvServicesPort)
	var oldPort model.TenantEnvServicesPort
	if ok := t.DB.Where("service_id = ? and container_port = ?", port.ServiceID, port.ContainerPort).Find(&oldPort).RecordNotFound(); ok {
		if err := t.DB.Create(port).Error; err != nil {
			return err
		}
	} else {
		return errors.ErrRecordAlreadyExist
	}
	return nil
}

// UpdateModel 更新租户
func (t *TenantEnvServicesPortDaoImpl) UpdateModel(mo model.Interface) error {
	port := mo.(*model.TenantEnvServicesPort)
	if port.ID == 0 {
		return fmt.Errorf("port id can not be empty when update ")
	}
	if err := t.DB.Save(port).Error; err != nil {
		return err
	}
	return nil
}

// CreateOrUpdatePortsInBatch Batch insert or update ports variables
func (t *TenantEnvServicesPortDaoImpl) CreateOrUpdatePortsInBatch(ports []*model.TenantEnvServicesPort) error {
	var objects []interface{}
	// dedup
	existPorts := make(map[string]struct{})
	for _, port := range ports {
		if _, ok := existPorts[port.Key()]; ok {
			continue
		}
		existPorts[port.Key()] = struct{}{}

		objects = append(objects, *port)
	}
	if err := gormbulkups.BulkUpsert(t.DB, objects, 2000); err != nil {
		return pkgerr.Wrap(err, "create or update ports in batch")
	}
	return nil
}

// DeleteModel 删除端口
func (t *TenantEnvServicesPortDaoImpl) DeleteModel(serviceID string, args ...interface{}) error {
	if len(args) < 1 {
		return fmt.Errorf("can not provide containerPort")
	}
	containerPort := args[0].(int)
	tsp := &model.TenantEnvServicesPort{
		ServiceID:     serviceID,
		ContainerPort: containerPort,
		//Protocol:      protocol,
	}
	if err := t.DB.Where("service_id=? and container_port=?", serviceID, containerPort).Delete(tsp).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return pkgerr.Wrap(bcode.ErrPortNotFound, "delete component port")
		}
		return err
	}
	return nil
}

// GetByTenantEnvAndName -
func (t *TenantEnvServicesPortDaoImpl) GetByTenantEnvAndName(tenantEnvID, name string) (*model.TenantEnvServicesPort, error) {
	var port model.TenantEnvServicesPort
	if err := t.DB.Where("tenant_env_id=? and k8s_service_name=?", tenantEnvID, name).Find(&port).Error; err != nil {
		return nil, err
	}
	return &port, nil
}

// GetPortsByServiceID 通过服务获取port
func (t *TenantEnvServicesPortDaoImpl) GetPortsByServiceID(serviceID string) ([]*model.TenantEnvServicesPort, error) {
	var oldPort []*model.TenantEnvServicesPort
	if err := t.DB.Where("service_id = ?", serviceID).Find(&oldPort).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return oldPort, nil
		}
		return nil, err
	}
	return oldPort, nil
}

// GetOuterPorts  获取对外端口
func (t *TenantEnvServicesPortDaoImpl) GetOuterPorts(serviceID string) ([]*model.TenantEnvServicesPort, error) {
	var oldPort []*model.TenantEnvServicesPort
	if err := t.DB.Where("service_id = ? and is_outer_service=?", serviceID, true).Find(&oldPort).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return oldPort, nil
		}
		return nil, err
	}
	return oldPort, nil
}

// GetInnerPorts 获取对内端口
func (t *TenantEnvServicesPortDaoImpl) GetInnerPorts(serviceID string) ([]*model.TenantEnvServicesPort, error) {
	var oldPort []*model.TenantEnvServicesPort
	if err := t.DB.Where("service_id = ? and is_inner_service=?", serviceID, true).Find(&oldPort).Error; err != nil {
		return nil, err
	}
	return oldPort, nil
}

// GetPort get port
func (t *TenantEnvServicesPortDaoImpl) GetPort(serviceID string, port int) (*model.TenantEnvServicesPort, error) {
	var oldPort model.TenantEnvServicesPort
	if err := t.DB.Where("service_id = ? and container_port=?", serviceID, port).Find(&oldPort).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, pkgerr.Wrap(bcode.ErrPortNotFound, fmt.Sprintf("component id: %s; port: %d; get port: %v", serviceID, port, err))
		}
		return nil, err
	}
	return &oldPort, nil
}

// GetOpenedPorts returns opened ports.
func (t *TenantEnvServicesPortDaoImpl) GetOpenedPorts(serviceID string) ([]*model.TenantEnvServicesPort, error) {
	var ports []*model.TenantEnvServicesPort
	if err := t.DB.Where("service_id = ? and (is_inner_service=1 or is_outer_service=1)", serviceID).
		Find(&ports).Error; err != nil {
		return nil, err
	}
	return ports, nil
}

// DELPortsByServiceID DELPortsByServiceID
func (t *TenantEnvServicesPortDaoImpl) DELPortsByServiceID(serviceID string) error {
	var port model.TenantEnvServicesPort
	if err := t.DB.Where("service_id=?", serviceID).Delete(&port).Error; err != nil {
		return err
	}
	return nil
}

// HasOpenPort checks if the given service(according to sid) has open port.
func (t *TenantEnvServicesPortDaoImpl) HasOpenPort(sid string) bool {
	var port model.TenantEnvServicesPort
	if err := t.DB.Where("service_id = ? and (is_outer_service=1 or is_inner_service=1)", sid).
		Find(&port).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			logrus.Warningf("error getting TenantEnvServicesPort: %v", err)
		}
		return false
	}
	return true
}

// GetDepUDPPort get all depend service udp port
func (t *TenantEnvServicesPortDaoImpl) GetDepUDPPort(serviceID string) ([]*model.TenantEnvServicesPort, error) {
	var portInfos []*model.TenantEnvServicesPort
	var port model.TenantEnvServicesPort
	var relation model.TenantEnvServiceRelation
	if err := t.DB.Raw(fmt.Sprintf("select * from %s where protocol=? and service_id in (select dep_service_id from %s where service_id=?)", port.TableName(), relation.TableName()), "udp", serviceID).Scan(&portInfos).Error; err != nil {
		return nil, err
	}
	return portInfos, nil
}

// DelByServiceID deletes TenantEnvServicesPort matching sid(service_id).
func (t *TenantEnvServicesPortDaoImpl) DelByServiceID(sid string) error {
	return t.DB.Where("service_id=?", sid).Delete(&model.TenantEnvServicesPort{}).Error
}

// DeleteByComponentIDs -
func (t *TenantEnvServicesPortDaoImpl) DeleteByComponentIDs(componentIDs []string) error {
	return t.DB.Where("service_id in (?)", componentIDs).Delete(&model.TenantEnvServicesPort{}).Error
}

// ListInnerPortsByServiceIDs -
func (t *TenantEnvServicesPortDaoImpl) ListInnerPortsByServiceIDs(serviceIDs []string) ([]*model.TenantEnvServicesPort, error) {
	var ports []*model.TenantEnvServicesPort
	if err := t.DB.Where("service_id in (?) and is_inner_service=?", serviceIDs, true).Find(&ports).Error; err != nil {
		return nil, err
	}

	return ports, nil
}

// ListByK8sServiceNames -
func (t *TenantEnvServicesPortDaoImpl) ListByK8sServiceNames(k8sServiceNames []string) ([]*model.TenantEnvServicesPort, error) {
	var ports []*model.TenantEnvServicesPort
	if err := t.DB.Where("k8s_service_name in (?)", k8sServiceNames).Find(&ports).Error; err != nil {
		return nil, err
	}
	return ports, nil
}

// TenantEnvServiceRelationDaoImpl TenantEnvServiceRelationDaoImpl
type TenantEnvServiceRelationDaoImpl struct {
	DB *gorm.DB
}

// AddModel 添加应用依赖关系
func (t *TenantEnvServiceRelationDaoImpl) AddModel(mo model.Interface) error {
	relation := mo.(*model.TenantEnvServiceRelation)
	var oldRelation model.TenantEnvServiceRelation
	if ok := t.DB.Where("service_id = ? and dep_service_id = ?", relation.ServiceID, relation.DependServiceID).Find(&oldRelation).RecordNotFound(); ok {
		if err := t.DB.Create(relation).Error; err != nil {
			return err
		}
	} else {
		return errors.ErrRecordAlreadyExist
	}
	return nil
}

// UpdateModel 更新应用依赖关系
func (t *TenantEnvServiceRelationDaoImpl) UpdateModel(mo model.Interface) error {
	relation := mo.(*model.TenantEnvServiceRelation)
	if relation.ID == 0 {
		return fmt.Errorf("relation id can not be empty when update ")
	}
	if err := t.DB.Save(relation).Error; err != nil {
		return err
	}
	return nil
}

// DeleteModel 删除依赖
func (t *TenantEnvServiceRelationDaoImpl) DeleteModel(serviceID string, args ...interface{}) error {
	depServiceID := args[0].(string)
	relation := &model.TenantEnvServiceRelation{
		ServiceID:       serviceID,
		DependServiceID: depServiceID,
	}
	logrus.Infof("service: %v, depend: %v", serviceID, depServiceID)
	if err := t.DB.Where("service_id=? and dep_service_id=?", serviceID, depServiceID).Delete(relation).Error; err != nil {
		return err
	}
	return nil
}

// DeleteRelationByDepID DeleteRelationByDepID
func (t *TenantEnvServiceRelationDaoImpl) DeleteRelationByDepID(serviceID, depID string) error {
	relation := &model.TenantEnvServiceRelation{
		ServiceID:       serviceID,
		DependServiceID: depID,
	}
	if err := t.DB.Where("service_id=? and dep_service_id=?", serviceID, depID).Delete(relation).Error; err != nil {
		return err
	}
	return nil
}

// DeleteByComponentIDs -
func (t *TenantEnvServiceRelationDaoImpl) DeleteByComponentIDs(componentIDs []string) error {
	return t.DB.Where("service_id in (?)", componentIDs).Delete(&model.TenantEnvServiceRelation{}).Error
}

// CreateOrUpdateRelationsInBatch -
func (t *TenantEnvServiceRelationDaoImpl) CreateOrUpdateRelationsInBatch(relations []*model.TenantEnvServiceRelation) error {
	var objects []interface{}
	for _, relation := range relations {
		objects = append(objects, *relation)
	}
	if err := gormbulkups.BulkUpsert(t.DB, objects, 2000); err != nil {
		return pkgerr.Wrap(err, "create or update relation in batch")
	}
	return nil
}

// GetTenantEnvServiceRelations 获取应用依赖关系
func (t *TenantEnvServiceRelationDaoImpl) GetTenantEnvServiceRelations(serviceID string) ([]*model.TenantEnvServiceRelation, error) {
	var oldRelation []*model.TenantEnvServiceRelation
	if err := t.DB.Where("service_id = ?", serviceID).Find(&oldRelation).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return oldRelation, nil
		}
		return nil, err
	}
	return oldRelation, nil
}

// ListByServiceIDs -
func (t *TenantEnvServiceRelationDaoImpl) ListByServiceIDs(serviceIDs []string) ([]*model.TenantEnvServiceRelation, error) {
	var relations []*model.TenantEnvServiceRelation
	if err := t.DB.Where("service_id in (?)", serviceIDs).Find(&relations).Error; err != nil {
		return nil, err
	}

	return relations, nil
}

// HaveRelations 是否有依赖
func (t *TenantEnvServiceRelationDaoImpl) HaveRelations(serviceID string) bool {
	var oldRelation []*model.TenantEnvServiceRelation
	if err := t.DB.Where("service_id = ?", serviceID).Find(&oldRelation).Error; err != nil {
		return false
	}
	if len(oldRelation) > 0 {
		return true
	}
	return false
}

// DELRelationsByServiceID DELRelationsByServiceID
func (t *TenantEnvServiceRelationDaoImpl) DELRelationsByServiceID(serviceID string) error {
	relation := &model.TenantEnvServiceRelation{
		ServiceID: serviceID,
	}
	if err := t.DB.Where("service_id=?", serviceID).Delete(relation).Error; err != nil {
		return err
	}
	logrus.Debugf("service id: %s; delete service relation successfully", serviceID)
	return nil
}

// GetTenantEnvServiceRelationsByDependServiceID 获取全部依赖当前服务的应用
func (t *TenantEnvServiceRelationDaoImpl) GetTenantEnvServiceRelationsByDependServiceID(dependServiceID string) ([]*model.TenantEnvServiceRelation, error) {
	var oldRelation []*model.TenantEnvServiceRelation
	if err := t.DB.Where("dep_service_id = ?", dependServiceID).Find(&oldRelation).Error; err != nil {
		return nil, err
	}
	return oldRelation, nil
}

// TenantEnvServiceEnvVarDaoImpl TenantEnvServiceEnvVarDaoImpl
type TenantEnvServiceEnvVarDaoImpl struct {
	DB *gorm.DB
}

// AddModel 添加应用环境变量
func (t *TenantEnvServiceEnvVarDaoImpl) AddModel(mo model.Interface) error {
	relation := mo.(*model.TenantEnvServiceEnvVar)
	var oldRelation model.TenantEnvServiceEnvVar
	if ok := t.DB.Where("service_id = ? and attr_name = ?", relation.ServiceID, relation.AttrName).Find(&oldRelation).RecordNotFound(); ok {
		if len(relation.AttrValue) > 65532 {
			relation.AttrValue = relation.AttrValue[:65532]
		}
		if err := t.DB.Create(relation).Error; err != nil {
			return err
		}
	} else {
		return errors.ErrRecordAlreadyExist
	}
	return nil
}

// UpdateModel update env support attr_value\is_change\scope
func (t *TenantEnvServiceEnvVarDaoImpl) UpdateModel(mo model.Interface) error {
	env := mo.(*model.TenantEnvServiceEnvVar)
	return t.DB.Table(env.TableName()).Where("service_id=? and attr_name = ?", env.ServiceID, env.AttrName).Update(map[string]interface{}{
		"attr_value": env.AttrValue,
		"is_change":  env.IsChange,
		"scope":      env.Scope,
	}).Error
}

// DeleteByComponentIDs -
func (t *TenantEnvServiceEnvVarDaoImpl) DeleteByComponentIDs(componentIDs []string) error {
	return t.DB.Where("service_id in (?)", componentIDs).Delete(&model.TenantEnvServiceEnvVar{}).Error
}

// CreateOrUpdateEnvsInBatch Batch insert or update environment variables
func (t *TenantEnvServiceEnvVarDaoImpl) CreateOrUpdateEnvsInBatch(envs []*model.TenantEnvServiceEnvVar) error {
	var objects []interface{}
	existEnvs := make(map[string]struct{})
	for _, env := range envs {
		key := fmt.Sprintf("%s+%s+%s", env.TenantEnvID, env.ServiceID, env.AttrName)
		if _, ok := existEnvs[key]; ok {
			continue
		}
		existEnvs[key] = struct{}{}

		objects = append(objects, *env)
	}
	if err := gormbulkups.BulkUpsert(t.DB, objects, 2000); err != nil {
		return pkgerr.Wrap(err, "create or update envs in batch")
	}
	return nil
}

// DeleteModel 删除env
func (t *TenantEnvServiceEnvVarDaoImpl) DeleteModel(serviceID string, args ...interface{}) error {
	envName := args[0].(string)
	relation := &model.TenantEnvServiceEnvVar{
		ServiceID: serviceID,
		AttrName:  envName,
	}
	if err := t.DB.Where("service_id=? and attr_name=?", serviceID, envName).Delete(relation).Error; err != nil {
		return err
	}
	return nil
}

// GetDependServiceEnvs 获取依赖服务的环境变量
func (t *TenantEnvServiceEnvVarDaoImpl) GetDependServiceEnvs(serviceIDs []string, scopes []string) ([]*model.TenantEnvServiceEnvVar, error) {
	var envs []*model.TenantEnvServiceEnvVar
	if err := t.DB.Where("service_id in (?) and scope in (?)", serviceIDs, scopes).Find(&envs).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return envs, nil
		}
		return nil, err
	}
	return envs, nil
}

// GetServiceEnvs 获取服务环境变量
func (t *TenantEnvServiceEnvVarDaoImpl) GetServiceEnvs(serviceID string, scopes []string) ([]*model.TenantEnvServiceEnvVar, error) {
	var envs []*model.TenantEnvServiceEnvVar
	if scopes == nil {
		if err := t.DB.Where("service_id=?", serviceID).Find(&envs).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return envs, nil
			}
			return nil, err
		}
	} else {
		if err := t.DB.Where("service_id=? and scope in (?)", serviceID, scopes).Find(&envs).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return envs, nil
			}
			return nil, err
		}
	}
	return envs, nil
}

// GetEnv 获取某个环境变量
func (t *TenantEnvServiceEnvVarDaoImpl) GetEnv(serviceID, envName string) (*model.TenantEnvServiceEnvVar, error) {
	var env model.TenantEnvServiceEnvVar
	if err := t.DB.Where("service_id=? and attr_name=? ", serviceID, envName).Find(&env).Error; err != nil {
		return nil, err
	}
	return &env, nil
}

// DELServiceEnvsByServiceID 通过serviceID 删除envs
func (t *TenantEnvServiceEnvVarDaoImpl) DELServiceEnvsByServiceID(serviceID string) error {
	var env model.TenantEnvServiceEnvVar
	if err := t.DB.Where("service_id=?", serviceID).Find(&env).Error; err != nil {
		return err
	}
	if err := t.DB.Where("service_id=?", serviceID).Delete(&env).Error; err != nil {
		return err
	}
	return nil
}

// DelByServiceIDAndScope deletes TenantEnvServiceEnvVar based on sid(service_id) and scope.
func (t *TenantEnvServiceEnvVarDaoImpl) DelByServiceIDAndScope(sid, scope string) error {
	var env model.TenantEnvServiceEnvVar
	if err := t.DB.Where("service_id=? and scope=?", sid, scope).Delete(&env).Error; err != nil {
		return err
	}
	return nil
}

// TenantEnvServiceMountRelationDaoImpl 依赖存储
type TenantEnvServiceMountRelationDaoImpl struct {
	DB *gorm.DB
}

// AddModel 添加应用依赖挂载
func (t *TenantEnvServiceMountRelationDaoImpl) AddModel(mo model.Interface) error {
	relation := mo.(*model.TenantEnvServiceMountRelation)
	var oldRelation model.TenantEnvServiceMountRelation
	if ok := t.DB.Where("service_id = ? and dep_service_id = ? and volume_name=?", relation.ServiceID, relation.DependServiceID, relation.VolumeName).Find(&oldRelation).RecordNotFound(); ok {
		if err := t.DB.Create(relation).Error; err != nil {
			return err
		}
	} else {
		return errors.ErrRecordAlreadyExist
	}
	return nil
}

// UpdateModel 更新应用依赖挂载
func (t *TenantEnvServiceMountRelationDaoImpl) UpdateModel(mo model.Interface) error {
	relation := mo.(*model.TenantEnvServiceMountRelation)
	if relation.ID == 0 {
		return fmt.Errorf("mount relation id can not be empty when update ")
	}
	if err := t.DB.Save(relation).Error; err != nil {
		return err
	}
	return nil
}

// DElTenantEnvServiceMountRelationByServiceAndName DElTenantEnvServiceMountRelationByServiceAndName
func (t *TenantEnvServiceMountRelationDaoImpl) DElTenantEnvServiceMountRelationByServiceAndName(serviceID, name string) error {
	var relation model.TenantEnvServiceMountRelation
	if err := t.DB.Where("service_id=? and volume_name=? ", serviceID, name).Find(&relation).Error; err != nil {
		return err
	}
	if err := t.DB.Where("service_id=? and volume_name=? ", serviceID, name).Delete(&relation).Error; err != nil {
		return err
	}
	return nil
}

// DElTenantEnvServiceMountRelationByDepService del mount relation
func (t *TenantEnvServiceMountRelationDaoImpl) DElTenantEnvServiceMountRelationByDepService(serviceID, depServiceID string) error {
	var relation model.TenantEnvServiceMountRelation
	if err := t.DB.Where("service_id=? and dep_service_id=?", serviceID, depServiceID).Find(&relation).Error; err != nil {
		return err
	}
	if err := t.DB.Where("service_id=? and dep_service_id=?", serviceID, depServiceID).Delete(&relation).Error; err != nil {
		return err
	}
	return nil
}

// DELTenantEnvServiceMountRelationByServiceID DELTenantEnvServiceMountRelationByServiceID
func (t *TenantEnvServiceMountRelationDaoImpl) DELTenantEnvServiceMountRelationByServiceID(serviceID string) error {
	var relation model.TenantEnvServiceMountRelation
	if err := t.DB.Where("service_id=?", serviceID).Delete(&relation).Error; err != nil {
		return err
	}
	return nil
}

// GetTenantEnvServiceMountRelationsByService 获取应用的所有挂载依赖
func (t *TenantEnvServiceMountRelationDaoImpl) GetTenantEnvServiceMountRelationsByService(serviceID string) ([]*model.TenantEnvServiceMountRelation, error) {
	var relations []*model.TenantEnvServiceMountRelation
	if err := t.DB.Where("service_id=? ", serviceID).Find(&relations).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return relations, nil
		}
		return nil, err
	}
	return relations, nil
}

// DeleteByComponentIDs -
func (t *TenantEnvServiceMountRelationDaoImpl) DeleteByComponentIDs(componentIDs []string) error {
	return t.DB.Where("service_id in (?)", componentIDs).Delete(&model.TenantEnvServiceMountRelation{}).Error
}

// CreateOrUpdateVolumeRelsInBatch -
func (t *TenantEnvServiceMountRelationDaoImpl) CreateOrUpdateVolumeRelsInBatch(volRels []*model.TenantEnvServiceMountRelation) error {
	var objects []interface{}
	for _, volRel := range volRels {
		objects = append(objects, *volRel)
	}
	if err := gormbulkups.BulkUpsert(t.DB, objects, 2000); err != nil {
		return pkgerr.Wrap(err, "create or update volume relation in batch")
	}
	return nil
}

// TenantEnvServiceVolumeDaoImpl 应用存储
type TenantEnvServiceVolumeDaoImpl struct {
	DB *gorm.DB
}

// GetAllVolumes 获取全部存储信息
func (t *TenantEnvServiceVolumeDaoImpl) GetAllVolumes() ([]*model.TenantEnvServiceVolume, error) {
	var volumes []*model.TenantEnvServiceVolume
	if err := t.DB.Find(&volumes).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return volumes, nil
		}
		return nil, err
	}
	return volumes, nil
}

// AddModel 添加应用挂载
func (t *TenantEnvServiceVolumeDaoImpl) AddModel(mo model.Interface) error {
	volume := mo.(*model.TenantEnvServiceVolume)
	var oldvolume model.TenantEnvServiceVolume
	if ok := t.DB.Where("(volume_name=? or volume_path = ?) and service_id=?", volume.VolumeName, volume.VolumePath, volume.ServiceID).Find(&oldvolume).RecordNotFound(); ok {
		if err := t.DB.Create(volume).Error; err != nil {
			return err
		}
	} else {
		return fmt.Errorf("service  %s volume name %s  path  %s is exist ", volume.ServiceID, volume.VolumeName, volume.VolumePath)
	}
	return nil
}

// UpdateModel 更��应用挂载
func (t *TenantEnvServiceVolumeDaoImpl) UpdateModel(mo model.Interface) error {
	volume := mo.(*model.TenantEnvServiceVolume)
	if volume.ID == 0 {
		return fmt.Errorf("volume id can not be empty when update ")
	}
	if err := t.DB.Save(volume).Error; err != nil {
		return err
	}
	return nil
}

// GetTenantEnvServiceVolumesByServiceID 获取应用挂载
func (t *TenantEnvServiceVolumeDaoImpl) GetTenantEnvServiceVolumesByServiceID(serviceID string) ([]*model.TenantEnvServiceVolume, error) {
	var volumes []*model.TenantEnvServiceVolume
	if err := t.DB.Where("service_id=? ", serviceID).Find(&volumes).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return volumes, nil
		}
		return nil, err
	}
	return volumes, nil
}

// ListVolumesByComponentIDs -
func (t *TenantEnvServiceVolumeDaoImpl) ListVolumesByComponentIDs(componentIDs []string) ([]*model.TenantEnvServiceVolume, error) {
	var volumes []*model.TenantEnvServiceVolume
	if err := t.DB.Where("service_id in (?)", componentIDs).Find(&volumes).Error; err != nil {
		return nil, err
	}
	return volumes, nil
}

// DeleteByVolumeIDs -
func (t *TenantEnvServiceVolumeDaoImpl) DeleteByVolumeIDs(volumeIDs []uint) error {
	return t.DB.Where("ID in (?)", volumeIDs).Delete(&model.TenantEnvServiceVolume{}).Error
}

// DeleteByComponentIDs -
func (t *TenantEnvServiceVolumeDaoImpl) DeleteByComponentIDs(componentIDs []string) error {
	return t.DB.Where("service_id in (?)", componentIDs).Delete(&model.TenantEnvServiceVolume{}).Error
}

// CreateOrUpdateVolumesInBatch -
func (t *TenantEnvServiceVolumeDaoImpl) CreateOrUpdateVolumesInBatch(volumes []*model.TenantEnvServiceVolume) error {
	var objects []interface{}
	for _, volume := range volumes {
		objects = append(objects, *volume)
	}
	if err := gormbulkups.BulkUpsert(t.DB, objects, 2000); err != nil {
		return pkgerr.Wrap(err, "create or update volumes in batch")
	}
	return nil
}

// DeleteModel 删除挂载
func (t *TenantEnvServiceVolumeDaoImpl) DeleteModel(serviceID string, args ...interface{}) error {
	var volume model.TenantEnvServiceVolume
	volumeName := args[0].(string)
	if err := t.DB.Where("volume_name = ? and service_id=?", volumeName, serviceID).Find(&volume).Error; err != nil {
		return err
	}
	if err := t.DB.Where("volume_name = ? and service_id=?", volumeName, serviceID).Delete(&volume).Error; err != nil {
		return err
	}
	return nil
}

// DeleteByServiceIDAndVolumePath 删除挂载通过挂载的目录
func (t *TenantEnvServiceVolumeDaoImpl) DeleteByServiceIDAndVolumePath(serviceID string, volumePath string) error {
	var volume model.TenantEnvServiceVolume
	if err := t.DB.Where("volume_path = ? and service_id=?", volumePath, serviceID).Find(&volume).Error; err != nil {
		return err
	}
	if err := t.DB.Where("volume_path = ? and service_id=?", volumePath, serviceID).Delete(&volume).Error; err != nil {
		return err
	}
	return nil
}

// GetVolumeByServiceIDAndName 获取存储信息
func (t *TenantEnvServiceVolumeDaoImpl) GetVolumeByServiceIDAndName(serviceID, name string) (*model.TenantEnvServiceVolume, error) {
	var volume model.TenantEnvServiceVolume
	if err := t.DB.Where("service_id=? and volume_name=? ", serviceID, name).Find(&volume).Error; err != nil {
		return nil, err
	}
	return &volume, nil
}

// GetVolumeByID get volume by id
func (t *TenantEnvServiceVolumeDaoImpl) GetVolumeByID(id int) (*model.TenantEnvServiceVolume, error) {
	var volume model.TenantEnvServiceVolume
	if err := t.DB.Where("ID=?", id).Find(&volume).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, dao.ErrVolumeNotFound
		}
		return nil, err
	}
	return &volume, nil
}

// DeleteTenantEnvServiceVolumesByServiceID 删除挂载
func (t *TenantEnvServiceVolumeDaoImpl) DeleteTenantEnvServiceVolumesByServiceID(serviceID string) error {
	var volume model.TenantEnvServiceVolume
	if err := t.DB.Where("service_id=? ", serviceID).Delete(&volume).Error; err != nil {
		return err
	}
	return nil
}

// DelShareableBySID deletes shareable volumes based on sid(service_id)
func (t *TenantEnvServiceVolumeDaoImpl) DelShareableBySID(sid string) error {
	query := "service_id=? and volume_type in ('share-file', 'config-file')"
	return t.DB.Where(query, sid).Delete(&model.TenantEnvServiceVolume{}).Error
}

// TenantEnvServiceConfigFileDaoImpl is a implementation of TenantEnvServiceConfigFileDao
type TenantEnvServiceConfigFileDaoImpl struct {
	DB *gorm.DB
}

// AddModel creates a new TenantEnvServiceConfigFile
func (t *TenantEnvServiceConfigFileDaoImpl) AddModel(mo model.Interface) error {
	configFile, ok := mo.(*model.TenantEnvServiceConfigFile)
	if !ok {
		return fmt.Errorf("can't convert %s to *model.TenantEnvServiceConfigFile", reflect.TypeOf(mo))
	}
	var old model.TenantEnvServiceConfigFile
	if ok := t.DB.Where("service_id=? and volume_name=?", configFile.ServiceID,
		configFile.VolumeName).Find(&old).RecordNotFound(); ok {
		if err := t.DB.Create(configFile).Error; err != nil {
			return err
		}
	} else {
		old.FileContent = configFile.FileContent
		if err := t.DB.Save(&old).Error; err != nil {
			return err
		}
	}
	return nil
}

// UpdateModel updates config file
func (t *TenantEnvServiceConfigFileDaoImpl) UpdateModel(mo model.Interface) error {
	configFile, ok := mo.(*model.TenantEnvServiceConfigFile)
	if !ok {
		return fmt.Errorf("can't convert %s to *model.TenantEnvServiceConfigFile", reflect.TypeOf(mo))
	}
	return t.DB.Table(configFile.TableName()).
		Where("service_id=? and volume_name=?", configFile.ServiceID, configFile.VolumeName).
		Update(configFile).Error
}

// GetConfigFileByServiceID -
func (t *TenantEnvServiceConfigFileDaoImpl) GetConfigFileByServiceID(serviceID string) ([]*model.TenantEnvServiceConfigFile, error) {
	var configFiles []*model.TenantEnvServiceConfigFile
	if err := t.DB.Where("service_id=?", serviceID).Find(&configFiles).Error; err != nil {
		return nil, err
	}
	return configFiles, nil
}

// GetByVolumeName get config file by volume name
func (t *TenantEnvServiceConfigFileDaoImpl) GetByVolumeName(sid string, volumeName string) (*model.TenantEnvServiceConfigFile, error) {
	var res model.TenantEnvServiceConfigFile
	if err := t.DB.Where("service_id=? and volume_name = ?", sid, volumeName).
		Find(&res).Error; err != nil {
		return nil, err
	}
	return &res, nil
}

// DelByVolumeID deletes config files according to service id and volume id.
func (t *TenantEnvServiceConfigFileDaoImpl) DelByVolumeID(sid, volumeName string) error {
	var cfs []model.TenantEnvServiceConfigFile
	return t.DB.Where("service_id=? and volume_name = ?", sid, volumeName).Delete(&cfs).Error
}

// DelByServiceID deletes config files according to service id.
func (t *TenantEnvServiceConfigFileDaoImpl) DelByServiceID(sid string) error {
	return t.DB.Where("service_id=?", sid).Delete(&model.TenantEnvServiceConfigFile{}).Error
}

// DeleteByComponentIDs -
func (t *TenantEnvServiceConfigFileDaoImpl) DeleteByComponentIDs(componentIDs []string) error {
	return t.DB.Where("service_id in (?)", componentIDs).Delete(&model.TenantEnvServiceConfigFile{}).Error
}

// CreateOrUpdateConfigFilesInBatch -
func (t *TenantEnvServiceConfigFileDaoImpl) CreateOrUpdateConfigFilesInBatch(configFiles []*model.TenantEnvServiceConfigFile) error {
	var objects []interface{}
	for _, configFile := range configFiles {
		objects = append(objects, *configFile)
	}
	if err := gormbulkups.BulkUpsert(t.DB, objects, 2000); err != nil {
		return pkgerr.Wrap(err, "create or update config files in batch")
	}
	return nil
}

// TenantEnvServiceLBMappingPortDaoImpl stream服务映射
type TenantEnvServiceLBMappingPortDaoImpl struct {
	DB *gorm.DB
}

// AddModel 添加应用端口映射
func (t *TenantEnvServiceLBMappingPortDaoImpl) AddModel(mo model.Interface) error {
	mapPort := mo.(*model.TenantEnvServiceLBMappingPort)
	var oldMapPort model.TenantEnvServiceLBMappingPort
	if ok := t.DB.Where("port=? ", mapPort.Port).Find(&oldMapPort).RecordNotFound(); ok {
		if err := t.DB.Create(mapPort).Error; err != nil {
			return err
		}
	} else {
		return fmt.Errorf("external port(%d) is exist ", mapPort.Port)
	}
	return nil
}

// UpdateModel 更新应用端口映射
func (t *TenantEnvServiceLBMappingPortDaoImpl) UpdateModel(mo model.Interface) error {
	mapPort := mo.(*model.TenantEnvServiceLBMappingPort)
	if mapPort.ID == 0 {
		return fmt.Errorf("mapport id can not be empty when update ")
	}
	if err := t.DB.Save(mapPort).Error; err != nil {
		return err
	}
	return nil
}

// GetTenantEnvServiceLBMappingPort 获取端口映射
func (t *TenantEnvServiceLBMappingPortDaoImpl) GetTenantEnvServiceLBMappingPort(serviceID string, containerPort int) (*model.TenantEnvServiceLBMappingPort, error) {
	var mapPort model.TenantEnvServiceLBMappingPort
	if err := t.DB.Where("service_id=? and container_port=?", serviceID, containerPort).Find(&mapPort).Error; err != nil {
		return nil, err
	}
	return &mapPort, nil
}

// GetLBMappingPortByServiceIDAndPort returns a LBMappingPort by serviceID and port
func (t *TenantEnvServiceLBMappingPortDaoImpl) GetLBMappingPortByServiceIDAndPort(serviceID string, port int) (*model.TenantEnvServiceLBMappingPort, error) {
	var mapPort model.TenantEnvServiceLBMappingPort
	if err := t.DB.Where("service_id=? and port=?", serviceID, port).Find(&mapPort).Error; err != nil {
		return nil, err
	}
	return &mapPort, nil
}

// GetLBPortsASC gets all LBMappingPorts ascending
func (t *TenantEnvServiceLBMappingPortDaoImpl) GetLBPortsASC() ([]*model.TenantEnvServiceLBMappingPort, error) {
	var ports []*model.TenantEnvServiceLBMappingPort
	if err := t.DB.Order("port asc").Find(&ports).Error; err != nil {
		return nil, fmt.Errorf("select all exist port error,%s", err.Error())
	}
	return ports, nil
}

// CreateTenantEnvServiceLBMappingPort 创建负载均衡VS端口,如果端口分配已存在，直接返回
func (t *TenantEnvServiceLBMappingPortDaoImpl) CreateTenantEnvServiceLBMappingPort(serviceID string, containerPort int) (*model.TenantEnvServiceLBMappingPort, error) {
	var mapPorts []*model.TenantEnvServiceLBMappingPort
	var mapPort model.TenantEnvServiceLBMappingPort
	err := t.DB.Where("service_id=? and container_port=?", serviceID, containerPort).Find(&mapPort).Error
	if err == nil {
		return &mapPort, nil
	}
	//分配端口
	var ports []int
	err = t.DB.Order("port asc").Find(&mapPorts).Error
	if err != nil {
		return nil, fmt.Errorf("select all exist port error,%s", err.Error())
	}
	for _, p := range mapPorts {
		ports = append(ports, p.Port)
	}
	maxPort, _ := strconv.Atoi(os.Getenv("MIN_LB_PORT"))
	minPort, _ := strconv.Atoi(os.Getenv("MAX_LB_PORT"))
	if minPort == 0 {
		minPort = 20001
	}
	if maxPort == 0 {
		maxPort = 35000
	}
	var maxUsePort int
	if len(ports) > 0 {
		maxUsePort = ports[len(ports)-1]
	} else {
		maxUsePort = 20001
	}
	//顺序分配端口
	selectPort := maxUsePort + 1
	if selectPort <= maxPort {
		mp := &model.TenantEnvServiceLBMappingPort{
			ServiceID:     serviceID,
			Port:          selectPort,
			ContainerPort: containerPort,
		}
		if err := t.DB.Save(mp).Error; err == nil {
			return mp, nil
		}
	}
	//捡漏以前端口
	selectPort = minPort
	errCount := 0
	for _, p := range ports {
		if p == selectPort {
			selectPort = selectPort + 1
			continue
		}
		if p > selectPort {
			mp := &model.TenantEnvServiceLBMappingPort{
				ServiceID:     serviceID,
				Port:          selectPort,
				ContainerPort: containerPort,
			}
			if err := t.DB.Save(mp).Error; err != nil {
				logrus.Errorf("save select map vs port %d error %s", selectPort, err.Error())
				errCount++
				if errCount > 2 { //尝试3次
					break
				}
			} else {
				return mp, nil
			}
		}
		selectPort = selectPort + 1
	}
	if selectPort <= maxPort {
		mp := &model.TenantEnvServiceLBMappingPort{
			ServiceID:     serviceID,
			Port:          selectPort,
			ContainerPort: containerPort,
		}
		if err := t.DB.Save(mp).Error; err != nil {
			logrus.Errorf("save select map vs port %d error %s", selectPort, err.Error())
			return nil, fmt.Errorf("can not select a good port for service stream port")
		}
		return mp, nil
	}
	logrus.Errorf("no more lb port can be use,max port is %d", maxPort)
	return nil, fmt.Errorf("no more lb port can be use,max port is %d", maxPort)
}

// GetTenantEnvServiceLBMappingPortByService 获取端口映射
func (t *TenantEnvServiceLBMappingPortDaoImpl) GetTenantEnvServiceLBMappingPortByService(serviceID string) ([]*model.TenantEnvServiceLBMappingPort, error) {
	var mapPort []*model.TenantEnvServiceLBMappingPort
	if err := t.DB.Where("service_id=?", serviceID).Find(&mapPort).Error; err != nil {
		return nil, err
	}
	return mapPort, nil
}

// DELServiceLBMappingPortByServiceID DELServiceLBMappingPortByServiceID
func (t *TenantEnvServiceLBMappingPortDaoImpl) DELServiceLBMappingPortByServiceID(serviceID string) error {
	mapPorts := &model.TenantEnvServiceLBMappingPort{
		ServiceID: serviceID,
	}
	if err := t.DB.Where("service_id=?", serviceID).Delete(mapPorts).Error; err != nil {
		return err
	}
	return nil
}

// DELServiceLBMappingPortByServiceIDAndPort DELServiceLBMappingPortByServiceIDAndPort
func (t *TenantEnvServiceLBMappingPortDaoImpl) DELServiceLBMappingPortByServiceIDAndPort(serviceID string, lbport int) error {
	var mapPorts model.TenantEnvServiceLBMappingPort
	if err := t.DB.Where("service_id=? and port=?", serviceID, lbport).Delete(&mapPorts).Error; err != nil {
		return err
	}
	return nil
}

// GetLBPortByTenantEnvAndPort  GetLBPortByTenantEnvAndPort
func (t *TenantEnvServiceLBMappingPortDaoImpl) GetLBPortByTenantEnvAndPort(tenantEnvID string, lbport int) (*model.TenantEnvServiceLBMappingPort, error) {
	var mapPort model.TenantEnvServiceLBMappingPort
	if err := t.DB.Raw("select * from tenant_env_lb_mapping_port where port=? and service_id in(select service_id from tenant_env_services where tenant_env_id=?)", lbport, tenantEnvID).Scan(&mapPort).Error; err != nil {
		return nil, err
	}
	return &mapPort, nil
}

// PortExists checks if the port exists
func (t *TenantEnvServiceLBMappingPortDaoImpl) PortExists(port int) bool {
	var mapPorts model.TenantEnvServiceLBMappingPort
	return !t.DB.Where("port=?", port).Find(&mapPorts).RecordNotFound()
}

// ServiceLabelDaoImpl ServiceLabelDaoImpl
type ServiceLabelDaoImpl struct {
	DB *gorm.DB
}

// AddModel 添加应用Label
func (t *ServiceLabelDaoImpl) AddModel(mo model.Interface) error {
	label := mo.(*model.TenantEnvServiceLable)
	var oldLabel model.TenantEnvServiceLable
	if ok := t.DB.Where("service_id = ? and label_key=? and label_value=?", label.ServiceID, label.LabelKey, label.LabelValue).Find(&oldLabel).RecordNotFound(); ok {
		if err := t.DB.Create(label).Error; err != nil {
			return err
		}
	} else {
		return fmt.Errorf("label key %s value %s of service %s is exist", label.LabelKey, label.LabelValue, label.ServiceID)
	}
	return nil
}

// UpdateModel 更新应用Label
func (t *ServiceLabelDaoImpl) UpdateModel(mo model.Interface) error {
	label := mo.(*model.TenantEnvServiceLable)
	if label.ID == 0 {
		return fmt.Errorf("label id can not be empty when update ")
	}
	if err := t.DB.Save(label).Error; err != nil {
		return err
	}
	return nil
}

// DeleteModel 删除应用label
func (t *ServiceLabelDaoImpl) DeleteModel(serviceID string, args ...interface{}) error {
	label := &model.TenantEnvServiceLable{
		ServiceID:  serviceID,
		LabelKey:   args[0].(string),
		LabelValue: args[1].(string),
	}
	if err := t.DB.Where("service_id=? and label_key=? and label_value=?",
		serviceID, label.LabelKey, label.LabelValue).Delete(label).Error; err != nil {
		return err
	}
	return nil
}

// DeleteLabelByServiceID 删除应用全部label
func (t *ServiceLabelDaoImpl) DeleteLabelByServiceID(serviceID string) error {
	label := &model.TenantEnvServiceLable{
		ServiceID: serviceID,
	}
	if err := t.DB.Where("service_id=?", serviceID).Delete(label).Error; err != nil {
		return err
	}
	return nil
}

// GetTenantEnvServiceLabel GetTenantEnvServiceLabel
func (t *ServiceLabelDaoImpl) GetTenantEnvServiceLabel(serviceID string) ([]*model.TenantEnvServiceLable, error) {
	var labels []*model.TenantEnvServiceLable
	if err := t.DB.Where("service_id=?", serviceID).Find(&labels).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return labels, nil
		}
		return nil, err
	}
	return labels, nil
}

// GetTenantEnvServiceNodeSelectorLabel GetTenantEnvServiceNodeSelectorLabel
func (t *ServiceLabelDaoImpl) GetTenantEnvServiceNodeSelectorLabel(serviceID string) ([]*model.TenantEnvServiceLable, error) {
	var labels []*model.TenantEnvServiceLable
	if err := t.DB.Where("service_id=? and label_key=?", serviceID, model.LabelKeyNodeSelector).Find(&labels).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return labels, nil
		}
		return nil, err
	}
	return labels, nil
}

// GetLabelByNodeSelectorKey returns a label by node-selector and label_value
func (t *ServiceLabelDaoImpl) GetLabelByNodeSelectorKey(serviceID string, labelValue string) (*model.TenantEnvServiceLable, error) {
	var label model.TenantEnvServiceLable
	if err := t.DB.Where("service_id=? and label_key = ? and label_value=?", serviceID, model.LabelKeyNodeSelector,
		labelValue).Find(&label).Error; err != nil {
		return nil, err
	}
	return &label, nil
}

// GetTenantEnvNodeAffinityLabel returns TenantEnvServiceLable matching serviceID and LabelKeyNodeAffinity
func (t *ServiceLabelDaoImpl) GetTenantEnvNodeAffinityLabel(serviceID string) (*model.TenantEnvServiceLable, error) {
	var label model.TenantEnvServiceLable
	if err := t.DB.Where("service_id=? and label_key = ?", serviceID, model.LabelKeyNodeAffinity).
		Find(&label).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return &label, nil
		}
		return nil, err
	}
	return &label, nil
}

// GetTenantEnvServiceAffinityLabel GetTenantEnvServiceAffinityLabel
func (t *ServiceLabelDaoImpl) GetTenantEnvServiceAffinityLabel(serviceID string) ([]*model.TenantEnvServiceLable, error) {
	var labels []*model.TenantEnvServiceLable
	if err := t.DB.Where("service_id=? and label_key in (?)", serviceID, []string{model.LabelKeyNodeSelector, model.LabelKeyNodeAffinity,
		model.LabelKeyServiceAffinity, model.LabelKeyServiceAntyAffinity}).Find(&labels).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return labels, nil
		}
		return nil, err
	}
	return labels, nil
}

// GetTenantEnvServiceTypeLabel GetTenantEnvServiceTypeLabel
// no usages func. get tenant env service type use TenantEnvServiceDao.GetServiceTypeByID(serviceID string)
func (t *ServiceLabelDaoImpl) GetTenantEnvServiceTypeLabel(serviceID string) (*model.TenantEnvServiceLable, error) {
	var label model.TenantEnvServiceLable
	return &label, nil
}

// GetPrivilegedLabel -
func (t *ServiceLabelDaoImpl) GetPrivilegedLabel(serviceID string) (*model.TenantEnvServiceLable, error) {
	var label model.TenantEnvServiceLable
	if err := t.DB.Where("service_id=? and label_value=?", serviceID, model.LabelKeyServicePrivileged).Find(&label).Error; err != nil {
		return nil, err
	}
	return &label, nil
}

// DelTenantEnvServiceLabelsByLabelValuesAndServiceID DELTenantEnvServiceLabelsByLabelvaluesAndServiceID
func (t *ServiceLabelDaoImpl) DelTenantEnvServiceLabelsByLabelValuesAndServiceID(serviceID string) error {
	var label model.TenantEnvServiceLable
	if err := t.DB.Where("service_id=? and label_value=?", serviceID, model.LabelKeyNodeSelector).Delete(&label).Error; err != nil {
		return err
	}
	return nil
}

// DelTenantEnvServiceLabelsByServiceIDKeyValue deletes labels
func (t *ServiceLabelDaoImpl) DelTenantEnvServiceLabelsByServiceIDKeyValue(serviceID string, labelKey string,
	labelValue string) error {
	var label model.TenantEnvServiceLable
	if err := t.DB.Where("service_id=? and label_key=? and label_value=?", serviceID, labelKey,
		labelValue).Delete(&label).Error; err != nil {
		return err
	}
	return nil
}

// DelTenantEnvServiceLabelsByServiceIDKey deletes labels by serviceID and labelKey
func (t *ServiceLabelDaoImpl) DelTenantEnvServiceLabelsByServiceIDKey(serviceID string, labelKey string) error {
	var label model.TenantEnvServiceLable
	if err := t.DB.Where("service_id=? and label_key=?", serviceID, labelKey).Delete(&label).Error; err != nil {
		return err
	}
	return nil
}

// DeleteByComponentIDs deletes labels based on componentIDs
func (t *ServiceLabelDaoImpl) DeleteByComponentIDs(componentIDs []string) error {
	return t.DB.Where("service_id in (?)", componentIDs).Delete(&model.TenantEnvServiceLable{}).Error
}

// CreateOrUpdateLabelsInBatch -
func (t *ServiceLabelDaoImpl) CreateOrUpdateLabelsInBatch(labels []*model.TenantEnvServiceLable) error {
	var objects []interface{}
	for _, label := range labels {
		objects = append(objects, *label)
	}
	if err := gormbulkups.BulkUpsert(t.DB, objects, 2000); err != nil {
		return pkgerr.Wrap(err, "create or update label in batch")
	}
	return nil
}

// TenantEnvServceAutoscalerRulesDaoImpl -
type TenantEnvServceAutoscalerRulesDaoImpl struct {
	DB *gorm.DB
}

// AddModel -
func (t *TenantEnvServceAutoscalerRulesDaoImpl) AddModel(mo model.Interface) error {
	rule := mo.(*model.TenantEnvServiceAutoscalerRules)
	var old model.TenantEnvServiceAutoscalerRules
	if ok := t.DB.Where("rule_id = ?", rule.RuleID).Find(&old).RecordNotFound(); ok {
		if err := t.DB.Create(rule).Error; err != nil {
			return err
		}
	} else {
		return errors.ErrRecordAlreadyExist
	}
	return nil
}

// UpdateModel -
func (t *TenantEnvServceAutoscalerRulesDaoImpl) UpdateModel(mo model.Interface) error {
	rule := mo.(*model.TenantEnvServiceAutoscalerRules)
	if err := t.DB.Save(rule).Error; err != nil {
		return err
	}
	return nil
}

// GetByRuleID -
func (t *TenantEnvServceAutoscalerRulesDaoImpl) GetByRuleID(ruleID string) (*model.TenantEnvServiceAutoscalerRules, error) {
	var rule model.TenantEnvServiceAutoscalerRules
	if err := t.DB.Where("rule_id=?", ruleID).Find(&rule).Error; err != nil {
		return nil, err
	}
	return &rule, nil
}

// ListByServiceID -
func (t *TenantEnvServceAutoscalerRulesDaoImpl) ListByServiceID(serviceID string) ([]*model.TenantEnvServiceAutoscalerRules, error) {
	var rules []*model.TenantEnvServiceAutoscalerRules
	if err := t.DB.Where("service_id=?", serviceID).Find(&rules).Error; err != nil {
		return nil, err
	}
	return rules, nil
}

// ListEnableOnesByServiceID -
func (t *TenantEnvServceAutoscalerRulesDaoImpl) ListEnableOnesByServiceID(serviceID string) ([]*model.TenantEnvServiceAutoscalerRules, error) {
	var rules []*model.TenantEnvServiceAutoscalerRules
	if err := t.DB.Where("service_id=? and enable=?", serviceID, true).Find(&rules).Error; err != nil {
		return nil, err
	}
	return rules, nil
}

// ListByComponentIDs -
func (t *TenantEnvServceAutoscalerRulesDaoImpl) ListByComponentIDs(componentIDs []string) ([]*model.TenantEnvServiceAutoscalerRules, error) {
	var rules []*model.TenantEnvServiceAutoscalerRules
	if err := t.DB.Where("service_id in (?)", componentIDs).Find(&rules).Error; err != nil {
		return nil, err
	}
	return rules, nil
}

// DeleteByComponentIDs deletes rule based on componentIDs
func (t *TenantEnvServceAutoscalerRulesDaoImpl) DeleteByComponentIDs(componentIDs []string) error {
	return t.DB.Where("service_id in (?)", componentIDs).Delete(&model.TenantEnvServiceAutoscalerRules{}).Error
}

// CreateOrUpdateScaleRulesInBatch -
func (t *TenantEnvServceAutoscalerRulesDaoImpl) CreateOrUpdateScaleRulesInBatch(rules []*model.TenantEnvServiceAutoscalerRules) error {
	var objects []interface{}
	for _, rule := range rules {
		objects = append(objects, *rule)
	}
	if err := gormbulkups.BulkUpsert(t.DB, objects, 2000); err != nil {
		return pkgerr.Wrap(err, "create or update scale rule in batch")
	}
	return nil
}

// TenantEnvServceAutoscalerRuleMetricsDaoImpl -
type TenantEnvServceAutoscalerRuleMetricsDaoImpl struct {
	DB *gorm.DB
}

// AddModel -
func (t *TenantEnvServceAutoscalerRuleMetricsDaoImpl) AddModel(mo model.Interface) error {
	metric := mo.(*model.TenantEnvServiceAutoscalerRuleMetrics)
	var old model.TenantEnvServiceAutoscalerRuleMetrics
	if ok := t.DB.Where("rule_id=? and metric_type=? and metric_name=?", metric.RuleID, metric.MetricsType, metric.MetricsName).Find(&old).RecordNotFound(); ok {
		if err := t.DB.Create(metric).Error; err != nil {
			return err
		}
	} else {
		return errors.ErrRecordAlreadyExist
	}
	return nil
}

// UpdateModel -
func (t *TenantEnvServceAutoscalerRuleMetricsDaoImpl) UpdateModel(mo model.Interface) error {
	metric := mo.(*model.TenantEnvServiceAutoscalerRuleMetrics)
	if err := t.DB.Save(metric).Error; err != nil {
		return err
	}
	return nil
}

// UpdateOrCreate -
func (t *TenantEnvServceAutoscalerRuleMetricsDaoImpl) UpdateOrCreate(metric *model.TenantEnvServiceAutoscalerRuleMetrics) error {
	var old model.TenantEnvServiceAutoscalerRuleMetrics
	if ok := t.DB.Where("rule_id=? and metric_type=? and metric_name=?", metric.RuleID, metric.MetricsType, metric.MetricsName).Find(&old).RecordNotFound(); ok {
		if err := t.DB.Create(metric).Error; err != nil {
			return err
		}
	} else {
		old.MetricTargetType = metric.MetricTargetType
		old.MetricTargetValue = metric.MetricTargetValue
		if err := t.DB.Save(&old).Error; err != nil {
			return err
		}
	}
	return nil
}

// ListByRuleID -
func (t *TenantEnvServceAutoscalerRuleMetricsDaoImpl) ListByRuleID(ruleID string) ([]*model.TenantEnvServiceAutoscalerRuleMetrics, error) {
	var metrics []*model.TenantEnvServiceAutoscalerRuleMetrics
	if err := t.DB.Where("rule_id=?", ruleID).Find(&metrics).Error; err != nil {
		return nil, err
	}
	return metrics, nil
}

// DeleteByRuleID -
func (t *TenantEnvServceAutoscalerRuleMetricsDaoImpl) DeleteByRuleID(ruldID string) error {
	if err := t.DB.Where("rule_id=?", ruldID).Delete(&model.TenantEnvServiceAutoscalerRuleMetrics{}).Error; err != nil {
		return err
	}

	return nil
}

// DeleteByRuleIDs deletes rule metrics based on componentIDs
func (t *TenantEnvServceAutoscalerRuleMetricsDaoImpl) DeleteByRuleIDs(ruleIDs []string) error {
	return t.DB.Where("rule_id in (?)", ruleIDs).Delete(&model.TenantEnvServiceAutoscalerRuleMetrics{}).Error
}

// CreateOrUpdateScaleRuleMetricsInBatch -
func (t *TenantEnvServceAutoscalerRuleMetricsDaoImpl) CreateOrUpdateScaleRuleMetricsInBatch(metrics []*model.TenantEnvServiceAutoscalerRuleMetrics) error {
	var objects []interface{}
	for _, metric := range metrics {
		objects = append(objects, *metric)
	}
	if err := gormbulkups.BulkUpsert(t.DB, objects, 2000); err != nil {
		return pkgerr.Wrap(err, "create or update rule metric in batch")
	}
	return nil
}

// TenantEnvServiceScalingRecordsDaoImpl -
type TenantEnvServiceScalingRecordsDaoImpl struct {
	DB *gorm.DB
}

// AddModel -
func (t *TenantEnvServiceScalingRecordsDaoImpl) AddModel(mo model.Interface) error {
	record := mo.(*model.TenantEnvServiceScalingRecords)
	var old model.TenantEnvServiceScalingRecords
	if ok := t.DB.Where("event_name=?", record.EventName).Find(&old).RecordNotFound(); ok {
		if err := t.DB.Create(record).Error; err != nil {
			return err
		}
	} else {
		return errors.ErrRecordAlreadyExist
	}
	return nil
}

// UpdateModel -
func (t *TenantEnvServiceScalingRecordsDaoImpl) UpdateModel(mo model.Interface) error {
	record := mo.(*model.TenantEnvServiceScalingRecords)
	if err := t.DB.Save(record).Error; err != nil {
		return err
	}
	return nil
}

// UpdateOrCreate -
func (t *TenantEnvServiceScalingRecordsDaoImpl) UpdateOrCreate(new *model.TenantEnvServiceScalingRecords) error {
	var old model.TenantEnvServiceScalingRecords

	if ok := t.DB.Where("event_name=?", new.EventName).Find(&old).RecordNotFound(); ok {
		return t.DB.Create(new).Error
	}

	old.Count = new.Count
	old.LastTime = new.LastTime
	return t.DB.Save(&old).Error
}

// ListByServiceID -
func (t *TenantEnvServiceScalingRecordsDaoImpl) ListByServiceID(serviceID string, offset, limit int) ([]*model.TenantEnvServiceScalingRecords, error) {
	var records []*model.TenantEnvServiceScalingRecords
	if err := t.DB.Where("service_id=?", serviceID).Offset(offset).Limit(limit).Order("last_time desc").Find(&records).Error; err != nil {
		return nil, err
	}

	return records, nil
}

// CountByServiceID -
func (t *TenantEnvServiceScalingRecordsDaoImpl) CountByServiceID(serviceID string) (int, error) {
	record := model.TenantEnvServiceScalingRecords{}
	var count int
	if err := t.DB.Table(record.TableName()).Where("service_id=?", serviceID).Count(&count).Error; err != nil {
		return 0, err
	}

	return count, nil
}
