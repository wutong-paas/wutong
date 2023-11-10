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

import "time"

type VMProfile struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	Desc        string `json:"desc"`
	// Size   VMSize            `json:"size"`
	OSSourceFrom  OSSourceFrom `json:"osSourceFrom"`
	OSSourceURL   string       `json:"osSourceURL"`
	OSDiskSize    int64        `json:"osDiskSize"`
	RequestCPU    int64        `json:"requestCPU"`
	RequestMemory int64        `json:"requestMemory"`
	// Labels        map[string]string `json:"labels"`
	Status         string    `json:"status"`
	IP             string    `json:"ip"`
	OSInfo         VMOSInfo  `json:"osInfo"`
	ScheduleNode   string    `json:"scheduleNode"`
	CreatedAt      time.Time `json:"createdAt"`
	LastModifiedAt time.Time `json:"lastModifiedAt"`
	CreatedBy      string    `json:"createdBy"`
	LastModifiedBy string    `json:"lastModifiedBy"`
}

type VMOSInfo struct {
	Name          string `json:"name"`
	Version       string `json:"version"`
	Arch          string `json:"arch"`
	KernelRelease string `json:"kernelRelease"`
	KernelVersion string `json:"kernelVersion"`
}

type VMPortProtocol string

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

// type VMSize string

// const (
// 	VMSizeSmall  VMSize = "small"
// 	VMSizeMedium VMSize = "medium"
// 	VMSizeLarge  VMSize = "large"
// )

type OSSourceFrom string

const (
	OSSourceFromHTTP     OSSourceFrom = "http"
	OSSourceFromRegistry OSSourceFrom = "registry"
)

// CreateVMRequest
type CreateVMRequest struct {
	Name          string       `json:"name" validate:"name|required"`
	DisplayName   string       `json:"displayName" validate:"displayName|required"`
	Desc          string       `json:"desc"`
	OSSourceFrom  OSSourceFrom `json:"osSourceFrom" validate:"osSourceFrom|required"`
	OSSourceURL   string       `json:"osSourceURL" validate:"osSourceURL|required"`
	OSDiskSize    int64        `json:"osDiskSize" validate:"osDiskSize|required"`
	RequestCPU    int64        `json:"requestCPU" validate:"requestCPU|required"`
	RequestMemory int64        `json:"requestMemory" validate:"requestMemory|required"`
	// Size        VMSize            `json:"size"`
	User     string `json:"user"`
	Password string `json:"password"`
	// Labels   map[string]string `json:"labels"`
	Operator string `json:"operator"`
}

// CreateVMResponse
type CreateVMResponse struct {
	VMProfile
}

// GetVMResponse
type GetVMResponse struct {
	VMProfile
}

// UpdateVMRequest
type UpdateVMRequest struct {
	DisplayName   string `json:"displayName"`
	Desc          string `json:"desc"`
	RequestCPU    int64  `json:"requestCPU"`
	RequestMemory int64  `json:"requestMemory"`
	// Size        VMSize            `json:"size"`
	// OSSourceFrom OSSourceFrom      `json:"osSourceFrom"`
	// OSSourceURL  string            `json:"osSourceURL"`
	// Labels   map[string]string `json:"labels"`
	Operator string `json:"operator"`
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
	// VMPort    int            `json:"vmPort" validate:"vmPort|required"`
	// Protocol      VMPortProtocol `json:"protocol" validate:"protocol|required"`
	VMPortGateway `json:",inline"`
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
	VMs   []VMProfile `json:"vms"`
	Total int         `json:"total"`
}
