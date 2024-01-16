package kube

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

func NodeRoles(clientset kubernetes.Interface, node *corev1.Node) []string {
	var result []string
	if _, ok := node.Labels["node-role.kubernetes.io/control-plane"]; ok {
		result = append(result, "master")
	} else {
		result = append(result, "worker")
	}

	etcdPod, err := GetCachedResources(clientset).PodLister.Pods("kube-system").Get("etcd-" + node.Name)
	if err == nil && etcdPod != nil && etcdPod.Labels["component"] == "etcd" {
		result = append(result, "etcd")
	}
	return result
}
