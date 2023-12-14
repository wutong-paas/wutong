package handler

import (
	"slices"
	"strings"

	"github.com/wutong-paas/wutong/api/client/kube"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

// NodeHandler -
type NodeHandler interface {
	ListVMNodeSelectorLabels() ([]string, error)
}

// NewClusterHandler -
func NewNodeHandler(clientset kubernetes.Interface) NodeHandler {
	return &nodeAction{
		clientset: clientset,
	}
}

type nodeAction struct {
	clientset kubernetes.Interface
}

func (a *nodeAction) ListVMNodeSelectorLabels() ([]string, error) {
	var result []string
	nodes, err := kube.GetCachedResources(a.clientset).NodeLister.List(labels.SelectorFromSet(labels.Set{
		"wutong.io/vm-schedulable": "true",
	}))

	for _, node := range nodes {
		for k := range node.Labels {
			if label, ok := strings.CutPrefix(k, "vm-node-selector.wutong.io/"); ok && !slices.Contains(result, label) {
				result = append(result, label)
			}
		}
	}

	return result, err
}
