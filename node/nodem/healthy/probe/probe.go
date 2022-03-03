// WUTONG, Application Management Platform
// Copyright (C) 2014-2017 Wutong Co., Ltd.

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

package probe

import (
	"context"
	"fmt"
	"strings"

	"github.com/wutong-paas/wutong/node/nodem/client"
	"github.com/wutong-paas/wutong/node/nodem/service"
)

//Probe probe
type Probe interface {
	Check()
	Stop()
}

//CreateProbe create probe
func CreateProbe(ctx context.Context, hostNode *client.HostNode, statusChan chan *service.HealthStatus, v *service.Service) (Probe, error) {
	ctx, cancel := context.WithCancel(ctx)
	model := strings.ToLower(strings.TrimSpace(v.ServiceHealth.Model))
	switch model {
	case "http":
		return &HttpProbe{
			Name:         v.ServiceHealth.Name,
			Address:      v.ServiceHealth.Address,
			Ctx:          ctx,
			Cancel:       cancel,
			ResultsChan:  statusChan,
			TimeInterval: v.ServiceHealth.TimeInterval,
			HostNode:     hostNode,
			MaxErrorsNum: v.ServiceHealth.MaxErrorsNum,
		}, nil
	case "tcp":
		return &TcpProbe{
			Name:         v.ServiceHealth.Name,
			Address:      v.ServiceHealth.Address,
			Ctx:          ctx,
			Cancel:       cancel,
			ResultsChan:  statusChan,
			TimeInterval: v.ServiceHealth.TimeInterval,
			HostNode:     hostNode,
			MaxErrorsNum: v.ServiceHealth.MaxErrorsNum,
		}, nil
	case "cmd":
		return &ShellProbe{
			Name:         v.ServiceHealth.Name,
			Address:      v.ServiceHealth.Address,
			Ctx:          ctx,
			Cancel:       cancel,
			ResultsChan:  statusChan,
			TimeInterval: v.ServiceHealth.TimeInterval,
			HostNode:     hostNode,
			MaxErrorsNum: v.ServiceHealth.MaxErrorsNum,
		}, nil
	default:
		cancel()
		return nil, fmt.Errorf("service %s probe mode %s not support ", v.Name, model)
	}
}
