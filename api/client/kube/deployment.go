package kube

import (
	api_model "github.com/wutong-paas/wutong/api/model"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

type Deployments struct {
	kubernetes.Interface
	Deployments []*appsv1.Deployment `json:"deployments"`
}

func (d *Deployments) SetClientset(clientset kubernetes.Interface) {
	d.Interface = clientset
}

func (d *Deployments) Migrate(namespace string, seletcor labels.Selector) {
	deployments, err := GetCachedResources(d).DeploymentLister.Deployments(namespace).List(seletcor)
	if err == nil {
		d.Deployments = deployments
	}
}

func (d *Deployments) Decorate(setting *api_model.KubeResourceCustomSetting) {
	for i := 0; i < len(d.Deployments); i++ {
		labels := map[string]string{
			"app":           d.Deployments[i].Labels["app"],
			"app_id":        d.Deployments[i].Labels["app_id"],
			"name":          d.Deployments[i].Labels["name"],
			"service_alias": d.Deployments[i].Labels["service_alias"],
			"service_id":    d.Deployments[i].Labels["service_id"],
			"tenant_id":     d.Deployments[i].Labels["tenant_id"],
			"tenant_name":   d.Deployments[i].Labels["tenant_name"],
		}
		if d.Deployments[i] != nil {
			d.Deployments[i].APIVersion = "apps/v1"
			d.Deployments[i].Kind = "Deployment"
			d.Deployments[i].ObjectMeta = v1.ObjectMeta{
				Name:   d.Deployments[i].Name,
				Labels: labels,
			}
			d.Deployments[i].Spec.ProgressDeadlineSeconds = nil
			d.Deployments[i].Spec.RevisionHistoryLimit = nil
			d.Deployments[i].Spec.Strategy = appsv1.DeploymentStrategy{}
			d.Deployments[i].Spec.Template.ObjectMeta = v1.ObjectMeta{
				CreationTimestamp: v1.Time{},
				Labels:            labels,
			}
			d.Deployments[i].Spec.Template.Spec.SchedulerName = ""
			d.Deployments[i].Spec.Template.Spec.DNSPolicy = ""
			d.Deployments[i].Status = appsv1.DeploymentStatus{}
		}
		if setting != nil {
			if setting.Namespace != "" {
				d.Deployments[i].Namespace = setting.Namespace
			}
		}
	}
}

func (d *Deployments) AppendTo(objs []interface{}) []interface{} {
	for _, deployment := range d.Deployments {
		objs = append(objs, deployment)
	}
	return objs
}
