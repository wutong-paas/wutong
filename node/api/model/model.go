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

package model

import (
	//"github.com/sirupsen/logrus"
	"fmt"
	"io"
	"net/http"
	url2 "net/url"
	"strings"

	"github.com/pquerna/ffjson/ffjson"
	"github.com/wutong-paas/wutong/node/utils"
	"go.etcd.io/etcd/api/v3/mvccpb"
	v1 "k8s.io/api/core/v1" //"github.com/sirupsen/logrus"
)

// Resource 资源
type Resource struct {
	CPU  int `json:"cpu"`
	MemR int `json:"mem"`
}

// NodePodResource -
type NodePodResource struct {
	AllocatedResources `json:"allocatedresources"`
	Resource           `json:"allocatable"`
}

// AllocatedResources -
type AllocatedResources struct {
	CPURequests     int64
	CPULimits       int64
	MemoryRequests  int64
	MemoryLimits    int64
	MemoryRequestsR string
	MemoryLimitsR   string
	CPURequestsR    string
	CPULimitsR      string
}

// InitStatus -
type InitStatus struct {
	Status   int    `json:"status"`
	StatusCN string `json:"cn"`
	HostID   string `json:"uuid"`
}

// InstallStatus -
type InstallStatus struct {
	Status   int           `json:"status"`
	StatusCN string        `json:"cn"`
	Tasks    []*ExecedTask `json:"tasks"`
}

// ExecedTask -
type ExecedTask struct {
	ID             string   `json:"id"`
	Seq            int      `json:"seq"`
	Desc           string   `json:"desc"`
	Status         string   `json:"status"`
	CompleteStatus string   `json:"complete_status"`
	ErrorMsg       string   `json:"err_msg"`
	Depends        []string `json:"dep"`
	Next           []string `json:"next"`
}

// Prome -
type Prome struct {
	Status string    `json:"status"`
	Data   PromeData `json:"data"`
}

// PromeData -
type PromeData struct {
	ResultType string             `json:"resultType"`
	Result     []*PromeResultCore `json:"result"`
}

// PromeResultCore -
type PromeResultCore struct {
	Metric map[string]string `json:"metric"`
	Value  []interface{}     `json:"value"`
	Values []interface{}     `json:"values"`
}

// Expr swagger:parameters createToken
type Expr struct {
	Body struct {
		// expr
		// in: body
		// required: true
		Expr string `json:"expr" validate:"expr|required"`
	}
}

// LabelsResp -
type LabelsResp struct {
	SysLabels    map[string]string `json:"sys_labels"`
	CustomLabels map[string]string `json:"custom_labels"`
}

// PrometheusInterface -
type PrometheusInterface interface {
	Query(query string) *Prome
	QueryRange(query string, start, end, step string) *Prome
}

// PrometheusAPI -
type PrometheusAPI struct {
	API string
}

// Query Get
func (s *PrometheusAPI) Query(query string) (*Prome, *utils.APIHandleError) {
	resp, code, err := DoRequest(s.API, query, "query", "GET", nil)
	if err != nil {
		return nil, utils.CreateAPIHandleError(400, err)
	}
	if code == 422 {
		return nil, utils.CreateAPIHandleError(422, fmt.Errorf("unprocessable entity,expression %s can't be executed", query))
	}
	if code == 400 {
		return nil, utils.CreateAPIHandleError(400, fmt.Errorf("bad request,error to request query %s", query))
	}
	if code == 503 {
		return nil, utils.CreateAPIHandleError(503, fmt.Errorf("service unavailable"))
	}
	var prome Prome
	err = ffjson.Unmarshal(resp, &prome)
	if err != nil {
		return nil, utils.CreateAPIHandleError(500, err)
	}
	return &prome, nil
}

// QueryRange Get
func (s *PrometheusAPI) QueryRange(query string, start, end, step string) (*Prome, *utils.APIHandleError) {
	//logrus.Infof("prometheus api is %s",s.API)
	uri := fmt.Sprintf("%v&start=%v&end=%v&step=%v", query, start, end, step)
	resp, code, err := DoRequest(s.API, uri, "query_range", "GET", nil)
	if err != nil {
		return nil, utils.CreateAPIHandleError(400, err)
	}
	if code == 422 {
		return nil, utils.CreateAPIHandleError(422, fmt.Errorf("unprocessable entity,expression %s can't be executed", query))
	}
	if code == 400 {
		return nil, utils.CreateAPIHandleError(400, fmt.Errorf("bad request,error to request query %s", query))
	}
	if code == 503 {
		return nil, utils.CreateAPIHandleError(503, fmt.Errorf("service unavailable"))
	}
	var prome Prome
	err = ffjson.Unmarshal(resp, &prome)
	if err != nil {
		return nil, utils.CreateAPIHandleError(500, err)
	}
	return &prome, nil
}

