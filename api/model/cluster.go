package model

import (
	"fmt"
	"strconv"

	corev1 "k8s.io/api/core/v1"
)

//ClusterResource -
type ClusterResource struct {
	AllNode              int             `json:"all_node"`
	NotReadyNode         int             `json:"notready_node"`
	ComputeNode          int             `json:"compute_node"`
	CapCPU               float32         `json:"cap_cpu"`
	CapMem               float32         `json:"cap_mem"`
	HealthCapCPU         float32         `json:"health_cap_cpu"`
	HealthCapMem         float32         `json:"health_cap_mem"`
	UnhealthCapCPU       float32         `json:"unhealth_cap_cpu"`
	UnhealthCapMem       float32         `json:"unhealth_cap_mem"`
	ReqCPU               float32         `json:"req_cpu"`
	ReqMem               float32         `json:"req_mem"`
	WutongReqMem         float32         `json:"wt_req_mem"` // Resources to embody wutong scheduling
	WutongReqCPU         float32         `json:"wt_req_cpu"`
	HealthReqCPU         float32         `json:"health_req_cpu"`
	HealthReqMem         float32         `json:"health_req_mem"`
	UnhealthReqCPU       float32         `json:"unhealth_req_cpu"`
	UnhealthReqMem       float32         `json:"unhealth_req_mem"`
	TotalCapacityPods    int64           `json:"total_capacity_pods"`
	TotalUsedPods        int64           `json:"total_used_pods"`
	TotalCapacityStorage float32         `json:"total_capacity_storage"`
	TotalUsedStorage     float32         `json:"total_used_storage"`
	NodeResources        []*NodeResource `json:"node_resources"`
	TenantPods           map[string]int  `json:"tenant_pods"`
}

// NodeResource is a collection of compute resource.
type NodeResource struct {
	Name            string  `json:"node_name"`
	RawUsedCPU      float32 `json:"-"`
	RawUsedMem      float32 `json:"-"`
	CapacityCPU     float32 `json:"capacity_cpu"`
	CapacityMem     float32 `json:"capacity_mem"`
	CapacityStorage float32 `json:"capacity_storage"`
	CapacityPods    int64   `json:"capacity_pod"`
	UsedCPU         float32 `json:"used_cpu"`
	UsedMem         float32 `json:"used_mem"`
	UsedStorage     float32 `json:"used_storage"`
	UsedPods        int64   `json:"used_pod"`
	DiskPressure    bool    `json:"disk_pressure"`
	MemoryPressure  bool    `json:"memory_pressure"`
	PIDPressure     bool    `json:"pid_pressure"`
	Ready           bool    `json:"ready"`
}

func NewNodeResource(name string, rl corev1.NodeStatus) *NodeResource {
	r := &NodeResource{Name: name}
	for rName, rQuant := range rl.Capacity {
		switch rName {
		case corev1.ResourceCPU:
			r.CapacityCPU = DecimalFromFloat32(float32(rQuant.MilliValue()) / 1000)
		case corev1.ResourceMemory:
			r.CapacityMem = DecimalFromFloat32(float32(rQuant.Value()) / 1024 / 1024 / 1024)
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

func DecimalFromFloat32(f float32) float32 {
	res, err := strconv.ParseFloat(fmt.Sprintf("%.2f", f), 32)
	if err != nil {
		return 0
	}
	return float32(res)
}
