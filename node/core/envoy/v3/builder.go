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

package v3

import (
	"fmt"
	"strings"

	configclusterv3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	endpointv3 "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	configratelimitv3 "github.com/envoyproxy/go-control-plane/envoy/config/ratelimit/v3"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	ratelimitv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/ratelimit/v3"
	httpconnectionmanagerv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	tcpproxyv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/tcp_proxy/v3"
	udpproxyv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/udp/udp_proxy/v3"
	typev3 "github.com/envoyproxy/go-control-plane/envoy/type/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/sirupsen/logrus"
	v1 "github.com/wutong-paas/wutong/node/core/envoy/v1"
	durationpb "google.golang.org/protobuf/types/known/durationpb"
	corev1 "k8s.io/api/core/v1"
)

// DefaultLocalhostListenerAddress -
var DefaultLocalhostListenerAddress = "127.0.0.1"

// DefaultLocalhostListenerPort -
var DefaultLocalhostListenerPort uint32 = 80

// CreateTCPListener listener builder
func CreateTCPListener(name, clusterName, address, statPrefix string, port uint32, idleTimeout int64) *listenerv3.Listener {
	if address == "" {
		address = DefaultLocalhostListenerAddress
	}
	tcpProxy := &tcpproxyv3.TcpProxy{
		StatPrefix: statPrefix,
		//todo:TcpProxy_WeightedClusters
		ClusterSpecifier: &tcpproxyv3.TcpProxy_Cluster{
			Cluster: clusterName,
		},
		IdleTimeout: ConverTimeDuration(idleTimeout),
	}
	if err := tcpProxy.Validate(); err != nil {
		logrus.Errorf("validate listener tcp proxy config failure %s", err.Error())
		return nil
	}
	listener := &listenerv3.Listener{
		Name:    name,
		Address: CreateSocketAddress("tcp", address, port),
		FilterChains: []*listenerv3.FilterChain{
			{
				Filters: []*listenerv3.Filter{
					{
						Name:       wellknown.TCPProxy,
						ConfigType: &listenerv3.Filter_TypedConfig{TypedConfig: Message2Any(tcpProxy)},
					},
				},
			},
		},
	}
	if err := listener.Validate(); err != nil {
		logrus.Errorf("validate listener config failure %s", err.Error())
		return nil
	}
	return listener
}

// CreateUDPListener create udp listenner
func CreateUDPListener(name, clusterName, address, statPrefix string, port uint32) *listenerv3.Listener {
	if address == "" {
		address = DefaultLocalhostListenerAddress
	}
	config := &udpproxyv3.UdpProxyConfig{
		StatPrefix: statPrefix,
		RouteSpecifier: &udpproxyv3.UdpProxyConfig_Cluster{
			Cluster: clusterName,
		},
	}
	if err := config.Validate(); err != nil {
		logrus.Errorf("validate listener udp config failure %s", err.Error())
		return nil
	}

	anyConfig := Message2Any(config)

	listener := &listenerv3.Listener{
		Name:    name,
		Address: CreateSocketAddress("udp", address, port),
		ListenerFilters: []*listenerv3.ListenerFilter{
			{
				Name: "envoy.filters.udp_listener.udp_proxy",
				ConfigType: &listenerv3.ListenerFilter_TypedConfig{
					TypedConfig: anyConfig,
				},
			},
		},
		// Listening on UDP without SO_REUSEPORT socket option may result to unstable packet proxying. Consider configuring the reuse_port listener option.
		ReusePort: true,
	}
	if err := listener.Validate(); err != nil {
		logrus.Errorf("validate listener config failure %s", err.Error())
		return nil
	}
	return listener
}

// RateLimitOptions rate limit options
type RateLimitOptions struct {
	Enable                bool
	Domain                string
	RateServerClusterName string
	Stage                 uint32
}

// DefaultRateLimitServerClusterName default rate limit server cluster name
var DefaultRateLimitServerClusterName = "rate_limit_service_cluster"

