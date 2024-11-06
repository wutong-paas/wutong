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

package client

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/wutong-paas/wutong/util"
	"github.com/wutong-paas/wutong/worker/server/pb"
)

// GetServicePods get service pods list
func (a *AppRuntimeSyncClient) GetServicePods(serviceID string) (*pb.ServiceAppPodList, error) {
	ctx, cancel := context.WithTimeout(a.ctx, time.Second*5)
	defer cancel()
	res, err := a.GrpcClient.GetAppPods(ctx, &pb.ServiceRequest{ServiceId: serviceID})
	if err != nil {
		a.TryResetGrpcClient(err)
		return nil, err
	}
	return res, nil
}

// GetMultiServicePods get multi service pods list
func (a *AppRuntimeSyncClient) GetMultiServicePods(serviceIDs []string) (*pb.MultiServiceAppPodList, error) {
	if logrus.IsLevelEnabled(logrus.DebugLevel) {
		defer util.Elapsed(fmt.Sprintf("[AppRuntimeSyncClient] [GetMultiServicePods] component nums: %d", len(serviceIDs)))()
	}

	ctx, cancel := context.WithTimeout(a.ctx, time.Second*5)
	defer cancel()
	res, err := a.GrpcClient.GetMultiAppPods(ctx, &pb.ServicesRequest{ServiceIds: strings.Join(serviceIDs, ",")})
	if err != nil {
		a.TryResetGrpcClient(err)
		return nil, err
	}
	return res, nil
}

// GetComponentPodNums -
func (a *AppRuntimeSyncClient) GetComponentPodNums(ctx context.Context, componentIDs []string) (map[string]int32, error) {
	if logrus.IsLevelEnabled(logrus.DebugLevel) {
		defer util.Elapsed(fmt.Sprintf("[AppRuntimeSyncClient] get component pod nums: %d", len(componentIDs)))()
	}

	res, err := a.GrpcClient.GetComponentPodNums(ctx, &pb.ServicesRequest{ServiceIds: strings.Join(componentIDs, ",")})
	if err != nil {
		a.TryResetGrpcClient(err)
		return nil, errors.Wrap(err, "get component pod nums")
	}

	return res.PodNums, nil
}

// GetPodDetail -
func (a *AppRuntimeSyncClient) GetPodDetail(sid, name string) (*pb.PodDetail, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	res, err := a.GrpcClient.GetPodDetail(ctx, &pb.GetPodDetailReq{
		Sid:     sid,
		PodName: name,
	})
	if err != nil {
		a.TryResetGrpcClient(err)
		return nil, err
	}
	return res, nil
}
