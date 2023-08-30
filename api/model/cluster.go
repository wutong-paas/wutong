package model

import (
	"fmt"
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

// ClusterResource -
type ClusterResource struct {
	AllNode              int             `json:"all_node"`
	NotReadyNode         int             `json:"notready_node"`
	ComputeNode          int             `json:"compute_node"`
	CapCPU               float32         `json:"cap_cpu"`
	CapMem               float32         `json:"cap_mem"`
	ReqCPU               float32         `json:"req_cpu"`
	ReqMem               float32         `json:"req_mem"`
	WutongReqMem         float32         `json:"wt_req_mem"` // Resources to embody wutong scheduling
	WutongReqCPU         float32         `json:"wt_req_cpu"`
	TotalCapacityPods    int64           `json:"total_capacity_pods"`
	TotalUsedPods        int64           `json:"total_used_pods"`
	TotalCapacityStorage float32         `json:"total_capacity_storage"`
	TotalUsedStorage     float32         `json:"total_used_storage"`
	NodeResources        []*NodeResource `json:"node_resources"`
	TenantEnvPods        map[string]int  `json:"tenant_env_pods"`
}

type ClusterEventLevel string

func (l ClusterEventLevel) String() string {
	return string(l)
}

const (
	ClusterEventLevelNormal  ClusterEventLevel = "Normal"
	ClusterEventLevelGeneral ClusterEventLevel = "General"
	ClusterEventLevelWarning ClusterEventLevel = "Warning"
	ClusterEventLevelFatal   ClusterEventLevel = "Urgent"
)

// ClusterEvent
type ClusterEvent struct {
	Level   ClusterEventLevel `json:"level"`
	Message string            `json:"message"`
}

func ClusterEventFrom(event *corev1.Event, clientset *kubernetes.Clientset) *ClusterEvent {
	if event.Type == ClusterEventLevelNormal.String() {
		return nil
	}

	switch event.InvolvedObject.Kind {
	case "Pod":
		return podEvent(event, clientset)
	case "Node":
		return nodeEvent(event, clientset)
	}
	return nil
}

func podEvent(event *corev1.Event, clientset *kubernetes.Clientset) *ClusterEvent {
	var message string
	switch event.Reason {
	case "FailedKillPod":
		message = fmt.Sprintf("容器[%s/%s]退出失败", event.InvolvedObject.Namespace, event.InvolvedObject.Name)
	case "BackOff":
		message = fmt.Sprintf("容器[%s/%s]意外退出", event.InvolvedObject.Namespace, event.InvolvedObject.Name)
	case "FailedMount", "FailedAttachVolume":
		message = fmt.Sprintf("容器[%s/%s]挂载错误", event.InvolvedObject.Namespace, event.InvolvedObject.Name)
	case "Unhealthy":
		message = fmt.Sprintf("容器[%s/%s]未通过健康检查", event.InvolvedObject.Namespace, event.InvolvedObject.Name)
	case "FailedScheduling":
		message = fmt.Sprintf("容器[%s/%s]调度失败", event.InvolvedObject.Namespace, event.InvolvedObject.Name)
	default:
		return nil
	}
	return &ClusterEvent{
		Level:   ClusterEventLevelWarning,
		Message: message,
	}
}

func nodeEvent(event *corev1.Event, clientset *kubernetes.Clientset) *ClusterEvent {
	var message string
	if strings.Contains(event.Reason, "bind: address already in use") {
		reasonParts := strings.Split(event.Reason, ":")
		if len(reasonParts) < 4 {
			return nil
		}
		if _, err := strconv.Atoi(reasonParts[1]); err != nil {
			return nil
		}
		message = fmt.Sprintf("节点[%s]端口[:%s]已被占用", event.InvolvedObject.Name, reasonParts[1])
		return &ClusterEvent{
			Level:   ClusterEventLevelWarning,
			Message: message,
		}
	}

	switch event.Reason {
	case "NodeHasInsufficientMemory":
		message = fmt.Sprintf("节点[%s]内存不足", event.InvolvedObject.Name)
	case "NodeHasDiskPressure":
		message = fmt.Sprintf("节点[%s]磁盘不足", event.InvolvedObject.Name)
	case "NodeHasInsufficientPID":
		message = fmt.Sprintf("节点[%s]PID不足", event.InvolvedObject.Name)
	default:
		return nil
	}
	return &ClusterEvent{
		Level:   ClusterEventLevelWarning,
		Message: message,
	}
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
