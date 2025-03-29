package kube

import (
	api_model "github.com/wutong-paas/wutong/api/model"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

type HorizontalPodAutoscalers struct {
	kubernetes.Interface
	HorizontalPodAutoscalers   []*autoscalingv1.HorizontalPodAutoscaler `json:"horizontalpodautoscalers"`
	V2HorizontalPodAutoscalers []*autoscalingv2.HorizontalPodAutoscaler `json:"v2horizontalpodautoscalers"`
}

func (h *HorizontalPodAutoscalers) SetClientset(clientset kubernetes.Interface) {
	h.Interface = clientset
}

func (h *HorizontalPodAutoscalers) Migrate(namespace string, seletcor labels.Selector) {
	if VersionGTE(1, 23) {
		// 使用v2版本
		hpas, err := GetCachedResources(h).HPAV2Lister.HorizontalPodAutoscalers(namespace).List(seletcor)
		if err == nil {
			h.V2HorizontalPodAutoscalers = hpas
		}
	} else {
		// 使用v1版本
		hpas, err := GetCachedResources(h).HPAV1Lister.HorizontalPodAutoscalers(namespace).List(seletcor)
		if err == nil {
			h.HorizontalPodAutoscalers = hpas
		}
	}
}

func (h *HorizontalPodAutoscalers) Decorate(setting *api_model.KubeResourceCustomSetting) {
	if VersionGTE(1, 23) {
		for i := range h.V2HorizontalPodAutoscalers {
			labels := map[string]string{
				"app":             h.V2HorizontalPodAutoscalers[i].Labels["app"],
				"app_id":          h.V2HorizontalPodAutoscalers[i].Labels["app_id"],
				"service_alias":   h.V2HorizontalPodAutoscalers[i].Labels["service_alias"],
				"service_id":      h.V2HorizontalPodAutoscalers[i].Labels["service_id"],
				"tenant_id":       h.V2HorizontalPodAutoscalers[i].Labels["tenant_id"],
				"tenant_name":     h.V2HorizontalPodAutoscalers[i].Labels["tenant_name"],
				"tenant_env_id":   h.V2HorizontalPodAutoscalers[i].Labels["tenant_env_id"],
				"tenant_env_name": h.V2HorizontalPodAutoscalers[i].Labels["tenant_env_name"],
			}
			if h.V2HorizontalPodAutoscalers[i] != nil {
				h.V2HorizontalPodAutoscalers[i].APIVersion = "autoscaling/v2"
				h.V2HorizontalPodAutoscalers[i].Kind = "HorizontalPodAutoscaler"
				h.V2HorizontalPodAutoscalers[i].ObjectMeta = v1.ObjectMeta{
					Name:   h.V2HorizontalPodAutoscalers[i].Name,
					Labels: labels,
				}
				h.V2HorizontalPodAutoscalers[i].Status = autoscalingv2.HorizontalPodAutoscalerStatus{}
			}
			if setting != nil {
				if setting.Namespace != "" {
					h.V2HorizontalPodAutoscalers[i].Namespace = setting.Namespace
				}
			}
		}
	} else {
		for i := range h.HorizontalPodAutoscalers {
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
}

func (h *HorizontalPodAutoscalers) AppendTo(objs []interface{}) []interface{} {
	if VersionGTE(1, 23) {
		for _, v2hpa := range h.V2HorizontalPodAutoscalers {
			objs = append(objs, v2hpa)
		}
	} else {
		for _, hpa := range h.HorizontalPodAutoscalers {
			objs = append(objs, hpa)
		}
	}

	return objs
}
