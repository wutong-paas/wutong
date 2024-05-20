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

package handler

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	api_model "github.com/wutong-paas/wutong/api/model"
	"github.com/wutong-paas/wutong/api/util"
	"github.com/wutong-paas/wutong/api/util/bcode"
	"github.com/wutong-paas/wutong/cmd/api/option"
	"github.com/wutong-paas/wutong/db"
	dbmodel "github.com/wutong-paas/wutong/db/model"
	mqclient "github.com/wutong-paas/wutong/mq/client"
	"github.com/wutong-paas/wutong/pkg/apis/wutong/v1alpha1"
	"github.com/wutong-paas/wutong/pkg/kube"
	"github.com/wutong-paas/wutong/pkg/prometheus"
	rutil "github.com/wutong-paas/wutong/util"
	"github.com/wutong-paas/wutong/worker/client"
	"github.com/wutong-paas/wutong/worker/server/pb"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

// TenantEnvAction tenant env act
type TenantEnvAction struct {
	MQClient                  mqclient.MQClient
	statusCli                 *client.AppRuntimeSyncClient
	OptCfg                    *option.Config
	restConfig                *rest.Config
	kubeClient                kubernetes.Interface
	cacheClusterResourceStats *ClusterResourceStats
	cacheTime                 time.Time
	prometheusCli             prometheus.Interface
	k8sClient                 k8sclient.Client
	resources                 map[string]k8sclient.Object
}

// CreateTenantEnvManager create Manger
func CreateTenantEnvManager(mqc mqclient.MQClient, statusCli *client.AppRuntimeSyncClient,
	optCfg *option.Config,
	config *rest.Config,
	kubeClient kubernetes.Interface,
	prometheusCli prometheus.Interface,
	k8sClient k8sclient.Client) *TenantEnvAction {

	resources := map[string]k8sclient.Object{
		"helmApp": &v1alpha1.HelmApp{},
		"service": &corev1.Service{},
	}

	return &TenantEnvAction{
		MQClient:      mqc,
		statusCli:     statusCli,
		OptCfg:        optCfg,
		restConfig:    config,
		kubeClient:    kubeClient,
		prometheusCli: prometheusCli,
		k8sClient:     k8sClient,
		resources:     resources,
	}
}

// BindTenantEnvsResource query tenant env resource used and sort
func (t *TenantEnvAction) BindTenantEnvsResource(source []*dbmodel.TenantEnvs) api_model.TenantEnvList {
	var list api_model.TenantEnvList
	var resources = make(map[string]*pb.TenantEnvResource, len(source))
	if len(source) == 1 {
		re, err := t.statusCli.GetTenantEnvResource(source[0].UUID)
		if err != nil {
			logrus.Errorf("get tenant env %s resource failure %s", source[0].UUID, err.Error())
		}
		if re != nil {
			resources[source[0].UUID] = re
		}
	} else {
		res, err := t.statusCli.GetAllTenantEnvResource()
		if err != nil {
			logrus.Errorf("get all tenant env resource failure %s", err.Error())
		}
		if res != nil {
			resources = res.Resources
		}
	}
	for i, ten := range source {
		var item = &api_model.TenantEnvAndResource{
			TenantEnvs: *source[i],
		}
		re := resources[ten.UUID]
		if re != nil {
			item.CPULimit = re.CpuLimit
			item.CPURequest = re.CpuRequest
			item.MemoryLimit = re.MemoryLimit
			item.MemoryRequest = re.MemoryRequest
			item.RunningAppNum = re.RunningAppNum
			item.RunningAppInternalNum = re.RunningAppInternalNum
			item.RunningAppThirdNum = re.RunningAppThirdNum
		}
		list.Add(item)
	}
	sort.Sort(list)
	return list
}

// CreateTenantEnv create tenant env
func (t *TenantEnvAction) GetAllTenantEnvs(query string) ([]*dbmodel.TenantEnvs, error) {
	tenantEnvs, err := db.GetManager().TenantEnvDao().GetAllTenantEnvs(query)
	if err != nil {
		return nil, err
	}
	return tenantEnvs, err
}

// GetTenantEnvs get tenant envs
func (t *TenantEnvAction) GetTenantEnvs(tenantName, query string) ([]*dbmodel.TenantEnvs, error) {
	tenantEnvs, err := db.GetManager().TenantEnvDao().GetTenantEnvs(tenantName, query)
	if err != nil {
		return nil, err
	}
	return tenantEnvs, err
}

