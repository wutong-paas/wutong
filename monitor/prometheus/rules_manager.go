package prometheus

import (
	"os"

	"github.com/sirupsen/logrus"
	"github.com/wutong-paas/wutong/cmd/monitor/option"
	"gopkg.in/yaml.v3"
)

// AlertingRulesConfig alerting rule config
type AlertingRulesConfig struct {
	Groups []*AlertingNameConfig `yaml:"groups" json:"groups"`
}

// AlertingNameConfig alerting config
type AlertingNameConfig struct {
	Name  string         `yaml:"name" json:"name"`
	Rules []*RulesConfig `yaml:"rules" json:"rules"`
}

// RulesConfig rule config
type RulesConfig struct {
	Alert       string            `yaml:"alert" json:"alert"`
	Expr        string            `yaml:"expr" json:"expr"`
	For         string            `yaml:"for" json:"for"`
	Labels      map[string]string `yaml:"labels" json:"labels"`
	Annotations map[string]string `yaml:"annotations" json:"annotations"`
}

// AlertingRulesManager alerting rule manage
type AlertingRulesManager struct {
	RulesConfig *AlertingRulesConfig
	config      *option.Config
}

// NewRulesManager new rule manager
func NewRulesManager(config *option.Config) *AlertingRulesManager {
	region := os.Getenv("REGION_NAME")
	if region == "" {
		region = "default"
	}
	commonLables := map[string]string{
		"Alert":  "Wutong",
		"Region": region,
	}
	getseverityLables := func(severity string) map[string]string {
		return map[string]string{
			"Alert":    "Wutong",
			"severity": severity,
			"Region":   region,
		}
	}
	a := &AlertingRulesManager{
		RulesConfig: &AlertingRulesConfig{
			Groups: []*AlertingNameConfig{
				{
					Name: "GatewayHealth",
					Rules: []*RulesConfig{
						{
							Alert:  "GatewayDown",
							Expr:   "absent(up{job=\"gateway\"}) or up{job=\"gateway\"}==0",
							For:    "20s",
							Labels: commonLables,
							Annotations: map[string]string{
								"description": "gateway node {{ $labels.instance }} maybe down",
								"summary":     "gateway is down",
							},
						},
						{
							Alert:  "RequestSizeTooMuch",
							Expr:   "sum by (instance, host) (rate(gateway_request_size_sum[5m])) > 1024*1024*10",
							For:    "20s",
							Labels: commonLables,
							Annotations: map[string]string{
								"description": "http doamin {{ $labels.host }} per-second request size {{ humanize $value }}, more than 10M",
								"summary":     "Too much traffic",
							},
						},
						{
							Alert:  "ResponseSizeTooMuch",
							Expr:   "sum by (instance, host) (rate(gateway_response_size_sum[5m])) > 1024*1024*10",
							For:    "20s",
							Labels: commonLables,
							Annotations: map[string]string{
								"description": "http doamin {{ $labels.host }} per-second response size {{ humanize $value }}, more than 10M",
								"summary":     "Too much traffic",
							},
						},
						{
							Alert:       "RequestMany",
							Expr:        "rate(gateway_requests[5m]) > 200",
							For:         "10s",
							Labels:      commonLables,
							Annotations: map[string]string{"description": "http doamin {{ $labels.host }} per-second requests {{ humanize $value }}, more than 200"},
						},
						{
							Alert:       "FailureRequestMany",
							Expr:        "rate(gateway_requests{status=~\"5..\"}[5m]) > 5",
							For:         "10s",
							Labels:      commonLables,
							Annotations: map[string]string{"description": "http doamin {{ $labels.host }} per-second failure requests {{ humanize $value }}, more than 5"},
						},
					},
				},
				{
					Name: "BuilderHealth",
					Rules: []*RulesConfig{
						{
							Alert:  "BuilderDown",
							Expr:   "absent(up{component=\"builder\"}) or up{component=\"builder\"}==0",
							For:    "1m",
							Labels: commonLables,
							Annotations: map[string]string{
								"description": "builder(wt-chaos) node {{ $labels.instance }} maybe down",
								"summary":     "builder(wt-chaos) is down",
							},
						},
						{
							Alert:       "BuilderUnhealthy",
							Expr:        "builder_exporter_health_status == 0",
							For:         "3m",
							Labels:      commonLables,
							Annotations: map[string]string{"description": "builder unhealthy"},
						},
						{
							Alert:       "BuilderTaskError",
							Expr:        "builder_exporter_builder_current_concurrent_task == builder_exporter_builder_max_concurrent_task",
							For:         "20s",
							Labels:      commonLables,
							Annotations: map[string]string{"summary": "The build service is performing a maximum number of tasks"},
						},
					},
				},
				{
					Name: "WorkerHealth",
					Rules: []*RulesConfig{
						{
							Alert:  "WorkerDown",
							Expr:   "absent(up{component=\"worker\"}) or up{component=\"worker\"}==0",
							For:    "5m",
							Labels: commonLables,
							Annotations: map[string]string{
								"description": "worker node {{ $labels.instance }} maybe down",
								"summary":     "worker is down",
							},
						},
						{
							Alert:  "WorkerUnhealthy",
							Expr:   "app_resource_exporter_health_status == 0",
							For:    "5m",
							Labels: commonLables,
							Annotations: map[string]string{
								"summary":     "worker unhealthy",
								"description": "worker node {{ $labels.instance }} is unhealthy",
							},
						},
						{
							Alert:  "WorkerTaskError",
							Expr:   "app_resource_exporter_worker_task_error > 50",
							For:    "5m",
							Labels: commonLables,
							Annotations: map[string]string{
								"description": "worker node {{ $labels.instance }} execution task error number is greater than 50",
							},
						},
					},
				},
				{
					Name: "MqHealth",
					Rules: []*RulesConfig{
						{
							Alert:  "MqDown",
							Expr:   "absent(up{component=\"mq\"}) or up{component=\"mq\"}==0",
							For:    "2m",
							Labels: commonLables,
							Annotations: map[string]string{
								"description": "mq node {{ $labels.instance }} maybe down",
								"summary":     "mq is down",
							},
						},
						{
							Alert:       "MqUnhealthy",
							Expr:        "acp_mq_exporter_health_status == 0",
							For:         "3m",
							Labels:      commonLables,
							Annotations: map[string]string{"summary": "mq unhealthy"},
						},
						{
							Alert:  "MqMessageQueueBlock",
							Expr:   "acp_mq_queue_message_number > 0",
							For:    "1m",
							Labels: commonLables,
							Annotations: map[string]string{
								"summary":     "message queue blocked",
								"description": "mq topic {{ $labels.topic }} message queue may be blocked, message size is {{ humanize $value }}",
							},
						},
					},
				},
				{
					Name: "EventlogHealth",
					Rules: []*RulesConfig{
						{
							Alert:       "EventLogUnhealthy",
							Expr:        "event_log_exporter_health_status == 0",
							For:         "3m",
							Labels:      commonLables,
							Annotations: map[string]string{"summary": "eventlog unhealthy"},
						},
						{
							Alert:  "EventLogDown",
							Expr:   "absent(up{component=\"eventlog\"}) or up{component=\"eventlog\"}==0",
							For:    "3m",
							Labels: commonLables,
							Annotations: map[string]string{
								"description": "worker node {{ $labels.instance }} maybe down",
								"summary":     "eventlog service down",
							},
						},
					},
				},
				{
					Name: "WebcliHealth",
					Rules: []*RulesConfig{
						{
							Alert:  "WebcliDown",
							Expr:   "absent(up{component=\"webcli\"}) or up{component=\"webcli\"}==0",
							For:    "20s",
							Labels: commonLables,
							Annotations: map[string]string{
								"description": "webcli node {{ $labels.instance }} maybe down",
								"summary":     "webcli is down",
							},
						},
						{
							Alert:       "WebcliUnhealthy",
							Expr:        "webcli_exporter_health_status == 0",
							For:         "3m",
							Labels:      commonLables,
							Annotations: map[string]string{"summary": "webcli unhealthy"},
						},
						{
							Alert:       "WebcliUnhealthy",
							Expr:        "rate(webcli_exporter_execute_command_failed[5m]) > 5",
							For:         "3m",
							Labels:      commonLables,
							Annotations: map[string]string{"summary": "The number of errors that occurred while executing the command was greater than 5 per-second."},
						},
					},
				},
				{
					Name: "NodeHealth",
					Rules: []*RulesConfig{
						{
							Alert:  "NodeDown",
							Expr:   "absent(up{component=\"wt_node\"}) or up{component=\"wt_node\"} == 0",
							For:    "30s",
							Labels: commonLables,
							Annotations: map[string]string{
								"description": "node {{ $labels.instance }} may be down",
								"summary":     "wt_node is down",
							},
						},
						{
							Alert:       "HighCpuUsageOnNode",
							Expr:        "sum by(instance) (rate(process_cpu_seconds_total[5m])) * 100 > 70",
							For:         "5m",
							Labels:      commonLables,
							Annotations: map[string]string{"description": "{{ $labels.instance }} is using a LOT of CPU. CPU usage is {{ humanize $value}}%.", "summary": "HIGH CPU USAGE WARNING ON '{{ $labels.instance }}'"},
						},
						{
							Alert:       "HighLoadOnNode",
							Expr:        "count by (instance) (node_load5) > count by(instance)(count by(job, instance, cpu)(node_cpu))",
							For:         "5m",
							Labels:      commonLables,
							Annotations: map[string]string{"description": "{{ $labels.instance }} has a high load average. Load Average 5m is {{ humanize $value}}.", "summary": "HIGH LOAD AVERAGE WARNING ON '{{ $labels.instance }}'"},
						},
						{
							Alert:       "InodeFreerateLow",
							Expr:        "node_filesystem_files_free{fstype=~\"ext4|xfs\"} / node_filesystem_files{fstype=~\"ext4|xfs\"} < 0.3",
							For:         "5m",
							Labels:      commonLables,
							Annotations: map[string]string{"description": "the inode free rate is low of node {{ $labels.instance }}, current value is {{ humanize $value}}."},
						},
						{
							Alert:       "HighRootdiskUsageOnNode",
							Expr:        "(node_filesystem_size{mountpoint='/'} - node_filesystem_free{mountpoint='/'}) * 100 / node_filesystem_size{mountpoint='/'} > 85",
							For:         "5m",
							Labels:      commonLables,
							Annotations: map[string]string{"description": "More than 85% of disk used. Disk usage {{ humanize $value }} mountpoint {{ $labels.mountpoint }}%.", "summary": "LOW DISK SPACE WARING:NODE '{{ $labels.instance }}"},
						},
						{
							Alert:       "HighDockerdiskUsageOnNode",
							Expr:        "(node_filesystem_size{mountpoint='/var/lib/docker'} - node_filesystem_free{mountpoint='/var/lib/docker'}) * 100 / node_filesystem_size{mountpoint='/var/lib/docker'} > 85",
							For:         "5m",
							Labels:      commonLables,
							Annotations: map[string]string{"description": "More than 85% of disk used. Disk usage {{ humanize $value }} mountpoint {{ $labels.mountpoint }}%.", "summary": "LOW DISK SPACE WARING:NODE '{{ $labels.instance }}"},
						},
						{
							Alert:       "HighMemoryUsageOnNode",
							Expr:        "((node_memory_MemTotal - node_memory_MemAvailable) / node_memory_MemTotal) * 100 > 80",
							For:         "5m",
							Labels:      commonLables,
							Annotations: map[string]string{"description": "{{ $labels.instance }} is using a LOT of MEMORY. MEMORY usage is over {{ humanize $value}}%.", "summary": "HIGH MEMORY USAGE WARNING TASK ON '{{ $labels.instance }}'"},
						},
					},
				},
				{
					Name: "ClusterHealth",
					Rules: []*RulesConfig{
						{
							Alert:  "InsufficientClusteMemoryResources",
							Expr:   "max(wt_api_exporter_cluster_memory_total) - max(sum(namespace_resource_memory_request) by (instance)) < 2048",
							For:    "2m",
							Labels: commonLables,
							Annotations: map[string]string{
								"description": "Cluster residual scheduled memory is {{ humanize $value }} MB, less than 2048 MB",
								"summary":     "Insufficient Cluster Memory Resources",
							},
						},
						{
							Alert:  "InsufficientClusteCPUResources",
							Expr:   "max(wt_api_exporter_cluster_cpu_total) - max(sum(namespace_resource_cpu_request) by (instance)) < 500",
							For:    "2m",
							Labels: commonLables,
							Annotations: map[string]string{
								"description": "Cluster residual scheduled cpu is {{ humanize $value }}, less than 500",
								"summary":     "Insufficient Cluster CPU Resources",
							},
						},
						{
							Alert:  "InsufficientTenantEnvResources",
							Expr:   "sum(wt_api_exporter_tenant_env_memory_limit) by(namespace) - sum(namespace_resource_memory_request)by (namespace) < sum(wt_api_exporter_tenant_env_memory_limit) by(namespace) *0.2 and sum(wt_api_exporter_tenant_env_memory_limit) by(namespace) > 0",
							For:    "2m",
							Labels: commonLables,
							Annotations: map[string]string{
								"description": "TenantEnv available memory capacity {{ humanize $value }} MB, less than 20% of the limit",
								"summary":     "Insufficient TenantEnv memory Resources",
							},
						},
					},
				},
				{
					Name: "EtcdHealth",
					Rules: []*RulesConfig{
						{
							Alert:  "EtcdDown",
							Expr:   "absent(up{component=\"etcd\"}) or up{component=\"etcd\"}==0",
							For:    "1m",
							Labels: commonLables,
							Annotations: map[string]string{
								"description": "etcd node {{ $labels.instance }} may be down",
								"summary":     "etcd node is down",
							},
						},
						{
							Alert:  "EtcdLoseLeader",
							Expr:   "etcd_server_has_leader == 0",
							For:    "1m",
							Labels: commonLables,
							Annotations: map[string]string{
								"description": "etcd node {{ $labels.instance }} is lose leader",
								"summary":     "etcd lose leader",
							},
						},
						{
							Alert:  "InsufficientMembers",
							Expr:   "count(up{job=\"etcd\"} == 0) > (count(up{job=\"etcd\"}) / 2 - 1)",
							For:    "1m",
							Labels: getseverityLables("critical"),
							Annotations: map[string]string{
								"description": "If one more etcd member goes down the cluster will be unavailable",
								"summary":     "etcd cluster insufficient members",
							},
						},
						{
							Alert:  "HighNumberOfLeaderChanges",
							Expr:   "increase(etcd_server_leader_changes_seen_total{job=\"etcd\"}[1h]) > 3",
							For:    "1m",
							Labels: getseverityLables("warning"),
							Annotations: map[string]string{
								"description": "etcd instance {{ $labels.instance }} has seen {{ $value }} leader changes within the last hour",
								"summary":     "a high number of leader changes within the etcd cluster are happening",
							},
						},
						{
							Alert:  "HighNumberOfFailedGRPCRequests",
							Expr:   "sum(rate(etcd_grpc_requests_failed_total{job=\"etcd\"}[5m])) BY (grpc_method) / sum(rate(etcd_grpc_total{job=\"etcd\"}[5m])) BY (grpc_method) > 0.05",
							For:    "5m",
							Labels: getseverityLables("critical"),
							Annotations: map[string]string{
								"description": "{{ $value }}% of requests for {{ $labels.grpc_method }} failed on etcd instance {{ $labels.instance }}",
								"summary":     "a high number of gRPC requests are failing",
							},
						},
						{
							Alert:  "HighNumberOfFailedHTTPRequests",
							Expr:   "sum(rate(etcd_http_failed_total{job=\"etcd\"}[5m])) BY (method) / sum(rate(etcd_http_received_total{job=\"etcd\"}[5m]))BY (method) > 0.05",
							For:    "1m",
							Labels: getseverityLables("critical"),
							Annotations: map[string]string{
								"description": "{{ $value }}% of requests for {{ $labels.method }} failed on etcd instance {{ $labels.instance }}",
								"summary":     "a high number of HTTP requests are failing",
							},
						},
						{
							Alert:  "GRPCRequestsSlow",
							Expr:   "histogram_quantile(0.99, rate(etcd_grpc_unary_requests_duration_seconds_bucket[5m])) > 0.15",
							For:    "1m",
							Labels: getseverityLables("critical"),
							Annotations: map[string]string{
								"description": "on etcd instance {{ $labels.instance }} gRPC requests to {{ $labels.grpc_method}} are slow",
								"summary":     "slow gRPC requests",
							},
						},
						{
							Alert:  "HighNumberOfFailedHTTPRequests",
							Expr:   "sum(rate(etcd_http_failed_total{job=\"etcd\"}[5m])) BY (method) / sum(rate(etcd_http_received_total{job=\"etcd\"}[5m]))BY (method) > 0.05",
							For:    "1m",
							Labels: getseverityLables("critical"),
							Annotations: map[string]string{
								"description": "{{ $value }}% of requests for {{ $labels.method }} failed on etcd instance {{ $labels.instance }}",
								"summary":     "a high number of HTTP requests are failing",
							},
						},
						{
							Alert:  "HighNumberOfFailedHTTPRequests",
							Expr:   "sum(rate(etcd_http_failed_total{job=\"etcd\"}[5m])) BY (method) / sum(rate(etcd_http_received_total{job=\"etcd\"}[5m]))BY (method) > 0.05",
							For:    "1m",
							Labels: getseverityLables("critical"),
							Annotations: map[string]string{
								"description": "{{ $value }}% of requests for {{ $labels.method }} failed on etcd instance {{ $labels.instance }}",
								"summary":     "a high number of HTTP requests are failing",
							},
						},
						{
							Alert:  "DatabaseSpaceExceeded",
							Expr:   "etcd_mvcc_db_total_size_in_bytes/etcd_server_quota_backend_bytes > 0.80",
							For:    "1m",
							Labels: getseverityLables("critical"),
							Annotations: map[string]string{
								"description": "{{ $labels.instance }}, {{ $labels.job }} of etcd DB space uses more than 80%",
								"summary":     "Etcd DB space is overused",
								"runbook":     "Please consider manual compaction and defrag. https://github.com/etcd-io/etcd/blob/master/Documentation/op-guide/maintenance.md",
							},
						},
					},
				},
				{
					Name: "APIHealth",
					Rules: []*RulesConfig{
						{
							Alert:  "APIDown",
							Expr:   "absent(up{job=\"wtapi\"}) or up{job=\"wtapi\"}==0",
							For:    "1m",
							Labels: commonLables,
							Annotations: map[string]string{
								"description": "wt api node {{ $labels.instance }} maybe down",
								"summary":     "wt api node is down",
							},
						},
					},
				},
			},
		},
		config: config,
	}
	return a
}