// CreateHTTPRateLimit create http rate limit
func CreateHTTPRateLimit(option RateLimitOptions) *ratelimitv3.RateLimit {
	httpRateLimit := &ratelimitv3.RateLimit{
		Domain: option.Domain,
		Stage:  option.Stage,
		RateLimitService: &configratelimitv3.RateLimitServiceConfig{
			GrpcService: &corev3.GrpcService{
				TargetSpecifier: &corev3.GrpcService_EnvoyGrpc_{
					EnvoyGrpc: &corev3.GrpcService_EnvoyGrpc{
						ClusterName: option.RateServerClusterName,
					},
				},
			},
		},
	}
	if err := httpRateLimit.Validate(); err != nil {
		logrus.Errorf("create http rate limit failure %s", err.Error())
		return nil
	}
	logrus.Debugf("service http rate limit for domain %s", httpRateLimit.Domain)
	return httpRateLimit
}

// CreateHTTPConnectionManager create http connection manager
func CreateHTTPConnectionManager(name, statPrefix string, rateOpt *RateLimitOptions, routes ...*routev3.VirtualHost) *httpconnectionmanagerv3.HttpConnectionManager {
	var httpFilters []*httpconnectionmanagerv3.HttpFilter
	if rateOpt != nil && rateOpt.Enable {
		httpFilters = append(httpFilters, &httpconnectionmanagerv3.HttpFilter{
			Name: wellknown.HTTPRateLimit,
			ConfigType: &httpconnectionmanagerv3.HttpFilter_TypedConfig{
				TypedConfig: Message2Any(CreateHTTPRateLimit(*rateOpt)),
			},
		})
	}
	httpFilters = append(httpFilters, &httpconnectionmanagerv3.HttpFilter{
		Name: wellknown.Router,
	})
	hcm := &httpconnectionmanagerv3.HttpConnectionManager{
		StatPrefix: statPrefix,
		RouteSpecifier: &httpconnectionmanagerv3.HttpConnectionManager_RouteConfig{
			RouteConfig: &routev3.RouteConfiguration{
				Name:         name,
				VirtualHosts: routes,
			},
		},
		HttpFilters: httpFilters,
	}
	if err := hcm.Validate(); err != nil {
		logrus.Errorf("validate http connertion manager config failure %s", err.Error())
		return nil
	}
	return hcm
}

// CreateHTTPListener create http manager listener
func CreateHTTPListener(name, address, statPrefix string, port uint32, rateOpt *RateLimitOptions, routes ...*routev3.VirtualHost) *listenerv3.Listener {
	hcm := CreateHTTPConnectionManager(name, statPrefix, rateOpt, routes...)
	if hcm == nil {
		logrus.Warningf("create http connection manager failure %s", name)
		return nil
	}
	listener := &listenerv3.Listener{
		Name: name,
		Address: &corev3.Address{
			Address: &corev3.Address_SocketAddress{
				SocketAddress: &corev3.SocketAddress{
					Protocol: corev3.SocketAddress_TCP,
					Address:  address,
					PortSpecifier: &corev3.SocketAddress_PortValue{
						PortValue: port,
					},
				},
			},
		},

		FilterChains: []*listenerv3.FilterChain{
			{
				Filters: []*listenerv3.Filter{
					{
						Name:       wellknown.HTTPConnectionManager,
						ConfigType: &listenerv3.Filter_TypedConfig{TypedConfig: Message2Any(hcm)},
					},
				},
			},
		},
	}
	if err := listener.Validate(); err != nil {
		logrus.Errorf("validate listener config failure %s", err.Error())
		return nil
	}
	return listener
}

// CreateSocketAddress create socket address
func CreateSocketAddress(protocol, address string, port uint32) *corev3.Address {
	if strings.HasPrefix(address, "https://") {
		address = strings.Split(address, "https://")[1]
	}
	if strings.HasPrefix(address, "http://") {
		address = strings.Split(address, "http://")[1]
	}
	return &corev3.Address{
		Address: &corev3.Address_SocketAddress{
			SocketAddress: &corev3.SocketAddress{
				Protocol: func(protocol string) corev3.SocketAddress_Protocol {
					if protocol == "udp" {
						return corev3.SocketAddress_UDP
					}
					return corev3.SocketAddress_TCP
				}(protocol),
				Address: address,
				PortSpecifier: &corev3.SocketAddress_PortValue{
					PortValue: port,
				},
			},
		},
	}
}

