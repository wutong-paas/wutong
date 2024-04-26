package grpc

import (
	"context"

	"github.com/wutong-paas/wutong/config/configs"
	"github.com/wutong-paas/wutong/worker/client"
)

var defaultGrpcComponent *Component

// Component -
type Component struct {
	StatusClient *client.AppRuntimeSyncClient
}

// Start -
func (c *Component) Start(ctx context.Context, cfg *configs.Config) (err error) {
	c.StatusClient, err = client.NewClient(ctx, cfg.APIConfig.WtWorker)
	return err
}

// CloseHandle -
func (c *Component) CloseHandle() {
}

// Grpc -
func Grpc() *Component {
	defaultGrpcComponent = &Component{}
	return defaultGrpcComponent
}

// Default -
func Default() *Component {
	return defaultGrpcComponent
}
