package handler

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/wutong-paas/wutong/api/model"
	"github.com/wutong-paas/wutong/pkg/kube"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

// TestCordonNode 测试禁止节点调度
func TestCordonNode(t *testing.T) {
	testdata := []struct {
		nodeName  string
		returnErr error
	}{
		{
			// 集群节点名称，正常节点测试
			nodeName:  "kind-01-control-plane",
			returnErr: nil,
		},
		{
			// 一个不存在的集群节点名称，错误测试
			nodeName:  "kind-01-node-not-exist",
			returnErr: errors.New("nodes \"kind-01-node-not-exist\" not found"),
		},
	}

	for _, test := range testdata {
		clientset := kube.RegionClientset()
		nodeAction := NewNodeHandler(clientset, nil)
		// 禁止节点调度
		err := nodeAction.CordonNode(test.nodeName, &model.CordonNodeRequest{
			EvictPods: false,
		})
		if err != test.returnErr {
			t.Fatalf("error: %v, expected: %v", err, test.returnErr)
		}

		// 禁止节点调度成功，确认节点状态
		if err == nil {
			node, err := kube.RegionClientset().CoreV1().Nodes().Get(context.Background(), test.nodeName, metav1.GetOptions{})
			if err != nil {
				t.Fatal(err)
			}
			// 如果节点不可调度状态不为 true，测试失败
			if node.Spec.Unschedulable != true {
				t.Fatalf("error: %v, expected: %v", node.Spec.Unschedulable, true)
			}
		}
	}
}

// TestUncordonNode 测试允许节点调度
func TestUncordonNode(t *testing.T) {
	testdata := []struct {
		nodeName  string
		returnErr error
	}{
		{
			// 集群节点名称，正常节点测试
			nodeName:  "kind-01-control-plane",
			returnErr: nil,
		},
		{
			// 一个不存在的集群节点名称，错误测试
			nodeName:  "kind-01-node-not-exist",
			returnErr: errors.New("nodes \"kind-01-node-not-exist\" not found"),
		},
	}

	for _, test := range testdata {
		clientset := kube.RegionClientset()
		nodeAction := NewNodeHandler(clientset, nil)
		// 允许节点调度
		err := nodeAction.UncordonNode(test.nodeName)
		if err != test.returnErr {
			t.Fatalf("error: %v, expected: %v", err, test.returnErr)
		}

		// 允许节点调度成功，确认节点状态
		if err == nil {
			node, err := kube.RegionClientset().CoreV1().Nodes().Get(context.Background(), test.nodeName, metav1.GetOptions{})
			if err != nil {
				t.Fatal(err)
			}
			// 如果节点不可调度状态不为 false，测试失败
			if node.Spec.Unschedulable != false {
				t.Fatalf("error: %v, expected: %v", node.Spec.Unschedulable, true)
			}
		}
	}
}
