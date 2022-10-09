package kube

import (
	api_model "github.com/wutong-paas/wutong/api/model"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

type Services struct {
	kubernetes.Clientset
	Services []*corev1.Service `json:"services"`
}

func (s *Services) Migrate(namespace string, seletcor labels.Selector) {
	services, err := GetCachedResources(s).ServiceLister.Services(namespace).List(seletcor)
	if err != nil {
		s.Services = services
	}
}

func (s *Services) Decorate(setting *api_model.KubeResourceCustomSetting) {
	for i := 0; i < len(s.Services); i++ {
		labels := map[string]string{
			"app":           s.Services[i].Labels["app"],
			"app_id":        s.Services[i].Labels["app_id"],
			"name":          s.Services[i].Labels["name"],
			"service_alias": s.Services[i].Labels["service_alias"],
			"service_id":    s.Services[i].Labels["service_id"],
			"tenant_id":     s.Services[i].Labels["tenant_id"],
			"tenant_name":   s.Services[i].Labels["tenant_name"],
		}
		if s.Services[i] != nil {
			s.Services[i].APIVersion = "v1"
			s.Services[i].Kind = "Service"
			s.Services[i].ObjectMeta = v1.ObjectMeta{
				Name:   s.Services[i].Name,
				Labels: labels,
			}
			s.Services[i].Spec = corev1.ServiceSpec{
				Ports: s.Services[i].Spec.Ports,
				Type:  s.Services[i].Spec.Type,
				Selector: map[string]string{
					"name": s.Services[i].Labels["name"],
				},
			}
			s.Services[i].Status = corev1.ServiceStatus{}
		}
		if setting != nil {
			if setting.Namespace != "" {
				s.Services[i].Namespace = setting.Namespace
			}
		}
	}
}

func (s *Services) AppendTo(objs []interface{}) {
	for _, service := range s.Services {
		objs = append(objs, service)
	}
}
