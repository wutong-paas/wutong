// WUTONG, Application Management Platform
// Copyright (C) 2014-2019 Wutong Co., Ltd.

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

package cmd

import (
	"context"
	"fmt"
	"strings"

	corev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	endpointv3 "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	discoveryv3 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	serviceendpointv3 "github.com/envoyproxy/go-control-plane/envoy/service/endpoint/v3"
	"github.com/gosuri/uitable"
	"github.com/urfave/cli"
	envoyv3 "github.com/wutong-paas/wutong/node/core/envoy/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// NewCmdEnvoy envoy cmd
func NewCmdEnvoy() cli.Command {
	c := cli.Command{
		Name:  "envoy",
		Usage: "envoy management related commands",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "address",
				Usage: "node envoy api address",
				Value: "127.0.0.1:6101",
			},
			cli.StringFlag{
				Name:  "node",
				Usage: "envoy node name",
			},
		},
		Subcommands: []cli.Command{
			{
				Name:  "endpoints",
				Usage: "list envoy node endpoints",
				Action: func(c *cli.Context) error {
					return listEnvoyEndpoint(c)
				},
			},
		},
	}
	return c
}

func listEnvoyEndpoint(c *cli.Context) error {
	if c.GlobalString("node") == "" {
		showError("node name can not be empty,please define by --node")
	}
	cli, err := grpc.NewClient(c.GlobalString("address"), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		showError(err.Error())
	}
	endpointDiscover := serviceendpointv3.NewEndpointDiscoveryServiceClient(cli)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	res, err := endpointDiscover.FetchEndpoints(ctx, &discoveryv3.DiscoveryRequest{
		Node: &corev3.Node{
			Cluster: c.GlobalString("node"),
			Id:      c.GlobalString("node"),
		},
	})
	if err != nil {
		showError(err.Error())
	}
	if len(res.Resources) == 0 {
		showError("not find endpoints")
	}
	endpoints := envoyv3.ParseLocalityLbEndpointsResource(res.Resources)
	table := uitable.New()
	table.Wrap = true // wrap columns
	for _, end := range endpoints {
		table.AddRow(end.ClusterName, strings.Join(func() []string {
			var re []string
			for _, e := range end.Endpoints {
				for _, a := range e.LbEndpoints {
					if lbe, ok := a.HostIdentifier.(*endpointv3.LbEndpoint_Endpoint); ok && lbe != nil {
						if address, ok := lbe.Endpoint.Address.Address.(*corev3.Address_SocketAddress); ok && address != nil {
							if port, ok := address.SocketAddress.PortSpecifier.(*corev3.SocketAddress_PortValue); ok && port != nil {
								re = append(re, fmt.Sprintf("%s:%d", address.SocketAddress.Address, port.PortValue))
							}
						}
					}
				}
			}
			return re
		}(), ";"))
	}
	fmt.Println(table)
	return nil
}
