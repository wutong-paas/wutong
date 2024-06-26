package util

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

func dns2Config(endpoint *corev1.Endpoints, podNamespace string) (podDNSConfig *corev1.PodDNSConfig) {
	servers := make([]string, 0)
	for _, sub := range endpoint.Subsets {
		for _, addr := range sub.Addresses {
			servers = append(servers, addr.IP)
		}
	}
	searchWTDNS := fmt.Sprintf("%s.svc.cluster.local", podNamespace)
	ndotsValue := "5"
	return &corev1.PodDNSConfig{
		Nameservers: servers,
		Options:     []corev1.PodDNSConfigOption{{Name: "ndots", Value: &ndotsValue}},
		Searches:    []string{searchWTDNS, "svc.cluster.local", "cluster.local"},
	}
}

// MakePodDNSConfig make pod dns config
func MakePodDNSConfig(clientset kubernetes.Interface, podNamespace, wtNamespace, wtEndpointDNSName string) (podDNSConfig *corev1.PodDNSConfig) {
	endpoints, err := clientset.CoreV1().Endpoints(wtNamespace).Get(context.Background(), wtEndpointDNSName, metav1.GetOptions{})
	if err != nil {
		logrus.Warningf("get wt-dns[namespace: %s, name: %s] endpoints error: %s", wtNamespace, wtEndpointDNSName, err.Error())
		return nil
	}
	return dns2Config(endpoints, podNamespace)
}
