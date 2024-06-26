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

package provider

import (
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/wutong-paas/wutong/db"
	"github.com/wutong-paas/wutong/util"
	"github.com/wutong-paas/wutong/worker/master/volumes/provider/lib/controller"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type wutongssscProvisioner struct {
	// The directory to create PV-backing directories in
	pvDir string
	name  string
}

// NewWutongssscProvisioner creates a new Wutong statefulset share volume provisioner
func NewWutongssscProvisioner() controller.Provisioner {
	sharePath := os.Getenv("SHARE_DATA_PATH")
	if sharePath == "" {
		sharePath = "/wtdata"
	}
	if os.Getenv("ALLINONE_MODE") == "true" {
		return &wutongssscProvisioner{
			pvDir: sharePath,
			name:  "rancher.io/local-path",
		}
	}
	return &wutongssscProvisioner{
		pvDir: sharePath,
		name:  "wutong.io/provisioner-sssc",
	}
}

var _ controller.Provisioner = &wutongssscProvisioner{}

// Provision creates a storage asset and returns a PV object representing it.
func (p *wutongssscProvisioner) Provision(options controller.VolumeOptions) (*v1.PersistentVolume, error) {
	logrus.Debugf("[wutongssscProvisioner] start creating PV object. paramters: %+v", options.Parameters)

	tenantEnvID := options.PVC.Labels["tenant_env_id"]
	serviceID := options.PVC.Labels["service_id"]
	_, stateless := options.PVC.Labels["stateless"]
	// v5.0.4 Previous versions
	hostpath := path.Join(p.pvDir, "tenant_env", tenantEnvID, "service", serviceID, options.PVC.Name)

	// after v5.0.4,change host path
	// Directory path has nothing to do with volume ID
	// Directory path bound to volume mount path
	if util.DirIsEmpty(hostpath) {
		podName := getPodNameByPVCName(options.PVC.Name)
		volumeID := getVolumeIDByPVCName(options.PVC.Name)
		if volumeID != 0 {
			volume, err := db.GetManager().TenantEnvServiceVolumeDao().GetVolumeByID(volumeID)
			if err != nil {
				logrus.Errorf("get volume by id %d failure %s", volumeID, err.Error())
				return nil, err
			}
			hostpath = volume.HostPath
			if !stateless {
				hostpath = path.Join(volume.HostPath, podName)
			}
		} else {
			return nil, fmt.Errorf("can not parse volume id")
		}
	}
	if err := util.CheckAndCreateDirByMode(hostpath, 0777); err != nil {
		return nil, err
	}
	// new volume path
	persistentVolumeSource, err := updatePathForPersistentVolumeSource(&options.PersistentVolumeSource, hostpath)
	if err != nil {
		return nil, err
	}

	pv := &v1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name:   options.PVName,
			Labels: options.PVC.Labels,
		},
		Spec: v1.PersistentVolumeSpec{
			PersistentVolumeReclaimPolicy: options.PersistentVolumeReclaimPolicy,
			AccessModes:                   options.PVC.Spec.AccessModes,
			Capacity: v1.ResourceList{
				v1.ResourceStorage: options.PVC.Spec.Resources.Requests[v1.ResourceStorage],
			},
			MountOptions:           options.MountOptions,
			PersistentVolumeSource: *persistentVolumeSource,
		},
	}
	logrus.Infof("create wutongsssc pv %s for pvc %s", pv.Name, options.PVC.Name)
	return pv, nil
}

// Delete removes the storage asset that was created by Provision represented
// by the given PV.
func (p *wutongssscProvisioner) Delete(volume *v1.PersistentVolume) error {

	return nil
}

func (p *wutongssscProvisioner) Name() string {
	return p.name
}

func getPodNameByPVCName(pvcName string) string {
	pvcNames := strings.SplitN(pvcName, "-", 2)
	if len(pvcNames) == 2 {
		return pvcNames[1]
	}
	return pvcName
}

func getVolumeIDByPVCName(pvcName string) int {
	logrus.Debug("parse volume id from pvc name", pvcName)
	pvcNames := strings.SplitN(pvcName, "-", 2)
	if len(pvcNames) == 2 {
		idStr := pvcNames[0][6:]
		id, _ := strconv.Atoi(idStr)
		return id
	}
	if strings.HasPrefix(pvcName, "manual") {
		idStr := strings.TrimPrefix(pvcName, "manual")
		id, _ := strconv.Atoi(idStr)
		return id
	}
	return 0
}

func updatePathForPersistentVolumeSource(persistentVolumeSource *v1.PersistentVolumeSource, hostpath string) (*v1.PersistentVolumeSource, error) {
	newPath := func(new string) string {
		p := strings.Replace(hostpath, "/wtdata", "", 1)
		return path.Join(new, p)
	}
	source := &v1.PersistentVolumeSource{}
	switch {
	case persistentVolumeSource.NFS != nil:
		source.NFS = persistentVolumeSource.NFS
		source.NFS.Path = newPath(persistentVolumeSource.NFS.Path)
	case persistentVolumeSource.CSI != nil && persistentVolumeSource.CSI.Driver == "nasplugin.csi.alibabacloud.com":
		// convert aliyun nas to nfs
		if persistentVolumeSource.CSI.VolumeAttributes != nil {
			source.NFS = &v1.NFSVolumeSource{
				Server: persistentVolumeSource.CSI.VolumeAttributes["server"],
				Path:   newPath(persistentVolumeSource.CSI.VolumeAttributes["path"]),
			}
		}
	case persistentVolumeSource.Glusterfs != nil:
		//glusterfs:
		//	endpoints: glusterfs-cluster
		//	path: myVol1
		glusterfs := &v1.GlusterfsPersistentVolumeSource{
			EndpointsName:      persistentVolumeSource.Glusterfs.EndpointsName,
			EndpointsNamespace: persistentVolumeSource.Glusterfs.EndpointsNamespace,
			Path:               newPath(persistentVolumeSource.Glusterfs.Path),
		}
		source.Glusterfs = glusterfs
	case persistentVolumeSource.HostPath != nil:
		source.HostPath = &v1.HostPathVolumeSource{
			Path: newPath(persistentVolumeSource.HostPath.Path),
			Type: persistentVolumeSource.HostPath.Type,
		}
	default:
		return nil, fmt.Errorf("unsupported persistence volume source")
	}
	return source, nil
}
