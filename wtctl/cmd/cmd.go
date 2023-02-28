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
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	conf "github.com/wutong-paas/wutong/cmd/wtctl/option"
	"github.com/wutong-paas/wutong/wtctl/clients"
)

// GetCmds GetCmds
func GetCmds() []cli.Command {
	cmds := []cli.Command{}
	cmds = append(cmds, NewCmdInstall())
	cmds = append(cmds, NewCmdService())
	cmds = append(cmds, NewCmdTenantEnv())
	cmds = append(cmds, NewCmdNode())
	cmds = append(cmds, NewCmdCluster())
	cmds = append(cmds, NewSourceBuildCmd())
	cmds = append(cmds, NewCmdAnsible())
	cmds = append(cmds, NewCmdLicense())
	cmds = append(cmds, NewCmdGateway())
	cmds = append(cmds, NewCmdEnvoy())
	cmds = append(cmds, NewCmdConfig())
	cmds = append(cmds, NewCmdRegistry())
	return cmds
}

// Common Common
func Common(c *cli.Context) {
	config, err := conf.LoadConfig(c)
	if err != nil {
		logrus.Warn("Load config file error.", err.Error())
	}
	kc := c.GlobalString("kubeconfig")
	if kc != "" {
		config.Kubernets.KubeConf = kc
	}
	if err := clients.InitClient(config.Kubernets.KubeConf); err != nil {
		logrus.Errorf("error config k8s,details %s", err.Error())
	}
	//clients.SetInfo(config.RegionAPI.URL, config.RegionAPI.Token)
	if err := clients.InitRegionClient(config.RegionAPI); err != nil {
		logrus.Fatal("error config region")
	}

}

// CommonWithoutRegion Common
func CommonWithoutRegion(c *cli.Context) {
	config, err := conf.LoadConfig(c)
	if err != nil {
		logrus.Warn("Load config file error.", err.Error())
	}
	kc := c.GlobalString("kubeconfig")
	if kc != "" {
		config.Kubernets.KubeConf = kc
	}
	if err := clients.InitClient(config.Kubernets.KubeConf); err != nil {
		logrus.Errorf("error config k8s,details %s", err.Error())
	}
}

// fatal prints the message (if provided) and then exits. If V(2) or greater,
// glog.Fatal is invoked for extended information.
func fatal(msg string, code int) {
	if len(msg) > 0 {
		// add newline if needed
		if !strings.HasSuffix(msg, "\n") {
			msg += "\n"
		}
		fmt.Fprint(os.Stderr, msg)
	}
	os.Exit(code)
}

// GetTenantEnvNamePath Get TenantEnvname Path
func GetTenantEnvNamePath() string {
	tenantEnvnamepath, err := conf.GetTenantEnvNamePath()
	if err != nil {
		logrus.Warn("Ger Home error", err.Error())
		return tenantEnvnamepath
	}
	return tenantEnvnamepath
}
