package handler

import (
	"context"
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/wutong-paas/wutong/api/model"
	"github.com/wutong-paas/wutong/pkg/kube"
	"github.com/wutong-paas/wutong/pkg/prometheus"
	"github.com/wutong-paas/wutong/util"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"
)

// NodeHandler -
type NodeHandler interface {
	ListNodes(query string) (*model.ListNodeResponse, error)
	GetNode(nodeName string) (*model.GetNodeResponse, error)
	GetNodeLabels(nodeName string) ([]model.Label, error)
	GetCommonLabels(nodeName string) ([]model.Label, error)
	GetVMSchedulingLabels(nodeName string) ([]model.Label, error)
	SetNodeLabel(nodeName string, req *model.SetNodeLabelRequest) error
	DeleteNodeLabel(nodeName string, req *model.DeleteNodeLabelRequest) error
	GetNodeAnnotations(nodeName string) ([]model.Annotation, error)
	SetNodeAnnotation(nodeName string, req *model.SetNodeAnnotationRequest) error
	DeleteNodeAnnotation(nodeName string, req *model.DeleteNodeAnnotationRequest) error
	GetNodeTaints(nodeName string) ([]model.Taint, error)
	TaintNode(nodeName string, req *model.TaintNodeRequest) error
	DeleteTaintNode(nodeName string, req *model.DeleteTaintNodeRequest) error
	CordonNode(nodeName string, req *model.CordonNodeRequest) error
	UncordonNode(nodeName string) error
	SetVMSchedulingLabel(nodeName string, req *model.SetVMSchedulingLabelRequest) error
	DeleteVMSchedulingLabel(nodeName string, req *model.DeleteVMSchedulingLabelRequest) error
}

// NewNodeHandler -
func NewNodeHandler(clientset kubernetes.Interface, promcli prometheus.Interface) NodeHandler {
	return &nodeAction{
		clientset: clientset,
		promcli:   promcli,
	}
}

type nodeAction struct {
	clientset kubernetes.Interface
	promcli   prometheus.Interface
}

func (a *nodeAction) ListNodes(query string) (*model.ListNodeResponse, error) {
	var result model.ListNodeResponse
	nodes, err := kube.GetCachedResources(a.clientset).NodeLister.List(labels.Everything())

	sort.Slice(nodes, func(i, j int) bool {
		return !nodes[i].CreationTimestamp.After(nodes[j].CreationTimestamp.Time)
	})

	if query != "" {
		query = strings.ToLower(query)
		var filteredNodes []*corev1.Node
		for _, node := range nodes {
			if strings.Contains(strings.ToLower(node.Name), query) {
				filteredNodes = append(filteredNodes, node)
			}
		}
		nodes = filteredNodes
	}

	for _, node := range nodes {
		nodeInfo := a.nodeInfo(node)
		result.Nodes = append(result.Nodes, nodeInfo)
	}

	result.Total = len(result.Nodes)
	return &result, err
}

func (a *nodeAction) GetNode(nodeName string) (*model.GetNodeResponse, error) {
	node, err := kube.GetCachedResources(a.clientset).NodeLister.Get(nodeName)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			return nil, fmt.Errorf("节点 %s 不存在", nodeName)

		}
	}

	nodeInfo := a.nodeInfo(node)

	nodeProfile := model.NodeProfile{
		NodeInfo:    nodeInfo,
		Labels:      nodeLabels(node),
		Annotations: nodeAnnotations(node),
		Taints:      nodeTaints(node),
	}
	result := model.GetNodeResponse{
		NodeProfile: nodeProfile,
	}
	return &result, nil
}

func (a *nodeAction) GetNodeLabels(nodeName string) ([]model.Label, error) {
	node, err := kube.GetCachedResources(a.clientset).NodeLister.Get(nodeName)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			return nil, fmt.Errorf("节点 %s 不存在", nodeName)

		}
	}

	return nodeLabels(node), nil
}

