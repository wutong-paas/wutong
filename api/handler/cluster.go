package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/wutong-paas/wutong/api/client/kube"
	"github.com/wutong-paas/wutong/api/model"
	"github.com/wutong-paas/wutong/api/util"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

// ClusterHandler -
type ClusterHandler interface {
	GetClusterInfo(ctx context.Context) (*model.ClusterResource, error)
	GetClusterEvents(ctx context.Context) ([]model.ClusterEvent, error)
	MavenSettingAdd(ctx context.Context, ms *MavenSetting) *util.APIHandleError
	MavenSettingList(ctx context.Context) (re []MavenSetting)
	MavenSettingUpdate(ctx context.Context, ms *MavenSetting) *util.APIHandleError
	MavenSettingDelete(ctx context.Context, name string) *util.APIHandleError
	MavenSettingDetail(ctx context.Context, name string) (*MavenSetting, *util.APIHandleError)
	Features(ctx context.Context) map[string]bool
}

// NewClusterHandler -
func NewClusterHandler(clientset kubernetes.Interface, WtNamespace, prometheusEndpoint string) ClusterHandler {
	return &clusterAction{
		namespace:          WtNamespace,
		clientset:          clientset,
		prometheusEndpoint: prometheusEndpoint,
	}
}

type clusterAction struct {
	namespace          string
	clientset          kubernetes.Interface
	clusterInfoCache   *model.ClusterResource
	cacheTime          time.Time
	prometheusEndpoint string
}

func (c *clusterAction) GetClusterInfo(ctx context.Context) (*model.ClusterResource, error) {
	timeout, _ := strconv.Atoi(os.Getenv("CLUSTER_INFO_CACHE_TIME"))
	if timeout == 0 {
		// default is 10 minutes
		timeout = 10
	}
	if c.clusterInfoCache != nil && c.cacheTime.Add(time.Minute*time.Duration(timeout)).After(time.Now()) {
		return c.clusterInfoCache, nil
	}
	if c.clusterInfoCache != nil {
		logrus.Debugf("cluster info cache is timeout, will calculate a new value")
	}

	nodes, err := c.listNodes(ctx)
	if err != nil {
		return nil, fmt.Errorf("[GetClusterInfo] list nodes: %v", err)
	}

	nodeCapaticyMetrics, nodeFreeStorageMetrics := c.GetNodeStorageMetrics(NodeCapacityStorageMetric), c.GetNodeStorageMetrics(NodFreeStorageMetric)

	usedNodeList := make([]*corev1.Node, len(nodes))
	for i := range nodes {
		node := nodes[i]
		if !node.Spec.Unschedulable {
			usedNodeList[i] = node
		}
	}

	var wtMemR, wtCPUR int64
	var nodeResources []*model.NodeResource
	var totalCapacityPods, totalUsedPods int64
	var totalCapacityStorage, totalUsedStorage float32
	var totalCapCPU, totalCapMem float32
	var totalReqCPU, totalReqMem float32
	tenantEnvPods := make(map[string]int)

	for i := range usedNodeList {
		node := usedNodeList[i]
		pods, err := c.listPods(ctx, node.Name)
		if err != nil {
			return nil, fmt.Errorf("list pods: %v", err)
		}

		nodeResource := model.NewNodeResource(node.Name, node.Status)
		if ip, ok := internalIPFromNode(node); ok {
			rawCapacity, rawFree := nodeCapaticyMetrics[ip], nodeFreeStorageMetrics[ip]
			if rawCapacity != 0 {
				capacity := rawCapacity / 1024 / 1024 / 1024
				totalCapacityStorage += capacity
				nodeResource.CapacityStorage = util.DecimalFromFloat32(capacity)
				if rawFree != 0 {
					usedStorage := (rawCapacity - rawFree) / 1024 / 1024 / 1024
					nodeResource.UsedStorage = util.DecimalFromFloat32(usedStorage)
					totalUsedStorage += usedStorage
				}
			}
		}
		totalCapacityPods += nodeResource.CapacityPods
		for _, pod := range pods {
			if pod.Status.Phase == corev1.PodRunning || pod.Status.Phase == corev1.PodPending {
				nodeResource.UsedPods++
				for _, c := range pod.Spec.Containers {
					nodeResource.RawUsedCPU += float32(c.Resources.Requests.Cpu().MilliValue())
					nodeResource.RawUsedMem += float32(c.Resources.Requests.Memory().Value())

					if pod.Labels["creator"] == "Wutong" {
						wtMemR += c.Resources.Requests.Memory().Value()
						wtCPUR += c.Resources.Requests.Cpu().MilliValue()
					}
					if pod.Labels["tenant_env_id"] != "" {
						tenantEnvPods[pod.Labels["tenant_env_id"]]++
					}
				}
			}
		}

		nodeResource.UsedCPU = util.DecimalFromFloat32(nodeResource.RawUsedCPU / 1000)
		nodeResource.UsedMem = util.DecimalFromFloat32(nodeResource.RawUsedMem / 1024 / 1024 / 1024)
		nodeResources = append(nodeResources, nodeResource)
		totalUsedPods += nodeResource.UsedPods
		totalCapCPU += nodeResource.CapacityCPU
		totalCapMem += nodeResource.CapacityMem
		totalReqCPU += nodeResource.UsedCPU
		totalReqMem += nodeResource.UsedMem
	}

	result := &model.ClusterResource{
		CapCPU:               util.DecimalFromFloat32(totalCapCPU),
		CapMem:               util.DecimalFromFloat32(totalCapMem),
		ReqCPU:               util.DecimalFromFloat32(totalReqCPU),
		ReqMem:               util.DecimalFromFloat32(totalReqMem),
		WutongReqCPU:         util.DecimalFromFloat32(float32(wtCPUR / 1000)),
		WutongReqMem:         util.DecimalFromFloat32(float32(wtMemR / 1024 / 1024 / 1024)),
		ComputeNode:          len(nodes),
		TotalCapacityPods:    totalCapacityPods,
		TotalUsedPods:        totalUsedPods,
		TotalCapacityStorage: util.DecimalFromFloat32(totalCapacityStorage),
		TotalUsedStorage:     util.DecimalFromFloat32(totalUsedStorage),
		NodeResources:        nodeResources,
		TenantEnvPods:        tenantEnvPods,
	}

	result.AllNode = len(nodes)
	for _, node := range nodes {
		if !isNodeReady(node) {
			result.NotReadyNode++
		}
	}
	c.clusterInfoCache = result
	c.cacheTime = time.Now()
	return result, nil
}

