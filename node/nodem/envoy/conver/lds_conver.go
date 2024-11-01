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

	listenerv3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	routev3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"github.com/pquerna/ffjson/ffjson"
	"github.com/sirupsen/logrus"
	api_model "github.com/wutong-paas/wutong/api/model"
	envoyv3 "github.com/wutong-paas/wutong/node/core/envoy/v3"
	corev1 "k8s.io/api/core/v1"
)

// GetPluginConfigs get plugin config model
func GetPluginConfigs(configs *corev1.ConfigMap) (*api_model.ResourceSpec, string, error) {
	if configs == nil {
		return nil, "", fmt.Errorf("no config for mesh")
	}
	var rs api_model.ResourceSpec
	if err := ffjson.Unmarshal([]byte(configs.Data["plugin-config"]), &rs); err != nil {
		logrus.Errorf("unmashal etcd v error, %v", err)
		return nil, "", err
	}
	return &rs, configs.Labels["plugin_id"], nil
}

// OneNodeListerner conver listerner of on envoy node
func OneNodeListerner(serviceAlias, namespace string, configs *corev1.ConfigMap, services []*corev1.Service) ([]types.Resource, error) {
	resources, _, err := GetPluginConfigs(configs)
	if err != nil {
		return nil, err
	}
	var listener []types.Resource
	var notCreateCommonHTTPListener = func() bool {
		if configs.Annotations["disable_create_http_common_listener"] == "true" {
			return true
		}
		if strings.Contains(configs.Name, "def-mesh") {
			return true
		}
		return false
	}()
	if len(resources.BaseServices) > 0 {
		for _, l := range upstreamListener(serviceAlias, namespace, resources.BaseServices, services, !notCreateCommonHTTPListener) {
			if err := l.Validate(); err != nil {
				logrus.Errorf("listener validate failure %s", err.Error())
			} else {
				logrus.Debugf("create listener %s for service %s", l.Name, serviceAlias)
				listener = append(listener, l)
			}
		}
	}
	if len(resources.BasePorts) > 0 {
		for _, l := range downstreamListener(serviceAlias, namespace, resources.BasePorts) {
			if err := l.Validate(); err != nil {
				logrus.Errorf("listener validate failure %s", err.Error())
			} else {
				logrus.Debugf("create listener %s for service %s", l.Name, serviceAlias)
				listener = append(listener, l)
			}
		}
	}
	if len(listener) == 0 {
		logrus.Warningf("configmap name: %s; plugin-config: %s; create listener zero length", configs.Name, configs.Data["plugin-config"])
	}
	return listener, nil
}

