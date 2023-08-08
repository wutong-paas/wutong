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

package option

import (
	"os"
	"path"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"github.com/wutong-paas/wutong/api/region"
	"github.com/wutong-paas/wutong/chaos/sources"
	yaml "gopkg.in/yaml.v2"
	//"strings"
)

var config Config

// Config Config
type Config struct {
	Kubernets     Kubernets      `yaml:"kube"`
	RegionAPI     region.APIConf `yaml:"region_api"`
	DockerLogPath string         `yaml:"docker_log_path"`
}

// RegionMysql RegionMysql
type RegionMysql struct {
	URL      string `yaml:"url"`
	Pass     string `yaml:"pass"`
	User     string `yaml:"user"`
	Database string `yaml:"database"`
}

// Kubernets Kubernets
type Kubernets struct {
	KubeConf string `yaml:"kube-conf"`
}

// LoadConfig 加载配置
func LoadConfig(ctx *cli.Context) (Config, error) {
	config = Config{
		RegionAPI: region.APIConf{
			Endpoints: []string{"http://127.0.0.1:8888"},
		},
	}
	configfile := ctx.GlobalString("config")
	if configfile == "" {
		home, _ := sources.Home()
		configfile = path.Join(home, ".wt", "wtctl.yaml")
	}
	_, err := os.Stat(configfile)
	if err != nil {
		return config, nil
	}
	data, err := os.ReadFile(configfile)
	if err != nil {
		logrus.Warning("Read config file error ,will get config from region.", err.Error())
		return config, err
	}
	if err := yaml.Unmarshal(data, &config); err != nil {
		logrus.Warning("Read config file error ,will get config from region.", err.Error())
		return config, err
	}
	return config, nil
}

// GetConfig GetConfig
func GetConfig() Config {
	return config
}

// Get TenantEnvNamePath
func GetTenantEnvNamePath() (tenantEnvnamepath string, err error) {
	home, err := sources.Home()
	if err != nil {
		logrus.Warn("Get Home Dir error.", err.Error())
		return tenantEnvnamepath, err
	}
	tenantEnvnamepath = path.Join(home, ".wt", "tenantEnv.txt")
	return tenantEnvnamepath, err
}
