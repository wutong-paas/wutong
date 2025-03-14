// WUTONG, Application Management Platform
// Copyright (C) 2014-2017 Wutong Co., Ltd.

// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version. For any non-GPL usage of Wutong,
// one or multiple Commercial Licenses authorized by Wutong Co., Ltd.
// must be obtained first.

// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.

// You should have received a copy of the GNU General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package kubecache

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/eapache/channels"
	"github.com/pquerna/ffjson/ffjson"
	"github.com/sirupsen/logrus"
	conf "github.com/wutong-paas/wutong/cmd/node/option"
	"github.com/wutong-paas/wutong/node/nodem/client"
	v1 "k8s.io/api/core/v1"
	"k8s.io/api/policy/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
)

// EventType -
type EventType string

const (
	//EvictionKind EvictionKind
	EvictionKind = "Eviction"
	//EvictionSubresource EvictionSubresource
	EvictionSubresource = "pods/eviction"
	// CreateEvent event associated with new objects in an informer
	CreateEvent EventType = "CREATE"
	// UpdateEvent event associated with an object update in an informer
	UpdateEvent EventType = "UPDATE"
	// DeleteEvent event associated when an object is removed from an informer
	DeleteEvent EventType = "DELETE"
)

// Event holds the context of an event.
type Event struct {
	Type EventType
	Obj  interface{}
}

type l map[string]string

func (l l) contains(k, v string) bool {
	if l == nil {
		return false
	}
	if val, ok := l[k]; !ok || val != v {
		return false
	}
	return true
}

// KubeClient KubeClient
type KubeClient interface {
	UpK8sNode(*client.HostNode) (*v1.Node, error)
	DownK8sNode(nodename string) error
	GetAllPods() (pods []*v1.Pod, err error)
	GetPods(namespace string) (pods []*v1.Pod, err error)
	GetPodsBySelector(namespace string, selector labels.Selector) (pods []*v1.Pod, err error)
	GetNodeByName(nodename string) (*v1.Node, error)
	GetNodes() ([]*v1.Node, error)
	GetNode(nodeName string) (*v1.Node, error)
	CordonOrUnCordon(nodeName string, drain bool) (*v1.Node, error)
	UpdateLabels(nodeName string, labels map[string]string) (*v1.Node, error)
	DeleteOrEvictPodsSimple(nodeName string) error
	GetPodsByNodes(nodeName string) (pods []v1.Pod, err error)
	GetEndpoints(namespace string, selector labels.Selector) ([]*v1.Endpoints, error)
	GetServices(namespace string, selector labels.Selector) ([]*v1.Service, error)
	GetConfig(namespace string, selector labels.Selector) ([]*v1.ConfigMap, error)
	Stop()
}

// NewKubeClient NewKubeClient
func NewKubeClient(cfg *conf.Conf, clientset kubernetes.Interface) (KubeClient, error) {
	stop := make(chan struct{})
	sharedInformers := informers.NewSharedInformerFactoryWithOptions(clientset, cfg.MinResyncPeriod)

	sharedInformers.Core().V1().Endpoints().Informer()
	sharedInformers.Core().V1().Services().Informer()
	sharedInformers.Core().V1().ConfigMaps().Informer()
	sharedInformers.Core().V1().Nodes().Informer()
	sharedInformers.Core().V1().Pods().Informer()
	sharedInformers.Start(stop)
	return &kubeClient{
		kubeclient:      clientset,
		stop:            stop,
		sharedInformers: sharedInformers,
	}, nil
}

type kubeClient struct {
	kubeclient      kubernetes.Interface
	sharedInformers informers.SharedInformerFactory
	stop            chan struct{}
	updateCh        *channels.RingChannel
}

func (k *kubeClient) Stop() {
	if k.stop != nil {
		close(k.stop)
	}
}

// GetNodeByName get node
func (k *kubeClient) GetNodeByName(nodename string) (*v1.Node, error) {
	return k.sharedInformers.Core().V1().Nodes().Lister().Get(nodename)
}

// CordonOrUnCordon node scheduler
// drain:true can't scheduler ,false can scheduler
func (k *kubeClient) CordonOrUnCordon(nodeName string, drain bool) (*v1.Node, error) {
	data := fmt.Sprintf(`{"spec":{"unschedulable":%t}}`, drain)
	node, err := k.kubeclient.CoreV1().Nodes().Patch(context.Background(), nodeName, types.StrategicMergePatchType, []byte(data), metav1.PatchOptions{})
	if err != nil {
		return node, err
	}
	return node, nil
}

