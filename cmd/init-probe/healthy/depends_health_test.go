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
	"testing"

	configclusterv3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	clusterv3 "github.com/envoyproxy/go-control-plane/envoy/service/cluster/v3"
	discoveryv3 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	serviceendpointv3 "github.com/envoyproxy/go-control-plane/envoy/service/endpoint/v3"
	servicelistenerv3 "github.com/envoyproxy/go-control-plane/envoy/service/listener/v3"
	envoyv3 "github.com/wutong-paas/wutong/node/core/envoy/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"gopkg.in/yaml.v3"
)

var testClusterID = "8cd9214e6b3d4476942b600f41bfefea_tcpmeshd3d6a722b632b854b6c232e4895e0cc6_gr5e0cc6"

var testXDSHost = "39.104.66.227:6101"

// var testClusterID = "2bf54c5a0b5a48a890e2dda8635cb507_tcpmeshed6827c0afdda50599b4108105c9e8e3_grc9e8e3"
//var testXDSHost = "127.0.0.1:6101"

func TestClientListener(t *testing.T) {
	conn, err := grpc.Dial(testXDSHost, grpc.WithTransportCredentials(
		insecure.NewCredentials(),
	))
	if err != nil {
		t.Fatal(err)
	}
	listenerDiscover := servicelistenerv3.NewListenerDiscoveryServiceClient(conn)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	res, err := listenerDiscover.FetchListeners(ctx, &discoveryv3.DiscoveryRequest{
		Node: &corev3.Node{
			Cluster: testClusterID,
			Id:      testClusterID,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Resources) == 0 {
		t.Fatal("no listeners")
	}
	t.Logf("version %s", res.GetVersionInfo())
	listeners := envoyv3.ParseListenerResource(res.Resources)
	printYaml(t, listeners)
}

func TestClientCluster(t *testing.T) {
	conn, err := grpc.Dial(testXDSHost, grpc.WithTransportCredentials(
		insecure.NewCredentials(),
	))
	if err != nil {
		t.Fatal(err)
	}
	clusterDiscover := clusterv3.NewClusterDiscoveryServiceClient(conn)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	res, err := clusterDiscover.FetchClusters(ctx, &discoveryv3.DiscoveryRequest{
		Node: &corev3.Node{
			Cluster: testClusterID,
			Id:      testClusterID,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Resources) == 0 {
		t.Fatal("no clusters")
	}
	t.Logf("version %s", res.GetVersionInfo())
	clusters := envoyv3.ParseClustersResource(res.Resources)
	for _, cluster := range clusters {
		if cluster.GetType() == configclusterv3.Cluster_LOGICAL_DNS {
			fmt.Println(cluster.Name)
		}
		printYaml(t, cluster)
	}
}

func printYaml(t *testing.T, data interface{}) {
	out, _ := yaml.Marshal(data)
	t.Log(string(out))
}

func TestClientEndpoint(t *testing.T) {
	conn, err := grpc.Dial(testXDSHost, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	endpointDiscover := serviceendpointv3.NewEndpointDiscoveryServiceClient(conn)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	res, err := endpointDiscover.FetchEndpoints(ctx, &discoveryv3.DiscoveryRequest{
		Node: &corev3.Node{
			Cluster: testClusterID,
			Id:      testClusterID,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Resources) == 0 {
		t.Fatal("no endpoints")
	}
	t.Logf("version %s", res.GetVersionInfo())
	endpoints := envoyv3.ParseLocalityLbEndpointsResource(res.Resources)
	for _, e := range endpoints {
		fmt.Println(e.GetClusterName())
	}
	printYaml(t, endpoints)
}

func TestNewDependServiceHealthController(t *testing.T) {
	controller, err := NewDependServiceHealthController()
	if err != nil {
		t.Fatal(err)
	}
	controller.Check()
}
