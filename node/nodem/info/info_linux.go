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

//go:build linux
// +build linux

package info

import (
	"bufio"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"syscall"

	"github.com/wutong-paas/wutong/node/nodem/client"
)

// GetSystemInfo GetSystemInfo
func GetSystemInfo() (info client.NodeSystemInfo) {
	info.Architecture = runtime.GOARCH
	b, _ := os.ReadFile("/etc/machine-id")
	info.MachineID = string(b)
	output, _ := exec.Command("uname", "-r").Output()
	info.KernelVersion = string(output)
	osInfo := readOS()
	if name, ok := osInfo["NAME"]; ok {
		info.OSImage = name
	}
	info.OperatingSystem = runtime.GOOS
	info.MemorySize, _ = getMemory()
	info.NumCPU = int64(runtime.NumCPU())
	return info
}

func readOS() map[string]string {
	f, err := os.Open("/etc/os-release")
	if err != nil {
		return nil
	}
	defer f.Close()
	var info = make(map[string]string)
	r := bufio.NewReader(f)
	for {
		line, _, err := r.ReadLine()
		if err != nil {
			return info
		}
		lines := strings.Split(string(line), "=")
		if len(lines) >= 2 {
			info[lines[0]] = lines[1]
		}
	}
}

func getMemory() (total uint64, free uint64) {
	sysInfo := new(syscall.Sysinfo_t)
	err := syscall.Sysinfo(sysInfo)
	if err == nil {
		return uint64(sysInfo.Totalram) * uint64(sysInfo.Unit), sysInfo.Freeram * uint64(syscall.Getpagesize())
	}
	return 0, 0
}
