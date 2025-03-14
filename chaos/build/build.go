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

package build

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/docker/docker/api/types/registry"
	"github.com/wutong-paas/wutong/chaos"
	"github.com/wutong-paas/wutong/chaos/parser/code"
	"github.com/wutong-paas/wutong/chaos/sources"
	"github.com/wutong-paas/wutong/db"
	"github.com/wutong-paas/wutong/event"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func init() {
	buildcreaters = make(map[code.Lang]CreaterBuild)
	buildcreaters[code.Dockerfile] = dockerfileBuilder
	buildcreaters[code.Docker] = dockerfileBuilder
	buildcreaters[code.NetCore] = netcoreBuilder
	buildcreaters[code.JavaJar] = slugBuilder
	buildcreaters[code.JavaMaven] = slugBuilder
	buildcreaters[code.JaveWar] = slugBuilder
	buildcreaters[code.PHP] = slugBuilder
	buildcreaters[code.Python] = slugBuilder
	buildcreaters[code.Nodejs] = slugBuilder
	buildcreaters[code.Golang] = slugBuilder
	buildcreaters[code.OSS] = slugBuilder
}

var buildcreaters map[code.Lang]CreaterBuild

// Build app build pack
type Build interface {
	Build(*Request) (*Response, error)
}

// CreaterBuild CreaterBuild
type CreaterBuild func() (Build, error)

// MediumType Build output medium type
type MediumType string

// ImageMediumType image type
var ImageMediumType MediumType = "image"

// SlugMediumType slug type
var SlugMediumType MediumType = "slug"

// ImageBuildHostNetworkMode image build host network mode
var ImageBuildHostNetworkMode = "host"

// Response build result
type Response struct {
	MediumPath string
	MediumType MediumType
}

// Request build input
type Request struct {
	KanikoImage   string
	WtNamespace   string
	WTDataPVCName string
	CachePVCName  string
	CacheMode     string
	CachePath     string
	TenantEnvID   string
	SourceDir     string
	CacheDir      string
	TGZDir        string
	RepositoryURL string
	CodeSouceInfo sources.CodeSourceInfo
	Branch        string
	ServiceAlias  string
	ServiceID     string
	DeployVersion string
	Runtime       string
	ServerType    string
	Commit        Commit
	Lang          code.Lang
	BuildEnvs     map[string]string
	Logger        event.Logger
	ImageClient   sources.ImageClient
	KubeClient    kubernetes.Interface
	ExtraHosts    []string
	HostAlias     []HostAlias
	Ctx           context.Context
}

// HostAlias holds the mapping between IP and hostnames that will be injected as an entry in the
// pod's hosts file.
type HostAlias struct {
	// IP address of the host file entry.
	IP string `json:"ip,omitempty" protobuf:"bytes,1,opt,name=ip"`
	// Hostnames for the above IP address.
	Hostnames []string `json:"hostnames,omitempty" protobuf:"bytes,2,rep,name=hostnames"`
}

// Commit Commit
type Commit struct {
	User    string
	Message string
	Hash    string
}

// GetBuild GetBuild
func GetBuild(lang code.Lang) (Build, error) {
	if fun, ok := buildcreaters[lang]; ok {
		return fun()
	}
	return slugBuilder()
}

// CreateImageName create image name
func CreateImageName(serviceID, deployversion string) string {
	imageName := strings.ToLower(fmt.Sprintf("%s/%s:%s", chaos.REGISTRYDOMAIN, serviceID, deployversion))
	component, err := db.GetManager().TenantEnvServiceDao().GetServiceByID(serviceID)
	if err != nil {
		return imageName
	}
	app, err := db.GetManager().ApplicationDao().GetByServiceID(serviceID)
	if err != nil {
		return imageName
	}
	tenantEnv, err := db.GetManager().TenantEnvDao().GetTenantEnvByUUID(component.TenantEnvID)
	if err != nil {
		return imageName
	}
	workloadName := fmt.Sprintf("%s-%s-%s", tenantEnv.Namespace, app.K8sApp, component.K8sComponentName)
	return strings.ToLower(fmt.Sprintf("%s/%s:%s", chaos.REGISTRYDOMAIN, workloadName, deployversion))
}

func GetTenantEnvRegistryAuthSecrets(kcli kubernetes.Interface, ctx context.Context, tenantEnvID string) map[string]registry.AuthConfig {
	auths := make(map[string]registry.AuthConfig)
	tenantEnv, err := db.GetManager().TenantEnvDao().GetTenantEnvByUUID(tenantEnvID)
	if err != nil {
		return auths
	}
	registrySecrets, err := kcli.CoreV1().Secrets(tenantEnv.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: "wutong.io/registry-auth-secret=true",
	})
	if err == nil {
		for _, secret := range registrySecrets.Items {
			d := string(secret.Data["Domain"])
			u := string(secret.Data["Username"])
			p := string(secret.Data["Password"])
			auths[d] = registry.AuthConfig{
				Username: u,
				Password: p,
				Auth:     base64.StdEncoding.EncodeToString([]byte(u + ":" + p)),
			}
		}
	}
	return auths
}
