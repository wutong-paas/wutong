package kube

import (
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	appsv1 "k8s.io/client-go/listers/apps/v1"
	autoscalingv1 "k8s.io/client-go/listers/autoscaling/v1"
	corev1 "k8s.io/client-go/listers/core/v1"
	networkingv1 "k8s.io/client-go/listers/networking/v1"
	storagev1 "k8s.io/client-go/listers/storage/v1"
	"k8s.io/client-go/tools/cache"
)

var cachedResources *CachedResources

type CachedResources struct {
	DeploymentLister            appsv1.DeploymentLister
	StatefuleSetLister          appsv1.StatefulSetLister
	PodLister                   corev1.PodLister
	ConfigMapLister             corev1.ConfigMapLister
	SecretLister                corev1.SecretLister
	ServiceLister               corev1.ServiceLister
	IngressV1Lister             networkingv1.IngressLister
	HPAV1Lister                 autoscalingv1.HorizontalPodAutoscalerLister
	EventLister                 corev1.EventLister
	StorageClassLister          storagev1.StorageClassLister
	NodeLister                  corev1.NodeLister
	PersistentVolumeClaimLister corev1.PersistentVolumeClaimLister
}

func GetCachedResources(clientset kubernetes.Interface) *CachedResources {
	if cachedResources == nil {
		cachedResources = initializeCachedResources(clientset)
	}
	return cachedResources
}

func GetDefaultStorageClass(clientset kubernetes.Interface) string {
	scs, _ := GetCachedResources(clientset).StorageClassLister.List(labels.Everything())
	for _, sc := range scs {
		if sc.Annotations["storageclass.kubernetes.io/is-default-class"] == "true" {
			return sc.Name
		}
	}
	return ""
}

func initializeCachedResources(clientset kubernetes.Interface) *CachedResources {
	clientset.Discovery().ServerGroupsAndResources()
	sharedInformers := informers.NewSharedInformerFactory(clientset, time.Hour*8)

	// store event
	filteredSharedInformer := informers.NewFilteredSharedInformerFactory(clientset, time.Hour*8, "", func(options *metav1.ListOptions) {
		options.FieldSelector = "type=Warning"
	})

	// informer
	deploymentInformer := sharedInformers.Apps().V1().Deployments()
	statefuleSetInformer := sharedInformers.Apps().V1().StatefulSets()
	podInformer := sharedInformers.Core().V1().Pods()
	configMapInformer := sharedInformers.Core().V1().ConfigMaps()
	secretInformer := sharedInformers.Core().V1().Secrets()
	serviceInformer := sharedInformers.Core().V1().Services()
	ingressV1Informer := sharedInformers.Networking().V1().Ingresses()
	hpaV1Informer := sharedInformers.Autoscaling().V1().HorizontalPodAutoscalers()
	eventInformer := filteredSharedInformer.Core().V1().Events()
	storageClassInformer := sharedInformers.Storage().V1().StorageClasses()
	nodeInformer := sharedInformers.Core().V1().Nodes()
	persistentVolumeClaimInformer := sharedInformers.Core().V1().PersistentVolumeClaims()

	// shared informers
	deploymentSharedInformer := deploymentInformer.Informer()
	statefuleSetSharedInformer := statefuleSetInformer.Informer()
	podSharedInformer := podInformer.Informer()
	podSharedInformer.AddEventHandler(podEventHandlerForMetrics())
	configMapSharedInformer := configMapInformer.Informer()
	secretSharedInformer := secretInformer.Informer()
	serviceSharedInformer := serviceInformer.Informer()
	ingressV1SharedInformer := ingressV1Informer.Informer()
	hpaV1SharedInformer := hpaV1Informer.Informer()
	eventSharedInformer := eventInformer.Informer()
	storageClassSharedInformer := storageClassInformer.Informer()
	nodeSharedInformer := nodeInformer.Informer()
	persistentVolumeClaimSharedInformer := persistentVolumeClaimInformer.Informer()

	informers := map[string]cache.SharedInformer{
		"deploymentSharedInformer":            deploymentSharedInformer,
		"statefuleSetSharedInformer":          statefuleSetSharedInformer,
		"podSharedInformer":                   podSharedInformer,
		"configMapSharedInformer":             configMapSharedInformer,
		"secretSharedInformer":                secretSharedInformer,
		"serviceSharedInformer":               serviceSharedInformer,
		"ingressV1SharedInformer":             ingressV1SharedInformer,
		"hpaV1SharedInformer":                 hpaV1SharedInformer,
		"eventSharedInformer":                 eventSharedInformer,
		"storageClassSharedInformer":          storageClassSharedInformer,
		"nodeSharedInformer":                  nodeSharedInformer,
		"persistentVolumeClaimSharedInformer": persistentVolumeClaimSharedInformer,
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
	filteredSharedInformer.Start(wait.NeverStop)
	sharedInformers.WaitForCacheSync(wait.NeverStop)
	filteredSharedInformer.WaitForCacheSync(wait.NeverStop)
	return &CachedResources{
		DeploymentLister:            deploymentInformer.Lister(),
		StatefuleSetLister:          statefuleSetInformer.Lister(),
		PodLister:                   podInformer.Lister(),
		ConfigMapLister:             configMapInformer.Lister(),
		SecretLister:                secretInformer.Lister(),
		ServiceLister:               serviceInformer.Lister(),
		IngressV1Lister:             ingressV1Informer.Lister(),
		HPAV1Lister:                 hpaV1Informer.Lister(),
		EventLister:                 eventInformer.Lister(),
		StorageClassLister:          storageClassInformer.Lister(),
		NodeLister:                  nodeInformer.Lister(),
		PersistentVolumeClaimLister: persistentVolumeClaimInformer.Lister(),
	}
}
