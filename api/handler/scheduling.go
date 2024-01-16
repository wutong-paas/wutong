package handler

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/wutong-paas/wutong/api/client/kube"
	"github.com/wutong-paas/wutong/api/model"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

// SchedulingHandler -
type SchedulingHandler interface {
	ListSchedulingNodes() (*model.ListSchedulingNodesResponse, error)
	ListSchedulingTaints() (*model.ListSchedulingTaintsResponse, error)

	ListVMSchedulingLabels() ([]string, error)
	ListSchedulingLabels() (*model.ListSchedulingLabelsResponse, error)
}

// NewSchedulingHandler -
func NewSchedulingHandler(clientset kubernetes.Interface) SchedulingHandler {
	return &schedulingAction{
		clientset: clientset,
	}
}

type schedulingAction struct {
	clientset kubernetes.Interface
}

func (a *schedulingAction) ListSchedulingNodes() (*model.ListSchedulingNodesResponse, error) {
	var result model.ListSchedulingNodesResponse
	nodes, err := kube.GetCachedResources(a.clientset).NodeLister.List(labels.Everything())

	sort.Slice(nodes, func(i, j int) bool {
		return !nodes[i].CreationTimestamp.After(nodes[j].CreationTimestamp.Time)
	})

	for _, node := range nodes {
		item := model.NodeBaseInfo{
			Name:       node.Name,
			ExternalIP: nodeExternalIP(node),
			Roles:      kube.NodeRoles(a.clientset, node),
			OS:         node.Status.NodeInfo.OperatingSystem,
			Arch:       node.Status.NodeInfo.Architecture,
		}
		result.Nodes = append(result.Nodes, item)
	}

	result.Total = len(result.Nodes)
	return &result, err
}

func (a *schedulingAction) ListSchedulingTaints() (*model.ListSchedulingTaintsResponse, error) {
	var result model.ListSchedulingTaintsResponse
	nodes, err := kube.GetCachedResources(a.clientset).NodeLister.List(labels.Everything())

	for _, node := range nodes {
		if len(node.Spec.Taints) > 0 {
			for _, taint := range node.Spec.Taints {
				result.Taints = result.Taints.TryAppend(taint)
			}
		}
	}
	return &result, err
}

func (a *schedulingAction) ListVMSchedulingLabels() ([]string, error) {
	var result []string
	nodes, err := kube.GetCachedResources(a.clientset).NodeLister.List(labels.Everything())

	for _, node := range nodes {
		for k, v := range node.Labels {
			if label, ok := strings.CutPrefix(k, "vm-scheduling-label.wutong.io/"); ok {
				if v != "" {
					label = fmt.Sprintf("%s=%s", label, v)
				}
				if !slices.Contains(result, label) {
					result = append(result, label)
				}
			}
		}
	}

	slices.Sort(result)

	return result, err
}

func (a *schedulingAction) ListSchedulingLabels() (*model.ListSchedulingLabelsResponse, error) {
	var labelList []model.KeyValue
	nodes, err := kube.GetCachedResources(a.clientset).NodeLister.List(labels.Everything())

	for _, node := range nodes {
		for k, v := range node.Labels {
			labelList = append(labelList, model.KeyValue{
				Key:   k,
				Value: v,
			})
		}
	}

	slices.SortFunc(labelList, func(a, b model.KeyValue) int {
		if a.Key < b.Key {
			return -1
		} else {
			return 1
		}
	})

	return &model.ListSchedulingLabelsResponse{
		Labels: labelList,
	}, err
}
