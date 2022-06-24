package model

import (
	corev1 "k8s.io/api/core/v1"
)

//ClusterResource -
type ClusterResource struct {
	AllNode                          int           `json:"all_node"`
	NotReadyNode                     int           `json:"notready_node"`
	ComputeNode                      int           `json:"compute_node"`
	Tenant                           int           `json:"tenant"`
	CapCPU                           int           `json:"cap_cpu"`
	CapMem                           int           `json:"cap_mem"`
	HealthCapCPU                     int           `json:"health_cap_cpu"`
	HealthCapMem                     int           `json:"health_cap_mem"`
	UnhealthCapCPU                   int           `json:"unhealth_cap_cpu"`
	UnhealthCapMem                   int           `json:"unhealth_cap_mem"`
	ReqCPU                           float32       `json:"req_cpu"`
	ReqMem                           int           `json:"req_mem"`
	WutongReqMem                     int           `json:"wt_req_mem"` // Resources to embody wutong scheduling
	WutongReqCPU                     float32       `json:"wt_req_cpu"`
	HealthReqCPU                     float32       `json:"health_req_cpu"`
	HealthReqMem                     int           `json:"health_req_mem"`
	UnhealthReqCPU                   float32       `json:"unhealth_req_cpu"`
	UnhealthReqMem                   int           `json:"unhealth_req_mem"`
	CapDisk                          uint64        `json:"cap_disk"`
	ReqDisk                          uint64        `json:"req_disk"`
	MaxAllocatableMemoryNodeResource *NodeResource `json:"max_allocatable_memory_node_resource"`

	TotalCapacityPods int64           `json:"total_capacity_pods"`
	TotalUsedPods     int64           `json:"total_used_pods"`
	NodeResources     []*NodeResource `json:"node_resources"`
}

// NodeResource is a collection of compute resource.
type NodeResource struct {
	MilliCPU         int64 `json:"milli_cpu"`
	Memory           int64 `json:"memory"`
	NvidiaGPU        int64 `json:"nvidia_gpu"`
	EphemeralStorage int64 `json:"ephemeral_storage"`
	// We store allowedPodNumber (which is Node.Status.Allocatable.Pods().Value())
	// explicitly as int, to avoid conversions and improve performance.
	AllowedPodNumber int `json:"allowed_pod_number"`

	Name           string `json:"node_name"`
	RawUsedCPU     int64  `json:"-"`
	RawUsedMem     int64  `json:"-"`
	RawUsedStorage int64  `json:"-"`

	CapacityCPU     int64 `json:"capacity_cpu"`
	CapacityMem     int64 `json:"capacity_mem"`
	CapacityStorage int64 `json:"capacity_storage"`
	CapacityPods    int64 `json:"capacity_pod"`
	UsedCPU         int64 `json:"used_cpu"`
	UsedMem         int64 `json:"used_mem"`
	UsedStorage     int64 `json:"used_storage"`
	UsedPods        int64 `json:"used_pod"`
	DiskPressure    bool  `json:"disk_pressure"`
	MemoryPressure  bool  `json:"memory_pressure"`
	PIDPressure     bool  `json:"pid_pressure"`
	Ready           bool  `json:"ready"`
}

// NewResource creates a Resource from ResourceList
func NewResource(rl corev1.ResourceList) *NodeResource {
	r := &NodeResource{}
	r.Add(rl)
	return r
}

func NewNodeResource(name string, rl corev1.NodeStatus) *NodeResource {
	r := &NodeResource{Name: name}
	for rName, rQuant := range rl.Capacity {
		switch rName {
		case corev1.ResourceCPU:
			r.CapacityCPU = rQuant.MilliValue() / 1000
		case corev1.ResourceMemory:
			r.CapacityMem = rQuant.Value() / 1024 / 1024
		case corev1.ResourceEphemeralStorage:
			r.CapacityStorage = rQuant.Value() / 1024 / 1024
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

// Add adds ResourceList into Resource.
func (r *NodeResource) Add(rl corev1.ResourceList) {
	if r == nil {
		return
	}

	for rName, rQuant := range rl {
		switch rName {
		case corev1.ResourceCPU:
			r.MilliCPU += rQuant.MilliValue()
		case corev1.ResourceMemory:
			r.Memory += rQuant.Value()
		case corev1.ResourcePods:
			r.AllowedPodNumber += int(rQuant.Value())
		case corev1.ResourceEphemeralStorage:
			r.EphemeralStorage += rQuant.Value()
		}
	}
}
