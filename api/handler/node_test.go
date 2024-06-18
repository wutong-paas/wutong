package handler

import (
	"encoding/json"
	"testing"

	"github.com/wutong-paas/wutong/api/model"
	"github.com/wutong-paas/wutong/pkg/kube"
)

func TestListNodes(t *testing.T) {
	clientset := kube.RegionClientset()
	nodeAction := NewNodeHandler(clientset, nil)
	nodes, err := nodeAction.ListNodes("")
	if err != nil {
		t.Fatal(err)
	}
	for _, node := range nodes.Nodes {
		t.Log(node.Name)
	}
	b, _ := json.Marshal(nodes)
	t.Log(string(b))
	t.Log("success")
}

func TestCordonNode(t *testing.T) {
	clientset := kube.RegionClientset()
	nodeAction := NewNodeHandler(clientset, nil)
	err := nodeAction.CordonNode("kind-01-control-plane", &model.CordonNodeRequest{
		EvictPods: false,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Log("success")
}

func TestUncordonNode(t *testing.T) {
	clientset := kube.RegionClientset()
	nodeAction := NewNodeHandler(clientset, nil)
	err := nodeAction.UncordonNode("kind-01-control-plane")
	if err != nil {
		t.Fatal(err)
	}
	t.Log("success")
}
