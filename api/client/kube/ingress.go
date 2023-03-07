package kube

import (
	api_model "github.com/wutong-paas/wutong/api/model"
	networkingv1 "k8s.io/api/networking/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

type Ingresses struct {
	kubernetes.Interface
	Ingresses []*networkingv1.Ingress `json:"ingresses"`
}

func (i *Ingresses) SetClientset(clientset kubernetes.Interface) {
	i.Interface = clientset
}

func (i *Ingresses) Migrate(namespace string, seletcor labels.Selector) {
	ingresses, err := GetCachedResources(i).IngressV1Lister.Ingresses(namespace).List(seletcor)
	if err == nil {
		i.Ingresses = ingresses
	}
}

func (i *Ingresses) Decorate(setting *api_model.KubeResourceCustomSetting) {
	for n := 0; n < len(i.Ingresses); n++ {
		labels := map[string]string{
			"app":             i.Ingresses[n].Labels["app"],
			"app_id":          i.Ingresses[n].Labels["app_id"],
			"service_alias":   i.Ingresses[n].Labels["service_alias"],
			"service_id":      i.Ingresses[n].Labels["service_id"],
			"tenant_id":       i.Ingresses[n].Labels["tenant_id"],
			"tenant_name":     i.Ingresses[n].Labels["tenant_name"],
			"tenant_env_id":   i.Ingresses[n].Labels["tenant_env_id"],
			"tenant_env_name": i.Ingresses[n].Labels["tenant_env_name"],
		}
		if i.Ingresses[n] != nil {
			i.Ingresses[n].APIVersion = "networking.k8s.io/v1"
			i.Ingresses[n].Kind = "Ingress"
			i.Ingresses[n].ObjectMeta = v1.ObjectMeta{
				Name:   i.Ingresses[n].Name,
				Labels: labels,
			}
			i.Ingresses[n].Status = networkingv1.IngressStatus{}
		}
		if setting != nil {
			if setting.Namespace != "" {
				i.Ingresses[n].Namespace = setting.Namespace
			}
		}
	}
}

func (i *Ingresses) AppendTo(objs []interface{}) []interface{} {
	for _, ingress := range i.Ingresses {
		objs = append(objs, ingress)
	}
	return objs
}
