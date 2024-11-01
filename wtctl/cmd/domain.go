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
	"bytes"
	"os/exec"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

// NewCmdDomain domain cmd
// v5.2 need refactoring
func NewCmdDomain() cli.Command {
	c := cli.Command{
		Name: "domain",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "ip",
				Usage: "ip address",
			},
			cli.StringFlag{
				Name:  "domain",
				Usage: "domain",
			},
		},
		Usage: "Default *.wtapps.cn domain resolution",
		Action: func(c *cli.Context) error {
			ip := c.String("ip")
			if len(ip) == 0 {
				logrus.Errorf("ip is required")
				return nil
			}
			domain := c.String("domain")
			cmd := exec.Command("bash", "/opt/wutong/bin/.domain.sh", ip, domain)
			outbuf := bytes.NewBuffer(nil)
			cmd.Stdout = outbuf
			cmd.Run()
			out := outbuf.String()
			logrus.Info(out)
			return nil
		},
	}
	return c
}
