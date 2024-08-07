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

package handler

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"time"
)

// Info license 信息
type Info struct {
	Code       string   `json:"code"`
	Company    string   `json:"company"`
	Node       int64    `json:"node"`
	CPU        int64    `json:"cpu"`
	Memory     int64    `json:"memory"`
	TenantEnv  int64    `json:"tenant_env"`
	EndTime    string   `json:"end_time"`
	StartTime  string   `json:"start_time"`
	DataCenter int64    `json:"data_center"`
	ModuleList []string `json:"module_list"`
}

var key = []byte("qa123zxswe3532crfvtg123bnhymjuki")

// decrypt 解密算法
func decrypt(key []byte, encrypted string) ([]byte, error) {
	return []byte{}, nil
}

// ReadLicenseFromFile 从文件获取license
func ReadLicenseFromFile(licenseFile string) (Info, error) {

	info := Info{}
	//step1 read license file
	_, err := os.Stat(licenseFile)
	if err != nil {
		return info, err
	}
	infoBody, err := os.ReadFile(licenseFile)
	if err != nil {
		return info, errors.New("LICENSE文件不可读")
	}

	//step2 decryption info
	infoData, err := decrypt(key, string(infoBody))
	if err != nil {
		return info, errors.New("LICENSE解密发生错误。")
	}
	err = json.Unmarshal(infoData, &info)
	if err != nil {
		return info, errors.New("解码LICENSE文件发生错误")
	}
	return info, nil
}

// BasePack base pack
func BasePack(text []byte) (string, error) {
	token := ""
	encodeStr := base64.StdEncoding.EncodeToString(text)
	begin := 0
	if len([]byte(encodeStr)) > 40 {
		begin = randInt(0, (len([]byte(encodeStr)) - 40))
	} else {
		return token, fmt.Errorf("error license")
	}
	token = string([]byte(encodeStr)[begin:(begin + 40)])
	return token, nil
}

func randInt(min int, max int) int {
	rand.Seed(time.Now().UTC().UnixNano())
	return min + rand.Intn(max-min)
}
