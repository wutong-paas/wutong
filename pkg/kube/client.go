package kube

import (
	veleroversioned "github.com/vmware-tanzu/velero/pkg/generated/clientset/versioned"
	wutongversioned "github.com/wutong-paas/wutong/pkg/generated/clientset/versioned"
	wutongscheme "github.com/wutong-paas/wutong/pkg/generated/clientset/versioned/scheme"
	apiext "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	controllerruntime "sigs.k8s.io/controller-runtime"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	regionRESTConfig      *rest.Config
	regionClientset       kubernetes.Interface
	regionAPIExtClientset apiext.Interface
	regionWutongClientset wutongversioned.Interface
	regionVeleroClientset veleroversioned.Interface
	regionDynamicClient   dynamic.Interface
	regionRuntimeClient   runtimeclient.Client
)

func RegionRESTConfig() *rest.Config {
	if regionRESTConfig == nil {
		regionRESTConfig = controllerruntime.GetConfigOrDie()
	}

	return regionRESTConfig
}

func RegionClientset() kubernetes.Interface {
	if regionClientset == nil {
		regionClientset = kubernetes.NewForConfigOrDie(RegionRESTConfig())
	}

	return regionClientset
}

func RegionAPIExtClientset() apiext.Interface {
	if regionAPIExtClientset == nil {
		regionAPIExtClientset = apiext.NewForConfigOrDie(RegionRESTConfig())
	}

	return regionAPIExtClientset
}

func RegionWutongClientset() wutongversioned.Interface {
	if regionWutongClientset == nil {
		regionWutongClientset = wutongversioned.NewForConfigOrDie(RegionRESTConfig())
	}

	return regionWutongClientset
}

func RegionVeleroClientset() veleroversioned.Interface {
	if regionVeleroClientset == nil {
		regionVeleroClientset = veleroversioned.NewForConfigOrDie(RegionRESTConfig())
	}

	return regionVeleroClientset
}

func RegionDynamicClient() dynamic.Interface {
	if regionDynamicClient == nil {
		regionDynamicClient = dynamic.NewForConfigOrDie(RegionRESTConfig())
	}

	return regionDynamicClient
}

func RegionRuntimeClient() runtimeclient.Client {
	if regionRuntimeClient == nil {
		// k8s runtime client
		scheme := runtime.NewScheme()
		clientgoscheme.AddToScheme(scheme)
		wutongscheme.AddToScheme(scheme)
		var err error
		regionRuntimeClient, err = runtimeclient.New(RegionRESTConfig(), runtimeclient.Options{
			Scheme: scheme,
		})
		if err != nil {
			panic("ERR: create k8s runtime client")
		}
	}

	return regionRuntimeClient
}