// DoRequest -
func DoRequest(baseAPI, query, queryType, method string, body []byte) ([]byte, int, error) {
	api := baseAPI + "/api/v1/" + queryType + "?"
	query = "query=" + query
	query = strings.Replace(query, "+", "%2B", -1)
	val, err := url2.ParseQuery(query)
	if err != nil {
		return nil, 0, err
	}
	encoded := val.Encode()
	//logrus.Infof("uri is %s",api+encoded)
	request, err := http.NewRequest(method, api+encoded, nil)
	if err != nil {
		return nil, 0, err
	}
	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, 0, err
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	return data, resp.StatusCode, nil
}

// ClusterResource 资源
type ClusterResource struct {
	AllNode                          int           `json:"all_node"`
	NotReadyNode                     int           `json:"notready_node"`
	ComputeNode                      int           `json:"compute_node"`
	TenantEnv                        int           `json:"tenant_env"`
	CapCPU                           int           `json:"cap_cpu"` //可分配CPU总额
	CapMem                           float32       `json:"cap_mem"` //可分配Mem总额
	ReqCPU                           float32       `json:"req_cpu"` //已使用CPU总额
	ReqMem                           float32       `json:"req_mem"` //已使用Mem总额
	CapDisk                          uint64        `json:"cap_disk"`
	ReqDisk                          uint64        `json:"req_disk"`
	MaxAllocatableMemoryNodeResource *NodeResource `json:"max_allocatable_memory_node_resource"`
}

// NodeResourceResponse 资源
type NodeResourceResponse struct {
	CapCPU int     `json:"cap_cpu"`
	CapMem int     `json:"cap_mem"`
	ReqCPU float32 `json:"req_cpu"`
	ReqMem int     `json:"req_mem"`
}

// FirstConfig -
type FirstConfig struct {
	StorageMode     string `json:"storage_mode"`
	StorageHost     string `json:"storage_host,omitempty"`
	StorageEndPoint string `json:"storage_endpoint,omitempty"`

	NetworkMode string `json:"network_mode"`
	ZKHosts     string `json:"zk_host,omitempty"`
	CassandraIP string `json:"cassandra_ip,omitempty"`
	K8SAPIAddr  string `json:"k8s_apiserver,omitempty"`
	MasterIP    string `json:"master_ip,omitempty"`
	DNS         string `json:"dns,omitempty"`
	ZMQSub      string `json:"zmq_sub,omitempty"`
	ZMQTo       string `json:"zmq_to,omitempty"`
	EtcdIP      string `json:"etcd_ip,omitempty"`
}

// Config -
type Config struct {
	Cn    string `json:"cn_name"`
	Name  string `json:"name"`
	Value string `json:"value"`
}

// ConfigUnit 一个配置单元
type ConfigUnit struct {
	//配置名称 例如:network
	Name   string `json:"name" validate:"name|required"`
	CNName string `json:"cn_name" validate:"cn_name"`
	//类型 例如:midonet
	Value     interface{} `json:"value" validate:"value|required"`
	ValueType string      `json:"value_type"`
	//可选类型 类型名称和需要的配置项
	OptionalValue []string                `json:"optional_value,omitempty"`
	DependConfig  map[string][]ConfigUnit `json:"depend_config,omitempty"`
	//是否用户可配置
	IsConfigurable bool `json:"is_configurable"`
}

func (c ConfigUnit) String() string {
	res, _ := ffjson.Marshal(&c)
	return string(res)
}

// GlobalConfig 全局配置
type GlobalConfig struct {
	Configs map[string]*ConfigUnit `json:"configs"`
}

// String String
func (g *GlobalConfig) String() string {
	res, _ := ffjson.Marshal(g)
	return string(res)
}

// Add 添加配置
func (g *GlobalConfig) Add(c ConfigUnit) {
	//具有依赖配置
	if c.DependConfig != nil || len(c.DependConfig) > 0 {
		if c.ValueType == "string" || c.ValueType == "" {
			if value, ok := c.Value.(string); ok {
				for _, dc := range c.DependConfig[value] {
					g.Add(dc)
				}
			}
		}
	}
	g.Configs[c.Name] = &c
}

