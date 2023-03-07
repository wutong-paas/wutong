package kube

import (
	api_model "github.com/wutong-paas/wutong/api/model"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

type Statefulsets struct {
	kubernetes.Interface
	StatefulSets []*appsv1.StatefulSet `json:"statefulsets"`
}

func (s *Statefulsets) SetClientset(clientset kubernetes.Interface) {
	s.Interface = clientset
}

func (s *Statefulsets) Migrate(namespace string, seletcor labels.Selector) {
	statefulsets, err := GetCachedResources(s).StatefuleSetLister.StatefulSets(namespace).List(seletcor)
	if err == nil {
		s.StatefulSets = statefulsets
	}
}

func (s *Statefulsets) Decorate(setting *api_model.KubeResourceCustomSetting) {
	for i := 0; i < len(s.StatefulSets); i++ {
		labels := map[string]string{
			"app":             s.StatefulSets[i].Labels["app"],
			"app_id":          s.StatefulSets[i].Labels["app_id"],
			"name":            s.StatefulSets[i].Labels["name"],
			"service_alias":   s.StatefulSets[i].Labels["service_alias"],
			"service_id":      s.StatefulSets[i].Labels["service_id"],
			"tenant_id":       s.StatefulSets[i].Labels["tenant_id"],
			"tenant_name":     s.StatefulSets[i].Labels["tenant_name"],
			"tenant_env_id":   s.StatefulSets[i].Labels["tenant_env_id"],
			"tenant_env_name": s.StatefulSets[i].Labels["tenant_env_name"],
		}
		if s.StatefulSets[i] != nil {
			s.StatefulSets[i].APIVersion = "apps/v1"
			s.StatefulSets[i].Kind = "StatefulSet"
			s.StatefulSets[i].ObjectMeta = v1.ObjectMeta{
				Name:   s.StatefulSets[i].Name,
				Labels: labels,
			}
			s.StatefulSets[i].Spec.Template.ObjectMeta = v1.ObjectMeta{
				Labels: labels,
			}
			s.StatefulSets[i].Spec.Template.Spec.SchedulerName = ""
			s.StatefulSets[i].Spec.Template.Spec.DNSPolicy = ""
			s.StatefulSets[i].Status = appsv1.StatefulSetStatus{}
		}
		if setting != nil {
			if setting.Namespace != "" {
				s.StatefulSets[i].Namespace = setting.Namespace
			}
		}
	}
}

func (s *Statefulsets) AppendTo(objs []interface{}) []interface{} {
	for _, statefulset := range s.StatefulSets {
		objs = append(objs, statefulset)
	}
	return objs
}
