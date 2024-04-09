package kube

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
)

type ClusterMetrics struct {
	NodeMetricsMap map[string]*NodeMetrics `json:"node_metrics_map"`
}

type NodeMetrics struct {
	PodMetricsMap map[string]*PodMetrics `json:"pod_metrics_map"`
}

type PodMetrics struct {
	CPURequest    int64 `json:"cpu_request"`
	MemoryRequest int64 `json:"memory_request"`
}

var clusterMetricsCache = &ClusterMetrics{
	NodeMetricsMap: make(map[string]*NodeMetrics),
}

func GetClusterCPURequest() int64 {
	var result int64
	if clusterMetricsCache == nil {
		return result
	}

	for node := range clusterMetricsCache.NodeMetricsMap {
		result += GetNodeCPURequest(node)
	}

	return result
}

func GetClusterMemoryRequest() int64 {
	var result int64
	if clusterMetricsCache == nil {
		return result
	}

	for node := range clusterMetricsCache.NodeMetricsMap {
		result += GetNodeMemoryRequest(node)
	}

	return result
}

func GetClusterPodCount() int64 {
	var result int64
	if clusterMetricsCache == nil {
		return result
	}

	for node := range clusterMetricsCache.NodeMetricsMap {
		result += GetNodePodCount(node)
	}

	return result
}

func GetNodeCPURequest(node string) int64 {
	var result int64
	if clusterMetricsCache == nil {
		return result
	}
	if nm, ok := clusterMetricsCache.NodeMetricsMap[node]; ok && nm != nil {
		for _, pm := range nm.PodMetricsMap {
			if pm != nil {
				result += pm.CPURequest
			}
		}
	}
	return result
}

func GetNodeMemoryRequest(node string) int64 {
	var result int64
	if clusterMetricsCache == nil {
		return result
	}
	if nm, ok := clusterMetricsCache.NodeMetricsMap[node]; ok && nm != nil {
		for _, pm := range nm.PodMetricsMap {
			if pm != nil {
				result += pm.MemoryRequest
			}
		}
	}
	return result
}

func GetNodePodCount(node string) int64 {
	var result int64
	if clusterMetricsCache == nil {
		return result
	}
	if nm, ok := clusterMetricsCache.NodeMetricsMap[node]; ok && nm != nil {
		result = int64(len(nm.PodMetricsMap))
	}
	return result
}

var podEventHandlerForMetrics = func() cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			p, ok := obj.(*corev1.Pod)
			if !ok || p.Spec.NodeName == "" {
				return
			}
			var requestCPU, requestMemory int64
			for _, c := range p.Spec.Containers {
				requestCPU += c.Resources.Requests.Cpu().MilliValue()
				requestMemory += c.Resources.Requests.Memory().Value()
			}
			pm := PodMetrics{
				CPURequest:    requestCPU,
				MemoryRequest: requestMemory,
			}
			if nm, ok := clusterMetricsCache.NodeMetricsMap[p.Spec.NodeName]; !ok || nm == nil {
				clusterMetricsCache.NodeMetricsMap[p.Spec.NodeName] = &NodeMetrics{
					PodMetricsMap: map[string]*PodMetrics{},
				}
			}

			clusterMetricsCache.NodeMetricsMap[p.Spec.NodeName].PodMetricsMap[podKey(p)] = &pm
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			p, ok := newObj.(*corev1.Pod)
			if !ok || p.Spec.NodeName == "" {
				return
			}
			var requestCPU, requestMemory int64
			for _, c := range p.Spec.Containers {
				requestCPU += c.Resources.Requests.Cpu().MilliValue()
				requestMemory += c.Resources.Requests.Memory().Value()
			}
			pm := PodMetrics{
				CPURequest:    requestCPU,
				MemoryRequest: requestMemory,
			}
			if nm, ok := clusterMetricsCache.NodeMetricsMap[p.Spec.NodeName]; !ok || nm == nil {
				clusterMetricsCache.NodeMetricsMap[p.Spec.NodeName] = &NodeMetrics{
					PodMetricsMap: map[string]*PodMetrics{},
				}
			}

			clusterMetricsCache.NodeMetricsMap[p.Spec.NodeName].PodMetricsMap[podKey(p)] = &pm
		},
		DeleteFunc: func(obj interface{}) {
			p, ok := obj.(*corev1.Pod)
			if !ok || p.Spec.NodeName == "" {
				return
			}
			if nm, ok := clusterMetricsCache.NodeMetricsMap[p.Spec.NodeName]; ok && nm != nil {
				delete(clusterMetricsCache.NodeMetricsMap[p.Spec.NodeName].PodMetricsMap, podKey(p))
			}
		},
	}
}

func podKey(p *corev1.Pod) string {
	return p.Namespace + "/" + p.Name
}
