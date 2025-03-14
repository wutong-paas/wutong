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

package conver

import (
	"fmt"
	"strconv"
	"strings"

	configclusterv3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	auth "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"github.com/sirupsen/logrus"
	api_model "github.com/wutong-paas/wutong/api/model"
	envoyv3 "github.com/wutong-paas/wutong/node/core/envoy/v3"
	"github.com/wutong-paas/wutong/node/utils"
	"google.golang.org/protobuf/types/known/anypb"
	corev1 "k8s.io/api/core/v1"
)

// OneNodeCluster conver cluster of on envoy node
func OneNodeCluster(serviceAlias, namespace string, configs *corev1.ConfigMap, services []*corev1.Service) ([]types.Resource, error) {
	resources, _, err := GetPluginConfigs(configs)
	if err != nil {
		return nil, err
	}
	var clusters []types.Resource
	if resources.BaseServices != nil && len(resources.BaseServices) > 0 {
		for _, cl := range upstreamClusters(serviceAlias, namespace, resources.BaseServices, services) {
			if err := cl.Validate(); err != nil {
				logrus.Errorf("cluster validate failure %s", err.Error())
			} else {
				clusters = append(clusters, cl)
			}
		}
	}
	if resources.BasePorts != nil && len(resources.BasePorts) > 0 {
		for _, cl := range downstreamClusters(serviceAlias, namespace, resources.BasePorts) {
			if err := cl.Validate(); err != nil {
				logrus.Errorf("cluster validate failure %s", err.Error())
			} else {
				clusters = append(clusters, cl)
			}
		}
	}
	if len(clusters) == 0 {
		logrus.Warningf("configmap name: %s; plugin-config: %s; create clusters zero length", configs.Name, configs.Data["plugin-config"])
	}
	return clusters, nil
}

// upstreamClusters handle upstream app cluster
// handle kubernetes inner service
func upstreamClusters(serviceAlias, namespace string, dependsServices []*api_model.BaseService, services []*corev1.Service) (cdsClusters []*configclusterv3.Cluster) {
	var clusterConfig = make(map[string]*api_model.BaseService, len(dependsServices))
	for i, dService := range dependsServices {
		depServiceIndex := fmt.Sprintf("%s_%s_%s_%d", namespace, serviceAlias, dService.DependServiceAlias, dService.Port)
		clusterConfig[depServiceIndex] = dependsServices[i]
	}
	for _, service := range services {
		inner, ok := service.Labels["service_type"]
		destServiceAlias := GetServiceAliasByService(service)
		port := service.Spec.Ports[0]
		if !ok || inner != "inner" {
			continue
		}
		getOptions := func() (d envoyv3.WutongPluginOptions) {
			relPort, _ := strconv.Atoi(service.Labels["origin_port"])
			if relPort == 0 {
				relPort = int(port.TargetPort.IntVal)
			}
			depServiceIndex := fmt.Sprintf("%s_%s_%s_%d", namespace, serviceAlias, GetServiceAliasByService(service), relPort)
			if _, ok := clusterConfig[depServiceIndex]; ok {
				return envoyv3.GetOptionValues(clusterConfig[depServiceIndex].Options)
			}
			return envoyv3.GetOptionValues(nil)
		}
		var clusterOption envoyv3.ClusterOptions
		clusterOption.Name = fmt.Sprintf("%s_%s_%s_%v", namespace, serviceAlias, GetServiceAliasByService(service), port.Port)
		options := getOptions()
		clusterOption.OutlierDetection = envoyv3.CreatOutlierDetection(options)
		clusterOption.CircuitBreakers = envoyv3.CreateCircuitBreaker(options)
		clusterOption.ServiceName = fmt.Sprintf("%s_%s_%s_%v", namespace, serviceAlias, destServiceAlias, port.Port)
		if domain, ok := service.Annotations["domain"]; ok && domain != "" {
			logrus.Debugf("domain endpoint[%s], create logical_dns cluster: ", domain)
			clusterOption.ClusterType = configclusterv3.Cluster_LOGICAL_DNS
			clusterOption.LoadAssignment = envoyv3.CreateDNSLoadAssignment(serviceAlias, namespace, domain, service)
			if strings.HasPrefix(domain, "https://") {
				splitDomain := strings.Split(domain, "https://")
				if len(splitDomain) == 2 {
					clusterOption.TransportSocket = transportSocket(clusterOption.Name, splitDomain[1])
				}
			}
		} else {
			clusterOption.ClusterType = configclusterv3.Cluster_EDS
		}
		clusterOption.HealthyPanicThreshold = options.HealthyPanicThreshold
		clusterOption.ConnectionTimeout = envoyv3.ConverTimeDuration(options.ConnectionTimeout)
		// set port realy protocol
		portProtocol := service.Labels["port_protocol"]
		clusterOption.Protocol = portProtocol
		clusterOption.GrpcHealthServiceName = options.GrpcHealthServiceName
		clusterOption.HealthTimeout = options.HealthCheckTimeout
		clusterOption.HealthInterval = options.HealthCheckInterval
		cluster := envoyv3.CreateCluster(clusterOption)
		if cluster != nil {
			logrus.Debugf("cluster is : %v", cluster)
			cdsClusters = append(cdsClusters, cluster)
		}
	}
	return
}

func transportSocket(name, domain string) *corev3.TransportSocket {
	logrus.Debugf("https domain tlsContext: %s", domain)
	// refer to: https://www.envoyproxy.io/docs/envoy/v1.17.2/api-v2/api/v2/auth/tls.proto#auth-upstreamtlscontext
	tlsContext := &anypb.Any{}
	err := tlsContext.MarshalFrom(&auth.UpstreamTlsContext{Sni: domain})
	// tlsContext, err := ptypes.MarshalAny(&auth.UpstreamTlsContext{Sni: domain})
	if err != nil {
		logrus.Errorf("error marshaling tls context to transport_socket config for cluster %s, err=%v",
			name, err)
		// no tls context for the cluster
		return nil
	}
	return &corev3.TransportSocket{
		Name: utils.EnvoyTLSSocketName,
		ConfigType: &corev3.TransportSocket_TypedConfig{
			TypedConfig: tlsContext,
		},
	}
}

// downstreamClusters handle app self cluster
// only local port
func downstreamClusters(serviceAlias, namespace string, ports []*api_model.BasePort) (cdsClusters []*configclusterv3.Cluster) {
	for i := range ports {
		port := ports[i]
		address := envoyv3.CreateSocketAddress(port.Protocol, "127.0.0.1", uint32(port.Port))
		clusterName := fmt.Sprintf("%s_%s_%v", namespace, serviceAlias, port.Port)
		option := envoyv3.GetOptionValues(port.Options)
		cluster := envoyv3.CreateCluster(envoyv3.ClusterOptions{
			Name:                     clusterName,
			ConnectionTimeout:        envoyv3.ConverTimeDuration(option.ConnectionTimeout),
			ServiceName:              "",
			ClusterType:              configclusterv3.Cluster_STATIC,
			CircuitBreakers:          envoyv3.CreateCircuitBreaker(option),
			OutlierDetection:         envoyv3.CreatOutlierDetection(option),
			MaxRequestsPerConnection: option.MaxRequestsPerConnection,
			DnsResolvers:             []*corev3.Address{address},
			HealthyPanicThreshold:    option.HealthyPanicThreshold,
		})
		if cluster != nil {
			cdsClusters = append(cdsClusters, cluster)
		}
	}
	return
}