// upstreamListener handle upstream app listener
// handle kubernetes inner service
func upstreamListener(serviceAlias, namespace string, dependsServices []*api_model.BaseService, services []*corev1.Service, createHTTPListen bool) (ldsL []*listenerv3.Listener) {
	var ListennerConfig = make(map[string]*api_model.BaseService, len(dependsServices))
	for i, dService := range dependsServices {
		protoccol := "tcp"
		if strings.ToLower(dService.Protocol) == "udp" {
			protoccol = "udp"
		}
		if strings.ToLower(dService.Protocol) == "sctp" {
			protoccol = "sctp"
		}
		listennerName := fmt.Sprintf("%s_%s_%s_%s_%d", namespace, serviceAlias, dService.DependServiceAlias, protoccol, dService.Port)
		ListennerConfig[listennerName] = dependsServices[i]
	}
	var portMap = make(map[int32]int)
	var uniqRoute = make(map[string]*routev3.Route, len(services))
	var newVHL []*routev3.VirtualHost
	var VHLDomainMap = make(map[string]*routev3.VirtualHost)
	for _, service := range services {
		inner, ok := service.Labels["service_type"]
		if !ok || inner != "inner" {
			continue
		}
		port := service.Spec.Ports[0].Port
		protocol := service.Spec.Ports[0].Protocol
		var ListenPort = port
		//listener real port
		if value, ok := service.Labels["origin_port"]; ok {
			origin, _ := strconv.Atoi(value)
			if origin != 0 {
				ListenPort = int32(origin)
			}
		}
		clusterName := fmt.Sprintf("%s_%s_%s_%d", namespace, serviceAlias, GetServiceAliasByService(service), port)
		listennerName := fmt.Sprintf("%s_%s_%s_%s_%d", namespace, serviceAlias, GetServiceAliasByService(service), strings.ToLower(string(protocol)), ListenPort)
		destService := ListennerConfig[listennerName]
		statPrefix := fmt.Sprintf("%s_%s", serviceAlias, GetServiceAliasByService(service))
		var options envoyv3.WutongPluginOptions
		if destService != nil {
			options = envoyv3.GetOptionValues(destService.Options)
		} else {
			logrus.Warningf("destService is nil for service %s listenner name %s", serviceAlias, listennerName)
		}
		// Unique by listen port
		if _, ok := portMap[ListenPort]; !ok {
			//listener name depend listner port
			listenerName := fmt.Sprintf("%s_%s_%d", namespace, serviceAlias, ListenPort)
			var listener *listenerv3.Listener
			protocol := service.Labels["port_protocol"]
			if domain, ok := service.Annotations["domain"]; ok && domain != "" && (protocol == "https" || protocol == "http" || protocol == "grpc") {
				route := envoyv3.CreateRouteWithHostRewrite(domain, clusterName, "/", nil, 0)
				if route != nil {
					pvh := envoyv3.CreateRouteVirtualHost(
						fmt.Sprintf("%s_%s_%s_%d", namespace, serviceAlias, GetServiceAliasByService(service), port),
						[]string{"*"},
						nil,
						route,
					)
					if pvh != nil {
						listener = envoyv3.CreateHTTPListener(fmt.Sprintf("%s_%s_http_%d", namespace, serviceAlias, port), envoyv3.DefaultLocalhostListenerAddress, fmt.Sprintf("%s_%d", serviceAlias, port), uint32(port), nil, pvh)
					} else {
						logrus.Warnf("create route virtual host of domain listener %s failure", fmt.Sprintf("%s_%s_http_%d", namespace, serviceAlias, port))
					}
				}
			} else if protocol == "udp" {
				listener = envoyv3.CreateUDPListener(listenerName, clusterName, envoyv3.DefaultLocalhostListenerAddress, statPrefix, uint32(ListenPort))
			} else {
				listener = envoyv3.CreateTCPListener(listenerName, clusterName, envoyv3.DefaultLocalhostListenerAddress, statPrefix, uint32(ListenPort), options.TCPIdleTimeout)
			}
			if listener != nil {
				ldsL = append(ldsL, listener)
			} else {
				logrus.Warningf("create tcp listenner %s failure", listenerName)
				continue
			}
			portMap[ListenPort] = len(ldsL) - 1
		}

		portProtocol := service.Labels["port_protocol"]
		if destService != nil && destService.Protocol != "" {
			portProtocol = destService.Protocol
		}

		if portProtocol != "" {
			//TODO: support more protocol
			switch portProtocol {
			case "http", "https", "grpc":
				hashKey := options.RouteBasicHash()
				if oldroute, ok := uniqRoute[hashKey]; ok {
					oldrr := oldroute.Action.(*routev3.Route_Route)
					if oldrrwc, ok := oldrr.Route.ClusterSpecifier.(*routev3.RouteAction_WeightedClusters); ok {
						weight := envoyv3.CheckWeightSum(oldrrwc.WeightedClusters.Clusters, options.Weight)
						oldrrwc.WeightedClusters.Clusters = append(oldrrwc.WeightedClusters.Clusters, &routev3.WeightedCluster_ClusterWeight{
							Name:   clusterName,
							Weight: envoyv3.ConversionUInt32(weight),
						})
					}
				} else {
					var headerMatchers []*routev3.HeaderMatcher
					for _, header := range options.Headers {
						headerMatcher := envoyv3.CreateHeaderMatcher(header)
						if headerMatcher != nil {
							headerMatchers = append(headerMatchers, headerMatcher)
						}
					}
					var route *routev3.Route
					if domain, ok := service.Annotations["domain"]; ok && domain != "" {
						route = envoyv3.CreateRouteWithHostRewrite(domain, clusterName, options.Prefix, headerMatchers, options.Weight)
					} else {
						route = envoyv3.CreateRoute(clusterName, options.Prefix, headerMatchers, options.Weight)
					}

					if route != nil {
						if pvh := VHLDomainMap[strings.Join(options.Domains, "")]; pvh != nil {
							pvh.Routes = append(pvh.Routes, route)
						} else {
							pvh := envoyv3.CreateRouteVirtualHost(fmt.Sprintf("%s_%s_%s_%d", namespace, serviceAlias,
								GetServiceAliasByService(service), port), envoyv3.CheckDomain(options.Domains, portProtocol), nil, route)
							if pvh != nil {
								newVHL = append(newVHL, pvh)
								uniqRoute[hashKey] = route
								VHLDomainMap[strings.Join(options.Domains, "")] = pvh
							}
						}
					}
				}
			default:
				continue
			}
		}
	}
	// Sum of weights in the weighted_cluster should add up to 100
	for _, vh := range newVHL {
		for _, r := range vh.Routes {
			oldrr := r.Action.(*routev3.Route_Route)
			if oldrrwc, ok := oldrr.Route.ClusterSpecifier.(*routev3.RouteAction_WeightedClusters); ok {
				var weightSum uint32 = 0
				for _, cluster := range oldrrwc.WeightedClusters.Clusters {
					weightSum += cluster.Weight.Value
				}
				if weightSum != 100 {
					oldrrwc.WeightedClusters.Clusters[len(oldrrwc.WeightedClusters.Clusters)-1].Weight = envoyv3.ConversionUInt32(
						uint32(oldrrwc.WeightedClusters.Clusters[len(oldrrwc.WeightedClusters.Clusters)-1].Weight.Value) + uint32(100-weightSum))
				}
			}
		}
	}
	logrus.Debugf("virtual host is : %v", newVHL)
	// create common http listener
	if len(newVHL) > 0 && createHTTPListen {
		defaultListenPort := envoyv3.DefaultLocalhostListenerPort
		//remove 80 tcp listener is exist
		if i, ok := portMap[int32(defaultListenPort)]; ok {
			ldsL = append(ldsL[:i], ldsL[i+1:]...)
		}
		statsPrefix := fmt.Sprintf("%s_%d", serviceAlias, defaultListenPort)
		plds := envoyv3.CreateHTTPListener(
			fmt.Sprintf("%s_%s_http_%d", namespace, serviceAlias, defaultListenPort),
			envoyv3.DefaultLocalhostListenerAddress, statsPrefix, defaultListenPort, nil, newVHL...)
		if plds != nil {
			ldsL = append(ldsL, plds)
		} else {
			logrus.Warnf("create listenner %s failure", fmt.Sprintf("%s_%s_http_%d", namespace, serviceAlias, defaultListenPort))
		}
	}
	return
}