// CreateCircuitBreaker create down cluster circuitbreaker
func CreateCircuitBreaker(options WutongPluginOptions) *configclusterv3.CircuitBreakers {
	circuitBreakers := &configclusterv3.CircuitBreakers{
		Thresholds: []*configclusterv3.CircuitBreakers_Thresholds{
			{
				Priority:           corev3.RoutingPriority_DEFAULT,
				MaxConnections:     ConversionUInt32(uint32(options.MaxConnections)),
				MaxRequests:        ConversionUInt32(uint32(options.MaxRequests)),
				MaxRetries:         ConversionUInt32(uint32(options.MaxActiveRetries)),
				MaxPendingRequests: ConversionUInt32(uint32(options.MaxPendingRequests)),
			},
		},
	}
	if err := circuitBreakers.Validate(); err != nil {
		logrus.Errorf("validate envoy config circuitBreakers failure %s", err.Error())
		return nil
	}
	return circuitBreakers
}

// CreatOutlierDetection create up cluster OutlierDetection
func CreatOutlierDetection(options WutongPluginOptions) *configclusterv3.OutlierDetection {
	outlierDetection := &configclusterv3.OutlierDetection{
		Interval:           ConverTimeDuration(options.Interval),
		BaseEjectionTime:   ConverTimeDuration(options.BaseEjectionTimeMS / 1000),
		MaxEjectionPercent: ConversionUInt32(uint32(options.MaxEjectionPercent)),
		Consecutive_5Xx:    ConversionUInt32(uint32(options.ConsecutiveErrors)),
	}
	if err := outlierDetection.Validate(); err != nil {
		logrus.Errorf("validate envoy config outlierDetection failure %s", err.Error())
		return nil
	}
	return outlierDetection
}

// CreateRouteVirtualHost create route virtual host
func CreateRouteVirtualHost(name string, domains []string, rateLimits []*routev3.RateLimit, routes ...*routev3.Route) *routev3.VirtualHost {
	pvh := &routev3.VirtualHost{
		Name:       name,
		Domains:    domains,
		Routes:     routes,
		RateLimits: rateLimits,
	}
	if err := pvh.Validate(); err != nil {
		logrus.Errorf("route virtualhost config validate failure %s domains %s", err.Error(), domains)
		return nil
	}
	return pvh
}

// CreateRouteWithHostRewrite create route with hostRewrite
func CreateRouteWithHostRewrite(host, clusterName, prefix string, headers []*routev3.HeaderMatcher, weight uint32) *routev3.Route {
	var rout *routev3.Route
	if host != "" {
		var hostRewriteSpecifier *routev3.RouteAction_HostRewriteLiteral
		var clusterSpecifier *routev3.RouteAction_Cluster
		if strings.HasPrefix(host, "https://") {
			host = strings.Split(host, "https://")[1]
		}
		if strings.HasPrefix(host, "http://") {
			host = strings.Split(host, "http://")[1]
		}
		hostRewriteSpecifier = &routev3.RouteAction_HostRewriteLiteral{
			HostRewriteLiteral: host,
		}
		clusterSpecifier = &routev3.RouteAction_Cluster{
			Cluster: clusterName,
		}
		rout = &routev3.Route{
			Match: &routev3.RouteMatch{
				PathSpecifier: &routev3.RouteMatch_Prefix{
					Prefix: prefix,
				},
				Headers: headers,
			},
			Action: &routev3.Route_Route{
				Route: &routev3.RouteAction{
					ClusterSpecifier:     clusterSpecifier,
					Priority:             corev3.RoutingPriority_DEFAULT,
					HostRewriteSpecifier: hostRewriteSpecifier,
				},
			},
		}
		if err := rout.Validate(); err != nil {
			logrus.Errorf("route http route config validate failure %s", err.Error())
			return nil
		}

	}
	return rout
}

