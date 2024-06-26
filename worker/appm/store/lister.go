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
	"github.com/wutong-paas/wutong/pkg/generated/listers/wutong/v1alpha1"
	crdlisters "k8s.io/apiextensions-apiserver/pkg/client/listers/apiextensions/v1"
	appsv1 "k8s.io/client-go/listers/apps/v1"
	autoscalingv1 "k8s.io/client-go/listers/autoscaling/v1"
	corev1 "k8s.io/client-go/listers/core/v1"
	networkingv1 "k8s.io/client-go/listers/networking/v1"
	betav1 "k8s.io/client-go/listers/networking/v1beta1"
	storagev1 "k8s.io/client-go/listers/storage/v1"
)

// Lister kube-api client cache
type Lister struct {
	Ingress                 networkingv1.IngressLister
	BetaIngress             betav1.IngressLister
	Service                 corev1.ServiceLister
	Secret                  corev1.SecretLister
	StatefulSet             appsv1.StatefulSetLister
	Deployment              appsv1.DeploymentLister
	Pod                     corev1.PodLister
	ReplicaSets             appsv1.ReplicaSetLister
	ConfigMap               corev1.ConfigMapLister
	Endpoints               corev1.EndpointsLister
	Nodes                   corev1.NodeLister
	StorageClass            storagev1.StorageClassLister
	Claims                  corev1.PersistentVolumeClaimLister
	HorizontalPodAutoscaler autoscalingv1.HorizontalPodAutoscalerLister
	CRD                     crdlisters.CustomResourceDefinitionLister
	HelmApp                 v1alpha1.HelmAppLister
	ComponentDefinition     v1alpha1.ComponentDefinitionLister
	ThirdComponent          v1alpha1.ThirdComponentLister
}
