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
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/wutong-paas/wutong/db/model"
	"github.com/wutong-paas/wutong/util"
	v1 "github.com/wutong-paas/wutong/worker/appm/types/v1"
	"github.com/wutong-paas/wutong/worker/server/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

// AppRuntimeSyncClient grpc client
type AppRuntimeSyncClient struct {
	target     string
	GrpcClient pb.AppRuntimeSyncClient
	conn       *grpc.ClientConn
	ctx        context.Context
}

func initGrpcConn(target string) (*grpc.ClientConn, error) {
	conn, err := grpc.NewClient(target, grpc.WithTransportCredentials(
		insecure.NewCredentials(),
	), grpc.WithDefaultCallOptions(grpc.WaitForReady(true)))
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func (a *AppRuntimeSyncClient) TryResetGrpcClient(err error) {
	if status, ok := status.FromError(err); ok {
		if status.Code() == codes.DeadlineExceeded || status.Code() == codes.Unavailable {
			logrus.Infof("reset grpc client connection on error: %v", err)
			conn, err := initGrpcConn(a.target)
			if err != nil {
				logrus.Errorf("reconnect grpc client failure %s", err)
				return
			}
			a.conn = conn
			a.GrpcClient = pb.NewAppRuntimeSyncClient(conn)
		}
	}
}

// NewClient new client (tx must be cancel where client not used)
func NewClient(ctx context.Context, grpcServer string) (c *AppRuntimeSyncClient, err error) {
	c = &AppRuntimeSyncClient{
		target: grpcServer,
		ctx:    ctx,
	}
	logrus.Infof("discover app runtime sync server address %s", grpcServer)

	c.conn, err = initGrpcConn(grpcServer)

	if err != nil {
		return nil, err
	}
	c.GrpcClient = pb.NewAppRuntimeSyncClient(c.conn)

	return c, nil
}

// when watch occurred error,will exec this method
func (a *AppRuntimeSyncClient) Error(err error) {
	logrus.Errorf("discover app runtime sync server address occurred err:%s", err.Error())
}

// GetStatus get status
func (a *AppRuntimeSyncClient) GetStatus(serviceID string) string {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	status, err := a.GrpcClient.GetAppStatusDeprecated(ctx, &pb.ServicesRequest{
		ServiceIds: serviceID,
	})
	if err != nil {
		a.TryResetGrpcClient(err)
		return v1.UNKNOW
	}
	return status.Status[serviceID]
}

// GetStatuss get multiple app status
func (a *AppRuntimeSyncClient) GetStatuss(serviceIDs string) map[string]string {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()
	status, err := a.GrpcClient.GetAppStatusDeprecated(ctx, &pb.ServicesRequest{
		ServiceIds: serviceIDs,
	})
	if err != nil {
		a.TryResetGrpcClient(err)
		logrus.Errorf("get service status failure %s", err.Error())
		re := make(map[string]string, len(serviceIDs))
		for _, id := range strings.Split(serviceIDs, ",") {
			re[id] = v1.UNKNOW
		}
		return re
	}
	return status.Status
}

// GetAllStatus get all status
func (a *AppRuntimeSyncClient) GetAllStatus() map[string]string {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()
	status, err := a.GrpcClient.GetAppStatusDeprecated(ctx, &pb.ServicesRequest{
		ServiceIds: "",
	})
	if err != nil {
		a.TryResetGrpcClient(err)
		return nil
	}
	return status.Status
}

// GetNeedBillingStatus get need billing status
func (a *AppRuntimeSyncClient) GetNeedBillingStatus() (map[string]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()
	re, err := a.GrpcClient.GetAppStatusDeprecated(ctx, &pb.ServicesRequest{})
	if err != nil {
		a.TryResetGrpcClient(err)
		return nil, err
	}
	var res = make(map[string]string)
	for k, v := range re.Status {
		if !a.IsClosedStatus(v) {
			res[k] = v
		}
	}
	return res, nil
}

// GetServiceDeployInfo get service deploy info
func (a *AppRuntimeSyncClient) GetServiceDeployInfo(serviceID string) (*pb.DeployInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	re, err := a.GrpcClient.GetDeployInfo(ctx, &pb.ServiceRequest{
		ServiceId: serviceID,
	})
	if err != nil {
		a.TryResetGrpcClient(err)
		return nil, err
	}
	return re, nil
}

// IsClosedStatus  check status
func (a *AppRuntimeSyncClient) IsClosedStatus(curStatus string) bool {
	return curStatus == "" || curStatus == v1.BUILDEFAILURE || curStatus == v1.CLOSED || curStatus == v1.UNDEPLOY || curStatus == v1.BUILDING || curStatus == v1.UNKNOW
}

// GetTenantEnvResource get tenant env resource
func (a *AppRuntimeSyncClient) GetTenantEnvResource(tenantEnvID string) (*pb.TenantEnvResource, error) {
	if logrus.IsLevelEnabled(logrus.DebugLevel) {
		defer util.Elapsed("[AppRuntimeSyncClient] get tenant env resource")()
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	res, err := a.GrpcClient.GetTenantEnvResource(ctx, &pb.TenantEnvRequest{TenantEnvId: tenantEnvID})
	if err != nil {
		a.TryResetGrpcClient(err)
		return nil, err
	}
	return res, nil
}

// GetAllTenantEnvResource get all tenant env resource
func (a *AppRuntimeSyncClient) GetAllTenantEnvResource() (*pb.TenantEnvResourceList, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	return a.GrpcClient.GetTenantEnvResources(ctx, &pb.Empty{})
}

// ListThirdPartyEndpoints -
func (a *AppRuntimeSyncClient) ListThirdPartyEndpoints(sid string) (*pb.ThirdPartyEndpoints, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	resp, err := a.GrpcClient.ListThirdPartyEndpoints(ctx, &pb.ServiceRequest{
		ServiceId: sid,
	})
	if err != nil {
		a.TryResetGrpcClient(err)
		return nil, err
	}
	return resp, nil
}

// AddThirdPartyEndpoint -
func (a *AppRuntimeSyncClient) AddThirdPartyEndpoint(req *model.Endpoint) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	if _, err := a.GrpcClient.AddThirdPartyEndpoint(ctx, &pb.AddThirdPartyEndpointsReq{
		Uuid: req.UUID,
		Sid:  req.ServiceID,
		Ip:   req.IP,
		Port: int32(req.Port),
	}); err != nil {
		a.TryResetGrpcClient(err)
	}
}

// UpdThirdPartyEndpoint -
func (a *AppRuntimeSyncClient) UpdThirdPartyEndpoint(req *model.Endpoint) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	if _, err := a.GrpcClient.UpdThirdPartyEndpoint(ctx, &pb.UpdThirdPartyEndpointsReq{
		Uuid: req.UUID,
		Sid:  req.ServiceID,
		Ip:   req.IP,
		Port: int32(req.Port),
	}); err != nil {
		a.TryResetGrpcClient(err)
	}
}

