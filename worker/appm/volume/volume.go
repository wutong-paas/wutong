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

package volume

import (
	"fmt"
	"path"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/wutong-paas/wutong/db"
	dbmodel "github.com/wutong-paas/wutong/db/model"
	v1 "github.com/wutong-paas/wutong/worker/appm/types/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Volume volume function interface
type Volume interface {
	CreateVolume(define *Define) error       // use serviceVolume
	CreateDependVolume(define *Define) error // use serviceMountR
	setBaseInfo(as *v1.AppService, serviceVolume *dbmodel.TenantEnvServiceVolume, serviceMountR *dbmodel.TenantEnvServiceMountRelation, version *dbmodel.VersionInfo, dbmanager db.Manager)
}

// NewVolumeManager create volume
func NewVolumeManager(as *v1.AppService,
	serviceVolume *dbmodel.TenantEnvServiceVolume,
	serviceMountR *dbmodel.TenantEnvServiceMountRelation,
	version *dbmodel.VersionInfo,
	envs []corev1.EnvVar,
	envVarSecrets []*corev1.Secret,
	dbmanager db.Manager) Volume {
	var v Volume
	volumeType := ""
	if serviceVolume != nil {
		volumeType = serviceVolume.VolumeType
	}
	if serviceMountR != nil {
		volumeType = serviceMountR.VolumeType
	}
	if volumeType == "" {
		logrus.Warn("unknown volume Type, can't create volume")
		return nil
	}
	switch volumeType {
	case dbmodel.ShareFileVolumeType.String():
		v = new(ShareFileVolume)
	case dbmodel.ConfigFileVolumeType.String():
		v = &ConfigFileVolume{envs: envs, envVarSecrets: envVarSecrets}
	case dbmodel.MemoryFSVolumeType.String():
		v = new(MemoryFSVolume)
	case dbmodel.LocalVolumeType.String():
		v = new(LocalVolume)
	default:
		logrus.Warnf("other volume type[%s]", volumeType)
		v = new(OtherVolume)
	}
	v.setBaseInfo(as, serviceVolume, serviceMountR, version, dbmanager)
	return v
}

// Base volume base
type Base struct {
	as        *v1.AppService
	svm       *dbmodel.TenantEnvServiceVolume
	smr       *dbmodel.TenantEnvServiceMountRelation
	version   *dbmodel.VersionInfo
	dbmanager db.Manager
}

func (b *Base) setBaseInfo(as *v1.AppService, serviceVolume *dbmodel.TenantEnvServiceVolume, serviceMountR *dbmodel.TenantEnvServiceMountRelation, version *dbmodel.VersionInfo, dbmanager db.Manager) {
	b.as = as
	b.svm = serviceVolume
	b.smr = serviceMountR
	b.version = version
	b.dbmanager = dbmanager
}

func newVolumeClaim(name, _, accessMode, storageClassName string, capacity int64, labels, annotations map[string]string) *corev1.PersistentVolumeClaim {
	logrus.Debugf("volume annotaion is %+v", annotations)
	if capacity == 0 {
		logrus.Warnf("claim[%s] capacity is 0, set 20G default", name)
		capacity = 20
	}
	resourceStorage, _ := resource.ParseQuantity(fmt.Sprintf("%dGi", capacity)) // 统一单位使用G
	return &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Labels:      labels,
			Annotations: annotations,
			Namespace:   "string",
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes:      []corev1.PersistentVolumeAccessMode{parseAccessMode(accessMode)},
			StorageClassName: &storageClassName,
			Resources: corev1.VolumeResourceRequirements{
				Requests: map[corev1.ResourceName]resource.Quantity{
					corev1.ResourceStorage: resourceStorage,
				},
			},
		},
	}
}

/*
RWO - ReadWriteOnce
ROX - ReadOnlyMany
RWX - ReadWriteMany
*/
func parseAccessMode(accessMode string) corev1.PersistentVolumeAccessMode {
	accessMode = strings.ToUpper(accessMode)
	switch accessMode {
	case "RWO":
		return corev1.ReadWriteOnce
	case "ROX":
		return corev1.ReadOnlyMany
	case "RWX":
		return corev1.ReadWriteMany
	default:
		return corev1.ReadWriteOnce
	}
}