// UpdateTenantEnv update tenant env info
func (t *TenantEnvAction) UpdateTenantEnv(tenantEnv *dbmodel.TenantEnvs) error {
	return db.GetManager().TenantEnvDao().UpdateModel(tenantEnv)
}

// DeleteTenantEnv deletes tenant env based on the given tenantEnvID.
//
// tenant env can only be deleted without service or plugin
func (t *TenantEnvAction) DeleteTenantEnv(ctx context.Context, tenantEnvID string) error {
	// check if there are still services
	services, err := db.GetManager().TenantEnvServiceDao().ListServicesByTenantEnvID(tenantEnvID)
	if err != nil {
		return err
	}
	if len(services) > 0 {
		for _, service := range services {
			GetServiceManager().TransServieToDelete(ctx, tenantEnvID, service.ServiceID)
		}
	}

	// check if there are still plugins
	plugins, err := db.GetManager().TenantEnvPluginDao().ListByTenantEnvID(tenantEnvID)
	if err != nil {
		return err
	}
	if len(plugins) > 0 {
		for _, plugin := range plugins {
			GetPluginManager().DeletePluginAct(plugin.PluginID, tenantEnvID)
		}
	}

	tenantEnv, err := db.GetManager().TenantEnvDao().GetTenantEnvByUUID(tenantEnvID)
	if err != nil {
		return err
	}
	oldStatus := tenantEnv.Status
	var rollback = func() {
		tenantEnv.Status = oldStatus
		_ = db.GetManager().TenantEnvDao().UpdateModel(tenantEnv)
	}
	tenantEnv.Status = dbmodel.TenantEnvStatusDeleting.String()
	if err := db.GetManager().TenantEnvDao().UpdateModel(tenantEnv); err != nil {
		return err
	}

	// delete namespace in k8s
	err = t.MQClient.SendBuilderTopic(mqclient.TaskStruct{
		TaskType: "delete_tenant_env",
		Topic:    mqclient.WorkerTopic,
		TaskBody: map[string]string{
			"tenant_env_id": tenantEnvID,
		},
	})
	if err != nil {
		rollback()
		logrus.Error("send task 'delete_tenant_env'", err)
		return err
	}

	return nil
}

// TotalMemCPU StatsMemCPU
func (t *TenantEnvAction) TotalMemCPU(services []*dbmodel.TenantEnvServices) (*api_model.StatsInfo, error) {
	cpus := 0
	mem := 0
	for _, service := range services {
		logrus.Debugf("service is %d, cpus is %d, mem is %v", service.ID, service.ContainerCPU, service.ContainerMemory)
		cpus += service.ContainerCPU
		mem += service.ContainerMemory
	}
	si := &api_model.StatsInfo{
		CPU: cpus,
		MEM: mem,
	}
	return si, nil
}

// GetTenantEnvsName get tenant envs name
func (t *TenantEnvAction) GetTenantEnvsName(tenantName string) ([]string, error) {
	tenantEnvs, err := db.GetManager().TenantEnvDao().GetTenantEnvs(tenantName, "")
	if err != nil {
		return nil, err
	}
	var result []string
	for _, v := range tenantEnvs {
		result = append(result, strings.ToLower(v.Name))
	}
	return result, err
}

// GetTenantEnvsByName get tenant envs
func (t *TenantEnvAction) GetTenantEnvsByName(tenantName, tenantEnvName string) (*dbmodel.TenantEnvs, error) {
	tenantEnv, err := db.GetManager().TenantEnvDao().GetTenantEnvIDByName(tenantName, tenantEnvName)
	if err != nil {
		return nil, err
	}
	return tenantEnv, err
}

// GetTenantEnvsByUUID get tenantEnvs
func (t *TenantEnvAction) GetTenantEnvsByUUID(uuid string) (*dbmodel.TenantEnvs, error) {
	tenantEnv, err := db.GetManager().TenantEnvDao().GetTenantEnvByUUID(uuid)
	if err != nil {
		return nil, err
	}

	return tenantEnv, err
}

