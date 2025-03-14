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

import "net/http"

type VMProfile struct {
	Name               string        `json:"name"`
	DisplayName        string        `json:"displayName"`
	Desc               string        `json:"desc"`
	OSSourceFrom       OSSourceFrom  `json:"osSourceFrom"`
	OSSourceURL        string        `json:"osSourceURL"`
	OSDiskSize         int64         `json:"osDiskSize"`
	RequestCPU         int64         `json:"requestCPU"`    // m
	RequestMemory      int64         `json:"requestMemory"` // GiB
	Namespace          string        `json:"namespace"`
	DefaultLoginUser   string        `json:"defaultLoginUser"`
	Status             string        `json:"status"`
	StatusMessage      string        `json:"statusMessage"`
	Conditions         []VMCondition `json:"conditions"`
	IP                 string        `json:"ip"`
	InternalDomainName string        `json:"internalDomainName"`
	OSInfo             VMOSInfo      `json:"osInfo"`
	ScheduleNode       string        `json:"scheduleNode"`
	CreatedAt          string        `json:"createdAt"`
	LastModifiedAt     string        `json:"lastModifiedAt"`
	CreatedBy          string        `json:"createdBy"`
	LastModifiedBy     string        `json:"lastModifiedBy"`
	NodeSelectorLabels []string      `json:"nodeSelectorLabels"`
	ContainsBootDisk   bool          `json:"containsBootDisk"`
}

type VMCondition struct {
	Type           string `json:"type"`
	Reason         string `json:"reason"`
	Message        string `json:"message"`
	Status         bool   `json:"status"`
	LastReportedAt string `json:"lastReportedAt"`
}

type VMOSInfo struct {
	Name          string `json:"name"`
	Version       string `json:"version"`
	Arch          string `json:"arch"`
	KernelRelease string `json:"kernelRelease"`
	KernelVersion string `json:"kernelVersion"`
}

type VMPortProtocol = string

const (
	VMPortProtocolHTTP  VMPortProtocol = "http"
	VMPortProtocolTCP   VMPortProtocol = "tcp"
	VMPortProtocolUDP   VMPortProtocol = "udp"
	VMPortProtocolSCTP  VMPortProtocol = "sctp"
	VMPortProtocolSSH   VMPortProtocol = "ssh"
	VMPortProtocolRDP   VMPortProtocol = "rdp"
	VMPortProtocolGrpc  VMPortProtocol = "grpc"
	VMPortProtocolMysql VMPortProtocol = "mysql"
)

var VMPortProtocols = []VMPortProtocol{
	VMPortProtocolHTTP,
	VMPortProtocolTCP,
	VMPortProtocolUDP,
	VMPortProtocolSSH,
	VMPortProtocolRDP,
	VMPortProtocolGrpc,
	VMPortProtocolMysql,
}

type VMPort struct {
	VMPort         int             `json:"vmPort"`
	Protocol       VMPortProtocol  `json:"protocol"`
	InnerService   string          `json:"innerService"`
	GatewayEnabled bool            `json:"gatewayEnabled"`
	Gateways       []VMPortGateway `json:"gateways"`
}

type VMPortGateway struct {
	GatewayID   string `json:"gatewayID"`
	GatewayIP   string `json:"gatewayIP"`
	GatewayPort int    `json:"gatewayPort"`
	GatewayHost string `json:"gatewayHost"`
	GatewayPath string `json:"gatewayPath"`
}

type OSSourceFrom = string

const (
	OSSourceFromHTTP     OSSourceFrom = "http"
	OSSourceFromRegistry OSSourceFrom = "registry"
)

type VMInstallType = string

const (
	VMInstallTypeISO VMInstallType = "iso"
)

// CreateVMRequest
type CreateVMRequest struct {
	Name          string       `json:"name" validate:"name|required"`
	DisplayName   string       `json:"displayName" validate:"displayName|required"`
	Desc          string       `json:"desc"`
	OSName        string       `json:"osName"`
	OSVersion     string       `json:"osVersion"`
	OSSourceFrom  OSSourceFrom `json:"osSourceFrom" validate:"osSourceFrom|required"`
	OSSourceURL   string       `json:"osSourceURL" validate:"osSourceURL|required"`
	OSDiskSize    uint32       `json:"osDiskSize" validate:"osDiskSize|required"`       // GiB
	RequestCPU    uint32       `json:"requestCPU" validate:"requestCPU|required"`       // m
	RequestMemory uint32       `json:"requestMemory" validate:"requestMemory|required"` // GiB
	Running       bool         `json:"running"`
	// Size        VMSize            `json:"size"`
	// HostNodeName string `json:"hostNodeName"`
	User     string `json:"user"`
	Password string `json:"password"`
	// Labels   map[string]string `json:"labels"`
	Operator           string   `json:"operator"`
	NodeSelectorLabels []string `json:"nodeSelectorLabels"`
	LoadVirtioDriver   bool     `json:"loadVirtioDriver"`
}

