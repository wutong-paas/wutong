// Copyright (C) 2014-2018 Wutong Co., Ltd.
// WUTONG, Application Management Platform

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

package clients

import (
	"fmt"
	"os"
	"path"

	"github.com/sirupsen/logrus"
	wutongv1alpha1 "github.com/wutong-paas/wutong-operator/api/v1alpha1"
	"github.com/wutong-paas/wutong/chaos/sources"
	k8sutil "github.com/wutong-paas/wutong/util/k8s"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(wutongv1alpha1.AddToScheme(scheme))
	utilruntime.Must(apiextensionsv1.AddToScheme(scheme))
}

// K8SClient K8SClient
var K8SClient kubernetes.Interface

// WutongKubeClient wutong custom resource client
var WutongKubeClient client.Client

// InitClient init k8s client
func InitClient(kubeconfig string) error {
	if kubeconfig == "" {
		homePath, _ := sources.Home()
		kubeconfig = path.Join(homePath, ".kube/config")
	}
	var config *rest.Config
	_, err := os.Stat(kubeconfig)
	if err != nil {
		fmt.Printf("Please make sure the kube-config file(%s) exists\n", kubeconfig)
		if config, err = rest.InClusterConfig(); err != nil {
			logrus.Error("get cluster config error:", err)
			return err
		}
	} else {
		// use the current context in kubeconfig
		config, err = k8sutil.NewRestConfig(kubeconfig)
		if err != nil {
			return err
		}
	}
	config.QPS = 50
	config.Burst = 100

	K8SClient, err = kubernetes.NewForConfig(config)
	if err != nil {
		logrus.Error("Create kubernetes client error.", err.Error())
		return err
	}
	mapper, err := apiutil.NewDynamicRESTMapper(config, apiutil.WithLazyDiscovery)
	if err != nil {
		return fmt.Errorf("NewDynamicRESTMapper failure %+v", err)
	}
	runtimeClient, err := client.New(config, client.Options{Scheme: scheme, Mapper: mapper})
	if err != nil {
		return fmt.Errorf("new kube client failure %+v", err)
	}
	WutongKubeClient = runtimeClient
	return nil
}

func K8SClientInitClient(k8sClient kubernetes.Interface, config *rest.Config) error {
	mapper, err := apiutil.NewDynamicRESTMapper(config, apiutil.WithLazyDiscovery)
	if err != nil {
		return fmt.Errorf("NewDynamicRESTMapper failure %+v", err)
	}
	runtimeClient, err := client.New(config, client.Options{Scheme: scheme, Mapper: mapper})
	if err != nil {
		return fmt.Errorf("new kube client failure %+v", err)
	}
	WutongKubeClient = runtimeClient
	return nil
}