// StatsMemCPU StatsMemCPU
func (t *TenantEnvAction) StatsMemCPU(services []*dbmodel.TenantEnvServices) (*api_model.StatsInfo, error) {
	cpus := 0
	mem := 0
	for _, service := range services {
		status := t.statusCli.GetStatus(service.ServiceID)
		if t.statusCli.IsClosedStatus(status) {
			continue
		}
		cpus += service.ContainerCPU
		mem += service.ContainerMemory
	}
	si := &api_model.StatsInfo{
		CPU: cpus,
		MEM: mem,
	}
	return si, nil
}

// QueryResult contains result data for a query.
type QueryResult struct {
	Data struct {
		Type   string                   `json:"resultType"`
		Result []map[string]interface{} `json:"result"`
	} `json:"data"`
	Status string `json:"status"`
}

// GetTenantEnvsResources Gets the resource usage of the specified tenantEnv.
func (t *TenantEnvAction) GetTenantEnvsResources(ctx context.Context, tr *api_model.TenantEnvResources) (map[string]map[string]interface{}, error) {
	ids, err := db.GetManager().TenantEnvDao().GetTenantEnvIDsByNames(tr.Body.TenantName, tr.Body.TenantEnvNames)
	if err != nil {
		return nil, err
	}
	limits, err := db.GetManager().TenantEnvDao().GetTenantEnvLimitsByNames(tr.Body.TenantName, tr.Body.TenantEnvNames)
	if err != nil {
		return nil, err
	}
	services, err := db.GetManager().TenantEnvServiceDao().GetServicesByTenantEnvIDs(ids)
	if err != nil {
		return nil, err
	}
	var serviceTenantEnvCount = make(map[string]int, len(ids))
	for _, s := range services {
		serviceTenantEnvCount[s.TenantEnvID]++
	}
	// get cluster resources
	clusterStats, err := t.GetAllocatableResources(ctx)
	if err != nil {
		return nil, fmt.Errorf("error getting allocatalbe cpu and memory: %v", err)
	}
	var result = make(map[string]map[string]interface{}, len(ids))
	var resources = make(map[string]*pb.TenantEnvResource, len(ids))
	if len(ids) == 1 {
		re, err := t.statusCli.GetTenantEnvResource(ids[0])
		if err != nil {
			logrus.Errorf("get tenant env %s resource failure %s", ids[0], err.Error())
		}
		if re != nil {
			resources[ids[0]] = re
		}
	} else {
		res, err := t.statusCli.GetAllTenantEnvResource()
		if err != nil {
			logrus.Errorf("get all tenant env resource failure %s", err.Error())
		}
		if res != nil {
			resources = res.Resources
		}
	}
	for _, tenantEnvID := range ids {
		var limitMemory int64
		if l, ok := limits[tenantEnvID]; ok && l != 0 {
			limitMemory = int64(l)
		} else {
			limitMemory = clusterStats.AllMemory
		}
		result[tenantEnvID] = map[string]interface{}{
			"tenant_env_id":       tenantEnvID,
			"limit_memory":        limitMemory,
			"limit_cpu":           clusterStats.AllCPU,
			"service_total_num":   serviceTenantEnvCount[tenantEnvID],
			"disk":                0,
			"service_running_num": 0,
			"cpu":                 0,
			"memory":              0,
		}
		tr := resources[tenantEnvID]
		if tr != nil {
			result[tenantEnvID]["service_running_num"] = tr.RunningAppNum
			result[tenantEnvID]["cpu"] = tr.CpuRequest
			result[tenantEnvID]["memory"] = tr.MemoryRequest
		}
	}
	//query disk used in prometheus
	query := fmt.Sprintf(`sum(app_resource_appfs{tenant_env_id=~"%s"}) by(tenant_env_id)`, strings.Join(ids, "|"))
	metric := t.prometheusCli.GetMetric(query, time.Now())
	for _, mv := range metric.MetricData.MetricValues {
		var tenantEnvID = mv.Metadata["tenant_env_id"]
		var disk int
		if mv.Sample != nil {
			disk = int(mv.Sample.Value() / 1024)
		}
		if tenantEnvID != "" {
			result[tenantEnvID]["disk"] = disk
		}
	}
	return result, nil
}

