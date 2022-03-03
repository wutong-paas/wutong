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

import "testing"

func TestMeta_Equals(t *testing.T) {
	m := newFakeMeta()
	c := newFakeMeta()

	if !m.Equals(&c) {
		t.Errorf("m should equal c")
	}
}

func newFakeMeta() Meta {
	return Meta{
		Index:      888,
		Name:       "foo-meta",
		Namespace:  "ns",
		PluginName: "Nginx",
	}
}
