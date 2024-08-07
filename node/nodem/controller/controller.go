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

package controller

import (
	"github.com/wutong-paas/wutong/node/nodem/service"
)

//Controller service daemon controller
type Controller interface {
	InitStart(services []*service.Service) error
	StartService(name string) error
	StopService(name string) error
	StartList(list []*service.Service) error
	StopList(list []*service.Service) error
	RestartService(s *service.Service) error
	WriteConfig(s *service.Service) (bool, error)
	RemoveConfig(name string) error
	EnableService(name string) error
	DisableService(name string) error
	CheckBeforeStart() bool
}
