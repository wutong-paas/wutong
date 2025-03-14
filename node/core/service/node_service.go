// Copyright (C) 2014-2018 Wutong Co., Ltd.
// WUTONG, Application Management Platform

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

package service

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/wutong-paas/wutong/cmd/node/option"
	"github.com/wutong-paas/wutong/event"
	"github.com/wutong-paas/wutong/node/api/model"
	"github.com/wutong-paas/wutong/node/kubecache"
	"github.com/wutong-paas/wutong/node/masterserver/node"
	"github.com/wutong-paas/wutong/node/nodem/client"
	"github.com/wutong-paas/wutong/node/utils"
	"github.com/wutong-paas/wutong/util"
	ansibleUtil "github.com/wutong-paas/wutong/util/ansible"
	// etcdutil "github.com/wutong-paas/wutong/util/etcd"
)

// NodeService node service
type NodeService struct {
	c           *option.Conf
	nodecluster *node.Cluster
	kubecli     kubecache.KubeClient
}

// CreateNodeService create
func CreateNodeService(c *option.Conf, nodecluster *node.Cluster, kubecli kubecache.KubeClient) *NodeService {
	// etcdClientArgs := &etcdutil.ClientArgs{
	// 	Endpoints:   c.EtcdEndpoints,
	// 	CaFile:      c.EtcdCaFile,
	// 	CertFile:    c.EtcdCertFile,
	// 	KeyFile:     c.EtcdKeyFile,
	// 	DialTimeout: c.EtcdDialTimeout,
	// }
	if _, err := event.NewLoggerManager(); err != nil {
		logrus.Errorf("create event manager faliure")
	}
	return &NodeService{
		c:           c,
		nodecluster: nodecluster,
		kubecli:     kubecli,
	}
}

// AddNode add node
func (n *NodeService) AddNode(node *client.APIHostNode) (*client.HostNode, *utils.APIHandleError) {
	if n.nodecluster == nil {
		return nil, utils.CreateAPIHandleError(400, fmt.Errorf("this node can not support this api"))
	}

	if err := node.Role.Validation(); err != nil {
		return nil, utils.CreateAPIHandleError(400, err)
	}
	if node.ID == "" {
		node.ID = uuid.New().String()
	}
	if node.InternalIP == "" {
		return nil, utils.CreateAPIHandleError(400, fmt.Errorf("node internal ip can not be empty"))
	}
	if node.RootPass != "" && node.Privatekey != "" {
		return nil, utils.CreateAPIHandleError(400, fmt.Errorf("options private-key and root-pass are conflicting"))
	}
	existNode := n.nodecluster.GetAllNode()
	for _, en := range existNode {
		if node.InternalIP == en.InternalIP {
			return nil, utils.CreateAPIHandleError(400, fmt.Errorf("node internal ip %s is exist", node.InternalIP))
		}
	}
	rbnode := node.Clone()
	rbnode.CreateTime = time.Now()
	n.nodecluster.UpdateNode(rbnode)
	return rbnode, nil
}

// InstallNode install node
func (n *NodeService) InstallNode(node *client.HostNode) *utils.APIHandleError {
	node.Status = client.Installing
	node.NodeStatus.Status = client.Installing
	node.Labels["event_id"] = util.NewUUID()
	n.nodecluster.UpdateNode(node)
	go n.AsynchronousInstall(node, node.Labels["event_id"])
	return nil
}

// write ansible hosts file
func (n *NodeService) writeHostsFile() error {
	hosts, err := n.GetAllNode()
	if err != nil {
		return err.Err
	}
	// use the value of environment if it is empty use default value
	hostsFilePath := os.Getenv("HOSTS_FILE_PATH")
	if hostsFilePath == "" {
		hostsFilePath = "/opt/wutong/wutong-ansible/inventory/hosts"
	}
	installConfPath := os.Getenv("INSTALL_CONF_PATH")
	if installConfPath == "" {
		installConfPath = "/opt/wutong/wutong-ansible/scripts/installer/global.sh"
	}
	erro := ansibleUtil.WriteHostsFile(hostsFilePath, installConfPath, hosts)
	if erro != nil {
		return err
	}
	return nil
}