// CreateRoute create http route
func CreateRoute(clusterName, prefix string, headers []*routev3.HeaderMatcher, weight uint32) *routev3.Route {
	rout := &routev3.Route{
		Match: &routev3.RouteMatch{
			PathSpecifier: &routev3.RouteMatch_Prefix{
				Prefix: prefix,
			},
			Headers: headers,
		},
		Action: &routev3.Route_Route{
			Route: &routev3.RouteAction{
				ClusterSpecifier: &routev3.RouteAction_WeightedClusters{
					WeightedClusters: &routev3.WeightedCluster{
						Clusters: []*routev3.WeightedCluster_ClusterWeight{
							{
								Name:   clusterName,
								Weight: ConversionUInt32(weight),
							},
						},
					},
				},
				Priority: corev3.RoutingPriority_DEFAULT,
			},
		},
	}

	if err := rout.Validate(); err != nil {
		logrus.Errorf("route http route config validate failure %s", err.Error())
		return nil
	}
	return rout
}

// CreateHeaderMatcher create http route config header matcher
func CreateHeaderMatcher(header v1.Header) *routev3.HeaderMatcher {
	if header.Name == "" {
		return nil
	}
	headerMatcher := &routev3.HeaderMatcher{
		Name: header.Name,
		HeaderMatchSpecifier: &routev3.HeaderMatcher_PrefixMatch{
			PrefixMatch: header.Value,
		},
	}
	if err := headerMatcher.Validate(); err != nil {
		logrus.Errorf("route http header(%s) matcher config validate failure %s", header.Name, err.Error())
		return nil
	}
	return headerMatcher
}

// CreateEDSClusterConfig create grpc eds cluster config
func CreateEDSClusterConfig(serviceName string) *configclusterv3.Cluster_EdsClusterConfig {
	edsClusterConfig := &configclusterv3.Cluster_EdsClusterConfig{
		EdsConfig: &corev3.ConfigSource{
			ConfigSourceSpecifier: &corev3.ConfigSource_ApiConfigSource{
				ApiConfigSource: &corev3.ApiConfigSource{
					ApiType: corev3.ApiConfigSource_GRPC,
					GrpcServices: []*corev3.GrpcService{
						{
							TargetSpecifier: &corev3.GrpcService_EnvoyGrpc_{
								EnvoyGrpc: &corev3.GrpcService_EnvoyGrpc{
									ClusterName: "wutong_xds_cluster",
								},
							},
						},
					},
				},
			},
		},
		ServiceName: serviceName,
	}
	if err := edsClusterConfig.Validate(); err != nil {
		logrus.Errorf("validate eds cluster config failure %s", err.Error())
		return nil
	}
	return edsClusterConfig
}

// ClusterOptions cluster options
type ClusterOptions struct {
	Name                     string
	ServiceName              string
	ConnectionTimeout        *durationpb.Duration
	ClusterType              configclusterv3.Cluster_DiscoveryType
	MaxRequestsPerConnection *uint32
	OutlierDetection         *configclusterv3.OutlierDetection
	CircuitBreakers          *configclusterv3.CircuitBreakers
	DnsResolvers             []*corev3.Address
	HealthyPanicThreshold    int64
	TransportSocket          *corev3.TransportSocket
	LoadAssignment           *endpointv3.ClusterLoadAssignment
	Protocol                 string
	// grpc service name of health check
	GrpcHealthServiceName string
	//health check
	HealthTimeout  int64
	HealthInterval int64
}