func (a *nodeAction) GetCommonLabels(nodeName string) ([]model.Label, error) {
	node, err := kube.GetCachedResources(a.clientset).NodeLister.Get(nodeName)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			return nil, fmt.Errorf("节点 %s 不存在", nodeName)

		}
	}

	var result []model.Label

	labels := nodeLabels(node)
	for _, label := range labels {
		if label.IsVMSchedulingLabel {
			continue
		}
		result = append(result, model.Label{
			Key:   label.Key,
			Value: label.Value,
		})
	}

	return result, nil
}

func (a *nodeAction) GetVMSchedulingLabels(nodeName string) ([]model.Label, error) {
	node, err := kube.GetCachedResources(a.clientset).NodeLister.Get(nodeName)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			return nil, fmt.Errorf("节点 %s 不存在", nodeName)
		}
	}

	var result []model.Label

	labels := nodeLabels(node)
	for _, label := range labels {
		if label.IsVMSchedulingLabel {
			result = append(result, model.Label{
				Key:   label.Key,
				Value: label.Value,
			})
		}
	}

	return result, nil
}

func (a *nodeAction) SetNodeLabel(nodeName string, req *model.SetNodeLabelRequest) error {
	req.Key = strings.TrimSpace(req.Key)
	if req.Key == "" {
		return fmt.Errorf("标签键不能为空！")
	}
	node, err := kube.GetCachedResources(a.clientset).NodeLister.Get(nodeName)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			return fmt.Errorf("节点 %s 不存在", nodeName)
		}
		logrus.Errorf("failed to get node %s: %v", nodeName, err)
		return fmt.Errorf("获取节点 %s 信息失败！", nodeName)
	}

	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		node.Labels[req.Key] = req.Value
		_, err = a.clientset.CoreV1().Nodes().Update(context.Background(), node, metav1.UpdateOptions{})
		if err != nil {
			node, err = a.clientset.CoreV1().Nodes().Get(context.Background(), nodeName, metav1.GetOptions{})
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		logrus.Errorf("failed to add label %s=%s to node %s: %v", req.Key, req.Value, nodeName, err)
		return fmt.Errorf("节点 %s 添加标签 %s=%s 失败！", nodeName, req.Key, req.Value)
	}
	return nil
}

func (a *nodeAction) DeleteNodeLabel(nodeName string, req *model.DeleteNodeLabelRequest) error {
	req.Key = strings.TrimSpace(req.Key)
	if req.Key == "" {
		return fmt.Errorf("标签键不能为空！")
	}
	node, err := kube.GetCachedResources(a.clientset).NodeLister.Get(nodeName)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			return fmt.Errorf("节点 %s 不存在", nodeName)
		}
		logrus.Errorf("failed to get node %s: %v", nodeName, err)
		return fmt.Errorf("获取节点 %s 信息失败！", nodeName)
	}

	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		delete(node.Labels, req.Key)
		_, err = a.clientset.CoreV1().Nodes().Update(context.Background(), node, metav1.UpdateOptions{})
		if err != nil {
			node, err = a.clientset.CoreV1().Nodes().Get(context.Background(), nodeName, metav1.GetOptions{})
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		logrus.Errorf("failed to delete label %s from node %s: %v", req.Key, nodeName, err)
		return fmt.Errorf("节点 %s 删除标签 %s 失败！", nodeName, req.Key)
	}
	return nil
}

func (a *nodeAction) GetNodeAnnotations(nodeName string) ([]model.Annotation, error) {
	node, err := kube.GetCachedResources(a.clientset).NodeLister.Get(nodeName)
	if err != nil {
		return nil, err
	}
	return nodeAnnotations(node), nil
}

func (a *nodeAction) SetNodeAnnotation(nodeName string, req *model.SetNodeAnnotationRequest) error {
	req.Key = strings.TrimSpace(req.Key)
	if req.Key == "" {
		return fmt.Errorf("注解键不能为空！")
	}
	node, err := kube.GetCachedResources(a.clientset).NodeLister.Get(nodeName)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			return fmt.Errorf("节点 %s 不存在", nodeName)
		}
		logrus.Errorf("failed to get node %s: %v", nodeName, err)
		return fmt.Errorf("获取节点 %s 信息失败！", nodeName)
	}

	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		node.Annotations[req.Key] = req.Value
		_, err = a.clientset.CoreV1().Nodes().Update(context.Background(), node, metav1.UpdateOptions{})
		if err != nil {
			node, err = a.clientset.CoreV1().Nodes().Get(context.Background(), nodeName, metav1.GetOptions{})
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		logrus.Errorf("failed to add annotation %s=%s to node %s: %v", req.Key, req.Value, nodeName, err)
		return fmt.Errorf("节点 %s 添加注解 %s=%s 失败！", nodeName, req.Key, req.Value)
	}
	return nil
}