// UpdateLabels update lables
func (k *kubeClient) UpdateLabels(nodeName string, labels map[string]string) (*v1.Node, error) {
	labelStr, err := ffjson.Marshal(labels)
	if err != nil {
		return nil, err
	}
	data := fmt.Sprintf(`{"metadata":{"labels":%s}}`, string(labelStr))
	node, err := k.kubeclient.CoreV1().Nodes().Patch(context.Background(), nodeName, types.StrategicMergePatchType, []byte(data), metav1.PatchOptions{})
	if err != nil {
		return node, err
	}
	return node, nil
}

// DeleteOrEvictPodsSimple Evict the Pod from a node
func (k *kubeClient) DeleteOrEvictPodsSimple(nodeName string) error {
	pods, err := k.GetPodsByNodes(nodeName)
	if err != nil {
		logrus.Infof("get pods of node %s failed ", nodeName)
		return err
	}
	policyGroupVersion, err := k.SupportEviction()
	if err != nil {
		return err
	}
	if policyGroupVersion == "" {
		return fmt.Errorf("the server can not support eviction subresource")
	}
	for _, v := range pods {
		k.evictPod(v, policyGroupVersion)
	}
	return nil
}
func (k *kubeClient) GetPodsByNodes(nodeName string) (pods []v1.Pod, err error) {
	podList, err := k.kubeclient.CoreV1().Pods(metav1.NamespaceAll).List(context.Background(), metav1.ListOptions{
		FieldSelector: fields.SelectorFromSet(fields.Set{"spec.nodeName": nodeName}).String()})
	if err != nil {
		return pods, err
	}
	pods = append(pods, podList.Items...)
	return pods, nil
}

// evictPod 驱离POD
func (k *kubeClient) evictPod(pod v1.Pod, policyGroupVersion string) error {
	deleteOptions := &metav1.DeleteOptions{}
	eviction := &v1beta1.Eviction{
		TypeMeta: metav1.TypeMeta{
			APIVersion: policyGroupVersion,
			Kind:       EvictionKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      pod.Name,
			Namespace: pod.Namespace,
		},
		DeleteOptions: deleteOptions,
	}
	// Remember to change change the URL manipulation func when Evction's version change
	return k.kubeclient.PolicyV1beta1().Evictions(eviction.Namespace).Evict(context.Background(), eviction)
}

func waitForDelete(pods []v1.Pod, interval, timeout time.Duration, usingEviction bool, getPodFn func(string, string) (*v1.Pod, error)) ([]v1.Pod, error) {
	var verbStr string
	if usingEviction {
		verbStr = "evicted"
	} else {
		verbStr = "deleted"
	}
	err := wait.PollImmediate(interval, timeout, func() (bool, error) {
		pendingPods := []v1.Pod{}
		for i, pod := range pods {
			p, err := getPodFn(pod.Namespace, pod.Name)
			if apierrors.IsNotFound(err) || (p != nil && p.ObjectMeta.UID != pod.ObjectMeta.UID) {
				fmt.Println(verbStr)
				//cmdutil.PrintSuccess(o.mapper, false, o.Out, "pod", pod.Name, false, verbStr)//todo
				continue
			} else if err != nil {
				return false, err
			} else {
				pendingPods = append(pendingPods, pods[i])
			}
		}
		pods = pendingPods
		if len(pendingPods) > 0 {
			return false, nil
		}
		return true, nil
	})
	return pods, err
}
func (k *kubeClient) deletePod(pod v1.Pod) error {
	deleteOptions := metav1.DeleteOptions{}
	//if GracePeriodSeconds >= 0 {
	//if 1 >= 0 {
	//	//gracePeriodSeconds := int64(GracePeriodSeconds)
	//	gracePeriodSeconds := int64(1)
	//	deleteOptions.GracePeriodSeconds = &gracePeriodSeconds
	//}
	gracePeriodSeconds := int64(1)
	deleteOptions.GracePeriodSeconds = &gracePeriodSeconds
	return k.kubeclient.CoreV1().Pods(pod.Namespace).Delete(context.Background(), pod.Name, deleteOptions)
}

// SupportEviction uses Discovery API to find out if the server support eviction subresource
// If support, it will return its groupVersion; Otherwise, it will return ""
func (k *kubeClient) SupportEviction() (string, error) {
	discoveryClient := k.kubeclient.Discovery()
	groupList, err := discoveryClient.ServerGroups()
	if err != nil {
		return "", err
	}
	foundPolicyGroup := false
	var policyGroupVersion string
	for _, group := range groupList.Groups {
		if group.Name == "policy" {
			foundPolicyGroup = true
			policyGroupVersion = group.PreferredVersion.GroupVersion
			break
		}
	}
	if !foundPolicyGroup {
		return "", nil
	}
	resourceList, err := discoveryClient.ServerResourcesForGroupVersion("v1")
	if err != nil {
		return "", err
	}
	for _, resource := range resourceList.APIResources {
		if resource.Name == EvictionSubresource && resource.Kind == EvictionKind {
			return policyGroupVersion, nil
		}
	}
	return "", nil
}

