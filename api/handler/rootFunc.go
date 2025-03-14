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
	"github.com/wutong-paas/wutong/cmd/api/option"
)

// RootAction  root function action struct
type RootAction struct{}

// CreateRootFuncManager get root func manager
func CreateRootFuncManager(conf option.Config) *RootAction {
	return &RootAction{}
}

// VersionInfo VersionInfo
type VersionInfo struct {
	Version []*LangInfo `json:"version"`
}

// LangInfo LangInfo
type LangInfo struct {
	Lang  string `json:"lang"`
	Major []*MajorInfo
}

// MajorInfo MajorInfo
type MajorInfo struct {
	Major int `json:"major"`
	Minor []*MinorInfo
}

// MinorInfo MinorInfo
type MinorInfo struct {
	Minor int   `json:"minor"`
	Patch []int `json:"patch"`
}
