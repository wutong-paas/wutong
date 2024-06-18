package kube

import (
	api_model "github.com/wutong-paas/wutong/api/model"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	&HorizontalPodAutoscalers{},
}

// GetResourcesYamlFormat
func GetResourcesYamlFormat(clientset kubernetes.Interface, namespace string, selectors []labels.Selector, customSetting *api_model.KubeResourceCustomSetting) string {
	objs := []interface{}{}

	if len(customSetting.Namespace) > 0 && !sliceContains(builtinNamespaces, customSetting.Namespace) {
		objs = append(objs, &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: customSetting.Namespace,
				Labels: map[string]string{
					"app.kubernetes.io/managed-by": "wutong",
				},
			},
		})
	}

	for _, selector := range selectors {
		for _, resource := range resources {
			resource.SetClientset(clientset)
			resource.Migrate(namespace, selector)
			resource.Decorate(customSetting)
			objs = resource.AppendTo(objs)
		}
	}

	// Secret 为共享配置组，可以为多个组件共享，只导出一份即可，所以该资源单独处理
	var secret = Secrets{}
	secret.SetClientset(clientset)
	secret.Migrate(namespace, nil)
	secret.Decorate(customSetting)
	objs = secret.AppendTo(objs)

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

// 内置命名空间
var builtinNamespaces = []string{
	"default",
	"kube-system",
	"kube-public",
	"kube-node-lease",
}

func sliceContains(s []string, v string) bool {
	for i := range s {
		if v == s[i] {
			return true
		}
	}
	return false
}
