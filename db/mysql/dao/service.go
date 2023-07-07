package dao

import "github.com/wutong-paas/wutong/db/model"

func (t *TenantEnvServicesDaoImpl) ListByAppID(appID string) ([]*model.TenantEnvServices, error) {
	var services []*model.TenantEnvServices
	if err := t.DB.Where("app_id=?", appID).Find(&services).Error; err != nil {
		return nil, err
	}
	return services, nil
}