// GetAllPods get all pods
func (k *kubeClient) GetAllPods() (pods []*v1.Pod, err error) {
	podList, err := k.sharedInformers.Core().V1().Pods().Lister().List(labels.Everything())
	if err != nil {
		return pods, err
	}
	return podList, nil
}

// GetAllPods get all pods
func (k *kubeClient) GetPods(namespace string) (pods []*v1.Pod, err error) {
	podList, err := k.sharedInformers.Core().V1().Pods().Lister().Pods(namespace).List(labels.Everything())
	if err != nil {
		return pods, err
	}
	return podList, nil
}

// DeleteNode  k8s节点下线
func (k *kubeClient) DownK8sNode(nodename string) error {
	_, err := k.GetNodeByName(nodename)
	if err != nil {
		logrus.Infof("get k8s node %s failed ", nodename)
		return err
	}
	//节点禁止调度
	_, err = k.CordonOrUnCordon(nodename, true)
	if err != nil {
		logrus.Infof("cordon node %s failed ", nodename)
		return err
	}
	//节点pod驱离
	err = k.DeleteOrEvictPodsSimple(nodename)
	if err != nil {
		logrus.Infof("delete or evict pods of node  %s failed ", nodename)
		return err
	}
	//删除节点
	err = k.deleteNodeWithoutPods(nodename)
	if err != nil {
		logrus.Infof("delete node with given name failed  %s failed ", nodename)
		return err
	}
	return nil
}

func (k *kubeClient) deleteNodeWithoutPods(name string) error {
	opt := metav1.DeleteOptions{}
	err := k.kubeclient.CoreV1().Nodes().Delete(context.Background(), name, opt)
	if err != nil {
		return err
	}
	return nil
}

// UpK8sNode create k8s node by wutong node info
func (k *kubeClient) UpK8sNode(wutongNode *client.HostNode) (*v1.Node, error) {
	capacity := make(v1.ResourceList)
	capacity[v1.ResourceCPU] = *resource.NewQuantity(wutongNode.AvailableCPU, resource.BinarySI)
	capacity[v1.ResourceMemory] = *resource.NewQuantity(wutongNode.AvailableMemory, resource.BinarySI)
	lbs := wutongNode.MergeLabels()
	node := &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:   strings.ToLower(wutongNode.ID),
			Labels: lbs,
		},
		Spec: v1.NodeSpec{
			Unschedulable: wutongNode.Unschedulable,
			PodCIDR:       wutongNode.PodCIDR,
		},
		Status: v1.NodeStatus{
			Capacity:    capacity,
			Allocatable: capacity,
			Addresses: []v1.NodeAddress{
				{Type: v1.NodeHostName, Address: wutongNode.HostName},
				{Type: v1.NodeInternalIP, Address: wutongNode.InternalIP},
				{Type: v1.NodeExternalIP, Address: wutongNode.ExternalIP},
			},
		},
	}
	//set wutong creator lable
	node.Labels["creator"] = "Wutong"
	savedNode, err := k.kubeclient.CoreV1().Nodes().Create(context.Background(), node, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	logrus.Infof("creating new node success , details: %v ", savedNode)
	return node, nil
}

func (k *kubeClient) GetPodsBySelector(namespace string, selector labels.Selector) ([]*v1.Pod, error) {
	return k.sharedInformers.Core().V1().Pods().Lister().Pods(namespace).List(selector)
}

func (k *kubeClient) GetEndpoints(namespace string, selector labels.Selector) ([]*v1.Endpoints, error) {
	return k.sharedInformers.Core().V1().Endpoints().Lister().Endpoints(namespace).List(selector)
}
func (k *kubeClient) GetServices(namespace string, selector labels.Selector) ([]*v1.Service, error) {
	return k.sharedInformers.Core().V1().Services().Lister().Services(namespace).List(selector)
}

func (k *kubeClient) GetNodes() ([]*v1.Node, error) {
	nodes, err := k.sharedInformers.Core().V1().Nodes().Lister().List(labels.Everything())
	if err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		list, err := k.kubeclient.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
		if err != nil {
			return nil, err
		}
		for idx := range list.Items {
			node := list.Items[idx]
			nodes = append(nodes, &node)
		}
	}
	return nodes, nil
}

func (k *kubeClient) GetNode(nodeName string) (*v1.Node, error) {
	return k.sharedInformers.Core().V1().Nodes().Lister().Get(nodeName)
}
func (k *kubeClient) GetConfig(namespace string, selector labels.Selector) ([]*v1.ConfigMap, error) {
	return k.sharedInformers.Core().V1().ConfigMaps().Lister().ConfigMaps(namespace).List(selector)
}
