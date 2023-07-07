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

package model

import (
	"sort"
	"testing"
)

func TestTenantEnvList(t *testing.T) {
	var tenantEnvs TenantEnvList
	t1 := &TenantEnvAndResource{
		MemoryRequest: 100,
	}
	t1.LimitMemory = 30
	tenantEnvs.Add(t1)

	t2 := &TenantEnvAndResource{
		MemoryRequest: 80,
	}
	t2.LimitMemory = 40
	tenantEnvs.Add(t2)

	t3 := &TenantEnvAndResource{
		MemoryRequest: 0,
	}
	t3.LimitMemory = 60
	t4 := &TenantEnvAndResource{
		MemoryRequest: 0,
	}
	t4.LimitMemory = 70

	t5 := &TenantEnvAndResource{
		RunningAppNum: 10,
	}
	t5.LimitMemory = 0

	tenantEnvs.Add(t3)
	tenantEnvs.Add(t4)
	tenantEnvs.Add(t5)
	sort.Sort(tenantEnvs)
	for _, ten := range tenantEnvs {
		t.Logf("%+v", ten)
	}
}
