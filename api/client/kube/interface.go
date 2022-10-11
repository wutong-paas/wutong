package kube

import (
	api_model "github.com/wutong-paas/wutong/api/model"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

type ResourceListInterface interface {
	SetClientset(clientset kubernetes.Interface)
	Migrate(namespace string, seletcor labels.Selector)
	Decorate(setting *api_model.KubeResourceCustomSetting)
	AppendTo([]interface{}) []interface{}
}
