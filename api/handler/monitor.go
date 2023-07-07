// WUTONG, Application Management Platform
// Copyright (C) 2020-2020 Wutong Co., Ltd.

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

package handler

import "github.com/wutong-paas/wutong/api/client/prometheus"

// MonitorHandler monitor api handler
type MonitorHandler interface {
	GetTenantEnvMonitorMetrics(tenantEnvID string) []prometheus.Metadata
	GetAppMonitorMetrics(tenantEnvID, appID string) []prometheus.Metadata
	GetComponentMonitorMetrics(tenantEnvID, componentID string) []prometheus.Metadata
}

// NewMonitorHandler new monitor handler
func NewMonitorHandler(cli prometheus.Interface) MonitorHandler {
	return &monitorHandler{cli: cli}
}

type monitorHandler struct {
	cli prometheus.Interface
}

func (m *monitorHandler) GetTenantEnvMonitorMetrics(tenantEnvID string) []prometheus.Metadata {
	return m.cli.GetMetadata(tenantEnvID)
}

func (m *monitorHandler) GetAppMonitorMetrics(tenantEnvID, appID string) []prometheus.Metadata {
	return m.cli.GetAppMetadata(tenantEnvID, appID)
}

func (m *monitorHandler) GetComponentMonitorMetrics(tenantEnvID, componentID string) []prometheus.Metadata {
	return m.cli.GetComponentMetadata(tenantEnvID, componentID)
}