// LoadAlertingRulesConfig load alerting rule config
func (a *AlertingRulesManager) LoadAlertingRulesConfig() error {
	logrus.Info("Load AlertingRules config file.")
	content, err := os.ReadFile(a.config.AlertingRulesFile)
	if err != nil {
		logrus.Error("Failed to read AlertingRules config file: ", err)
		logrus.Info("Init config file by default values.")
		return nil
	}
	if err := yaml.Unmarshal(content, a.RulesConfig); err != nil {
		logrus.Error("Unmarshal AlertingRulesConfig config string to object error.", err.Error())
		return err
	}
	logrus.Debugf("Loaded config file to memory: %+v", a)

	return nil
}

// SaveAlertingRulesConfig save alerting rule config
func (a *AlertingRulesManager) SaveAlertingRulesConfig() error {
	logrus.Info("Save alerting rules config file.")

	data, err := yaml.Marshal(a.RulesConfig)
	if err != nil {
		logrus.Error("Marshal alerting rules config to yaml error.", err.Error())
		return err
	}

	err = os.WriteFile(a.config.AlertingRulesFile, data, 0644)
	if err != nil {
		logrus.Error("Write alerting rules config file error.", err.Error())
		return err
	}

	return nil
}

// AddRules add rule
func (a *AlertingRulesManager) AddRules(val AlertingNameConfig) error {
	// group := a.RulesConfig.Groups
	// group = append(group, &val)
	return nil
}

// InitRulesConfig init rule config
func (a *AlertingRulesManager) InitRulesConfig() {
	_, err := os.Stat(a.config.AlertingRulesFile) //os.Stat获取文件信息
	if err != nil {
		if os.IsExist(err) {
			return
		}
		a.SaveAlertingRulesConfig()
		return
	}
}
