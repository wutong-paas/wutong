package criutil

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"
	"k8s.io/cri-client/pkg/util"
)

func getConnection(endPoints []string, timeout time.Duration) (*grpc.ClientConn, error) {
	if len(endPoints) == 0 {
		return nil, fmt.Errorf("endpoint is not set")
	}
	endPointsLen := len(endPoints)
	var conn *grpc.ClientConn
	for indx, endPoint := range endPoints {
		logrus.Debugf("connect using endpoint '%s' with '%s' timeout", endPoint, timeout)
		addr, dialer, err := util.GetAddressAndDialer(endPoint)
		if err != nil {
			if indx == endPointsLen-1 {
				return nil, err
			}
			logrus.Error(err)
			continue
		}
		// conn, err = grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock(), grpc.WithTimeout(timeout), grpc.WithContextDialer(dialer))
		conn, err = grpc.NewClient(addr, grpc.WithTransportCredentials(
			insecure.NewCredentials(),
		), grpc.WithContextDialer(dialer))
		if err != nil {
			errMsg := errors.Wrapf(err, "connect endpoint '%s', make sure you are running as root and the endpoint has been started", endPoint)
			if indx == endPointsLen-1 {
				return nil, errMsg
			}
			logrus.Error(errMsg)
		} else {
			logrus.Debugf("connected successfully using endpoint: %s", endPoint)
			break
		}
	}
	return conn, nil
}

func GetRuntimeClient(ctx context.Context, endpoint string, timeout time.Duration) (runtimeapi.RuntimeServiceClient, *grpc.ClientConn, error) {
	// Set up a connection to the server.
	conn, err := getRuntimeClientConnection(ctx, endpoint, timeout)
	if err != nil {
		return nil, nil, errors.Wrap(err, "connect")
	}
	runtimeClient := runtimeapi.NewRuntimeServiceClient(conn)
	return runtimeClient, conn, nil
}

func getRuntimeClientConnection(_ context.Context, endpoint string, timeout time.Duration) (*grpc.ClientConn, error) {
	return getConnection([]string{endpoint}, timeout)
}

func GetImageClient(ctx context.Context, endpoint string, timeout time.Duration) (runtimeapi.ImageServiceClient, *grpc.ClientConn, error) {
	// Set up a connection to the server.
	conn, err := getRuntimeClientConnection(ctx, endpoint, timeout)
	if err != nil {
		return nil, nil, errors.Wrap(err, "connect")
	}
	runtimeClient := runtimeapi.NewImageServiceClient(conn)
	return runtimeClient, conn, nil
}
