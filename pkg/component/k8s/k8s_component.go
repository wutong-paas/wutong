package k8s

import (
	"context"

	"github.com/sirupsen/logrus"
	"github.com/wutong-paas/wutong/config/configs"
	"github.com/wutong-paas/wutong/pkg/generated/clientset/versioned"
	wutongscheme "github.com/wutong-paas/wutong/pkg/generated/clientset/versioned/scheme"
	k8sutil "github.com/wutong-paas/wutong/util/k8s"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"

	// "kubevirt.io/client-go/kubecli"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	// "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned/typed/apis/v1beta1"
	// gateway "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned/typed/apis/v1beta1"
	veleroversioned "github.com/vmware-tanzu/velero/pkg/generated/clientset/versioned"
	apiextclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
)

// Component -
type Component struct {
	RestConfig *rest.Config
	Clientset  *kubernetes.Clientset
	// GatewayClient *v1beta1.GatewayV1beta1Client
	DynamicClient dynamic.Interface

	WutongClient versioned.Interface
	K8sClient    k8sclient.Client
	// KubevirtCli    kubecli.KubevirtClient

	Mapper       meta.RESTMapper
	ApiExtClient apiextclient.Interface
	VeleroClient veleroversioned.Interface
}

var defaultK8sComponent *Component

// Client -
func Client() *Component {
	defaultK8sComponent = &Component{}
	return defaultK8sComponent
}

// Start -
func (k *Component) Start(ctx context.Context, cfg *configs.Config) error {
	logrus.Infof("init k8s client...")
	config, err := k8sutil.NewRestConfig(cfg.APIConfig.KubeConfigPath)
	k.RestConfig = config
	if err != nil {
		logrus.Errorf("create k8s config failure: %v", err)
		return err
	}
	k.Clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		logrus.Errorf("create k8s client failure: %v", err)
		return err
	}
	// k.GatewayClient, err = gateway.NewForConfig(config)
	// if err != nil {
	// 	logrus.Errorf("create gateway client failure: %v", err)
	// 	return err
	// }
	k.DynamicClient, err = dynamic.NewForConfig(config)
	if err != nil {
		logrus.Errorf("create dynamic client failure: %v", err)
		return err
	}

	k.WutongClient = versioned.NewForConfigOrDie(config)
	k.ApiExtClient = apiextclient.NewForConfigOrDie(config)
	k.VeleroClient = veleroversioned.NewForConfigOrDie(config)

	scheme := runtime.NewScheme()
	clientgoscheme.AddToScheme(scheme)
	wutongscheme.AddToScheme(scheme)
	k.K8sClient, err = k8sclient.New(config, k8sclient.Options{
		Scheme: scheme,
	})
	if err != nil {
		logrus.Errorf("create k8s client failure: %v", err)
		return err
	}

	// k.KubevirtCli, err = kubecli.GetKubevirtClientFromRESTConfig(config)
	// if err != nil {
	// 	logrus.Errorf("create kubevirt cli failure: %v", err)
	// 	return err
	// }

	gr, err := restmapper.GetAPIGroupResources(k.Clientset)
	if err != nil {
		return err
	}
	k.Mapper = restmapper.NewDiscoveryRESTMapper(gr)
	logrus.Infof("init k8s client success")
	return nil
}

// CloseHandle -
func (k *Component) CloseHandle() {
}

// Default -
func Default() *Component {
	return defaultK8sComponent
}