func (a *nodeAction) DeleteNodeAnnotation(nodeName string, req *model.DeleteNodeAnnotationRequest) error {
	req.Key = strings.TrimSpace(req.Key)
	if req.Key == "" {
		return fmt.Errorf("注解键不能为空！")
	}
	node, err := kube.GetCachedResources(a.clientset).NodeLister.Get(nodeName)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			return fmt.Errorf("节点 %s 不存在", nodeName)
		}
		logrus.Errorf("failed to get node %s: %v", nodeName, err)
		return fmt.Errorf("获取节点 %s 信息失败！", nodeName)
	}

	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		delete(node.Annotations, req.Key)
		_, err = a.clientset.CoreV1().Nodes().Update(context.Background(), node, metav1.UpdateOptions{})
		if err != nil {
			node, err = a.clientset.CoreV1().Nodes().Get(context.Background(), nodeName, metav1.GetOptions{})
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		logrus.Errorf("failed to delete annotation %s from node %s: %v", req.Key, nodeName, err)
		return fmt.Errorf("节点 %s 删除注解 %s 失败！", nodeName, req.Key)
	}
	return nil
}

func (a *nodeAction) GetNodeTaints(nodeName string) ([]model.Taint, error) {
	node, err := kube.GetCachedResources(a.clientset).NodeLister.Get(nodeName)
	if err != nil {
		return nil, err
	}
	return nodeTaints(node), nil
}

func (a *nodeAction) TaintNode(nodeName string, req *model.TaintNodeRequest) error {
	req.Key = strings.TrimSpace(req.Key)
	if req.Key == "" {
		return fmt.Errorf("污点键不能为空！")
	}
	if req.Key == "node.kubernetes.io/unschedulable" {
		return fmt.Errorf("不允许设置 %s 污点！", req.Key)
	}
	if !slices.Contains([]string{"NoSchedule", "NoExecute", "PreferNoSchedule"}, req.Effect) {
		return fmt.Errorf("污点效果 %s 不合法！", req.Effect)
	}

	node, err := kube.GetCachedResources(a.clientset).NodeLister.Get(nodeName)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			return fmt.Errorf("节点 %s 不存在", nodeName)
		}
		logrus.Errorf("failed to get node %s: %v", nodeName, err)
		return fmt.Errorf("获取节点 %s 信息失败！", nodeName)
	}

	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		for i, taint := range node.Spec.Taints {
			if taint.Key == req.Key {
				// remove if exists
				node.Spec.Taints = append(node.Spec.Taints[:i], node.Spec.Taints[i+1:]...)
			}
		}
		// append
		node.Spec.Taints = append(node.Spec.Taints, corev1.Taint{
			Key:    req.Key,
			Value:  req.Value,
			Effect: corev1.TaintEffect(req.Effect),
		})
		_, err = a.clientset.CoreV1().Nodes().Update(context.Background(), node, metav1.UpdateOptions{})
		if err != nil {
			node, err = a.clientset.CoreV1().Nodes().Get(context.Background(), nodeName, metav1.GetOptions{})
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		logrus.Errorf("failed to taint node %s: %v", nodeName, err)
		return fmt.Errorf("节点 %s 标记污点失败！", nodeName)
	}
	return nil
}

