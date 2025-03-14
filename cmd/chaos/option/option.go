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

package option

import (
	"fmt"
	"runtime"

	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"github.com/wutong-paas/wutong/mq/client"
	"github.com/wutong-paas/wutong/util/containerutil"
)

// Config config server
type Config struct {
	EtcdEndPoints        []string
	EtcdCaFile           string
	EtcdCertFile         string
	EtcdKeyFile          string
	EtcdTimeout          int
	EtcdPrefix           string
	ClusterName          string
	MysqlConnectionInfo  string
	KanikoImage          string
	DBType               string
	PrometheusMetricPath string
	KubeConfig           string
	MaxTasks             int
	APIPort              int
	MQAPI                string
	DockerEndpoint       string
	HostIP               string
	CleanUp              bool
	Topic                string
	LogPath              string
	WtNamespace          string
	WtRepoName           string
	WTDataPVCName        string
	CachePVCName         string
	CacheMode            string
	CachePath            string
	ContainerRuntime     string
	RuntimeEndpoint      string
}

// Builder  builder server
type Builder struct {
	Config
	LogLevel string
	RunMode  string //default,sync
}

// NewBuilder new server
func NewBuilder() *Builder {
	return &Builder{}
}

// AddFlags config
func (a *Builder) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&a.LogLevel, "log-level", "info", "the builder log level")
	// fs.StringSliceVar(&a.EtcdEndPoints, "etcd-endpoints", []string{"http://127.0.0.1:2379"}, "etcd v3 cluster endpoints.")
	fs.StringVar(&a.EtcdCaFile, "etcd-ca", "", "")
	fs.StringVar(&a.EtcdCertFile, "etcd-cert", "", "")
	fs.StringVar(&a.EtcdKeyFile, "etcd-key", "", "")
	fs.IntVar(&a.EtcdTimeout, "etcd-timeout", 5, "etcd http timeout seconds")
	fs.StringVar(&a.EtcdPrefix, "etcd-prefix", "/store", "the etcd data save key prefix ")
	fs.StringVar(&a.PrometheusMetricPath, "metric", "/metrics", "prometheus metrics path")
	fs.StringVar(&a.KanikoImage, "kaniko-image", "swr.cn-southwest-2.myhuaweicloud.com/wutong/kaniko-executor:latest", "kaniko image version")
	fs.StringVar(&a.DBType, "db-type", "mysql", "db type mysql or etcd")
	fs.StringVar(&a.MysqlConnectionInfo, "mysql", "root:admin@tcp(127.0.0.1:3306)/region", "mysql db connection info")
	fs.StringVar(&a.KubeConfig, "kube-config", "", "kubernetes api server config file")
	fs.IntVar(&a.MaxTasks, "max-tasks", 50, "Maximum number of simultaneous build tasks")
	fs.IntVar(&a.APIPort, "api-port", 3228, "the port for api server")
	// fs.StringVar(&a.MQAPI, "mq-api", "127.0.0.1:6300", "acp_mq api")
	fs.StringVar(&a.RunMode, "run", "sync", "sync data when worker start")
	fs.StringVar(&a.DockerEndpoint, "dockerd", "127.0.0.1:2376", "dockerd endpoint")
	fs.StringVar(&a.HostIP, "hostIP", "", "Current node Intranet IP")
	fs.BoolVar(&a.CleanUp, "clean-up", true, "Turn on build version cleanup")
	fs.StringVar(&a.Topic, "topic", "builder", "Topic in mq,you coule choose `builder` or `windows_builder`")
	fs.StringVar(&a.LogPath, "log-path", "/wtdata/logs", "Where Docker log files and event log files are stored.")
	fs.StringVar(&a.WtNamespace, "wt-namespace", "wt-system", "wt component namespace")
	fs.StringVar(&a.WtRepoName, "wt-repo", "wt-repo", "wt component repo's name")
	fs.StringVar(&a.WTDataPVCName, "pvc-wtdata-name", "wtdata", "pvc name of wtdata")
	fs.StringVar(&a.CachePVCName, "pvc-cache-name", "cache", "pvc name of cache")
	fs.StringVar(&a.CacheMode, "cache-mode", "sharefile", "volume cache mount type, can be hostpath and sharefile, default is sharefile, which mount using pvc")
	fs.StringVar(&a.CachePath, "cache-path", "/cache", "volume cache mount path, when cache-mode using hostpath, default path is /cache")
	fs.StringVar(&a.ContainerRuntime, "container-runtime", containerutil.ContainerRuntimeDocker, "container runtime, support docker and containerd")
	fs.StringVar(&a.RuntimeEndpoint, "runtime-endpoint", containerutil.DefaultDockerSock, "container runtime endpoint")
	fs.StringVar(&a.MQAPI, "mq-api", "wt-mq:6300", "acp_mq api")
	fs.StringSliceVar(&a.EtcdEndPoints, "etcd-endpoints", []string{"http://wt-etcd:2379"}, "etcd v3 cluster endpoints.")
}

// SetLog 设置log
func (a *Builder) SetLog() {
	level, err := logrus.ParseLevel(a.LogLevel)
	if err != nil {
		logrus.Errorf("failed to parse log level: %s", err)
		return
	}
	logrus.SetLevel(level)
}

// CheckConfig check config
func (a *Builder) CheckConfig() error {
	if a.Topic != client.BuilderTopic && a.Topic != client.WindowsBuilderTopic {
		return fmt.Errorf("Topic is only suppory `%s` and `%s`", client.BuilderTopic, client.WindowsBuilderTopic)
	}
	if runtime.GOOS == "windows" {
		a.Topic = "windows_builder"
	}
	return nil
}
