package kube

import (
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	appsv1 "k8s.io/client-go/listers/apps/v1"
	autoscalingv1 "k8s.io/client-go/listers/autoscaling/v1"
	v1 "k8s.io/client-go/listers/core/v1"
	networkingv1 "k8s.io/client-go/listers/networking/v1"
	"k8s.io/client-go/tools/cache"
)

var wutongSelectorLabels = map[string]string{"creator": "Wutong"}

var cachedResources *CachedResources

type CachedResources struct {
	DeploymentLister   appsv1.DeploymentLister
	StatefuleSetLister appsv1.StatefulSetLister
	ConfigMapLister    v1.ConfigMapLister
	SecretLister       v1.SecretLister
	ServiceLister      v1.ServiceLister
	IngressV1Lister    networkingv1.IngressLister
	HPAV1Lister        autoscalingv1.HorizontalPodAutoscalerLister
}

func GetCachedResources(clientset kubernetes.Interface) *CachedResources {
	if cachedResources == nil {
		cachedResources = initializeCachedResources(clientset)
	}
	return cachedResources
}

func initializeCachedResources(clientset kubernetes.Interface) *CachedResources {
	clientset.Discovery().ServerGroupsAndResources()
	sharedInformers := informers.NewSharedInformerFactory(clientset, time.Hour*8)

	// informer
	deploymentInformer := sharedInformers.Apps().V1().Deployments()
	statefuleSetInformer := sharedInformers.Apps().V1().StatefulSets()
	configMapInformer := sharedInformers.Core().V1().ConfigMaps()
	secretInformer := sharedInformers.Core().V1().Secrets()
	serviceInformer := sharedInformers.Core().V1().Services()
	ingressV1Informer := sharedInformers.Networking().V1().Ingresses()
	hpaV1Informer := sharedInformers.Autoscaling().V1().HorizontalPodAutoscalers()

	// shared informers
	deploymentSharedInformer := deploymentInformer.Informer()
	statefuleSetSharedInformer := statefuleSetInformer.Informer()
	configMapSharedInformer := configMapInformer.Informer()
	secretSharedInformer := secretInformer.Informer()
	serviceSharedInformer := serviceInformer.Informer()
	ingressV1SharedInformer := ingressV1Informer.Informer()
	hpaV1SharedInformer := hpaV1Informer.Informer()

	//
	informers := map[string]cache.SharedInformer{
		"deploymentSharedInformer":   deploymentSharedInformer,
		"statefuleSetSharedInformer": statefuleSetSharedInformer,
		"configMapSharedInformer":    configMapSharedInformer,
		"secretSharedInformer":       secretSharedInformer,
		"serviceSharedInformer":      serviceSharedInformer,
		"ingressV1SharedInformer":    ingressV1SharedInformer,
		"hpaV1SharedInformer":        hpaV1SharedInformer,
	}
	var wg sync.WaitGroup
	wg.Add(len(informers))
	for k, v := range informers {
		go func(name string, informer cache.SharedInformer) {
			if !cache.WaitForCacheSync(wait.NeverStop, informer.HasSynced) {
				logrus.Warningln("wait for cached synced failed:", name)
			}
			wg.Done()
		}(k, v)
	}

	sharedInformers.Start(wait.NeverStop)
	sharedInformers.WaitForCacheSync(wait.NeverStop)
	return &CachedResources{
		DeploymentLister:   deploymentInformer.Lister(),
		StatefuleSetLister: statefuleSetInformer.Lister(),
		ConfigMapLister:    configMapInformer.Lister(),
		SecretLister:       secretInformer.Lister(),
		ServiceLister:      serviceInformer.Lister(),
		IngressV1Lister:    ingressV1Informer.Lister(),
		HPAV1Lister:        hpaV1Informer.Lister(),
	}
}