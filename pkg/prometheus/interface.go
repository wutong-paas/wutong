// WUTONG, Application Management Platform
// Copyright (C) 2020-2020 Wutong Co., Ltd.

// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version. For any non-GPL usage of Wutong
// one or multiple Commercial Licenses authorized by Wutong Co., Ltd.
// must be obtained first.

// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.

// You should have received a copy of the GNU General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package prometheus

import "time"

// Interface monitor interface
type Interface interface {
	GetMetric(expr string, time time.Time) Metric
	GetMetricOverTime(expr string, start, end time.Time, step time.Duration) Metric
	GetMetadata(tenantEnvID string) []Metadata
	//TODO Query dimensions to be improved
	GetAppMetadata(tenantEnvID, appID string) []Metadata
	GetComponentMetadata(tenantEnvID, componentID string) []Metadata
	GetMetricLabelSet(expr string, start, end time.Time) []map[string]string
	GetEndpoint() string
}
