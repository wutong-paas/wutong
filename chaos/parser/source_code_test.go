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

package parser

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/wutong-paas/wutong/chaos"
	"github.com/wutong-paas/wutong/chaos/parser/types"
	"github.com/wutong-paas/wutong/chaos/sources"
)

func TestParseDockerfileInfo(t *testing.T) {
	parse := &SourceCodeParse{
		source:  "source",
		ports:   make(map[int]*types.Port),
		volumes: make(map[string]*types.Volume),
		envs:    make(map[string]*types.Env),
		logger:  nil,
		image:   ParseImageName(chaos.RUNNERIMAGENAME),
		args:    []string{"start", "web"},
	}
	parse.parseDockerfileInfo("./Dockerfile")
	fmt.Println(parse.GetServiceInfo())
}

// ServiceCheckResult 应用检测结果
type ServiceCheckResult struct {
	//检测状态 Success Failure
	CheckStatus string         `json:"check_status"`
	ErrorInfos  ParseErrorList `json:"error_infos"`
	ServiceInfo []ServiceInfo  `json:"service_info"`
}

func TestSourceCode(t *testing.T) {
	sc := sources.CodeSourceInfo{
		ServerType:    "",
		RepositoryURL: "https://github.com/barnettZQG/fserver.git",
		Branch:        "master",
	}
	b, _ := json.Marshal(sc)
	p := CreateSourceCodeParse(string(b), nil)
	err := p.Parse()
	if err != nil && err.IsFatalError() {
		t.Fatal(err)
	}
	re := ServiceCheckResult{
		CheckStatus: "Failure",
		ErrorInfos:  err,
		ServiceInfo: p.GetServiceInfo(),
	}
	body, _ := json.Marshal(re)
	fmt.Printf("%s \n", string(body))
}

func TestOSSCheck(t *testing.T) {
	sc := sources.CodeSourceInfo{
		ServerType:    "oss",
		RepositoryURL: "http://8081.wt021644.64q1jlfb.17f4cc.wtapps.cn/artifactory/dev/java-war-demo-master.tar",
		User:          "demo",
		Password:      "wt123465!",
	}
	b, _ := json.Marshal(sc)
	p := CreateSourceCodeParse(string(b), nil)
	err := p.Parse()
	if err != nil && err.IsFatalError() {
		t.Fatal(err)
	}
	re := ServiceCheckResult{
		CheckStatus: "Success",
		ErrorInfos:  err,
		ServiceInfo: p.GetServiceInfo(),
	}
	body, _ := json.Marshal(re)
	fmt.Printf("%s \n", string(body))
}