func internalIPFromNode(node *corev1.Node) (string, bool) {
	if len(node.Status.Addresses) > 0 {
		for _, address := range node.Status.Addresses {
			if address.Type == corev1.NodeInternalIP {
				return address.Address, true
			}
		}
	}
	return "", false
}

func (c *clusterAction) listNodes(ctx context.Context) ([]*corev1.Node, error) {
	opts := metav1.ListOptions{}
	nodeList, err := c.clientset.CoreV1().Nodes().List(ctx, opts)
	if err != nil {
		return nil, err
	}

	var nodes []*corev1.Node
	for idx := range nodeList.Items {
		node := &nodeList.Items[idx]
		// check if node contains taints
		if containsTaints(node) {
			logrus.Debugf("[GetClusterInfo] node(%s) contains NoSchedule taints", node.GetName())
			continue
		}

		nodes = append(nodes, node)
	}

	return nodes, nil
}

func isNodeReady(node *corev1.Node) bool {
	for _, cond := range node.Status.Conditions {
		if cond.Type == corev1.NodeReady && cond.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

func containsTaints(node *corev1.Node) bool {
	for _, taint := range node.Spec.Taints {
		if taint.Effect == corev1.TaintEffectNoSchedule {
			return true
		}
	}
	return false
}

func (c *clusterAction) listPods(ctx context.Context, nodeName string) (pods []corev1.Pod, err error) {
	podList, err := c.clientset.CoreV1().Pods(metav1.NamespaceAll).List(ctx, metav1.ListOptions{
		FieldSelector: fields.SelectorFromSet(fields.Set{"spec.nodeName": nodeName}).String()})
	if err != nil {
		return pods, err
	}

	return podList.Items, nil
}

// MavenSetting maven setting
type MavenSetting struct {
	Name       string `json:"name" validate:"required"`
	CreateTime string `json:"create_time"`
	UpdateTime string `json:"update_time"`
	Content    string `json:"content" validate:"required"`
	IsDefault  bool   `json:"is_default"`
}

// MavenSettingList maven setting list
func (c *clusterAction) MavenSettingList(ctx context.Context) (re []MavenSetting) {
	cms, err := c.clientset.CoreV1().ConfigMaps(c.namespace).List(ctx, metav1.ListOptions{
		LabelSelector: "configtype=mavensetting",
	})
	if err != nil {
		logrus.Errorf("list maven setting config list failure %s", err.Error())
	}
	for _, sm := range cms.Items {
		isDefault := false
		if sm.Labels["default"] == "true" {
			isDefault = true
		}
		re = append(re, MavenSetting{
			Name:       sm.Name,
			CreateTime: sm.CreationTimestamp.Format(time.RFC3339),
			UpdateTime: sm.Labels["updateTime"],
			Content:    sm.Data["mavensetting"],
			IsDefault:  isDefault,
		})
	}
	return
}

// MavenSettingAdd maven setting add
func (c *clusterAction) MavenSettingAdd(ctx context.Context, ms *MavenSetting) *util.APIHandleError {
	config := &corev1.ConfigMap{}
	config.Name = ms.Name
	config.Namespace = c.namespace
	config.Labels = map[string]string{
		"creator":    "Wutong",
		"configtype": "mavensetting",
	}
	config.Annotations = map[string]string{
		"updateTime": time.Now().Format(time.RFC3339),
	}
	config.Data = map[string]string{
		"mavensetting": ms.Content,
	}
	_, err := c.clientset.CoreV1().ConfigMaps(c.namespace).Create(ctx, config, metav1.CreateOptions{})
	if err != nil {
		if apierrors.IsAlreadyExists(err) {
			return &util.APIHandleError{Code: 400, Err: fmt.Errorf("setting name is exist")}
		}
		logrus.Errorf("create maven setting configmap failure %s", err.Error())
		return &util.APIHandleError{Code: 500, Err: fmt.Errorf("create setting config failure")}
	}
	ms.CreateTime = time.Now().Format(time.RFC3339)
	ms.UpdateTime = time.Now().Format(time.RFC3339)
	return nil
}

// MavenSettingUpdate maven setting file update
func (c *clusterAction) MavenSettingUpdate(ctx context.Context, ms *MavenSetting) *util.APIHandleError {
	sm, err := c.clientset.CoreV1().ConfigMaps(c.namespace).Get(ctx, ms.Name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return &util.APIHandleError{Code: 404, Err: fmt.Errorf("setting name is not exist")}
		}
		logrus.Errorf("get maven setting config list failure %s", err.Error())
		return &util.APIHandleError{Code: 400, Err: fmt.Errorf("get setting failure")}
	}
	if sm.Data == nil {
		sm.Data = make(map[string]string)
	}
	if sm.Annotations == nil {
		sm.Annotations = make(map[string]string)
	}
	sm.Data["mavensetting"] = ms.Content
	sm.Annotations["updateTime"] = time.Now().Format(time.RFC3339)
	if _, err := c.clientset.CoreV1().ConfigMaps(c.namespace).Update(ctx, sm, metav1.UpdateOptions{}); err != nil {
		logrus.Errorf("update maven setting configmap failure %s", err.Error())
		return &util.APIHandleError{Code: 500, Err: fmt.Errorf("update setting config failure")}
	}
	ms.UpdateTime = sm.Annotations["updateTime"]
	ms.CreateTime = sm.CreationTimestamp.Format(time.RFC3339)
	return nil
}

// MavenSettingDelete maven setting file delete
func (c *clusterAction) MavenSettingDelete(ctx context.Context, name string) *util.APIHandleError {
	err := c.clientset.CoreV1().ConfigMaps(c.namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return &util.APIHandleError{Code: 404, Err: fmt.Errorf("setting not found")}
		}
		logrus.Errorf("delete maven setting config list failure %s", err.Error())
		return &util.APIHandleError{Code: 500, Err: fmt.Errorf("setting delete failure")}
	}
	return nil
}

