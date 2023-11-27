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

package v1

import (
	"github.com/wutong-paas/wutong/db"
)

// GetCommonLabels get common labels
func (a *AppService) GetCommonLabels(labels ...map[string]string) map[string]string {
	var resultLabel = make(map[string]string)
	for _, l := range labels {
		for k, v := range l {
			resultLabel[k] = v
		}
	}
	var tenantID, tenantName string
	tenantEnv, err := db.GetManager().TenantEnvDao().GetTenantEnvByUUID(a.TenantEnvID)
	if err == nil && tenantEnv != nil {
		tenantID = tenantEnv.TenantID
		tenantName = tenantEnv.TenantName
	}

	resultLabel["creator"] = "Wutong"
	resultLabel["creater_id"] = a.CreaterID
	resultLabel["service_id"] = a.ServiceID
	resultLabel["workload_name"] = a.GetK8sWorkloadName()
	resultLabel["service_alias"] = a.ServiceAlias
	resultLabel["tenant_id"] = tenantID
	resultLabel["tenant_name"] = tenantName
	resultLabel["tenant_env_id"] = a.TenantEnvID
	resultLabel["tenant_env_name"] = a.TenantEnvName
	resultLabel["app_id"] = a.AppID
	resultLabel["app"] = a.K8sApp
	return resultLabel
}
