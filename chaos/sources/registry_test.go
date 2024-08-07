// WUTONG, Application Management Platform
// Copyright (C) 2014-2019 Wutong Co., Ltd.

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

package sources

import "testing"

func TestPublicImageExist(t *testing.T) {
	exist, err := ImageExist("barnett/nextcloud-runtime:0.2", "", "")
	if err != nil {
		t.Fail()
	}
	if exist {
		t.Log("image exist")
	}
}

func TestPrivateImageExist(t *testing.T) {
	exist, err := ImageExist("harbor.smartqi.cn:80/library/nginx:1.11", "admin", "Harbor12345")
	if err != nil {
		t.Fatal(err)
	}
	if exist {
		t.Log("image exist")
	}
}
