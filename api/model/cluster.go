package model

import (
	corev1 "k8s.io/api/core/v1"
)

//ClusterResource -
type ClusterResource struct {
	AllNode              int             `json:"all_node"`
	NotReadyNode         int             `json:"notready_node"`
	ComputeNode          int             `json:"compute_node"`
	CapCPU               int             `json:"cap_cpu"`
	CapMem               int             `json:"cap_mem"`
	HealthCapCPU         int             `json:"health_cap_cpu"`
	HealthCapMem         int             `json:"health_cap_mem"`
	UnhealthCapCPU       int             `json:"unhealth_cap_cpu"`
	UnhealthCapMem       int             `json:"unhealth_cap_mem"`
	ReqCPU               float32         `json:"req_cpu"`
	ReqMem               int             `json:"req_mem"`
	WutongReqMem         int             `json:"wt_req_mem"` // Resources to embody wutong scheduling
	WutongReqCPU         float32         `json:"wt_req_cpu"`
	HealthReqCPU         float32         `json:"health_req_cpu"`
	HealthReqMem         int             `json:"health_req_mem"`
	UnhealthReqCPU       float32         `json:"unhealth_req_cpu"`
	UnhealthReqMem       int             `json:"unhealth_req_mem"`
	TotalCapacityPods    int64           `json:"total_capacity_pods"`
	TotalUsedPods        int64           `json:"total_used_pods"`
	TotalCapacityStorage int64           `json:"total_capacity_storage"`
	TotalUsedStorage     int64           `json:"total_used_storage"`
	NodeResources        []*NodeResource `json:"node_resources"`
	TenantPods           map[string]int  `json:"tenant_pods"`
}

// NodeResource is a collection of compute resource.
type NodeResource struct {
	Name            string `json:"node_name"`
	RawUsedCPU      int64  `json:"-"`
	RawUsedMem      int64  `json:"-"`
	CapacityCPU     int64  `json:"capacity_cpu"`
	CapacityMem     int64  `json:"capacity_mem"`
	CapacityStorage int64  `json:"capacity_storage"`
	CapacityPods    int64  `json:"capacity_pod"`
	UsedCPU         int64  `json:"used_cpu"`
	UsedMem         int64  `json:"used_mem"`
	UsedStorage     int64  `json:"used_storage"`
	UsedPods        int64  `json:"used_pod"`
	DiskPressure    bool   `json:"disk_pressure"`
	MemoryPressure  bool   `json:"memory_pressure"`
	PIDPressure     bool   `json:"pid_pressure"`
	Ready           bool   `json:"ready"`
}

func NewNodeResource(name string, rl corev1.NodeStatus) *NodeResource {
	r := &NodeResource{Name: name}
	for rName, rQuant := range rl.Capacity {
		switch rName {
		case corev1.ResourceCPU:
			r.CapacityCPU = rQuant.MilliValue() / 1000
		case corev1.ResourceMemory:
			r.CapacityMem = rQuant.Value() / 1024 / 1024
		case corev1.ResourcePods:
			r.CapacityPods = rQuant.Value()
		}
	}
	for conIndex := range rl.Conditions {
		switch rl.Conditions[conIndex].Type {
		case corev1.NodeReady:
			r.Ready = rl.Conditions[conIndex].Status == corev1.ConditionTrue
		case corev1.NodeDiskPressure:
			r.DiskPressure = rl.Conditions[conIndex].Status == corev1.ConditionTrue
		case corev1.NodeMemoryPressure:
			r.MemoryPressure = rl.Conditions[conIndex].Status == corev1.ConditionTrue
		case corev1.NodePIDPressure:
			r.PIDPressure = rl.Conditions[conIndex].Status == corev1.ConditionTrue
		}
	}
	return r
}
