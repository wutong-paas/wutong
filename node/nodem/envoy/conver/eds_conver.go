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

	endpointv3 "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"github.com/sirupsen/logrus"
	envoyv3 "github.com/wutong-paas/wutong/node/core/envoy/v3"
	corev1 "k8s.io/api/core/v1"
)

// OneNodeClusterLoadAssignment one envoy node endpoints
func OneNodeClusterLoadAssignment(serviceAlias, namespace string, endpoints []*corev1.Endpoints, services []*corev1.Service) (clusterLoadAssignment []types.Resource) {
	for i := range services {
		if domain, ok := services[i].Annotations["domain"]; ok && domain != "" {
			logrus.Warnf("service[sid: %s] endpoint id domain endpoint[domain: %s], use dns cluster type, do not create eds", services[i].GetUID(), domain)
			continue
		}
		service := services[i]
		destServiceAlias := GetServiceAliasByService(service)
		if destServiceAlias == "" {
			logrus.Errorf("service alias is empty in k8s service %s", service.Name)
			continue
		}
		clusterName := fmt.Sprintf("%s_%s_%s_%d", namespace, serviceAlias, destServiceAlias, service.Spec.Ports[0].Port)
		selectEndpoint := getEndpointsByServiceName(endpoints, service.Name)
		logrus.Debugf("select endpoints %d for service %s", len(selectEndpoint), service.Name)
		var lendpoints []*endpointv3.LocalityLbEndpoints // localityLbEndpoints just support only one content
		for _, en := range selectEndpoint {
			var notReadyAddress *corev1.EndpointAddress
			var notReadyPort *corev1.EndpointPort
			var notreadyToPort int
			for _, subset := range en.Subsets {
				for i, port := range subset.Ports {
					toport := int(port.Port)
					if serviceAlias == destServiceAlias {
						//use real port
						if originPort, ok := service.Labels["origin_port"]; ok {
							origin, err := strconv.Atoi(originPort)
							if err == nil {
								toport = origin
							}
						}
					}
					protocol := string(port.Protocol)
					if len(subset.Addresses) == 0 && len(subset.NotReadyAddresses) > 0 {
						notReadyAddress = &subset.NotReadyAddresses[0]
						notreadyToPort = toport
						notReadyPort = &subset.Ports[i]
					}
					getHealty := func() *endpointv3.Endpoint_HealthCheckConfig {
						return &endpointv3.Endpoint_HealthCheckConfig{
							PortValue: uint32(toport),
						}
					}
					if len(subset.Addresses) > 0 {
						var lbe []*endpointv3.LbEndpoint
						for _, address := range subset.Addresses {
							envoyAddress := envoyv3.CreateSocketAddress(protocol, address.IP, uint32(toport))
							lbe = append(lbe, &endpointv3.LbEndpoint{
								HostIdentifier: &endpointv3.LbEndpoint_Endpoint{
									Endpoint: &endpointv3.Endpoint{
										Address:           envoyAddress,
										HealthCheckConfig: getHealty(),
									},
								},
							})
						}
						if len(lbe) > 0 {
							lendpoints = append(lendpoints, &endpointv3.LocalityLbEndpoints{LbEndpoints: lbe})
						}
					}
				}
			}
			if len(lendpoints) == 0 && notReadyAddress != nil && notReadyPort != nil {
				var lbe []*endpointv3.LbEndpoint
				envoyAddress := envoyv3.CreateSocketAddress(string(notReadyPort.Protocol), notReadyAddress.IP, uint32(notreadyToPort))
				lbe = append(lbe, &endpointv3.LbEndpoint{
					HostIdentifier: &endpointv3.LbEndpoint_Endpoint{
						Endpoint: &endpointv3.Endpoint{
							Address: envoyAddress,
						},
					},
				})
				lendpoints = append(lendpoints, &endpointv3.LocalityLbEndpoints{LbEndpoints: lbe})
			}
		}
		cla := &endpointv3.ClusterLoadAssignment{
			ClusterName: clusterName,
			Endpoints:   lendpoints,
		}
		if err := cla.Validate(); err != nil {
			logrus.Errorf("endpoints discover validate failure %s", err.Error())
		} else {
			clusterLoadAssignment = append(clusterLoadAssignment, cla)
		}
	}
	if len(clusterLoadAssignment) == 0 {
		logrus.Warn("create clusterLoadAssignment zero length")
	}
	return clusterLoadAssignment
}

func getEndpointsByLables(endpoints []*corev1.Endpoints, slabels map[string]string) (re []*corev1.Endpoints) {
	for _, en := range endpoints {
		existLength := 0
		for k, v := range slabels {
			v2, ok := en.Labels[k]
			if ok && v == v2 {
				existLength++
			}
		}
		if existLength == len(slabels) {
			re = append(re, en)
		}
	}
	return
}

func getEndpointsByServiceName(endpoints []*corev1.Endpoints, serviceName string) (re []*corev1.Endpoints) {
	for _, en := range endpoints {
		if serviceName == en.Name {
			re = append(re, en)
		}
	}
	return
}