func (a *nodeAction) DeleteTaintNode(nodeName string, req *model.DeleteTaintNodeRequest) error {
	req.Key = strings.TrimSpace(req.Key)
	if req.Key == "" {
		return fmt.Errorf("污点键不能为空！")
	}
	if req.Key == "node.kubernetes.io/unschedulable" {
		return fmt.Errorf("不允许直接清除 %s 污点，可以通过标记节点为可调度来清除！", req.Key)
	}

	node, err := kube.GetCachedResources(a.clientset).NodeLister.Get(nodeName)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			return fmt.Errorf("节点 %s 不存在", nodeName)
		}
		logrus.Errorf("failed to get node %s: %v", nodeName, err)
		return fmt.Errorf("获取节点 %s 信息失败！", nodeName)
	}

	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		for i, taint := range node.Spec.Taints {
			if taint.Key == req.Key {
				// remove if exists
				node.Spec.Taints = append(node.Spec.Taints[:i], node.Spec.Taints[i+1:]...)
			}
		}
		_, err = a.clientset.CoreV1().Nodes().Update(context.Background(), node, metav1.UpdateOptions{})
		if err != nil {
			node, err = a.clientset.CoreV1().Nodes().Get(context.Background(), nodeName, metav1.GetOptions{})
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		logrus.Errorf("failed to delete taint from node %s: %v", nodeName, err)
		return fmt.Errorf("清除节点 %s 标记污点失败！", nodeName)
	}
	return nil
}

func (a *nodeAction) CordonNode(nodeName string, req *model.CordonNodeRequest) error {
	node, err := kube.GetCachedResources(a.clientset).NodeLister.Get(nodeName)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			return fmt.Errorf("节点 %s 不存在", nodeName)
		}
		logrus.Errorf("failed to get node %s: %v", nodeName, err)
		return fmt.Errorf("获取节点 %s 信息失败！", nodeName)
	}

	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		node.Spec.Taints = append(node.Spec.Taints, corev1.Taint{
			Key:       "node.kubernetes.io/unschedulable",
			Effect:    corev1.TaintEffectNoSchedule,
			TimeAdded: util.Ptr(metav1.Now()),
		})
		node.Spec.Unschedulable = true
		_, err = a.clientset.CoreV1().Nodes().Update(context.Background(), node, metav1.UpdateOptions{})
		if err != nil {
			node, err = a.clientset.CoreV1().Nodes().Get(context.Background(), nodeName, metav1.GetOptions{})
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		logrus.Errorf("failed to cordon node %s: %v", nodeName, err)
		return fmt.Errorf("节点 %s 标记为不可调度失败！", nodeName)
	}

	if req.EvictPods {
		podList, err := a.clientset.CoreV1().Pods("").List(context.Background(), metav1.ListOptions{
			FieldSelector: fmt.Sprintf("spec.nodeName=%s", nodeName),
		})
		if err != nil {
			logrus.Errorf("failed to list pods on node %s: %v", nodeName, err)
			return fmt.Errorf("获取节点 %s 上的 Pod 失败！", nodeName)
		}
		if podList != nil && len(podList.Items) > 0 {
			for _, pod := range podList.Items {
				// ingore daemonset pods
				if pod.OwnerReferences != nil {
					for _, owner := range pod.OwnerReferences {
						if owner.Kind == "DaemonSet" {
							continue
						}
					}
				}
				a.clientset.CoreV1().Pods(pod.Namespace).EvictV1(context.Background(), &policyv1.Eviction{
					ObjectMeta: metav1.ObjectMeta{
						Name:      pod.Name,
						Namespace: pod.Namespace,
					},
				})
			}
		}
	}
	return nil
}

func (a *nodeAction) UncordonNode(nodeName string) error {
	node, err := kube.GetCachedResources(a.clientset).NodeLister.Get(nodeName)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			return fmt.Errorf("节点 %s 不存在", nodeName)
		}
		logrus.Errorf("failed to get node %s: %v", nodeName, err)
		return fmt.Errorf("获取节点 %s 信息失败！", nodeName)
	}

	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		for i, taint := range node.Spec.Taints {
			if taint.Key == "node.kubernetes.io/unschedulable" {
				node.Spec.Taints = append(node.Spec.Taints[:i], node.Spec.Taints[i+1:]...)
				break
			}
		}
		node.Spec.Unschedulable = false
		_, err = a.clientset.CoreV1().Nodes().Update(context.Background(), node, metav1.UpdateOptions{})
		if err != nil {
			node, err = a.clientset.CoreV1().Nodes().Get(context.Background(), nodeName, metav1.GetOptions{})
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		logrus.Errorf("failed to cordon node %s: %v", nodeName, err)
		return fmt.Errorf("节点 %s 标记为可调度失败！", nodeName)
	}
	return nil
}