// DelThirdPartyEndpoint -
func (a *AppRuntimeSyncClient) DelThirdPartyEndpoint(req *model.Endpoint) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	if _, err := a.GrpcClient.DelThirdPartyEndpoint(ctx, &pb.DelThirdPartyEndpointsReq{
		Uuid: req.UUID,
		Sid:  req.ServiceID,
		Ip:   req.IP,
		Port: int32(req.Port),
	}); err != nil {
		a.TryResetGrpcClient(err)
	}
}

// GetStorageClasses client GetStorageClasses
func (a *AppRuntimeSyncClient) GetStorageClasses() (storageclasses *pb.StorageClasses, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	res, err := a.GrpcClient.GetStorageClasses(ctx, &pb.Empty{})
	if err != nil {
		a.TryResetGrpcClient(err)
		return nil, err
	}
	return res, nil
}

// GetAppVolumeStatus get app volume status
func (a *AppRuntimeSyncClient) GetAppVolumeStatus(serviceID string) (*pb.ServiceVolumeStatusMessage, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	res, err := a.GrpcClient.GetAppVolumeStatus(ctx, &pb.ServiceRequest{ServiceId: serviceID})
	if err != nil {
		a.TryResetGrpcClient(err)
		return nil, err
	}
	return res, nil
}

// GetAppResources -
func (a *AppRuntimeSyncClient) GetAppResources(appID string) (*pb.AppStatus, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	res, err := a.GrpcClient.GetAppStatus(ctx, &pb.AppStatusReq{AppId: appID})
	if err != nil {
		a.TryResetGrpcClient(err)
		return nil, err
	}
	return res, nil
}
