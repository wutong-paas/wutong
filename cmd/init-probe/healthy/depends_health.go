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

package healthy

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	configclusterv3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	v3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	configendpointv3 "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	clusterv3 "github.com/envoyproxy/go-control-plane/envoy/service/cluster/v3"
	discoveryv3 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	serviceendpointv3 "github.com/envoyproxy/go-control-plane/envoy/service/endpoint/v3"
	"github.com/sirupsen/logrus"
	envoyv3 "github.com/wutong-paas/wutong/node/core/envoy/v3"
	"github.com/wutong-paas/wutong/util"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// DependServiceHealthController Detect the health of the dependent service
// Health based conditions：
// ------- lds: discover all dependent services
// ------- cds: discover all dependent services
// ------- sds: every service has at least one Ready instance
type DependServiceHealthController struct {
	clusters                        []*configclusterv3.Cluster
	interval                        time.Duration
	checkFunc                       []func() bool
	endpointClient                  serviceendpointv3.EndpointDiscoveryServiceClient
	clusterClient                   clusterv3.ClusterDiscoveryServiceClient
	clusterID                       string
	dependServiceNames              []string
	ignoreCheckEndpointsClusterName []string
}

// NewDependServiceHealthController create a controller
func NewDependServiceHealthController() (*DependServiceHealthController, error) {
	clusterID := os.Getenv("ENVOY_NODE_ID")
	if clusterID == "" {
		clusterID = fmt.Sprintf("%s_%s_%s", os.Getenv("POD_NAMESPACE"), os.Getenv("WT_PLUGIN_ID"), os.Getenv("WT_SERVICE_ALIAS"))
	}
	dsc := DependServiceHealthController{
		interval:  time.Second * 5,
		clusterID: clusterID,
	}
	dsc.checkFunc = append(dsc.checkFunc, dsc.checkListener)
	dsc.checkFunc = append(dsc.checkFunc, dsc.checkClusters)
	dsc.checkFunc = append(dsc.checkFunc, dsc.checkEDS)
	xDSHost := os.Getenv("XDS_HOST_IP")
	xDSHostPort := os.Getenv("XDS_HOST_PORT")
	if xDSHostPort == "" {
		xDSHostPort = "6101"
	}

	conn, err := grpc.Dial(fmt.Sprintf("%s:%s", xDSHost, xDSHostPort), grpc.WithTransportCredentials(insecure.NewCredentials()))
	// conn, err := grpc.NewClient(fmt.Sprintf("%s:%s", xDSHost, xDSHostPort),
	// 	grpc.WithTransportCredentials(
	// 		insecure.NewCredentials(),
	// 	),
	// )
	if err != nil {
		return nil, err
	}
	dsc.endpointClient = serviceendpointv3.NewEndpointDiscoveryServiceClient(conn)
	dsc.clusterClient = clusterv3.NewClusterDiscoveryServiceClient(conn)
	dsc.dependServiceNames = strings.Split(os.Getenv("STARTUP_SEQUENCE_DEPENDENCIES"), ",")
	return &dsc, nil
}

// Check check all conditions
func (d *DependServiceHealthController) Check() {
	logrus.Info("start denpenent health check.")
	ticker := time.NewTicker(d.interval)
	defer ticker.Stop()
	check := func() bool {
		for _, check := range d.checkFunc {
			if !check() {
				return false
			}
		}
		return true
	}
	for {
		if check() {
			logrus.Info("Depend services all check passed, will start service")
			return
		}
		<-ticker.C
	}
}

func (d *DependServiceHealthController) checkListener() bool {
	return true
}

func (d *DependServiceHealthController) checkClusters() bool {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	res, err := d.clusterClient.FetchClusters(ctx, &discoveryv3.DiscoveryRequest{
		Node: &v3.Node{
			Cluster: d.clusterID,
			Id:      d.clusterID,
		},
	})
	if err != nil {
		logrus.Errorf("discover depend services cluster failure %s", err.Error())
		return false
	}

	clusters := envoyv3.ParseClustersResource(res.Resources)
	d.ignoreCheckEndpointsClusterName = nil
	for _, cluster := range clusters {
		if cluster.GetType() == configclusterv3.Cluster_LOGICAL_DNS {
			d.ignoreCheckEndpointsClusterName = append(d.ignoreCheckEndpointsClusterName, cluster.Name)
		}
	}
	d.clusters = clusters
	return true
}

func (d *DependServiceHealthController) checkEDS() bool {
	logrus.Infof("start checking eds; dependent service cluster names: %s", d.dependServiceNames)
	if len(d.clusters) == len(d.ignoreCheckEndpointsClusterName) {
		logrus.Info("all dependent services is domain third service.")
		return true
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	res, err := d.endpointClient.FetchEndpoints(ctx, &discoveryv3.DiscoveryRequest{
		Node: &v3.Node{
			Cluster: d.clusterID,
			Id:      d.clusterID,
		},
	})
	if err != nil {
		logrus.Errorf("discover depend services endpoint failure %s", err.Error())
		return false
	}
	clusterLoadAssignments := envoyv3.ParseLocalityLbEndpointsResource(res.Resources)
	readyClusters := make(map[string]bool, len(clusterLoadAssignments))
	for _, cla := range clusterLoadAssignments {
		// clusterName := fmt.Sprintf("%s_%s_%s_%d", namespace, serviceAlias, destServiceAlias, service.Spec.Ports[0].Port)
		serviceName := ""
		clusterNameInfo := strings.Split(cla.GetClusterName(), "_")
		if len(clusterNameInfo) == 4 {
			serviceName = clusterNameInfo[2]
		}
		if serviceName == "" {
			continue
		}
		if ready, exist := readyClusters[serviceName]; exist && ready {
			continue
		}

		ready := func() bool {
			if util.StringArrayContains(d.ignoreCheckEndpointsClusterName, cla.ClusterName) {
				return true
			}
			if len(cla.Endpoints) > 0 && len(cla.Endpoints[0].LbEndpoints) > 0 {
				// first LbEndpoints healthy is not nil. so endpoint is not notreadyaddress
				if host, ok := cla.Endpoints[0].LbEndpoints[0].HostIdentifier.(*configendpointv3.LbEndpoint_Endpoint); ok {
					if host.Endpoint != nil && host.Endpoint.HealthCheckConfig != nil {
						logrus.Infof("depend service (%s) start complete", cla.ClusterName)
						return true
					}
				}
			}
			return false
		}()
		logrus.Infof("cluster name: %s; ready: %v", serviceName, ready)
		readyClusters[serviceName] = ready
	}
	for _, ignoreCheckEndpointsClusterName := range d.ignoreCheckEndpointsClusterName {
		clusterNameInfo := strings.Split(ignoreCheckEndpointsClusterName, "_")
		if len(clusterNameInfo) == 4 {
			readyClusters[clusterNameInfo[2]] = true
		}
	}
	for _, cn := range d.dependServiceNames {
		if cn != "" {
			if ready := readyClusters[cn]; !ready {
				logrus.Infof("%s not ready.", cn)
				return false
			}
		}
	}
	logrus.Info("all dependent services have been started.")

	return true
}