// downstreamListener handle app self port listener
func downstreamListener(serviceAlias, namespace string, ports []*api_model.BasePort) (ls []*listenerv3.Listener) {
	var portMap = make(map[int32]int, 0)
	for i := range ports {
		p := ports[i]
		port := int32(p.Port)
		clusterName := fmt.Sprintf("%s_%s_%d", namespace, serviceAlias, port)
		listenerName := clusterName
		statsPrefix := fmt.Sprintf("%s_%d", serviceAlias, port)
		if _, ok := portMap[port]; !ok {
			inboundConfig := envoyv3.GetWutongInboundPluginOptions(p.Options)
			options := envoyv3.GetOptionValues(p.Options)
			if p.Protocol == "http" || p.Protocol == "https" || p.Protocol == "grpc" {
				var limit []*routev3.RateLimit
				if inboundConfig.OpenLimit {
					limit = []*routev3.RateLimit{
						{
							Actions: []*routev3.RateLimit_Action{
								{
									ActionSpecifier: &routev3.RateLimit_Action_RemoteAddress_{
										RemoteAddress: &routev3.RateLimit_Action_RemoteAddress{},
									},
								},
							},
						},
					}
				}
				route := envoyv3.CreateRoute(clusterName, "/", nil, 100)
				if route == nil {
					logrus.Warning("create route cirtual route failure")
					continue
				}
				virtuals := envoyv3.CreateRouteVirtualHost(listenerName, []string{"*"}, limit, route)
				if virtuals == nil {
					logrus.Warning("create route cirtual failure")
					continue
				}
				listener := envoyv3.CreateHTTPListener(listenerName, "0.0.0.0", statsPrefix, uint32(p.ListenPort), &envoyv3.RateLimitOptions{
					Enable:                inboundConfig.OpenLimit,
					Domain:                inboundConfig.LimitDomain,
					RateServerClusterName: envoyv3.DefaultRateLimitServerClusterName,
					Stage:                 0,
				}, virtuals)
				if listener != nil {
					ls = append(ls, listener)
				}
			} else if p.Protocol == "udp" {
				listener := envoyv3.CreateUDPListener(listenerName, clusterName, "0.0.0.0", statsPrefix, uint32(p.ListenPort))
				if listener != nil {
					ls = append(ls, listener)
				} else {
					logrus.Warningf("create udp listener %s failure", listenerName)
					continue
				}
			} else {
				listener := envoyv3.CreateTCPListener(listenerName, clusterName, "0.0.0.0", statsPrefix, uint32(p.ListenPort), options.TCPIdleTimeout)
				if listener != nil {
					ls = append(ls, listener)
				} else {
					logrus.Warningf("create tcp listener %s failure", listenerName)
					continue
				}
			}
			portMap[port] = 1
		}
	}
	return
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
