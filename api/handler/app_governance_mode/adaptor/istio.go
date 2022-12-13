package adaptor

import (
	"context"
	"os"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
)

type istioServiceMeshMode struct {
	kubeClient clientset.Interface
}

// NewIstioGoveranceMode -
func NewIstioGoveranceMode(kubeClient clientset.Interface) AppGoveranceModeHandler {
	return &istioServiceMeshMode{
		kubeClient: kubeClient,
	}
}

// IsInstalledControlPlane -
func (i *istioServiceMeshMode) IsInstalledControlPlane() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmName := os.Getenv("ISTIO_CM")
	if cmName == "" {
		cmName = "istio-ca-root-cert"
	}
	_, err := i.kubeClient.CoreV1().ConfigMaps("default").Get(ctx, cmName, metav1.GetOptions{})

	return err == nil
}

// GetInjectLabels -
func (i *istioServiceMeshMode) GetInjectLabels() map[string]string {
	return map[string]string{"sidecar.istio.io/inject": "true"}
}