// UpdateNodeStatus update node status
func (n *NodeService) UpdateNodeStatus(nodeID, status string) *utils.APIHandleError {
	node := n.nodecluster.GetNode(nodeID)
	if node == nil {
		return utils.CreateAPIHandleError(400, errors.New("node can not be found"))
	}
	if status != client.Installing && status != client.InstallFailed && status != client.InstallSuccess {
		return utils.CreateAPIHandleError(400, fmt.Errorf("node can not set status is %s", status))
	}
	node.Status = status
	node.NodeStatus.Status = status
	n.nodecluster.UpdateNode(node)
	return nil
}

// AsynchronousInstall AsynchronousInstall
func (n *NodeService) AsynchronousInstall(node *client.HostNode, eventID string) {
	// write ansible hosts file
	err := n.writeHostsFile()
	if err != nil {
		logrus.Error("write hosts file error ", err.Error())
		return
	}
	// start add node script
	logrus.Infof("Begin install node %s", node.ID)
	// write log to event log
	logger := event.GetLogger(eventID)
	option := ansibleUtil.NodeInstallOption{
		HostRole:   node.Role.String(),
		HostName:   node.HostName,
		InternalIP: node.InternalIP,
		RootPass:   node.RootPass,
		KeyPath:    node.KeyPath,
		NodeID:     node.ID,
		Stdin:      nil,
		Stdout:     logger.GetWriter("node-install", "info"),
		Stderr:     logger.GetWriter("node-install", "err"),
	}

	err = ansibleUtil.RunNodeInstallCmd(option)
	if err != nil {
		logrus.Error("Error executing shell script : ", err)
		node.Status = client.InstallFailed
		node.NodeStatus.Status = client.InstallFailed
		n.nodecluster.UpdateNode(node)
		return
	}
	node.Status = client.InstallSuccess
	node.NodeStatus.Status = client.InstallSuccess
	n.nodecluster.UpdateNode(node)
	logrus.Infof("Install node %s successful", node.ID)
}

// DeleteNode delete node
// only node status is offline and node can be deleted
func (n *NodeService) DeleteNode(nodeID string) *utils.APIHandleError {
	node := n.nodecluster.GetNode(nodeID)
	if node == nil {
		return utils.CreateAPIHandleError(404, fmt.Errorf("node is not found"))
	}
	if node.Status == "running" {
		return utils.CreateAPIHandleError(401, fmt.Errorf("node is running, you must closed node process in node %s", nodeID))
	}
	n.nodecluster.RemoveNode(node.ID)
	_, err := node.DeleteNode()
	if err != nil {
		return utils.CreateAPIHandleErrorFromDBError("delete node", err)
	}
	return nil
}

// GetNode get node info
func (n *NodeService) GetNode(nodeID string) (*client.HostNode, *utils.APIHandleError) {
	node := n.nodecluster.GetNode(nodeID)
	if node == nil {
		return nil, utils.CreateAPIHandleError(404, fmt.Errorf("node no found"))
	}
	return node, nil
}

// GetAllNode get all node
func (n *NodeService) GetAllNode() ([]*client.HostNode, *utils.APIHandleError) {
	if n.nodecluster == nil {
		return nil, utils.CreateAPIHandleError(400, fmt.Errorf("this node can not support this api"))
	}
	nodes := n.nodecluster.GetAllNode()
	sort.Sort(client.NodeList(nodes))
	return nodes, nil
}

// GetServicesHealthy get service health
func (n *NodeService) GetServicesHealthy() (map[string][]map[string]string, *utils.APIHandleError) {
	if n.nodecluster == nil {
		return nil, utils.CreateAPIHandleError(400, fmt.Errorf("this node can not support this api"))
	}
	StatusMap := make(map[string][]map[string]string, 30)
	nodes := n.nodecluster.GetAllNode()
	for _, n := range nodes {
		for _, v := range n.NodeStatus.Conditions {
			status, ok := StatusMap[string(v.Type)]
			if !ok {
				StatusMap[string(v.Type)] = []map[string]string{{"type": string(v.Type), "status": string(v.Status), "message": string(v.Message), "hostname": n.HostName}}
			} else {
				list := status
				list = append(list, map[string]string{"type": string(v.Type), "status": string(v.Status), "message": string(v.Message), "hostname": n.HostName})
				StatusMap[string(v.Type)] = list
			}

		}
	}
	return StatusMap, nil
}

