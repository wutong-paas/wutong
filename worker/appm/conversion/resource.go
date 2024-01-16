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

package conversion

import (
	"fmt"

	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func createResourcesBySetting(requestMemory, limitMemory, requestCPU, limitCPU int64, gpuType string, limitGPU int64) corev1.ResourceRequirements {
	limits := corev1.ResourceList{}
	request := corev1.ResourceList{}

	request[corev1.ResourceMemory] = *resource.NewQuantity(requestMemory*1024*1024, resource.BinarySI)
	if limitMemory > 0 {
		limits[corev1.ResourceMemory] = *resource.NewQuantity(limitMemory*1024*1024, resource.BinarySI)
	}

	request[corev1.ResourceCPU] = *resource.NewMilliQuantity(requestCPU, resource.DecimalSI)
	if limitCPU > 0 {
		limits[corev1.ResourceCPU] = *resource.NewMilliQuantity(limitCPU, resource.DecimalSI)
	}

	if gpuType != "" && limitGPU > 0 {
		gpuLimit, err := resource.ParseQuantity(fmt.Sprintf("%d", limitGPU))
		if err != nil {
			logrus.Errorf("gpu request is invalid")
		} else {
			limits[corev1.ResourceName(gpuType)] = gpuLimit
			limits[corev1.ResourceName(gpuType)] = gpuLimit
		}
	}

	return corev1.ResourceRequirements{
		Limits:   limits,
		Requests: request,
	}
}
