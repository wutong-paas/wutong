package kube

import (
	api_model "github.com/wutong-paas/wutong/api/model"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

type HorizontalPodAutoscalers struct {
	kubernetes.Interface
	HorizontalPodAutoscalers []*autoscalingv1.HorizontalPodAutoscaler `json:"horizontalpodautoscalers"`
}

func (h *HorizontalPodAutoscalers) SetClientset(clientset kubernetes.Interface) {
	h.Interface = clientset
}

func (h *HorizontalPodAutoscalers) Migrate(namespace string, seletcor labels.Selector) {
	hpas, err := GetCachedResources(h).HPAV1Lister.HorizontalPodAutoscalers(namespace).List(seletcor)
	if err == nil {
		h.HorizontalPodAutoscalers = hpas
	}
}

func (h *HorizontalPodAutoscalers) Decorate(setting *api_model.KubeResourceCustomSetting) {
	for i := 0; i < len(h.HorizontalPodAutoscalers); i++ {
		labels := map[string]string{
			"app":             h.HorizontalPodAutoscalers[i].Labels["app"],
			"app_id":          h.HorizontalPodAutoscalers[i].Labels["app_id"],
			"service_alias":   h.HorizontalPodAutoscalers[i].Labels["service_alias"],
			"service_id":      h.HorizontalPodAutoscalers[i].Labels["service_id"],
			"tenant_id":       h.HorizontalPodAutoscalers[i].Labels["tenant_id"],
			"tenant_name":     h.HorizontalPodAutoscalers[i].Labels["tenant_name"],
			"tenant_env_id":   h.HorizontalPodAutoscalers[i].Labels["tenant_env_id"],
			"tenant_env_name": h.HorizontalPodAutoscalers[i].Labels["tenant_env_name"],
		}
		if h.HorizontalPodAutoscalers[i] != nil {
			h.HorizontalPodAutoscalers[i].APIVersion = "autoscaling/v1"
			h.HorizontalPodAutoscalers[i].Kind = "HorizontalPodAutoscaler"
			h.HorizontalPodAutoscalers[i].ObjectMeta = v1.ObjectMeta{
				Name:   h.HorizontalPodAutoscalers[i].Name,
				Labels: labels,
			}
			h.HorizontalPodAutoscalers[i].Status = autoscalingv1.HorizontalPodAutoscalerStatus{}
		}
		if setting != nil {
			if setting.Namespace != "" {
				h.HorizontalPodAutoscalers[i].Namespace = setting.Namespace
			}
		}
	}
}

func (h *HorizontalPodAutoscalers) AppendTo(objs []interface{}) []interface{} {
	for _, hpa := range h.HorizontalPodAutoscalers {
		objs = append(objs, hpa)
	}
	return objs
}
