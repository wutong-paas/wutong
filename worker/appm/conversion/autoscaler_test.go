package conversion

import (
	"context"
	"testing"

	"github.com/wutong-paas/wutong/db/model"
	k8sutil "github.com/wutong-paas/wutong/util/k8s"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCreateMetricSpec(t *testing.T) {
	metric := &model.TenantEnvServiceAutoscalerRuleMetrics{
		MetricsType:       "resource_metrics",
		MetricsName:       "memory",
		MetricTargetType:  "average_value",
		MetricTargetValue: 60,
	}

	metricSpec := createResourceMetrics(metric)
	t.Logf("%#v", metricSpec)
}

func TestNewHPA(t *testing.T) {
	rule := &model.TenantEnvServiceAutoscalerRules{
		RuleID:      "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
		ServiceID:   "45197f4936cf45efa2ac4831ce42025a",
		MinReplicas: 1,
		MaxReplicas: 10,
	}

	namespace := "bab18e6b1c8640979b91f8dfdd211226"
	kind := "Deployment"
	name := "45197f4936cf45efa2ac4831ce42025a-deployment-6d84f798b4-tmvfc"

	hpa := newHPA(namespace, kind, name, nil, rule)

	clientset, err := k8sutil.NewClientset("/opt/wutong/etc/kubernetes/kubecfg/admin.kubeconfig")
	if err != nil {
		t.Fatalf("error creating k8s clientset: %s", err.Error())
	}

	// _, err = clientset.AutoscalingV2beta2().HorizontalPodAutoscalers(hpa.GetNamespace()).Create(context.Background(), hpa, metav1.CreateOptions{})
	_, err = clientset.AutoscalingV1().HorizontalPodAutoscalers(hpa.GetNamespace()).Create(context.Background(), hpa, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("create hpa: %v", err)
	}
}
