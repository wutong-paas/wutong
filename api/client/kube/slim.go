package kube

import (
	appsv1 "k8s.io/api/apps/v1"
	autosaclingv1 "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func slimDeployments(resources []*appsv1.Deployment) {
	for i := 0; i < len(resources); i++ {
		labels := map[string]string{
			"app":           resources[i].Labels["app"],
			"app_id":        resources[i].Labels["app_id"],
			"name":          resources[i].Labels["name"],
			"service_alias": resources[i].Labels["service_alias"],
			"service_id":    resources[i].Labels["service_id"],
			"tenant_id":     resources[i].Labels["tenant_id"],
			"tenant_name":   resources[i].Labels["tenant_name"],
		}
		if resources[i] != nil {
			resources[i].APIVersion = "apps/v1"
			resources[i].Kind = "Deployment"
			resources[i].ObjectMeta = v1.ObjectMeta{
				Name:   resources[i].Name,
				Labels: labels,
			}
			resources[i].Spec.ProgressDeadlineSeconds = nil
			resources[i].Spec.RevisionHistoryLimit = nil
			resources[i].Spec.Strategy = appsv1.DeploymentStrategy{}
			resources[i].Spec.Template.ObjectMeta = v1.ObjectMeta{
				CreationTimestamp: v1.Time{},
				Labels:            labels,
			}
			resources[i].Spec.Template.Spec.SchedulerName = ""
			resources[i].Spec.Template.Spec.DNSPolicy = ""
			resources[i].Status = appsv1.DeploymentStatus{}
		}
	}
}

func slimServices(resources []*corev1.Service) {
	for i := 0; i < len(resources); i++ {
		labels := map[string]string{
			"app":           resources[i].Labels["app"],
			"app_id":        resources[i].Labels["app_id"],
			"name":          resources[i].Labels["name"],
			"service_alias": resources[i].Labels["service_alias"],
			"service_id":    resources[i].Labels["service_id"],
			"tenant_id":     resources[i].Labels["tenant_id"],
			"tenant_name":   resources[i].Labels["tenant_name"],
		}
		if resources[i] != nil {
			resources[i].APIVersion = "v1"
			resources[i].Kind = "Service"
			resources[i].ObjectMeta = v1.ObjectMeta{
				Name:   resources[i].Name,
				Labels: labels,
			}
			resources[i].Spec = corev1.ServiceSpec{
				Ports: resources[i].Spec.Ports,
				Type:  resources[i].Spec.Type,
				Selector: map[string]string{
					"name": resources[i].Labels["name"],
				},
			}
			resources[i].Status = corev1.ServiceStatus{}
		}
	}
}

func slimStatefulSets(resources []*appsv1.StatefulSet) {
	for i := 0; i < len(resources); i++ {
		labels := map[string]string{
			"app":           resources[i].Labels["app"],
			"app_id":        resources[i].Labels["app_id"],
			"name":          resources[i].Labels["name"],
			"service_alias": resources[i].Labels["service_alias"],
			"service_id":    resources[i].Labels["service_id"],
			"tenant_id":     resources[i].Labels["tenant_id"],
			"tenant_name":   resources[i].Labels["tenant_name"],
		}
		if resources[i] != nil {
			resources[i].APIVersion = "apps/v1"
			resources[i].Kind = "StatefulSet"
			resources[i].ObjectMeta = v1.ObjectMeta{
				Name:   resources[i].Name,
				Labels: labels,
			}
			resources[i].Spec.Template.ObjectMeta = v1.ObjectMeta{
				Labels: labels,
			}
			resources[i].Spec.Template.Spec.SchedulerName = ""
			resources[i].Spec.Template.Spec.DNSPolicy = ""
			resources[i].Status = appsv1.StatefulSetStatus{}
		}
	}
}

func slimConfigMaps(resources []*corev1.ConfigMap) {
	for i := 0; i < len(resources); i++ {
		labels := map[string]string{
			"app":           resources[i].Labels["app"],
			"app_id":        resources[i].Labels["app_id"],
			"service_alias": resources[i].Labels["service_alias"],
			"service_id":    resources[i].Labels["service_id"],
			"tenant_id":     resources[i].Labels["tenant_id"],
			"tenant_name":   resources[i].Labels["tenant_name"],
		}
		if resources[i] != nil {
			resources[i].APIVersion = "v1"
			resources[i].Kind = "ConfigMap"
			resources[i].ObjectMeta = v1.ObjectMeta{
				Name:   resources[i].Name,
				Labels: labels,
			}
		}
	}
}

func slimSecrets(resources []*corev1.Secret) {
	for i := 0; i < len(resources); i++ {
		labels := map[string]string{
			"app":         resources[i].Labels["app"],
			"app_id":      resources[i].Labels["app_id"],
			"tenant_id":   resources[i].Labels["tenant_id"],
			"tenant_name": resources[i].Labels["tenant_name"],
		}
		if resources[i] != nil {
			resources[i].APIVersion = "v1"
			resources[i].Kind = "Secret"
			resources[i].ObjectMeta = v1.ObjectMeta{
				Name:   resources[i].Name,
				Labels: labels,
			}
		}
	}
}

func slimIngresses(resources []*networkingv1.Ingress) {
	for i := 0; i < len(resources); i++ {
		labels := map[string]string{
			"app":           resources[i].Labels["app"],
			"app_id":        resources[i].Labels["app_id"],
			"service_alias": resources[i].Labels["service_alias"],
			"service_id":    resources[i].Labels["service_id"],
			"tenant_id":     resources[i].Labels["tenant_id"],
			"tenant_name":   resources[i].Labels["tenant_name"],
		}
		if resources[i] != nil {
			resources[i].APIVersion = "networking.k8s.io/v1"
			resources[i].Kind = "Ingress"
			resources[i].ObjectMeta = v1.ObjectMeta{
				Name:   resources[i].Name,
				Labels: labels,
			}
			resources[i].Status = networkingv1.IngressStatus{}
		}
	}
}

func slimHPAs(resources []*autosaclingv1.HorizontalPodAutoscaler) {
	for i := 0; i < len(resources); i++ {
		labels := map[string]string{
			"app":           resources[i].Labels["app"],
			"app_id":        resources[i].Labels["app_id"],
			"service_alias": resources[i].Labels["service_alias"],
			"service_id":    resources[i].Labels["service_id"],
			"tenant_id":     resources[i].Labels["tenant_id"],
			"tenant_name":   resources[i].Labels["tenant_name"],
		}
		if resources[i] != nil {
			resources[i].APIVersion = "autoscaling/v1"
			resources[i].Kind = "HorizontalPodAutoscaler"
			resources[i].ObjectMeta = v1.ObjectMeta{
				Name:   resources[i].Name,
				Labels: labels,
			}
			resources[i].Status = autosaclingv1.HorizontalPodAutoscalerStatus{}
		}
	}
}
