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

package builder

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/wutong-paas/wutong/util/constants"
)

func init() {
	if os.Getenv("BUILD_IMAGE_REPOSTORY_DOMAIN") != "" {
		REGISTRYDOMAIN = os.Getenv("BUILD_IMAGE_REPOSTORY_DOMAIN")
	}
	if os.Getenv("BUILD_IMAGE_REPOSTORY_USER") != "" {
		REGISTRYUSER = os.Getenv("BUILD_IMAGE_REPOSTORY_USER")
	}
	if os.Getenv("BUILD_IMAGE_REPOSTORY_PASS") != "" {
		REGISTRYPASS = os.Getenv("BUILD_IMAGE_REPOSTORY_PASS")
	}

	// set runner image name
	RUNNERIMAGENAME = "runner"
	if os.Getenv("RUNNER_IMAGE_NAME") != "" {
		RUNNERIMAGENAME = os.Getenv("RUNNER_IMAGE_NAME")
	}

	CIVERSION = "v1.0.0-stable"
	if os.Getenv("CI_VERSION") != "" {
		CIVERSION = os.Getenv("CI_VERSION")
	}

	RUNNERIMAGENAME = fmt.Sprintf("%s:%s", path.Join(REGISTRYDOMAIN, RUNNERIMAGENAME), CIVERSION)
	ONLINERUNNERIMAGENAME = fmt.Sprintf("%s:%s", path.Join(ONLINEREGISTRYDOMAIN, "runner"), CIVERSION)

	// set builder image name
	BUILDERIMAGENAME = "builder"
	if os.Getenv("BUILDER_IMAGE_NAME") != "" {
		BUILDERIMAGENAME = os.Getenv("BUILDER_IMAGE_NAME")
	}

	BUILDERIMAGENAME = fmt.Sprintf("%s:%s", path.Join(REGISTRYDOMAIN, BUILDERIMAGENAME), CIVERSION)
	ONLINEBUILDERIMAGENAME = fmt.Sprintf("%s:%s", path.Join(ONLINEREGISTRYDOMAIN, "builder"), CIVERSION)
}

// GetImageUserInfoV2 -
func GetImageUserInfoV2(domain, user, pass string) (string, string) {
	if user != "" && pass != "" {
		return user, pass
	}
	if strings.HasPrefix(domain, REGISTRYDOMAIN) {
		return REGISTRYUSER, REGISTRYPASS
	}
	return "", ""
}

// GetImageRepo -
func GetImageRepo(imageRepo string) string {
	if imageRepo == "" {
		return REGISTRYDOMAIN
	}
	return imageRepo
}

// REGISTRYDOMAIN REGISTRY_DOMAIN
var REGISTRYDOMAIN = constants.DefImageRepository

// REGISTRYUSER REGISTRY USER NAME
var REGISTRYUSER = ""

// REGISTRYPASS REGISTRY PASSWORD
var REGISTRYPASS = ""

// RUNNERIMAGENAME runner image name
var RUNNERIMAGENAME string

// BUILDERIMAGENAME builder image name
var BUILDERIMAGENAME string

// ONLINEREGISTRYDOMAIN online REGISTRY_DOMAIN
var ONLINEREGISTRYDOMAIN = constants.DefOnlineImageRepository

// ONLINEBUILDERIMAGENAME online builder image name
var ONLINEBUILDERIMAGENAME string

// ONLINERUNNERIMAGENAME online runner image name
var ONLINERUNNERIMAGENAME string

// CIVERSION -
var CIVERSION string
