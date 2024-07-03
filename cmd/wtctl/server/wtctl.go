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

package server

import (
	"os"
	"sort"

	"github.com/urfave/cli"
	version "github.com/wutong-paas/wutong/cmd"
	"github.com/wutong-paas/wutong/wtctl/cmd"
)

// App wtctl command app
var App *cli.App

// Run Run
func Run() error {
	App = cli.NewApp()
	App.Version = version.GetVersion()
	App.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "config, c",
			Value: "",
			Usage: "default <USER_HOME>/.wt/wtctl.yaml",
		},
		cli.StringFlag{
			Name:  "kubeconfig, kube",
			Value: "",
			Usage: "default <USER_HOME>/.kube/config",
		},
	}
	App.Commands = cmd.GetCmds()
	sort.Sort(cli.FlagsByName(App.Flags))
	sort.Sort(cli.CommandsByName(App.Commands))
	return App.Run(os.Args)
}
