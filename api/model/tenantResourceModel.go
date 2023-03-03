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

package model

import (
	dbmodel "github.com/wutong-paas/wutong/db/model"
)

// TenantEnvResList TenantEnvResList
type TenantEnvResList []*TenantEnvResource

// PagedTenantEnvResList PagedTenantEnvResList
type PagedTenantEnvResList struct {
	List   []*TenantEnvResource `json:"list"`
	Length int                  `json:"length"`
}

// TenantEnvResource abandoned
type TenantEnvResource struct {
	//without plugin
	AllocatedCPU int `json:"alloc_cpu"`
	//without plugin
	AllocatedMEM int `json:"alloc_memory"`
	//with plugin
	UsedCPU int `json:"used_cpu"`
	//with plugin
	UsedMEM  int     `json:"used_memory"`
	UsedDisk float64 `json:"used_disk"`
	Name     string  `json:"name"`
	UUID     string  `json:"uuid"`
}

func (list TenantEnvResList) Len() int {
	return len(list)
}

func (list TenantEnvResList) Less(i, j int) bool {
	if list[i].UsedMEM > list[j].UsedMEM {
		return true
	} else if list[i].UsedMEM < list[j].UsedMEM {
		return false
	} else {
		return list[i].UsedCPU > list[j].UsedCPU
	}
}

func (list TenantEnvResList) Swap(i, j int) {
	temp := list[i]
	list[i] = list[j]
	list[j] = temp
}

// TenantEnvAndResource tenant env and resource strcut
type TenantEnvAndResource struct {
	dbmodel.TenantEnvs
	CPURequest            int64 `json:"cpu_request"`
	CPULimit              int64 `json:"cpu_limit"`
	MemoryRequest         int64 `json:"memory_request"`
	MemoryLimit           int64 `json:"memory_limit"`
	RunningAppNum         int64 `json:"running_app_num"`
	RunningAppInternalNum int64 `json:"running_app_internal_num"`
	RunningAppThirdNum    int64 `json:"running_app_third_num"`
}

// TenantEnvList TenantEnv list struct
type TenantEnvList []*TenantEnvAndResource

// Add add
func (list *TenantEnvList) Add(tr *TenantEnvAndResource) {
	*list = append(*list, tr)
}
func (list TenantEnvList) Len() int {
	return len(list)
}

func (list TenantEnvList) Less(i, j int) bool {
	// Highest priority
	if list[i].MemoryRequest > list[j].MemoryRequest {
		return true
	}
	if list[i].MemoryRequest == list[j].MemoryRequest {
		if list[i].CPURequest > list[j].CPURequest {
			return true
		}
		if list[i].CPURequest == list[j].CPURequest {
			if list[i].RunningAppNum > list[j].RunningAppNum {
				return true
			}
			if list[i].RunningAppNum == list[j].RunningAppNum {
				// Minimum priority
				if list[i].TenantEnvs.LimitMemory > list[j].TenantEnvs.LimitMemory {
					return true
				}
			}
		}
	}
	return false
}

func (list TenantEnvList) Swap(i, j int) {
	list[i], list[j] = list[j], list[i]
}

// Paging paging
func (list TenantEnvList) Paging(page, pageSize int) map[string]interface{} {
	startIndex := (page - 1) * pageSize
	endIndex := page * pageSize
	var relist TenantEnvList
	if startIndex < list.Len() && endIndex < list.Len() {
		relist = list[startIndex:endIndex]
	}
	if startIndex < list.Len() && endIndex >= list.Len() {
		relist = list[startIndex:]
	}
	return map[string]interface{}{
		"list":     relist,
		"page":     page,
		"pageSize": pageSize,
		"total":    list.Len(),
	}
}
