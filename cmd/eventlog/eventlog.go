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

package main

import (
	"os"

	_ "net/http/pprof"

	"github.com/spf13/pflag"
	"github.com/wutong-paas/wutong/cmd"
	"github.com/wutong-paas/wutong/cmd/eventlog/server"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "version" {
		cmd.ShowVersion("eventlog") // 打印版本信息
	}
	s := server.NewLogServer()
	s.AddFlags(pflag.CommandLine)
	pflag.Parse()
	s.InitConf()
	s.InitLog()

	if err := s.Run(); err != nil {
		s.Logger.Error("server run error.", err.Error())
		os.Exit(1)
	}
}