// Get 获取配置
func (g *GlobalConfig) Get(name string) *ConfigUnit {
	return g.Configs[name]
}

// Delete 删除配置
func (g *GlobalConfig) Delete(Name string) {
	delete(g.Configs, Name)
}

// Bytes Bytes
func (g GlobalConfig) Bytes() []byte {
	res, _ := ffjson.Marshal(&g)
	return res
}

// CreateDefaultGlobalConfig 生成默认配置
func CreateDefaultGlobalConfig() *GlobalConfig {
	gconfig := &GlobalConfig{
		Configs: make(map[string]*ConfigUnit),
	}
	gconfig.Add(ConfigUnit{
		Name:      "NETWORK_MODE",
		CNName:    "集群网络模式",
		Value:     "calico",
		ValueType: "string",
		DependConfig: map[string][]ConfigUnit{
			"calico": {{Name: "ETCD_ADDRS", CNName: "ETCD地址", ValueType: "array"}},
			"midonet": {
				{Name: "CASSANDRA_ADDRS", CNName: "CASSANDRA地址", ValueType: "array"},
				{Name: "ZOOKEEPER_ADDRS", CNName: "ZOOKEEPER地址", ValueType: "array"},
				{Name: "LB_CIDR", CNName: "负载均衡所在网段", ValueType: "string"},
			}},
		IsConfigurable: true,
	})
	gconfig.Add(ConfigUnit{
		Name:   "STORAGE_MODE",
		Value:  "nfs",
		CNName: "默认共享存储模式",
		DependConfig: map[string][]ConfigUnit{
			"nfs": {
				{Name: "NFS_SERVERS", CNName: "NFS服务端地址列表", ValueType: "array"},
				{Name: "NFS_ENDPOINT", CNName: "NFS挂载端点", ValueType: "string"},
			},
			"clusterfs": {},
		},
		IsConfigurable: true,
	})
	gconfig.Add(ConfigUnit{
		Name:          "DB_MODE",
		Value:         "mysql",
		CNName:        "管理节点数据库类型",
		OptionalValue: []string{"mysql", "cockroachdb"},
		DependConfig: map[string][]ConfigUnit{
			"mysql": {
				{Name: "MYSQL_HOST", CNName: "Mysql数据库地址", ValueType: "string", Value: "127.0.0.1"},
				{Name: "MYSQL_PASS", CNName: "Mysql数据库密码", ValueType: "string", Value: ""},
				{Name: "MYSQL_USER", CNName: "Mysql数据库用户名", ValueType: "string", Value: ""},
			},
			"cockroachdb": {
				{Name: "COCKROACH_HOST", CNName: "Mysql数据库地址", ValueType: "array"},
				{Name: "COCKROACH_PASS", CNName: "Mysql数据库密码", ValueType: "string"},
				{Name: "COCKROACH_USER", CNName: "Mysql数据库用户名", ValueType: "string"},
			},
		},
		IsConfigurable: true,
	})
	gconfig.Add(ConfigUnit{
		Name:           "LB_MODE",
		Value:          "nginx",
		ValueType:      "string",
		CNName:         "边缘负载均衡",
		OptionalValue:  []string{"nginx", "zeus"},
		IsConfigurable: true,
	})
	gconfig.Add(ConfigUnit{Name: "DOMAIN", CNName: "应用域名", ValueType: "string"})
	gconfig.Add(ConfigUnit{Name: "INSTALL_NODE", CNName: "安装节点", ValueType: "array"})
	gconfig.Add(ConfigUnit{
		Name:           "INSTALL_MODE",
		Value:          "online",
		ValueType:      "string",
		CNName:         "安装模式",
		OptionalValue:  []string{"online", "offine"},
		IsConfigurable: true,
	})
	gconfig.Add(ConfigUnit{
		Name:      "DNS_SERVER",
		Value:     []string{},
		CNName:    "集群DNS服务",
		ValueType: "array",
	})
	gconfig.Add(ConfigUnit{
		Name:      "KUBE_API",
		Value:     []string{},
		ValueType: "array",
		CNName:    "KubernetesAPI服务",
	})
	gconfig.Add(ConfigUnit{
		Name:      "MANAGE_NODE_ADDRESS",
		Value:     []string{},
		ValueType: "array",
		CNName:    "管理节点",
	})
	return gconfig
}

