package kube

import (
	api_model "github.com/wutong-paas/wutong/api/model"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/yaml"
)

var resources = []ResourceListInterface{
	&Deployments{},
	&Statefulsets{},
	&Services{},
	&Ingresses{},
	&ConfigMaps{},
	&Secrets{},
	&HorizontalPodAutoscalers{},
}

// GetResourcesYamlFormat
func GetResourcesYamlFormat(clientset kubernetes.Interface, namespace string, selector labels.Selector, customSetting *api_model.KubeResourceCustomSetting) string {
	objs := []interface{}{}

	for _, resource := range resources {
		resource.SetClientset(clientset)
		resource.Migrate(namespace, selector)
		resource.Decorate(customSetting)
		objs = resource.AppendTo(objs)
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
