package kube

import (
	api_model "github.com/wutong-paas/wutong/api/model"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

type ConfigMaps struct {
	kubernetes.Clientset
	ConfigMaps []*corev1.ConfigMap `json:"configmaps"`
}

func (c *ConfigMaps) Migrate(namespace string, seletcor labels.Selector) {
	configMaps, err := GetCachedResources(c).ConfigMapLister.ConfigMaps(namespace).List(seletcor)
	if err != nil {
		c.ConfigMaps = configMaps
	}
}

func (c *ConfigMaps) Decorate(setting *api_model.KubeResourceCustomSetting) {
	for i := 0; i < len(c.ConfigMaps); i++ {
		labels := map[string]string{
			"app":           c.ConfigMaps[i].Labels["app"],
			"app_id":        c.ConfigMaps[i].Labels["app_id"],
			"service_alias": c.ConfigMaps[i].Labels["service_alias"],
			"service_id":    c.ConfigMaps[i].Labels["service_id"],
			"tenant_id":     c.ConfigMaps[i].Labels["tenant_id"],
			"tenant_name":   c.ConfigMaps[i].Labels["tenant_name"],
		}
		if c.ConfigMaps[i] != nil {
			c.ConfigMaps[i].APIVersion = "v1"
			c.ConfigMaps[i].Kind = "ConfigMap"
			c.ConfigMaps[i].ObjectMeta = v1.ObjectMeta{
				Name:   c.ConfigMaps[i].Name,
				Labels: labels,
			}
		}
		if setting != nil {
			if setting.Namespace != "" {
				c.ConfigMaps[i].Namespace = setting.Namespace
			}
		}
	}
}

func (c *ConfigMaps) AppendTo(objs []interface{}) {
	for _, configMap := range c.ConfigMaps {
		objs = append(objs, configMap)
	}
}
