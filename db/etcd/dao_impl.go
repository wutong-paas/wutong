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

package etcd

import (
	"github.com/wutong-paas/wutong/db/dao"
)

// TenantEnvDao  tenantEnvDao
func (m *Manager) TenantEnvDao() dao.TenantEnvDao {
	return nil
}

// TenantEnvServiceDao TenantEnvServiceDao
func (m *Manager) TenantEnvServiceDao() dao.TenantEnvServiceDao {
	return nil
}

// TenantEnvServicesPortDao TenantEnvServicesPortDao
func (m *Manager) TenantEnvServicesPortDao() dao.TenantEnvServicesPortDao {
	return nil
}

// TenantEnvServiceRelationDao TenantEnvServiceRelationDao
func (m *Manager) TenantEnvServiceRelationDao() dao.TenantEnvServiceRelationDao {
	return nil
}

// TenantEnvServiceEnvVarDao TenantEnvServiceEnvVarDao
func (m *Manager) TenantEnvServiceEnvVarDao() dao.TenantEnvServiceEnvVarDao {
	return nil
}

// TenantEnvServiceMountRelationDao TenantEnvServiceMountRelationDao
func (m *Manager) TenantEnvServiceMountRelationDao() dao.TenantEnvServiceMountRelationDao {
	return nil
}

// TenantEnvServiceVolumeDao TenantEnvServiceVolumeDao
func (m *Manager) TenantEnvServiceVolumeDao() dao.TenantEnvServiceVolumeDao {
	return nil
}

// func (m *Manager) K8sServiceDao() dao.K8sServiceDao {
// 	return nil
// }
// func (m *Manager) K8sDeployReplicationDao() dao.K8sDeployReplicationDao {
// 	return nil
// }
// func (m *Manager) K8sPodDao() dao.K8sPodDao {
// 	return nil
// }

// ServiceProbeDao ServiceProbeDao
func (m *Manager) ServiceProbeDao() dao.ServiceProbeDao {
	return nil
}
