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

	"github.com/sirupsen/logrus"
	"github.com/wutong-paas/wutong/db/model"

	"github.com/jinzhu/gorm"
)

// VolumeTypeDaoImpl license model 管理
type VolumeTypeDaoImpl struct {
	DB *gorm.DB
}

// CreateOrUpdateVolumeType find or create volumeType, !!! attention：just for store sync storageclass from k8s
func (vtd *VolumeTypeDaoImpl) CreateOrUpdateVolumeType(vt *model.TenantEnvServiceVolumeType) (*model.TenantEnvServiceVolumeType, error) {
	if vt.VolumeType == model.ShareFileVolumeType.String() || vt.VolumeType == model.LocalVolumeType.String() || vt.VolumeType == model.MemoryFSVolumeType.String() {
		return vt, nil
	}
	volumeType, err := vtd.GetVolumeTypeByType(vt.VolumeType)
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	if err == gorm.ErrRecordNotFound || volumeType == nil {
		logrus.Debugf("volume type[%s] do not exists, create it", vt.VolumeType)
		err = vtd.AddModel(vt)
	} else {
		logrus.Debugf("volume type[%s] already exists, update it", vt.VolumeType)
		volumeType.Provisioner = vt.Provisioner
		volumeType.StorageClassDetail = vt.StorageClassDetail
		volumeType.NameShow = vt.NameShow
		err = vtd.UpdateModel(volumeType)
	}
	return volumeType, err
}

// AddModel AddModel
func (vtd *VolumeTypeDaoImpl) AddModel(mo model.Interface) error {
	volumeType := mo.(*model.TenantEnvServiceVolumeType)
	var oldVolumeType model.TenantEnvServiceVolumeType
	if ok := vtd.DB.Where("volume_type=?", volumeType.VolumeType).Find(&oldVolumeType).RecordNotFound(); ok {
		if err := vtd.DB.Create(volumeType).Error; err != nil {
			return err
		}
	} else {
		return fmt.Errorf("volumeType is exist")
	}
	return nil
}

// UpdateModel update model
func (vtd *VolumeTypeDaoImpl) UpdateModel(mo model.Interface) error {
	volumeType := mo.(*model.TenantEnvServiceVolumeType)
	if err := vtd.DB.Save(volumeType).Error; err != nil {
		return err
	}
	return nil
}

// GetAllVolumeTypes get all volumeTypes
func (vtd *VolumeTypeDaoImpl) GetAllVolumeTypes() ([]*model.TenantEnvServiceVolumeType, error) {
	var volumeTypes []*model.TenantEnvServiceVolumeType
	if err := vtd.DB.Find(&volumeTypes).Error; err != nil {
		return nil, err
	}
	return volumeTypes, nil
}

// GetAllVolumeTypesByPage get all volumeTypes by page
func (vtd *VolumeTypeDaoImpl) GetAllVolumeTypesByPage(page int, pageSize int) ([]*model.TenantEnvServiceVolumeType, error) {
	var volumeTypes []*model.TenantEnvServiceVolumeType
	if err := vtd.DB.Limit(pageSize).Offset((page - 1) * pageSize).Find(&volumeTypes).Error; err != nil {
		return nil, err
	}
	return volumeTypes, nil
}

// GetVolumeTypeByType get volume type by type
func (vtd *VolumeTypeDaoImpl) GetVolumeTypeByType(vt string) (*model.TenantEnvServiceVolumeType, error) {
	var volumeType model.TenantEnvServiceVolumeType
	if err := vtd.DB.Where("volume_type=?", vt).Find(&volumeType).Error; err != nil {
		return nil, err
	}
	return &volumeType, nil
}

// DeleteModelByVolumeTypes delete volume by type
func (vtd *VolumeTypeDaoImpl) DeleteModelByVolumeTypes(volumeType string) error {
	if err := vtd.DB.Where("volume_type=?", volumeType).Delete(&model.TenantEnvServiceVolumeType{}).Error; err != nil {
		return err
	}
	return nil
}