// CreateGlobalConfig 生成配置
func CreateGlobalConfig(kvs []*mvccpb.KeyValue) (*GlobalConfig, error) {
	dgc := &GlobalConfig{
		Configs: make(map[string]*ConfigUnit),
	}
	for _, kv := range kvs {
		var cn ConfigUnit
		if err := ffjson.Unmarshal(kv.Value, &cn); err == nil {
			dgc.Add(cn)
		}
	}
	return dgc, nil
}

// LoginResult -
type LoginResult struct {
	HostPort  string `json:"hostport"`
	LoginType bool   `json:"type"`
	Result    string `json:"result"`
}

// Login -
type Login struct {
	HostPort  string `json:"hostport"`
	LoginType bool   `json:"type"`
	HostType  string `json:"hosttype"`
	RootPwd   string `json:"pwd,omitempty"`
}

// Body -
type Body struct {
	List interface{} `json:"list"`
	Bean interface{} `json:"bean,omitempty"`
}

// ResponseBody -
type ResponseBody struct {
	Code  int    `json:"code"`
	Msg   string `json:"msg"`
	MsgCN string `json:"msgcn"`
	Body  Body   `json:"body,omitempty"`
}

// Pods -
type Pods struct {
	Namespace       string `json:"namespace"`
	Id              string `json:"id"`
	Name            string `json:"name"`
	TenantEnvName   string `json:"tenant_env_name"`
	CPURequests     string `json:"cpurequest"`
	CPURequestsR    string `json:"cpurequestr"`
	CPULimits       string `json:"cpulimits"`
	CPULimitsR      string `json:"cpulimitsr"`
	MemoryRequests  string `json:"memoryrequests"`
	MemoryRequestsR string `json:"memoryrequestsr"`
	MemoryLimits    string `json:"memorylimits"`
	MemoryLimitsR   string `json:"memorylimitsr"`
	Status          string `json:"status"`
}

// NodeDetails NodeDetails
type NodeDetails struct {
	Name               string              `json:"name"`
	Role               []string            `json:"role"`
	Status             string              `json:"status"`
	Labels             map[string]string   `json:"labels"`
	Annotations        map[string]string   `json:"annotations"`
	CreationTimestamp  string              `json:"creationtimestamp"`
	Conditions         []v1.NodeCondition  `json:"conditions"`
	Addresses          map[string]string   `json:"addresses"`
	Capacity           map[string]string   `json:"capacity"`
	Allocatable        map[string]string   `json:"allocatable"`
	SystemInfo         v1.NodeSystemInfo   `json:"systeminfo"`
	NonterminatedPods  []*Pods             `json:"nonterminatedpods"`
	AllocatedResources map[string]string   `json:"allocatedresources"`
	Events             map[string][]string `json:"events"`
}

// AlertingRulesConfig -
type AlertingRulesConfig struct {
	Groups []*AlertingNameConfig `yaml:"groups" json:"groups"`
}

// AlertingNameConfig -
type AlertingNameConfig struct {
	Name  string         `yaml:"name" json:"name"`
	Rules []*RulesConfig `yaml:"rules" json:"rules"`
}

// RulesConfig -
type RulesConfig struct {
	Alert       string            `yaml:"alert" json:"alert"`
	Expr        string            `yaml:"expr" json:"expr"`
	For         string            `yaml:"for" json:"for"`
	Labels      map[string]string `yaml:"labels" json:"labels"`
	Annotations map[string]string `yaml:"annotations" json:"annotations"`
}

// NotificationEvent NotificationEvent
type NotificationEvent struct {
	//Kind could be service, tenantEnv, cluster, node
	Kind string `json:"Kind"`
	//KindID could be service_id,tenant_env_id,cluster_id,node_id
	KindID string `json:"KindID"`
	Hash   string `json:"Hash"`
	//Type could be Normal UnNormal Notification
	Type          string `json:"Type"`
	Message       string `json:"Message"`
	Reason        string `json:"Reason"`
	Count         int    `json:"Count"`
	LastTime      string `json:"LastTime"`
	FirstTime     string `json:"FirstTime"`
	IsHandle      bool   `json:"IsHandle"`
	HandleMessage string `json:"HandleMessage"`
	ServiceName   string `json:"ServiceName"`
	TenantEnvName string `json:"TenantEnvName"`
}
