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

package code

import (
	"fmt"
	"os"
	"path"

	"github.com/sirupsen/logrus"
	"github.com/wutong-paas/wutong/util"
	yaml "gopkg.in/yaml.v2"
)

// WutongFileConfig 云帮源码配置文件
type WutongFileConfig struct {
	Language  string                 `yaml:"language"`
	BuildPath string                 `yaml:"buildpath"`
	Ports     []Port                 `yaml:"ports"`
	Envs      map[string]interface{} `yaml:"envs"`
	Cmd       string                 `yaml:"cmd"`
	Services  []*Service             `yaml:"services"`
}

// Service contains
type Service struct {
	Name  string            `yaml:"name"`
	Ports []Port            `yaml:"ports"`
	Envs  map[string]string `yaml:"envs"`
}

// Port Port
type Port struct {
	Port     int    `yaml:"port"`
	Protocol string `yaml:"protocol"`
}

// ReadWutongFile
func ReadWutongFile(homepath string) (*WutongFileConfig, error) {
	filename := "wutongfile"
	if ok, _ := util.FileExists(path.Join(homepath, filename)); !ok {
		filename = "rainbondfile"
		if ok, _ := util.FileExists(path.Join(homepath, filename)); !ok {
			return nil, ErrWutongFileNotFound
		}
	}
	body, err := os.ReadFile(path.Join(homepath, filename))
	if err != nil {
		logrus.Error("read wutong file error,", err.Error())
		return nil, fmt.Errorf("read wutong file error")
	}
	var wtfile WutongFileConfig
	if err := yaml.Unmarshal(body, &wtfile); err != nil {
		logrus.Error("marshal wutong file error,", err.Error())
		return nil, fmt.Errorf("marshal wutong file error")
	}
	return &wtfile, nil
}
