// WUTONG, Application Management Platform
// Copyright (C) 2014-2017 Wutong Co., Ltd.

// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version. For any non-GPL usage of Wutong,
// one or multiple Commercial Licenses authorized by Wutong Co., Ltd.
// must be obtained first.

// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.

// You should have received a copy of the GNU General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package conversion

import (
	"fmt"

	"github.com/sirupsen/logrus"

	autoscalingv1 "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/wutong-paas/wutong/db"
	"github.com/wutong-paas/wutong/db/model"
	"github.com/wutong-paas/wutong/util"
	v1 "github.com/wutong-paas/wutong/worker/appm/types/v1"
)

var str2ResourceName = map[string]corev1.ResourceName{
	"cpu":    corev1.ResourceCPU,
	"memory": corev1.ResourceMemory,
}

// TenantEnvServiceAutoscaler -
func TenantEnvServiceAutoscaler(as *v1.AppService, dbmanager db.Manager) error {
	hpas, err := newHPAs(as, dbmanager)
	if err != nil {
		return fmt.Errorf("create HPAs: %v", err)
	}
	logrus.Debugf("the numbers of HPAs: %d", len(hpas))

	as.SetHPAs(hpas)

	return nil
}

func newHPAs(as *v1.AppService, dbmanager db.Manager) ([]*autoscalingv1.HorizontalPodAutoscaler, error) {
	xpaRules, err := dbmanager.TenantEnvServceAutoscalerRulesDao().ListEnableOnesByServiceID(as.ServiceID)
	if err != nil {
		return nil, err
	}

	var hpas []*autoscalingv1.HorizontalPodAutoscaler
	for _, rule := range xpaRules {
		var kind, name string
		if as.GetStatefulSet() != nil {
			kind, name = "StatefulSet", as.GetStatefulSet().GetName()
		} else {
			kind, name = "Deployment", as.GetDeployment().GetName()
		}

		labels := as.GetCommonLabels(map[string]string{
			"rule_id": rule.RuleID,
			"version": as.DeployVersion,
		})

		hpa := newHPA(as.GetNamespace(), kind, name, labels, rule)

		hpas = append(hpas, hpa)
	}

	return hpas, nil
}

func createResourceMetrics(metric *model.TenantEnvServiceAutoscalerRuleMetrics) autoscalingv1.MetricSpec {
	ms := autoscalingv1.MetricSpec{
		Type: autoscalingv1.ResourceMetricSourceType,
		Resource: &autoscalingv1.ResourceMetricSource{
			Name: str2ResourceName[metric.MetricsName],
		},
	}

	if metric.MetricTargetType == "utilization" {
		value := int32(metric.MetricTargetValue)
		ms.Resource.TargetAverageUtilization = &value
		// ms.Resource.Target = autoscalingv1.MetricTarget{
		// 	Type:               autoscalingv1.UtilizationMetricType,
		// 	AverageUtilization: &value,
		// }
	}
	if metric.MetricTargetType == "average_value" {
		ms.Resource.TargetAverageValue = resource.NewMilliQuantity(int64(metric.MetricTargetValue), resource.DecimalSI)
		// ms.Resource.Target.Type = autoscalingv1.AverageValueMetricType
		// if metric.MetricsName == "cpu" {
		// 	ms.Resource.Target.AverageValue = resource.NewMilliQuantity(int64(metric.MetricTargetValue), resource.DecimalSI)
		// }
		// if metric.MetricsName == "memory" {
		// 	ms.Resource.Target.AverageValue = resource.NewQuantity(int64(metric.MetricTargetValue*1024*1024), resource.BinarySI)
		// }
	}

	return ms
}

func newHPA(namespace, kind, name string, labels map[string]string, rule *model.TenantEnvServiceAutoscalerRules) *autoscalingv1.HorizontalPodAutoscaler {
	hpa := &autoscalingv1.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rule.RuleID,
			Namespace: namespace,
			Labels:    labels,
		},
	}

	spec := autoscalingv1.HorizontalPodAutoscalerSpec{
		MinReplicas: util.Int32(int32(rule.MinReplicas)),
		MaxReplicas: int32(rule.MaxReplicas),
		ScaleTargetRef: autoscalingv1.CrossVersionObjectReference{
			Kind:       kind,
			Name:       name,
			APIVersion: "apps/v1",
		},
	}

	// for _, metric := range metrics {
	// 	if metric.MetricsType != "resource_metrics" {
	// 		logrus.Warningf("rule id:  %s; unsupported metric type: %s", rule.RuleID, metric.MetricsType)
	// 		continue
	// 	}
	// 	if metric.MetricTargetValue <= 0 {
	// 		// TODO: If the target value of cpu and memory is 0, it will not take effect.
	// 		// TODO: The target value of the custom indicator can be 0.
	// 		continue
	// 	}

	// 	ms := createResourceMetrics(metric)
	// 	spec.Metrics = append(spec.Metrics, ms)
	// }
	// if len(spec.Metrics) == 0 {
	// 	return nil
	// }
	hpa.Spec = spec

	return hpa
}
