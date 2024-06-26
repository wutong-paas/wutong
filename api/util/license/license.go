// WUTONG, Application Management Platform
// Copyright (C) 2021-2021 Wutong Co., Ltd.

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

package license

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
)

// LicenseInfo license data
type LicenseInfo struct {
	Code      string    `json:"code"`
	Company   string    `json:"company"`
	Node      int64     `json:"node"`
	Memory    int64     `json:"memory"`
	EndTime   string    `json:"end_time"`
	StartTime string    `json:"start_time"`
	Features  []Feature `json:"features"`
}

func (l *LicenseInfo) HaveFeature(code string) bool {
	for _, f := range l.Features {
		if f.Code == strings.ToUpper(code) {
			return true
		}
	}
	return false
}

type Feature struct {
	Name string `json:"name"`
	Code string `json:"code"`
}

var licenseInfo *LicenseInfo

// ReadLicense -
func ReadLicense() *LicenseInfo {
	if licenseInfo != nil {
		return licenseInfo
	}
	licenseFile := os.Getenv("LICENSE_PATH")
	if licenseFile == "" {
		return nil
	}
	//step1 read license file
	_, err := os.Stat(licenseFile)
	if err != nil {
		logrus.Error("read LICENSE file failure：" + err.Error())
		return nil
	}
	infoBody, err := os.ReadFile(licenseFile)
	if err != nil {
		logrus.Error("read LICENSE file failure：" + err.Error())
		return nil
	}

	//step2 decryption info
	key := os.Getenv("LICENSE_KEY")
	if key == "" {
		logrus.Error("not define license Key")
		return nil
	}
	infoData, err := Decrypt(getKey(key), string(infoBody))
	if err != nil {
		logrus.Error("decrypt LICENSE failure " + err.Error())
		return nil
	}
	info := LicenseInfo{}
	err = json.Unmarshal(infoData, &info)
	if err != nil {
		logrus.Error("decrypt LICENSE json failure " + err.Error())
		return nil
	}
	licenseInfo = &info
	return &info
}

func Decrypt(key []byte, encrypted string) ([]byte, error) {
	ciphertext, err := base64.RawURLEncoding.DecodeString(encrypted)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	if len(ciphertext) < aes.BlockSize {
		return nil, errors.New("ciphertext too short")
	}
	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]
	cfb := cipher.NewCFBDecrypter(block, iv)
	cfb.XORKeyStream(ciphertext, ciphertext)
	return ciphertext, nil
}
func getKey(source string) []byte {
	if len(source) > 32 {
		return []byte(source[:32])
	}
	return append(defaultKey[len(source):], []byte(source)...)
}

var defaultKey = []byte{113, 119, 101, 114, 116, 121, 117, 105, 111, 112, 97, 115, 100, 102, 103, 104, 106, 107, 108, 122, 120, 99, 118, 98, 110, 109, 49, 50, 51, 52, 53, 54}
