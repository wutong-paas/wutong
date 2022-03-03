// Copyright (C) 2014-2018 Wutong Co., Ltd.
// WUTONG, Application Management Platform

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

package controller

import (
	"net/http"

	"github.com/wutong-paas/wutong/api/util/license"
	httputil "github.com/wutong-paas/wutong/util/http"
)

//LicenseManager license manager
type LicenseManager struct{}

var licenseManager *LicenseManager

//GetLicenseManager get license Manager
func GetLicenseManager() *LicenseManager {
	if licenseManager != nil {
		return licenseManager
	}
	licenseManager = &LicenseManager{}
	return licenseManager
}

func (l *LicenseManager) GetlicenseFeature(w http.ResponseWriter, r *http.Request) {
	//The following features are designed for license control.
	// GPU Support
	// Windows Container Support
	// Gateway security control
	features := []license.Feature{}
	if lic := license.ReadLicense(); lic != nil {
		features = lic.Features
	}
	httputil.ReturnSuccess(r, w, features)
}