// CordonNode set node is unscheduler
func (n *NodeService) CordonNode(nodeID string, unschedulable bool) *utils.APIHandleError {
	hostNode, apierr := n.GetNode(nodeID)
	if apierr != nil {
		return apierr
	}
	if !hostNode.Role.HasRole(client.ComputeNode) {
		return utils.CreateAPIHandleError(400, fmt.Errorf("this node can not support this api"))
	}
	k8snode := hostNode.NodeStatus.KubeNode
	hostNode.Unschedulable = unschedulable
	//update k8s node unshcedulable status
	if k8snode != nil {
		node, err := n.kubecli.CordonOrUnCordon(hostNode.ID, unschedulable)
		if err != nil {
			return utils.CreateAPIHandleError(500, fmt.Errorf("set node schedulable info error,%s", err.Error()))
		}
		hostNode.UpdateK8sNodeStatus(*node)
	}
	n.nodecluster.UpdateNode(hostNode)
	return nil
}

// GetNodeLabels returns node labels, including system labels and custom labels
func (n *NodeService) GetNodeLabels(nodeID string) (*model.LabelsResp, *utils.APIHandleError) {
	node, err := n.GetNode(nodeID)
	if err != nil {
		return nil, err
	}
	labels := &model.LabelsResp{
		SysLabels:    node.Labels,
		CustomLabels: node.CustomLabels,
	}
	return labels, nil
}

// PutNodeLabel update node label
func (n *NodeService) PutNodeLabel(nodeID string, labels map[string]string) (map[string]string, *utils.APIHandleError) {
	hostNode, apierr := n.GetNode(nodeID)
	if apierr != nil {
		return nil, apierr
	}
	// api can only upate or create custom labels
	if hostNode.CustomLabels == nil {
		hostNode.CustomLabels = make(map[string]string)
	}
	for k, v := range labels {
		hostNode.CustomLabels[k] = v
	}
	if hostNode.Role.HasRole(client.ComputeNode) && hostNode.NodeStatus.KubeNode != nil {
		labels := hostNode.MergeLabels()
		node, err := n.kubecli.UpdateLabels(nodeID, labels)
		if err != nil {
			return nil, utils.CreateAPIHandleError(500, fmt.Errorf("update k8s node labels error,%s", err.Error()))
		}
		hostNode.UpdateK8sNodeStatus(*node)
	}
	n.nodecluster.UpdateNode(hostNode)
	return hostNode.CustomLabels, nil
}

// DeleteNodeLabel delete node label
func (n *NodeService) DeleteNodeLabel(nodeID string, labels map[string]string) (map[string]string, *utils.APIHandleError) {
	hostNode, apierr := n.GetNode(nodeID)
	if apierr != nil {
		return nil, apierr
	}

	newLabels := make(map[string]string)
	for k, v := range hostNode.CustomLabels {
		if _, ok := labels[k]; !ok {
			newLabels[k] = v
		}
	}
	hostNode.CustomLabels = newLabels
	if hostNode.Role.HasRole(client.ComputeNode) && hostNode.NodeStatus.KubeNode != nil {
		labels := hostNode.MergeLabels()
		node, err := n.kubecli.UpdateLabels(nodeID, labels)
		if err != nil {
			return nil, utils.CreateAPIHandleError(500, fmt.Errorf("update k8s node labels error,%s", err.Error()))
		}
		hostNode.UpdateK8sNodeStatus(*node)
	}
	n.nodecluster.UpdateNode(hostNode)
	return hostNode.CustomLabels, nil
}

// DownNode down node
func (n *NodeService) DownNode(nodeID string) (*client.HostNode, *utils.APIHandleError) {
	hostNode, apierr := n.GetNode(nodeID)
	if apierr != nil {
		return nil, apierr
	}
	// add the node from k8s if type is compute
	if hostNode.Role.HasRole(client.ComputeNode) && hostNode.NodeStatus.KubeNode != nil {
		err := n.kubecli.DownK8sNode(hostNode.ID)
		if err != nil {
			logrus.Error("Failed to down node: ", err)
			return nil, utils.CreateAPIHandleError(500, fmt.Errorf("k8s node down error,%s", err.Error()))
		}
	}
	hostNode.Status = client.Offline
	hostNode.NodeStatus.Status = client.Offline
	n.nodecluster.UpdateNode(hostNode)
	return hostNode, nil
}

