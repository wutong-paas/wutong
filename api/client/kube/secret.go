package kube

import (
	api_model "github.com/wutong-paas/wutong/api/model"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

type Secrets struct {
	kubernetes.Interface
	Secrets []*corev1.Secret `json:"secrets"`
}

func (s *Secrets) SetClientset(clientset kubernetes.Interface) {
	s.Interface = clientset
}

func (s *Secrets) Migrate(namespace string, seletcor labels.Selector) {
	secrets, err := GetCachedResources(s).SecretLister.Secrets(namespace).List(labels.SelectorFromSet(wutongSelectorLabels))
	if err == nil {
		s.Secrets = secrets
	}
}

func (s *Secrets) Decorate(setting *api_model.KubeResourceCustomSetting) {
	for i := 0; i < len(s.Secrets); i++ {
		labels := map[string]string{
			"app":             s.Secrets[i].Labels["app"],
			"app_id":          s.Secrets[i].Labels["app_id"],
			"tenant_env_id":   s.Secrets[i].Labels["tenant_env_id"],
			"tenant_env_name": s.Secrets[i].Labels["tenant_env_name"],
		}
		if s.Secrets[i] != nil {
			s.Secrets[i].APIVersion = "v1"
			s.Secrets[i].Kind = "Secret"
			s.Secrets[i].ObjectMeta = v1.ObjectMeta{
				Name:   s.Secrets[i].Name,
				Labels: labels,
			}
		}
		if setting != nil {
			if setting.Namespace != "" {
				s.Secrets[i].Namespace = setting.Namespace
			}
		}
	}
}

func (s *Secrets) AppendTo(objs []interface{}) []interface{} {
	for _, secret := range s.Secrets {
		objs = append(objs, secret)
	}
	return objs
}
