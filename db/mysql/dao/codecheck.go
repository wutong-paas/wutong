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
	"github.com/wutong-paas/wutong/db/model"

	"github.com/jinzhu/gorm"
)

// AddModel AddModel
func (c *CodeCheckResultDaoImpl) AddModel(mo model.Interface) error {
	result := mo.(*model.CodeCheckResult)
	var oldResult model.CodeCheckResult
	if ok := c.DB.Where("service_id=?", result.ServiceID).Find(&oldResult).RecordNotFound(); ok {
		if err := c.DB.Create(result).Error; err != nil {
			return err
		}
	} else {
		update(result, &oldResult)
		if err := c.DB.Save(&oldResult).Error; err != nil {
			return err
		}
		return nil
	}
	return nil
}

// UpdateModel UpdateModel
func (c *CodeCheckResultDaoImpl) UpdateModel(mo model.Interface) error {
	result := mo.(*model.CodeCheckResult)
	var oldResult model.CodeCheckResult
	if ok := c.DB.Where("service_id=?", result.ServiceID).Find(&oldResult).RecordNotFound(); !ok {
		update(result, &oldResult)
		if err := c.DB.Save(&oldResult).Error; err != nil {
			return err
		}
	}
	return nil
}

// CodeCheckResultDaoImpl EventLogMessageDaoImpl
type CodeCheckResultDaoImpl struct {
	DB *gorm.DB
}

func update(target, old *model.CodeCheckResult) {
	//o,_:=json.Marshal(old)
	//t,_:=json.Marshal(target)
	//logrus.Infof("before update,stared is %s,target is ",string(o),string(t))
	if target.DockerFileReady != old.DockerFileReady {

		old.DockerFileReady = !old.DockerFileReady
	}
	if target.VolumeList != "" && target.VolumeList != "null" {
		old.VolumeList = target.VolumeList
	}
	if target.PortList != "" && target.PortList != "null" {
		old.PortList = target.PortList
	}
	if target.BuildImageName != "" {
		old.BuildImageName = target.BuildImageName
	}
	if target.VolumeMountPath != "" {
		old.VolumeMountPath = target.VolumeMountPath
	}
	if target.InnerPort != "" {
		old.InnerPort = target.InnerPort
	}
	//o2,_:=json.Marshal(old)
	//t2,_:=json.Marshal(target)
	//logrus.Infof("after update,%s,%s",string(o2),string(t2))
}

// GetCodeCheckResult get event log message
func (c *CodeCheckResultDaoImpl) GetCodeCheckResult(serviceID string) (*model.CodeCheckResult, error) {
	var result model.CodeCheckResult
	if err := c.DB.Where("service_id=?", serviceID).Find(&result).Error; err != nil {
		return nil, err
	}
	return &result, nil
}

// DeleteByServiceID deletes a CodeCheckResult base on serviceID.
func (c *CodeCheckResultDaoImpl) DeleteByServiceID(serviceID string) error {
	return c.DB.Where("service_id=?", serviceID).Delete(&model.CodeCheckResult{}).Error
}
