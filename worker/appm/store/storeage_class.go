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

package store

import (
	"context"

	"github.com/sirupsen/logrus"
	v1 "github.com/wutong-paas/wutong/worker/appm/types/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// InitStorageclass init storage class
func (a *appRuntimeStore) initStorageclass() error {
	for _, storageclass := range v1.GetInitStorageClass() {
		old, err := a.conf.KubeClient.StorageV1().StorageClasses().Get(context.Background(), storageclass.Name, metav1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				_, err = a.conf.KubeClient.StorageV1().StorageClasses().Create(context.Background(), storageclass, metav1.CreateOptions{})
			}
			if err != nil {
				return err
			}
			logrus.Infof("create storageclass %s", storageclass.Name)
		} else {
			update := false
			if old.VolumeBindingMode == nil {
				update = true
			}
			if !update && string(*old.VolumeBindingMode) != string(*storageclass.VolumeBindingMode) {
				update = true
			}
			if update {
				err := a.conf.KubeClient.StorageV1().StorageClasses().Delete(context.Background(), storageclass.Name, metav1.DeleteOptions{})
				if err == nil {
					_, err := a.conf.KubeClient.StorageV1().StorageClasses().Create(context.Background(), storageclass, metav1.CreateOptions{})
					if err != nil {
						logrus.Errorf("recreate strageclass %s failure %s", storageclass.Name, err.Error())
					}
					logrus.Infof("update storageclass %s success", storageclass.Name)
				} else {
					logrus.Errorf("recreate strageclass %s failure %s", storageclass.Name, err.Error())
				}
			}
		}
	}
	return nil
}