// TenantEnvResourceStats tenant env resource stats
type TenantEnvResourceStats struct {
	TenantEnvID      string `json:"tenant_env_id,omitempty"`
	CPURequest       int64  `json:"cpu_request,omitempty"`
	CPULimit         int64  `json:"cpu_limit,omitempty"`
	MemoryRequest    int64  `json:"memory_request,omitempty"`
	MemoryLimit      int64  `json:"memory_limit,omitempty"`
	RunningAppNum    int64  `json:"running_app_num"`
	UnscdCPUReq      int64  `json:"unscd_cpu_req,omitempty"`
	UnscdCPULimit    int64  `json:"unscd_cpu_limit,omitempty"`
	UnscdMemoryReq   int64  `json:"unscd_memory_req,omitempty"`
	UnscdMemoryLimit int64  `json:"unscd_memory_limit,omitempty"`
}

// GetTenantEnvResource get tenant env resource
func (t *TenantEnvAction) GetTenantEnvResource(tenantEnvID string) (ts TenantEnvResourceStats, err error) {
	tr, err := t.statusCli.GetTenantEnvResource(tenantEnvID)
	if err != nil {
		return ts, err
	}
	ts.TenantEnvID = tenantEnvID
	ts.CPULimit = tr.CpuLimit
	ts.CPURequest = tr.CpuRequest
	ts.MemoryLimit = tr.MemoryLimit
	ts.MemoryRequest = tr.MemoryRequest
	ts.RunningAppNum = tr.RunningAppNum
	return
}

// ClusterResourceStats cluster resource stats
type ClusterResourceStats struct {
	AllCPU        int64
	AllMemory     int64
	RequestCPU    int64
	RequestMemory int64
}

func (t *TenantEnvAction) initClusterResource(ctx context.Context) error {
	if t.cacheClusterResourceStats == nil || t.cacheTime.Add(time.Minute*3).Before(time.Now()) {
		var crs ClusterResourceStats
		nodes, err := t.kubeClient.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
		if err != nil {
			logrus.Errorf("get cluster nodes failure %s", err.Error())
			return err
		}
		for _, node := range nodes.Items {
			// check if node contains taints
			if containsTaints(&node) {
				logrus.Debugf("[GetClusterInfo] node(%s) contains NoSchedule taints", node.GetName())
				continue
			}
			if node.Spec.Unschedulable {
				continue
			}
			for _, c := range node.Status.Conditions {
				if c.Type == corev1.NodeReady && c.Status != corev1.ConditionTrue {
					continue
				}
			}
			crs.AllMemory += node.Status.Allocatable.Memory().Value() / (1024 * 1024)
			crs.AllCPU += node.Status.Allocatable.Cpu().MilliValue()
		}
		t.cacheClusterResourceStats = &crs
		t.cacheTime = time.Now()
	}
	return nil
}

// GetAllocatableResources returns allocatable cpu and memory (MB)
func (t *TenantEnvAction) GetAllocatableResources(ctx context.Context) (*ClusterResourceStats, error) {
	var crs ClusterResourceStats
	if t.initClusterResource(ctx) != nil {
		return &crs, nil
	}
	ts, err := t.statusCli.GetAllTenantEnvResource()
	if err != nil {
		logrus.Errorf("get tenant env resource failure %s", err.Error())
	}
	re := t.cacheClusterResourceStats
	if ts != nil {
		crs.RequestCPU = 0
		crs.RequestMemory = 0
		for _, re := range ts.Resources {
			crs.RequestCPU += re.CpuRequest
			crs.RequestMemory += re.MemoryRequest
		}
	}
	return re, nil
}

// GetServicesResources Gets the resource usage of the specified service.
func (t *TenantEnvAction) GetServicesResources(tr *api_model.ServicesResources) (re map[string]map[string]interface{}, err error) {
	status := t.statusCli.GetStatuss(strings.Join(tr.Body.ServiceIDs, ","))
	var running, closed []string
	for k, v := range status {
		if !t.statusCli.IsClosedStatus(v) {
			running = append(running, k)
		} else {
			closed = append(closed, k)
		}
	}

	podList, err := t.statusCli.GetMultiServicePods(running)
	if err != nil {
		return nil, err
	}

	res := make(map[string]map[string]interface{})
	for serviceID, item := range podList.ServicePods {
		pods := item.NewPods
		pods = append(pods, item.OldPods...)
		var memory, cpu int64
		for _, pod := range pods {
			for _, c := range pod.Containers {
				memory += c.MemoryRequest
				cpu += c.CpuRequest
			}
		}
		res[serviceID] = map[string]interface{}{"memory": memory / 1024 / 1024, "cpu": cpu}
	}

	for _, c := range closed {
		res[c] = map[string]interface{}{"memory": 0, "cpu": 0}
	}

	disks := GetServicesDiskDeprecated(tr.Body.ServiceIDs, t.prometheusCli)
	for serviceID, disk := range disks {
		if _, ok := res[serviceID]; ok {
			res[serviceID]["disk"] = disk / 1024
		} else {
			res[serviceID] = make(map[string]interface{})
			res[serviceID]["disk"] = disk / 1024
		}
	}
	return res, nil
}