func (a *nodeAction) SetVMSchedulingLabel(nodeName string, req *model.SetVMSchedulingLabelRequest) error {
	req.Key = strings.TrimSpace(req.Key)
	if req.Key == "" {
		return fmt.Errorf("虚拟机调度标签键不能为空！")
	}
	node, err := kube.GetCachedResources(a.clientset).NodeLister.Get(nodeName)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			return fmt.Errorf("节点 %s 不存在", nodeName)
		}
		logrus.Errorf("failed to get node %s: %v", nodeName, err)
		return fmt.Errorf("获取节点 %s 信息失败！", nodeName)
	}

	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		node.Labels[fmt.Sprintf("vm-scheduling-label.wutong.io/%s", req.Key)] = req.Value
		_, err = a.clientset.CoreV1().Nodes().Update(context.Background(), node, metav1.UpdateOptions{})
		if err != nil {
			node, err = a.clientset.CoreV1().Nodes().Get(context.Background(), nodeName, metav1.GetOptions{})
			if err != nil {
				return err
			}
		}
		return nil
	})

	return nil
}

func (a *nodeAction) DeleteVMSchedulingLabel(nodeName string, req *model.DeleteVMSchedulingLabelRequest) error {
	req.Key = strings.TrimSpace(req.Key)
	if req.Key == "" {
		return fmt.Errorf("虚拟机调度标签键不能为空！")
	}
	node, err := kube.GetCachedResources(a.clientset).NodeLister.Get(nodeName)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			return fmt.Errorf("节点 %s 不存在", nodeName)
		}
		logrus.Errorf("failed to get node %s: %v", nodeName, err)
		return fmt.Errorf("获取节点 %s 信息失败！", nodeName)
	}

	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		delete(node.Labels, fmt.Sprintf("vm-scheduling-label.wutong.io/%s", req.Key))
		_, err = a.clientset.CoreV1().Nodes().Update(context.Background(), node, metav1.UpdateOptions{})
		if err != nil {
			node, err = a.clientset.CoreV1().Nodes().Get(context.Background(), nodeName, metav1.GetOptions{})
			if err != nil {
				return err
			}
		}
		return nil
	})

	return nil
}

func (a *nodeAction) nodeInfo(node *corev1.Node) model.NodeInfo {
	var result model.NodeInfo
	if node == nil {
		return result
	}

	containerRuntime, containerRuntimeVersion := a.nodeContainerRuntimeAndVersion(node)
	var cpuUsed = kube.GetNodeCPURequest(node.Name)
	var memoryUsed = kube.GetNodeMemoryRequest(node.Name)
	var podUsed = kube.GetNodePodCount(node.Name)
	nodeInternalIP := nodeInternalIP(node)
	result = model.NodeInfo{
		NodeBaseInfo: model.NodeBaseInfo{
			Name:       node.Name,
			ExternalIP: nodeExternalIP(node),
			InternalIP: nodeInternalIP,
			Roles:      kube.NodeRoles(a.clientset, node),
			OS:         node.Status.NodeInfo.OperatingSystem,
			Arch:       node.Status.NodeInfo.Architecture,
		},
		KubeVersion:             node.Status.NodeInfo.KubeletVersion,
		ContainerRuntime:        containerRuntime,
		ContainerRuntimeVersion: containerRuntimeVersion,
		OSVersion:               node.Status.NodeInfo.OSImage,
		KernelVersion:           node.Status.NodeInfo.KernelVersion,
		Status:                  nodeStatus(node),
		CreatedAt:               node.CreationTimestamp.Local().Format("2006-01-02 15:04:05"),
		PodCIDR:                 node.Spec.PodCIDR,
		CPUCap:                  util.DecimailFromFloat64(float64(node.Status.Capacity.Cpu().MilliValue()) / 1000),
		CPUUsed:                 util.DecimailFromFloat64(float64(cpuUsed) / 1000),
		MemoryCap:               util.DecimailFromFloat64(float64(node.Status.Capacity.Memory().Value()) / 1024 / 1024 / 1024),
		MemoryUsed:              util.DecimailFromFloat64(float64(memoryUsed) / 1024 / 1024 / 1024),
		PodCap:                  node.Status.Capacity.Pods().Value(),
		PodUsed:                 podUsed,
		DiskCap:                 util.DecimailFromFloat64(float64(node.Status.Capacity.StorageEphemeral().Value()) / 1024 / 1024 / 1024),
		Schedulable:             !node.Spec.Unschedulable,
	}
	result.CPUtilizationRate = util.DecimailFromFloat64(result.CPUUsed / result.CPUCap * 100)
	result.MemoryUtilizationRate = util.DecimailFromFloat64(result.MemoryUsed / result.MemoryCap * 100)
	result.PodUtilizationRate = util.DecimailFromFloat64(float64(result.PodUsed) / float64(result.PodCap) * 100)
	diskAvailable := util.DecimailFromFloat64(GetNodeDiskAvailable(node.Name, nodeInternalIP, a.promcli) / 1024 / 1024 / 1024)
	result.DiskUsed = util.DecimailFromFloat64(result.DiskCap - diskAvailable)
	if result.DiskUsed <= 0 {
		result.DiskUsed = 0
	}
	result.DiskUtilizationRate = util.DecimailFromFloat64(result.DiskUsed / result.DiskCap * 100)

	return result
}