// MavenSettingDetail maven setting file delete
func (c *clusterAction) MavenSettingDetail(ctx context.Context, name string) (*MavenSetting, *util.APIHandleError) {
	sm, err := c.clientset.CoreV1().ConfigMaps(c.namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		logrus.Errorf("get maven setting config failure %s", err.Error())
		return nil, &util.APIHandleError{Code: 404, Err: fmt.Errorf("setting not found")}
	}
	return &MavenSetting{
		Name:       sm.Name,
		CreateTime: sm.CreationTimestamp.Format(time.RFC3339),
		UpdateTime: sm.Annotations["updateTime"],
		Content:    sm.Data["mavensetting"],
	}, nil
}

// Features -
func (c *clusterAction) Features(ctx context.Context) map[string]bool {
	return map[string]bool{
		"velero":   kube.IsVeleroInstalled(kube.RegionClientset(), kube.RegionAPIExtClientset()),
		"kubevirt": kube.IsKubevirtInstalled(kube.RegionClientset(), kube.RegionAPIExtClientset()),
	}
}

type NodeStorageMetric struct {
	NodeName        string
	CapacityStorage int64
	UsedStorage     int64
}

type NodeStorageMetricsResponse struct {
	Status string                         `json:"status"`
	Data   NodeStorageMetricsResponseData `json:"data"`
}