// TenantEnvsSum TenantEnvsSum
func (t *TenantEnvAction) TenantEnvsSum(tenantName string) (int, error) {
	s, err := db.GetManager().TenantEnvDao().GetTenantEnvs(tenantName, "")
	if err != nil {
		return 0, err
	}
	return len(s), nil
}

// GetProtocols GetProtocols
func (t *TenantEnvAction) GetProtocols() ([]*dbmodel.RegionProcotols, *util.APIHandleError) {
	return []*dbmodel.RegionProcotols{
		{
			ProtocolGroup: "http",
			ProtocolChild: "http",
			APIVersion:    "v2",
			IsSupport:     true,
		},
		{
			ProtocolGroup: "http",
			ProtocolChild: "grpc",
			APIVersion:    "v2",
			IsSupport:     true,
		}, {
			ProtocolGroup: "stream",
			ProtocolChild: "tcp",
			APIVersion:    "v2",
			IsSupport:     true,
		}, {
			ProtocolGroup: "stream",
			ProtocolChild: "udp",
			APIVersion:    "v2",
			IsSupport:     true,
		}, {
			ProtocolGroup: "stream",
			ProtocolChild: "mysql",
			APIVersion:    "v2",
			IsSupport:     true,
		},
	}, nil
}

// TransPlugins TransPlugins
func (t *TenantEnvAction) TransPlugins(tenantEnvID, tenantEnvName, fromTenantEnv string, pluginList []string) *util.APIHandleError {
	// tenantEnvInfo, err := db.GetManager().TenantEnvDao().GetTenantEnvIDByName(fromTenantEnv)
	tenantEnvInfo, err := db.GetManager().TenantEnvDao().GetTenantEnvByUUID(tenantEnvID)
	if err != nil {
		return util.CreateAPIHandleErrorFromDBError("get tenant env infos", err)
	}
	tenantEnvUUID := tenantEnvInfo.UUID
	tx := db.GetManager().Begin()
	defer func() {
		if r := recover(); r != nil {
			logrus.Errorf("Unexpected panic occurred, rollback transaction: %v", r)
			tx.Rollback()
		}
	}()
	for _, p := range pluginList {
		pluginInfo, err := db.GetManager().TenantEnvPluginDao().GetPluginByID(p, tenantEnvUUID)
		if err != nil {
			tx.Rollback()
			return util.CreateAPIHandleErrorFromDBError("get plugin infos", err)
		}
		pluginInfo.TenantEnvID = tenantEnvID
		pluginInfo.Domain = tenantEnvName
		pluginInfo.ID = 0
		err = db.GetManager().TenantEnvPluginDaoTransactions(tx).AddModel(pluginInfo)
		if err != nil {
			if !strings.Contains(err.Error(), "is exist") {
				tx.Rollback()
				return util.CreateAPIHandleErrorFromDBError("add plugin Info", err)
			}
		}
	}
	if err := tx.Commit().Error; err != nil {
		return util.CreateAPIHandleErrorFromDBError("trans plugins infos", err)
	}
	return nil
}

// GetServicesStatus returns a list of service status matching ids.
func (t *TenantEnvAction) GetServicesStatus(ids string) map[string]string {
	return t.statusCli.GetStatuss(ids)
}

// IsClosedStatus checks if the status is closed status.
func (t *TenantEnvAction) IsClosedStatus(status string) bool {
	return t.statusCli.IsClosedStatus(status)
}

// GetClusterResource get cluster resource
func (t *TenantEnvAction) GetClusterResource(ctx context.Context) *ClusterResourceStats {
	if t.initClusterResource(ctx) != nil {
		return nil
	}
	return t.cacheClusterResourceStats
}