// UpNode up node
func (n *NodeService) UpNode(nodeID string) (*client.HostNode, *utils.APIHandleError) {
	hostNode, apierr := n.GetNode(nodeID)
	if apierr != nil {
		return nil, apierr
	}
	hostNode.Unschedulable = false
	// add the node to k8s if type is compute
	if hostNode.Role.HasRole(client.ComputeNode) {
		if k8snode, _ := n.kubecli.GetNode(hostNode.ID); k8snode == nil {
			node, err := n.kubecli.UpK8sNode(hostNode)
			if err != nil {
				return nil, utils.CreateAPIHandleError(500, fmt.Errorf("k8s node up error,%s", err.Error()))
			}
			hostNode.UpdateK8sNodeStatus(*node)
		}
	}
	hostNode.Status = client.Running
	hostNode.NodeStatus.Status = client.Running
	n.nodecluster.UpdateNode(hostNode)
	return hostNode, nil
}

// GetNodeResource get node resource
func (n *NodeService) GetNodeResource(nodeUID string) (*model.NodePodResource, *utils.APIHandleError) {
	node, err := n.GetNode(nodeUID)
	if err != nil {
		return nil, err
	}
	if !node.Role.HasRole("compute") {
		return nil, utils.CreateAPIHandleError(401, fmt.Errorf("node is not compute node"))
	}
	ps, error := n.kubecli.GetPodsByNodes(nodeUID)
	if error != nil {
		return nil, utils.CreateAPIHandleError(404, err)
	}
	var cpuTotal = node.AvailableCPU
	var memTotal = node.AvailableMemory
	var cpuLimit int64
	var cpuRequest int64
	var memLimit int64
	var memRequest int64
	for _, v := range ps {
		for _, v := range v.Spec.Containers {
			lc := v.Resources.Limits.Cpu().MilliValue()
			cpuLimit += lc
		}
		for _, v := range v.Spec.Containers {
			lm := v.Resources.Limits.Memory().Value()
			memLimit += lm
		}
		for _, v := range v.Spec.Containers {
			rc := v.Resources.Requests.Cpu().MilliValue()
			cpuRequest += rc
		}
		for _, v := range v.Spec.Containers {
			rm := v.Resources.Requests.Memory().Value()
			memRequest += rm
		}
	}
	var res model.NodePodResource
	res.CPULimits = cpuLimit
	//logrus.Infof("node %s cpu limit is %v",cpuLimit)
	res.CPURequests = cpuRequest
	res.CPU = int(cpuTotal)
	res.MemR = int(memTotal / 1024 / 1024)
	res.CPULimitsR = strconv.FormatFloat(float64(res.CPULimits*100)/float64(res.CPU*1000), 'f', 2, 64)
	res.CPURequestsR = strconv.FormatFloat(float64(res.CPURequests*100)/float64(res.CPU*1000), 'f', 2, 64)
	res.MemoryLimits = memLimit / 1024 / 1024
	res.MemoryLimitsR = strconv.FormatFloat(float64(res.MemoryLimits*100)/float64(res.MemR), 'f', 2, 64)
	res.MemoryRequests = memRequest / 1024 / 1024
	res.MemoryRequestsR = strconv.FormatFloat(float64(res.MemoryRequests*100)/float64(res.MemR), 'f', 2, 64)
	return &res, nil
}

// CheckNode check node install status
func (n *NodeService) CheckNode(nodeUID string) (*model.InstallStatus, *utils.APIHandleError) {

	return nil, nil
}

// DeleteNodeCondition delete node condition
func (n *NodeService) DeleteNodeCondition(nodeUID string, condition client.NodeConditionType) (*client.HostNode, *utils.APIHandleError) {
	node, err := n.GetNode(nodeUID)
	if err != nil {
		return nil, err
	}
	node.DeleteCondition(condition)
	n.nodecluster.UpdateNode(node)
	return node, nil
}
