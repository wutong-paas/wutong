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

package config

import (
	"fmt"
	"testing"
)

func TestResettingArray(t *testing.T) {
	c := CreateDataCenterConfig()
	c.Start()
	defer c.Stop()
	groupCtx := NewGroupContext("")
	groupCtx.Add("SADAS", "Test")
	result, err := ResettingArray(groupCtx, []string{"Sdd${sadas}asd", "${MYSQL_HOST}", "12_${MYSQL_PASS}_sd"})
	if err != nil {
		t.Fatal(err)
	}
	t.Log(result)
}

func TestResettingString(t *testing.T) {
	c := CreateDataCenterConfig()
	c.Start()
	defer c.Stop()
	groupCtx := NewGroupContext("")
	groupCtx.Add("SADAS", "Test")
	result, err := ResettingString(nil, "${MYSQL_HOST}Sdd${sadas}asd")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(result)
}

func TestGroupConfig(t *testing.T) {
	groupCtx := NewGroupContext("")
	v := groupCtx.Get("API")
	fmt.Println("asdasd:", v)
}
