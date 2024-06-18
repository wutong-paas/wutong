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

package chaos

import (
	"fmt"
	"os"
	"path"
	"runtime"
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
	CIVERSION = "latest"
	if runtime.GOARCH != "amd64" {
		CIVERSION = fmt.Sprintf("%s-arm64", CIVERSION)
	}
	if os.Getenv("CI_VERSION") != "" {
		CIVERSION = os.Getenv("CI_VERSION")
	}

	RUNNERIMAGENAME = fmt.Sprintf("%s:%s", path.Join(constants.WutongOnlineImageRepository, "runner"), "latest")
	BUILDERIMAGENAME = fmt.Sprintf("%s:%s", path.Join(constants.WutongOnlineImageRepository, "builder"), "latest")
	PROBEMESHIMAGENAME = fmt.Sprintf("%s:%s", path.Join(constants.WutongOnlineImageRepository, "wt-init-probe"), CIVERSION)
	TCPMESHIMAGENAME = fmt.Sprintf("%s:%s", path.Join(constants.WutongOnlineImageRepository, "wt-mesh-data-panel"), CIVERSION)
	NODESHELLIMAGENAME = fmt.Sprintf("%s:%s", path.Join(constants.WutongOnlineImageRepository, "node-shell"), "stable")
	WTCHANNELIMAGENAME = fmt.Sprintf("%s:%s", path.Join(constants.WutongOnlineImageRepository, "wt-channel"), "stable")
	VIRTVNCIMAGENAME = fmt.Sprintf("%s:%s", path.Join(constants.WutongOnlineImageRepository, "virt-vnc"), "stable")
	VIRTIOCONTAINERDISKIMAGENAME = fmt.Sprintf("%s:%s", path.Join(constants.WutongOnlineImageRepository, "virtio-container-disk"), "v1.3.0-alpha.0")
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
var REGISTRYDOMAIN = constants.WutongHubImageRepository

// REGISTRYUSER REGISTRY USER NAME
var REGISTRYUSER = ""

// REGISTRYPASS REGISTRY PASSWORD
var REGISTRYPASS = ""

// RUNNERIMAGENAME runner image name
var RUNNERIMAGENAME string

// BUILDERIMAGENAME builder image name
var BUILDERIMAGENAME string

// PROBEMESHIMAGENAME probemesh image name
var PROBEMESHIMAGENAME string

// TCPMESHIMAGENAME tcpmesh image name
var TCPMESHIMAGENAME string

// NODESHELLIMAGENAME nodeshell image name
var NODESHELLIMAGENAME string

// WTCHANNELIMAGENAME wt-channel image name
var WTCHANNELIMAGENAME string

// VIRTVNCIMAGENAME virt-vnc image name
var VIRTVNCIMAGENAME string

// VIRTIOCONTAINERDISKIMAGENAME virtio-container-disk image name
var VIRTIOCONTAINERDISKIMAGENAME string

// CIVERSION -
var CIVERSION string