type NodeStorageMetricsResponseData struct {
	Result []NodeStorageMetricsResponseDataResult `json:"result"`
}

type NodeStorageMetricsResponseDataResult struct {
	Metric NodeStorageMetricsResponseDataResultMetric `json:"metric"`
	Value  []interface{}                              `json:"value"`
}

type NodeStorageMetricsResponseDataResultMetric struct {
	Instance   string `json:"instance"`
	Mountpoint string `json:"mountpoint"`
}

const (
	NodeCapacityStorageMetric = "node_filesystem_size_bytes"
	NodFreeStorageMetric      = "node_filesystem_free_bytes"
)

func (c *clusterAction) GetNodeStorageMetrics(metricName string) map[string]float32 {
	url := fmt.Sprintf("http://%s/api/v1/query?query=%s&time=%d", c.prometheusEndpoint, metricName, time.Now().Unix())
	method := "GET"

	client := &http.Client{}
	req, err := http.NewRequest(method, url, nil)

	if err != nil {
		fmt.Println(err)
		return nil
	}
	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	var metricsResp NodeStorageMetricsResponse
	err = json.Unmarshal(body, &metricsResp)
	if err != nil {
		return nil
	}

	storageMetrics := make(map[string]float32)

	for _, result := range metricsResp.Data.Result {
		if result.Metric.Mountpoint == "/" && len(result.Value) == 2 {
			storage, err := strconv.ParseFloat(result.Value[1].(string), 32)
			if err != nil {
				continue
			}
			if ip := strings.Split(result.Metric.Instance, ":"); len(ip) == 2 {
				storageMetrics[ip[0]] = float32(storage)
			}
		}
	}

	return storageMetrics
}

type clusterEventsCache struct {
	cacheTime time.Time
	cacheData []model.ClusterEvent
}

var cachedClusterEvents *clusterEventsCache

func (c *clusterAction) GetClusterEvents(ctx context.Context) ([]model.ClusterEvent, error) {
	//  5 分钟内的事件缓存
	if cachedClusterEvents == nil || time.Since(cachedClusterEvents.cacheTime) > time.Minute*5 {
		events, err := kube.GetCachedResources(c.clientset).EventLister.List(labels.Everything())
		if err != nil {
			return nil, err
		}

		clusterEvents := make([]model.ClusterEvent, 0)
		for _, event := range events {
			clusterEvent := model.ClusterEventFrom(event, c.clientset)
			if clusterEvent == nil {
				continue
			}
			clusterEvents = append(clusterEvents, *clusterEvent)
		}
		cachedClusterEvents = &clusterEventsCache{
			cacheTime: time.Now(),
			cacheData: clusterEvents,
		}
	}
	return cachedClusterEvents.cacheData, nil
}
