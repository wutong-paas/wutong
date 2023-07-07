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

package cmd

import (
	"fmt"
	"os"

	"github.com/gosuri/uitable"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"github.com/wutong-paas/wutong/util/termtables"
	"github.com/wutong-paas/wutong/wtctl/clients"

	//"github.com/wutong-paas/wutong/eventlog/conf"
	"errors"

	config "github.com/wutong-paas/wutong/cmd/wtctl/option"
)

// NewCmdTenantEnv tenant env cmd
func NewCmdTenantEnv() cli.Command {
	c := cli.Command{
		Name:  "tenant_env",
		Usage: "wtctl tenant env -h",
		Subcommands: []cli.Command{
			{
				Name:  "list",
				Usage: "list all tenant env info",
				Action: func(c *cli.Context) error {
					Common(c)
					return getAllTenantEnv(c)
				},
			},
			{
				Name:  "get",
				Usage: "get all app details by specified tenant env name",
				Action: func(c *cli.Context) error {
					Common(c)
					return getTenantEnvInfo(c)
				},
			},
			{
				Name:  "res",
				Usage: "get tenant env resource details by specified tenant env name",
				Action: func(c *cli.Context) error {
					Common(c)
					return findTenantEnvResourceUsage(c)
				},
			},
			{
				Name:  "batchstop",
				Usage: "batch stop app by specified tenant env name",
				Flags: []cli.Flag{
					cli.BoolFlag{
						Name:  "f",
						Usage: "Continuous log output",
					},
					cli.StringFlag{
						Name:  "event_log_server",
						Usage: "event log server address",
					},
				},
				Action: func(c *cli.Context) error {
					Common(c)
					return stopTenantEnvService(c)
				},
			},
			{
				Name:  "setdefname",
				Usage: "set default tenant env name",
				Action: func(c *cli.Context) error {
					err := CreateTenantEnvFile(c.Args().First())
					if err != nil {
						logrus.Error("set default tenantEnvname fail", err.Error())
					}
					return nil
				},
			},
		},
	}
	return c
}

// wtctrl tenant env TENANT_NAME
func getTenantEnvInfo(c *cli.Context) error {
	tenantEnvID := c.Args().First()
	if tenantEnvID == "" {
		fmt.Println("Please provide tenant env name")
		os.Exit(1)
	}
	services, err := clients.RegionClient.TenantEnvs(tenantEnvID).Services("").List()
	handleErr(err)
	if services != nil {
		runtable := termtables.CreateTable()
		closedtable := termtables.CreateTable()
		runtable.AddHeaders("服务别名", "应用状态", "Deploy版本", "实例数量", "内存占用")
		closedtable.AddHeaders("租户ID", "服务ID", "服务别名", "应用状态", "Deploy版本")
		for _, service := range services {
			if service.CurStatus != "closed" && service.CurStatus != "closing" && service.CurStatus != "undeploy" && service.CurStatus != "deploying" {
				runtable.AddRow(service.ServiceAlias, service.CurStatus, service.DeployVersion, service.Replicas, fmt.Sprintf("%d Mb", service.ContainerMemory*service.Replicas))
			} else {
				closedtable.AddRow(service.TenantEnvID, service.ServiceID, service.ServiceAlias, service.CurStatus, service.DeployVersion)
			}
		}
		fmt.Println("运行中的应用：")
		fmt.Println(runtable.Render())
		fmt.Println("不在运行的应用：")
		fmt.Println(closedtable.Render())
		return nil
	}
	return nil
}
func findTenantEnvResourceUsage(c *cli.Context) error {
	tenantEnvName := c.Args().First()
	if tenantEnvName == "" {
		fmt.Println("Please provide tenant env name")
		os.Exit(1)
	}
	resources, err := clients.RegionClient.Resources().TenantEnvs(tenantEnvName).Get()
	handleErr(err)
	table := uitable.New()
	table.Wrap = true // wrap columns
	table.AddRow("租户名：", resources.Name)
	table.AddRow("租户ID：", resources.UUID)
	table.AddRow("正使用CPU资源：", fmt.Sprintf("%.2f Core", float64(resources.UsedCPU)/1000))
	table.AddRow("正使用内存资源：", fmt.Sprintf("%d %s", resources.UsedMEM, "Mb"))
	table.AddRow("正使用磁盘资源：", fmt.Sprintf("%.2f Mb", resources.UsedDisk/1024))
	table.AddRow("总分配CPU资源：", fmt.Sprintf("%.2f Core", float64(resources.AllocatedCPU)/1000))
	table.AddRow("总分配内存资源：", fmt.Sprintf("%d %s", resources.AllocatedMEM, "Mb"))
	fmt.Println(table)
	return nil
}

func getAllTenantEnv(c *cli.Context) error {
	tenantEnvs, err := clients.RegionClient.TenantEnvs("").List()
	handleErr(err)
	tenantEnvsTable := termtables.CreateTable()
	tenantEnvsTable.AddHeaders("TenantEnvAlias", "TenantEnvID", "TenantEnvLimit")
	for _, t := range tenantEnvs {
		tenantEnvsTable.AddRow(t.Name, t.UUID, fmt.Sprintf("%d GB", t.LimitMemory))
	}
	fmt.Print(tenantEnvsTable.Render())
	return nil
}

// CreateTenantEnvFile Create TenantEnv File
func CreateTenantEnvFile(tname string) error {
	filename, err := config.GetTenantEnvNamePath()
	if err != nil {
		logrus.Warn("Load config file error.")
		return errors.New("Load config file error")
	}
	f, err := os.OpenFile(filename, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
	if err != nil {
		logrus.Warn("load teantnamefile file", err.Error())
		f.Close()
		return err
	}
	_, err = f.WriteString(tname)
	if err != nil {
		logrus.Warn("write teantnamefile file", err.Error())
	}
	f.Close()
	return err
}