// CreateVMResponse
type CreateVMResponse struct {
	VMProfile
}

// GetVMResponse
type GetVMResponse struct {
	VMProfile
}

type GetVMConditionsResponse struct {
	Conditions []VMCondition `json:"conditions"`
}

// UpdateVMRequest
type UpdateVMRequest struct {
	DisplayName        string   `json:"displayName"`
	Desc               string   `json:"desc"`
	RequestCPU         uint32   `json:"requestCPU"`    // m
	RequestMemory      uint32   `json:"requestMemory"` // GiB
	DefaultLoginUser   string   `json:"defaultLoginUser"`
	Operator           string   `json:"operator"`
	NodeSelectorLabels []string `json:"nodeSelectorLabels"`
}

type AddVMPortRequest struct {
	VMPort   int            `json:"vmPort" validate:"vmPort|required"`
	Protocol VMPortProtocol `json:"protocol" validate:"protocol|required"`
}

type AddVMPortResponse struct {
	VMPort   int             `json:"vmPort"`
	Protocol VMPortProtocol  `json:"protocol"`
	Gateways []VMPortGateway `json:"gateways"`
}

type GetVMPortsResponse struct {
	Total int      `json:"total"`
	Ports []VMPort `json:"ports"`
}

type CreateVMPortGatewayRequest struct {
	VMPort        int            `json:"vmPort" validate:"vmPort|required"`
	Protocol      VMPortProtocol `json:"protocol" validate:"protocol|required"`
	VMPortGateway `json:",inline"`
}

type UpdateVMPortGatewayRequest struct {
	VMPortGateway `json:",inline"`
}

type EnableVMPortRequest struct {
	VMPort   int            `json:"vmPort" validate:"vmPort|required"`
	Protocol VMPortProtocol `json:"protocol" validate:"protocol|required"`
}

type DisableVMPortRequest struct {
	VMPort   int            `json:"vmPort" validate:"vmPort|required"`
	Protocol VMPortProtocol `json:"protocol" validate:"protocol|required"`
}

// UpdateVMResponse
type UpdateVMResponse struct {
	VMProfile
}

type StartVMResponse struct {
	VMProfile
}

type StopVMResponse struct {
	VMProfile
}

type RestartVMResponse struct {
	VMProfile
}

type DeleteVMPortRequest struct {
	VMPort   int            `json:"vmPort" validate:"vmPort|required"`
	Protocol VMPortProtocol `json:"protocol" validate:"protocol|required"`
}

type ListVMsResponse struct {
	VMs   []*VMProfile `json:"vms"`
	Total int          `json:"total"`
}

type ListVMVolumesResponse struct {
	VMVolumes []VMVolume `json:"volumes"`
	Total     int        `json:"total"`
}

type VMVolume struct {
	VolumeName   string `json:"volumeName"`
	StorageClass string `json:"storageClass"`
	VolumeSize   int64  `json:"volumeSize"`
	Status       string `json:"status"`
}

type AddVMVolumeRequest struct {
	VolumeName   string `json:"volumeName" validate:"volumeName|required"`
	StorageClass string `json:"storageClass" validate:"storageClass|required"`
	VolumeSize   int64  `json:"volumeSize" validate:"size|required"`
}

type ChangeServiceAppRequest struct {
	NewAppID string `json:"newAppId" validate:"newAppId|required"`
}

type CloneVMRequest struct {
	CloneName string `json:"cloneName" validate:"cloneName|required"`
	Operator  string `json:"operator"`
}

type CreateVMSnapshotRequest struct {
	Description string `json:"description"`
	Operator    string `json:"operator"`
}

type VMSnapshot struct {
	SnapshotName string `json:"snapshotName"`
	Description  string `json:"description"`
	Status       string `json:"status"`
	CreateTime   string `json:"createTime"`
	Creator      string `json:"creator"`
}

type ListVMSnapshotsResponse struct {
	Snapshots []VMSnapshot `json:"snapshots"`
}

type CreateVMRestoreRequest struct {
	SnapshotName string `json:"snapshotName" validate:"snapshotName|required"`
	Operator     string `json:"operator"`
}

type VMRestore struct {
	RestoreName  string `json:"restoreName"`
	SnapshotName string `json:"snapshotName"`
	Description  string `json:"description"`
	Status       string `json:"status"`
	CreateTime   string `json:"createTime"`
	Creator      string `json:"creator"`
}

type ListVMRestoresResponse struct {
	Restores []VMRestore `json:"restores"`
}

type ExportVMResponse struct{}

type VMExportFormat struct {
	DisplayName string `json:"displayName"`
	Format      string `json:"format"`
}

type VMExportItem struct {
	ExportID string `json:"exportId"`
	Formats  []VMExportFormat
}

type GetVMExportStatusResponse struct {
	ExportItems []VMExportItem `json:"exportItems"`
	Status      string         `json:"status"`
}

type DownloadVMExportRequest struct {
	ExportID       string
	Format         string // 下载虚拟机导出文件的格式
	ResponseWriter http.ResponseWriter
}