// Define define volume
type Define struct {
	volumeMounts []corev1.VolumeMount
	volumes      []corev1.Volume
}

// GetVolumes get define volumes
func (v *Define) GetVolumes() []corev1.Volume {
	return v.volumes
}

// GetVolumeMounts get define volume mounts
func (v *Define) GetVolumeMounts() []corev1.VolumeMount {
	return v.volumeMounts
}

// SetVolume define set volume
func (v *Define) SetVolume(VolumeType dbmodel.VolumeType, name, mountPath, hostPath string, hostPathType corev1.HostPathType, readOnly bool) {
	for _, m := range v.volumeMounts {
		if m.MountPath == mountPath {
			return
		}
	}
	switch VolumeType {
	case dbmodel.MemoryFSVolumeType:
		vo := corev1.Volume{Name: name}
		// V5.2 do not use memory as medium of emptyDir
		vo.EmptyDir = &corev1.EmptyDirVolumeSource{}
		v.volumes = append(v.volumes, vo)
		if mountPath != "" {
			vm := corev1.VolumeMount{
				MountPath: mountPath,
				Name:      name,
				ReadOnly:  readOnly,
				SubPath:   "",
			}
			v.volumeMounts = append(v.volumeMounts, vm)
		}
	case dbmodel.ShareFileVolumeType:
		if hostPath != "" {
			vo := corev1.Volume{
				Name: name,
			}
			vo.HostPath = &corev1.HostPathVolumeSource{
				Path: hostPath,
				Type: &hostPathType,
			}
			v.volumes = append(v.volumes, vo)
			if mountPath != "" {
				vm := corev1.VolumeMount{
					MountPath: mountPath,
					Name:      name,
					ReadOnly:  readOnly,
					SubPath:   "",
				}
				v.volumeMounts = append(v.volumeMounts, vm)
			}
		}
	case dbmodel.LocalVolumeType:
		//no support
		return
	}
}

// SetVolumeCMap sets volumes and volumeMounts. The type of volumes is configMap.
func (v *Define) SetVolumeCMap(cmap *corev1.ConfigMap, k, p string, isReadOnly bool, mode *int32) {
	hasSubPath := true
	if strings.HasSuffix(p, "/") {
		hasSubPath = false
	}

	vm := corev1.VolumeMount{
		MountPath: p,
		Name:      cmap.Name,
		ReadOnly:  false,
	}

	var defaultMode int32 = 0777
	if mode != nil {
		// convert int to octal
		octal, _ := strconv.ParseInt(strconv.Itoa(int(*mode)), 8, 64)
		defaultMode = int32(octal)
	}
	vo := corev1.Volume{
		Name: cmap.Name,
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: cmap.Name,
				},
				DefaultMode: &defaultMode,
			},
		},
	}

	if hasSubPath {
		vm.SubPath = path.Base(p)

		vo.VolumeSource.ConfigMap.Items = []corev1.KeyToPath{
			{
				Key:  k,
				Path: path.Base(p), // subpath
				Mode: &defaultMode,
			},
		}
	}

	v.volumeMounts = append(v.volumeMounts, vm)
	v.volumes = append(v.volumes, vo)
}

// RewriteHostPathInWindows rewrite host path
func RewriteHostPathInWindows(hostPath string) string {
	hostPath = strings.Replace(hostPath, "/wtdata", `z:`, 1)
	hostPath = strings.Replace(hostPath, "/", `\`, -1)
	return hostPath
}

// RewriteContainerPathInWindows mount path in windows
func RewriteContainerPathInWindows(mountPath string) string {
	if mountPath == "" {
		return ""
	}
	if mountPath[0] == '/' {
		mountPath = `c:\` + mountPath[1:]
	}
	mountPath = strings.Replace(mountPath, "/", `\`, -1)
	return mountPath
}