// CreateCluster create cluster config
func CreateCluster(options ClusterOptions) *configclusterv3.Cluster {
	var edsClusterConfig *configclusterv3.Cluster_EdsClusterConfig
	if options.ClusterType == configclusterv3.Cluster_EDS {
		edsClusterConfig = CreateEDSClusterConfig(options.ServiceName)
		if edsClusterConfig == nil {
			logrus.Errorf("create eds cluster config failure")
			return nil
		}
	}
	cluster := &configclusterv3.Cluster{
		Name:                 options.Name,
		ClusterDiscoveryType: &configclusterv3.Cluster_Type{Type: options.ClusterType},
		ConnectTimeout:       options.ConnectionTimeout,
		LbPolicy:             configclusterv3.Cluster_ROUND_ROBIN,
		EdsClusterConfig:     edsClusterConfig,
		DnsResolvers:         options.DnsResolvers,
		OutlierDetection:     options.OutlierDetection,
		CircuitBreakers:      options.CircuitBreakers,
		CommonLbConfig: &configclusterv3.Cluster_CommonLbConfig{
			HealthyPanicThreshold: &typev3.Percent{Value: float64(options.HealthyPanicThreshold) / 100},
		},
	}
	if options.Protocol == "http2" || options.Protocol == "grpc" {
		cluster.Http2ProtocolOptions = &corev3.Http2ProtocolOptions{}
		// set grpc health check
		if options.Protocol == "grpc" && options.GrpcHealthServiceName != "" {
			cluster.HealthChecks = append(cluster.HealthChecks, &corev3.HealthCheck{
				Timeout:  ConverTimeDuration(options.HealthTimeout),
				Interval: ConverTimeDuration(options.HealthInterval),
				//The number of unhealthy health checks required before a host is marked unhealthy.
				//Note that for http health checking if a host responds with 503 this threshold is ignored and the host is considered unhealthy immediately.
				UnhealthyThreshold: ConversionUInt32(2),
				//The number of healthy health checks required before a host is marked healthy.
				//Note that during startup, only a single successful health check is required to mark a host healthy.
				HealthyThreshold: ConversionUInt32(1),
				HealthChecker: &corev3.HealthCheck_GrpcHealthCheck_{
					GrpcHealthCheck: &corev3.HealthCheck_GrpcHealthCheck{
						ServiceName: options.GrpcHealthServiceName,
					},
				}})
		}
	}
	if options.TransportSocket != nil {
		cluster.TransportSocket = options.TransportSocket
	}
	if options.LoadAssignment != nil {
		cluster.LoadAssignment = options.LoadAssignment
	}
	if options.MaxRequestsPerConnection != nil {
		cluster.MaxRequestsPerConnection = ConversionUInt32(*options.MaxRequestsPerConnection)
	}
	if err := cluster.Validate(); err != nil {
		logrus.Errorf("validate cluster config failure %s", err.Error())
		return nil
	}
	return cluster
}

// GetServiceAliasByService get service alias from k8s service
func GetServiceAliasByService(service *corev1.Service) string {
	//v5.1 and later
	if serviceAlias, ok := service.Labels["service_alias"]; ok {
		return serviceAlias
	}
	//version before v5.1
	if serviceAlias, ok := service.Spec.Selector["name"]; ok {
		return serviceAlias
	}
	return ""
}

// CreateDNSLoadAssignment create dns loadAssignment
func CreateDNSLoadAssignment(serviceAlias, namespace, domain string, service *corev1.Service) *endpointv3.ClusterLoadAssignment {
	destServiceAlias := GetServiceAliasByService(service)
	if destServiceAlias == "" {
		logrus.Errorf("service alias is empty in k8s service %s", service.Name)
		return nil
	}

	clusterName := fmt.Sprintf("%s_%s_%s_%d", namespace, serviceAlias, destServiceAlias, service.Spec.Ports[0].Port)
	var lendpoints []*endpointv3.LocalityLbEndpoints
	protocol := service.Labels["port_protocol"]
	port := service.Spec.Ports[0].Port
	var lbe []*endpointv3.LbEndpoint
	envoyAddress := CreateSocketAddress(protocol, domain, uint32(port))
	lbe = append(lbe, &endpointv3.LbEndpoint{
		HostIdentifier: &endpointv3.LbEndpoint_Endpoint{
			Endpoint: &endpointv3.Endpoint{
				Address:           envoyAddress,
				HealthCheckConfig: &endpointv3.Endpoint_HealthCheckConfig{PortValue: uint32(port)},
			},
		},
	})
	lendpoints = append(lendpoints, &endpointv3.LocalityLbEndpoints{LbEndpoints: lbe})
	cla := &endpointv3.ClusterLoadAssignment{
		ClusterName: clusterName,
		Endpoints:   lendpoints,
	}
	if err := cla.Validate(); err != nil {
		logrus.Errorf("endpoints discover validate failure %s", err.Error())
	}

	return cla
}