// CheckResourceName checks resource name.
func (t *TenantEnvAction) CheckResourceName(ctx context.Context, namespace string, req *api_model.CheckResourceNameReq) (*api_model.CheckResourceNameResp, error) {
	obj, ok := t.resources[req.Type]
	if !ok {
		return nil, bcode.NewBadRequest("unsupported resource: " + req.Type)
	}

	nctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	retries := 3
	for i := 0; i < retries; i++ {
		if err := t.k8sClient.Get(nctx, types.NamespacedName{Namespace: namespace, Name: req.Name}, obj); err != nil {
			if k8sErrors.IsNotFound(err) {
				break
			}
			return nil, errors.Wrap(err, "ensure app name")
		}
		req.Name += "-" + rutil.NewUUID()[:5]
	}

	return &api_model.CheckResourceNameResp{
		Name: req.Name,
	}, nil
}

// GetKubeConfig get kubeconfig from tenant env namespace by default dev rbac
func (t *TenantEnvAction) GetKubeConfig(namespace string) (string, error) {
	sa, err := t.completeServiceAccount(namespace)
	if err != nil {
		return "", errors.Wrap(err, "get service account")
	}
	secret, err := t.completeSecretFromServiceAccount(sa)
	if err != nil {
		return "", errors.Wrap(err, "get secret")
	}
	cfgModel := buildConfigFromSecret(t.restConfig, secret)
	cfgContent, err := yaml.Marshal(cfgModel)
	if err != nil {
		return "", errors.Wrap(err, "marshal config")
	}
	return string(cfgContent), nil
}

// GetKubeResources get kube resources for tenantEnv
func (s *TenantEnvAction) GetKubeResources(namespace, tenantEnvID string, customSetting api_model.KubeResourceCustomSetting) (string, error) {
	if msgs := validation.IsDNS1123Label(customSetting.Namespace); len(msgs) > 0 {
		return "", fmt.Errorf("invalid namespace name: %s", customSetting.Namespace)
	}
	selectors := []labels.Selector{
		labels.SelectorFromSet(labels.Set{"tenant_env_id": tenantEnvID}),
	}
	resources := kube.GetResourcesYamlFormat(s.kubeClient, namespace, selectors, &customSetting)
	return resources, nil
}

// createTPServiceAccount create telepresence dev serviceaccount for specified namespace
func (t *TenantEnvAction) completeServiceAccount(namespace string) (*corev1.ServiceAccount, error) {
	tpns := "ambassador"
	saName := fmt.Sprintf("tpdev-%s", namespace)
	sa, err := t.kubeClient.CoreV1().ServiceAccounts(tpns).Get(context.TODO(), saName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return sa, nil
}

// completeSecretFromServiceAccount -
func (t *TenantEnvAction) completeSecretFromServiceAccount(sa *corev1.ServiceAccount) (*corev1.Secret, error) {
	secret, err := t.kubeClient.CoreV1().Secrets(sa.Namespace).Get(context.TODO(), sa.Secrets[0].Name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return secret, nil
}

// buildConfigFromSecret -
func buildConfigFromSecret(kubecfg *rest.Config, secret *corev1.Secret) api_model.Config {
	cfgContext := "context-" + rand.String(5)
	cfgCluster := "cluster-" + rand.String(5)
	cfgUser := "user-" + rand.String(5)
	cfg := api_model.Config{
		APIVersion:     "v1",
		Kind:           "Config",
		CurrentContext: cfgContext,
		Contexts: []*api_model.ContextItem{
			{
				Name: cfgContext,
				Context: &api_model.Context{
					Cluster:   cfgCluster,
					AuthInfo:  cfgUser,
					Namespace: string(secret.Data["namespace"]),
				},
			},
		},
		Clusters: []*api_model.ClusterItem{
			{
				Name: cfgCluster,
				Cluster: &api_model.Cluster{
					Server:                   kubecfg.Host,
					CertificateAuthorityData: secret.Data["ca.crt"],
				},
			},
		},
		AuthInfos: []*api_model.AuthInfoItem{
			{
				Name: cfgUser,
				AuthInfo: &api_model.AuthInfo{
					Token: string(secret.Data["token"]),
				},
			},
		},
	}
	return cfg
}