func nodeInternalIP(node *corev1.Node) string {
	for _, addr := range node.Status.Addresses {
		if addr.Type == corev1.NodeInternalIP {
			return addr.Address
		}
	}
	return ""
}

func nodeExternalIP(node *corev1.Node) string {
	for _, addr := range node.Status.Addresses {
		if addr.Type == corev1.NodeExternalIP {
			return addr.Address
		}
	}
	return ""
}

func (a *nodeAction) nodeContainerRuntimeAndVersion(node *corev1.Node) (string, string) {
	containerRuntimeVersionLabel := node.Status.NodeInfo.ContainerRuntimeVersion
	if containerRuntimeVersionLabel == "" {
		if criSocket := node.Labels["kubeadm.alpha.kubernetes.io/cri-socket"]; strings.HasSuffix(criSocket, "containerd.sock") {
			return "containerd", ""
		} else if strings.HasSuffix(criSocket, "docker.sock") || strings.HasSuffix(criSocket, "cri-dockerd.sock") {
			return "docker", ""
		}
		return "", ""
	}
	parts := strings.Split(containerRuntimeVersionLabel, "://")
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return containerRuntimeVersionLabel, ""
}

func nodeStatus(node *corev1.Node) string {
	for _, condition := range node.Status.Conditions {
		if condition.Type == corev1.NodeReady {
			if condition.Status == corev1.ConditionTrue {
				return "Ready"
			} else {
				return "NotReady"
			}
		}
	}
	return "Unknown"
}

func nodeLabels(node *corev1.Node) []model.Label {
	var result []model.Label
	for k, v := range node.Labels {
		var found bool
		k, found = strings.CutPrefix(k, "vm-scheduling-label.wutong.io/")
		result = append(result, model.Label{
			Key:                 k,
			Value:               v,
			IsVMSchedulingLabel: found,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Key < result[j].Key
	})

	return result
}

func nodeAnnotations(node *corev1.Node) []model.Annotation {
	var result = make([]model.Annotation, 0)
	for k, v := range node.Annotations {
		result = append(result, model.Annotation{
			Key:   k,
			Value: v,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Key < result[j].Key
	})

	return result
}

func nodeTaints(node *corev1.Node) []model.Taint {
	var result = make([]model.Taint, 0)
	for _, taint := range node.Spec.Taints {
		if taint.Key == "node.kubernetes.io/unschedulable" {
			continue
		}
		result = append(result, model.Taint{
			Key:    string(taint.Key),
			Value:  string(taint.Value),
			Effect: string(taint.Effect),
		})
	}

	return result
}
