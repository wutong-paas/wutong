package kube

import (
	api_model "github.com/wutong-paas/wutong/api/model"
	"github.com/wutong-paas/wutong/util/constants"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

type ConfigMaps struct {
	kubernetes.Interface
	ConfigMaps []*corev1.ConfigMap `json:"configmaps"`
}

func (c *ConfigMaps) SetClientset(clientset kubernetes.Interface) {
	c.Interface = clientset
}

func (c *ConfigMaps) Migrate(namespace string, seletcor labels.Selector) {
	configMaps, err := GetCachedResources(c).ConfigMapLister.ConfigMaps(namespace).List(seletcor)
	if err == nil {
		c.ConfigMaps = configMaps
	}
}

func (c *ConfigMaps) Decorate(setting *api_model.KubeResourceCustomSetting) {
	for i := 0; i < len(c.ConfigMaps); i++ {
		labels := map[string]string{
			constants.ResourceAppNameLabel:       c.ConfigMaps[i].Labels[constants.ResourceAppNameLabel],
			constants.ResourceAppIDLabel:         c.ConfigMaps[i].Labels[constants.ResourceAppIDLabel],
			constants.ResourceServiceAliasLabel:  c.ConfigMaps[i].Labels[constants.ResourceServiceAliasLabel],
			constants.ResourceServiceIDLabel:     c.ConfigMaps[i].Labels[constants.ResourceServiceIDLabel],
			constants.ResourceTenantEnvIDLabel:   c.ConfigMaps[i].Labels[constants.ResourceTenantEnvIDLabel],
			constants.ResourceTenantEnvNameLabel: c.ConfigMaps[i].Labels[constants.ResourceTenantEnvNameLabel],
			constants.ResourceTenantIDLabel:      c.ConfigMaps[i].Labels[constants.ResourceTenantIDLabel],
			constants.ResourceTenantNameLabel:    c.ConfigMaps[i].Labels[constants.ResourceTenantNameLabel],
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

func (c *ConfigMaps) AppendTo(objs []interface{}) []interface{} {
	for _, configMap := range c.ConfigMaps {
		objs = append(objs, configMap)
	}
	return objs
}
