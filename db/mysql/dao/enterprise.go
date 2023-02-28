package dao

import (
	"github.com/jinzhu/gorm"
	"github.com/wutong-paas/wutong/db/model"
)

// EnterpriseDaoImpl 租户环境信息管理
type EnterpriseDaoImpl struct {
	DB *gorm.DB
}

// GetEnterpriseTenantEnvs -
func (e *EnterpriseDaoImpl) GetEnterpriseTenantEnvs(enterpriseID string) ([]*model.TenantEnvs, error) {
	var tenantEnvs []*model.TenantEnvs
	if enterpriseID == "" {
		return []*model.TenantEnvs{}, nil
	}
	if err := e.DB.Where("eid= ?", enterpriseID).Find(&tenantEnvs).Error; err != nil {
		return nil, err
	}
	return tenantEnvs, nil
}
