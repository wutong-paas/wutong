package helm

import (
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
)

var (
	kubeConfig      *rest.Config
	kubeClient      *kubernetes.Clientset
	discoveryClient *discovery.DiscoveryClient
	dynamicClient   dynamic.Interface
	restClient      *rest.RESTClient
	apiResourcesMap meta.RESTMapper
)

func KubeClient() *kubernetes.Clientset {
	if kubeClient == nil {
		c, err := kubernetes.NewForConfig(KubeConfig())
		if err != nil {
			panic(err)
		}
		kubeClient = c
	}
	return kubeClient
}

func KubeConfig() *rest.Config {
	if kubeConfig == nil {
		c, err := rest.InClusterConfig()
		if err != nil {
			panic(err)
		}
		kubeConfig = c
	}
	return kubeConfig
}

func KubeDiscoveryClient() *discovery.DiscoveryClient {
	if discoveryClient == nil {
		c, err := discovery.NewDiscoveryClientForConfig(KubeConfig())
		if err != nil {
			panic(err)
		}
		discoveryClient = c
	}
	return discoveryClient
}

func KubeRestClient() *rest.RESTClient {
	if restClient == nil {
		c, err := rest.RESTClientFor(KubeConfig())
		if err != nil {
			panic(err)
		}
		restClient = c
	}
	return restClient
}

func KubeDynamicClient() dynamic.Interface {
	if dynamicClient == nil {
		c, err := dynamic.NewForConfig(KubeConfig())
		if err != nil {
			panic(err)
		}
		dynamicClient = c
	}
	return dynamicClient
}

func KubeGVPMapCache() meta.RESTMapper {
	if apiResourcesMap == nil {
		LoadKubeGVRMap()
	}
	return apiResourcesMap
}

func LoadKubeGVRMap() {
	grmap, err := restmapper.GetAPIGroupResources(KubeDiscoveryClient())
	if err != nil {
		return
	}
	apiResourcesMap = restmapper.NewDiscoveryRESTMapper(grmap)
}

func KubeGVRFromGK(gk schema.GroupKind) (schema.GroupVersionResource, bool) {
	mapping, err := KubeGVPMapCache().RESTMapping(schema.ParseGroupKind(gk.String()))
	if err != nil && meta.IsNoMatchError(err) {
		LoadKubeGVRMap()
		mapping, err = apiResourcesMap.RESTMapping(schema.ParseGroupKind(gk.String()))
		if err != nil {
			return schema.GroupVersionResource{}, false
		}
	}
	return mapping.Resource, true
}
