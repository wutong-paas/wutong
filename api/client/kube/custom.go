package kube

import (

	// "gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/yaml"
)

func GetResourcesYamlFormat(clientset kubernetes.Interface, namespace string, selector labels.Selector) string {
	objs := []interface{}{}

	deployments, _ := GetCachedResources(clientset).DeploymentLister.Deployments(namespace).List(selector)
	if len(deployments) > 0 {
		slimDeployments(deployments)
		for i := 0; i < len(deployments); i++ {
			objs = append(objs, deployments[i])
		}
	}

	statefulsets, _ := cachedResources.StatefuleSetLister.StatefulSets(namespace).List(selector)
	if len(statefulsets) > 0 {
		slimStatefulSets(statefulsets)
		for i := 0; i < len(statefulsets); i++ {
			objs = append(objs, statefulsets[i])
		}
	}
	configmaps, _ := cachedResources.ConfigMapLister.ConfigMaps(namespace).List(selector)
	if len(configmaps) > 0 {
		slimConfigMaps(configmaps)
		for i := 0; i < len(configmaps); i++ {
			objs = append(objs, configmaps[i])
		}
	}

	secrets, _ := cachedResources.SecretLister.Secrets(namespace).List(labels.SelectorFromSet(wutongSelectorLabels))
	if len(secrets) > 0 {
		slimSecrets(secrets)
		for i := 0; i < len(secrets); i++ {
			objs = append(objs, secrets[i])
		}
	}

	services, _ := cachedResources.ServiceLister.Services(namespace).List(selector)
	if len(services) > 0 {
		slimServices(services)
		for i := 0; i < len(services); i++ {
			objs = append(objs, services[i])
		}
	}

	v1ingresses, _ := cachedResources.IngressV1Lister.Ingresses(namespace).List(selector)
	if len(v1ingresses) > 0 {
		slimIngresses(v1ingresses)
		for i := 0; i < len(v1ingresses); i++ {
			objs = append(objs, v1ingresses[i])
		}
	}

	v1hpas, _ := cachedResources.HPAV1Lister.HorizontalPodAutoscalers(namespace).List(selector)
	if len(v1hpas) > 0 {
		slimHPAs(v1hpas)
		for i := 0; i < len(v1hpas); i++ {
			objs = append(objs, v1hpas[i])
		}
	}

	return marshal(objs)
}

func marshal(objs []interface{}) string {
	r := yamlResource{
		ApiVersion: "v1",
		Kind:       "List",
		Items:      objs,
	}

	b, _ := yaml.Marshal(r)

	return string(b)
}

type yamlResource struct {
	ApiVersion string        `json:"apiVersion"`
	Items      []interface{} `json:"items"`
	Kind       string        `json:"kind"`
}
