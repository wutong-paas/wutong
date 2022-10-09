package kube

import (
	api_model "github.com/wutong-paas/wutong/api/model"
	"k8s.io/apimachinery/pkg/labels"
)

type ResourceListInterface interface {
	Migrate(namespace string, seletcor labels.Selector)
	Decorate(setting *api_model.KubeResourceCustomSetting)
	AppendTo([]interface{})
}
