package kube

import (
	"github.com/sirupsen/logrus"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	wutongversioned "github.com/wutong-paas/wutong/pkg/generated/clientset/versioned"
	wutongscheme "github.com/wutong-paas/wutong/pkg/generated/clientset/versioned/scheme"
	apiext "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	kubevirtclient "kubevirt.io/client-go/kubecli"
	controllerruntime "sigs.k8s.io/controller-runtime"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	restConfig     *rest.Config
	kubeClient     kubernetes.Interface
	apiExtClient   apiext.Interface
	wutongClient   wutongversioned.Interface
	kubevirtClient kubevirtclient.KubevirtClient
	dynamicClient  dynamic.Interface
	runtimeClient  runtimeclient.Client
)

func RESTConfig() *rest.Config {
	if restConfig == nil {
		restConfig = controllerruntime.GetConfigOrDie()
	}

	return restConfig
}

func KubeClient() kubernetes.Interface {
	if kubeClient == nil {
		kubeClient = kubernetes.NewForConfigOrDie(RESTConfig())
	}

	return kubeClient
}

func APIExtClient() apiext.Interface {
	if apiExtClient == nil {
		apiExtClient = apiext.NewForConfigOrDie(RESTConfig())
	}

	return apiExtClient
}

func WutongClient() wutongversioned.Interface {
	if wutongClient == nil {
		wutongClient = wutongversioned.NewForConfigOrDie(RESTConfig())
	}

	return wutongClient
}

func KubevirtClient() kubevirtclient.KubevirtClient {
	if kubevirtClient == nil {
		var err error
		kubevirtClient, err = kubevirtclient.GetKubevirtClientFromRESTConfig(RESTConfig())
		if err != nil {
			logrus.Errorf("failed to create kubevirt client: %v", err)
		}
	}
	return kubevirtClient
}

func DynamicClient() dynamic.Interface {
	if dynamicClient == nil {
		dynamicClient = dynamic.NewForConfigOrDie(RESTConfig())
	}

	return dynamicClient
}

func RuntimeClient() runtimeclient.Client {
	if runtimeClient == nil {
		// k8s runtime client
		scheme := runtime.NewScheme()
		clientgoscheme.AddToScheme(scheme)
		wutongscheme.AddToScheme(scheme)
		velerov1.AddToScheme(scheme)
		var err error
		runtimeClient, err = runtimeclient.New(RESTConfig(), runtimeclient.Options{
			Scheme: scheme,
		})
		if err != nil {
			panic("ERR: create k8s runtime client")
		}
	}

	return runtimeClient
}
