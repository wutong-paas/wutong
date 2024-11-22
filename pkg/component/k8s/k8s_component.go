package k8s

import (
	"context"

	"github.com/sirupsen/logrus"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	wutongv1alpha1 "github.com/wutong-paas/wutong-operator/api/v1alpha1"
	"github.com/wutong-paas/wutong/config/configs"
	"github.com/wutong-paas/wutong/pkg/generated/clientset/versioned"
	wutongscheme "github.com/wutong-paas/wutong/pkg/generated/clientset/versioned/scheme"
	k8sutil "github.com/wutong-paas/wutong/util/k8s"
	apiextclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	ctrl "sigs.k8s.io/controller-runtime"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

// Component -
type Component struct {
	RestConfig    *rest.Config
	Clientset     *kubernetes.Clientset
	DynamicClient dynamic.Interface
	WutongClient  versioned.Interface
	K8sClient     k8sclient.Client
	Mapper        meta.RESTMapper
	ApiExtClient  apiextclient.Interface
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

	k.DynamicClient, err = dynamic.NewForConfig(config)
	if err != nil {
		logrus.Errorf("create dynamic client failure: %v", err)
		return err
	}

	k.WutongClient = versioned.NewForConfigOrDie(config)
	k.ApiExtClient = apiextclient.NewForConfigOrDie(config)

	scheme := runtime.NewScheme()
	clientgoscheme.AddToScheme(scheme)
	wutongscheme.AddToScheme(scheme)
	wutongv1alpha1.AddToScheme(scheme)
	velerov1.AddToScheme(scheme)
	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))
	k.K8sClient, err = k8sclient.New(config, k8sclient.Options{
		Scheme: scheme,
	})
	if err != nil {
		logrus.Errorf("create k8s client failure: %v", err)
		return err
	}

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
