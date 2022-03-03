package handler

import (
	"github.com/wutong-paas/wutong/worker/client"
	"github.com/wutong-paas/wutong/worker/server/pb"
)

// PodHandler defines handler methods about k8s pods.
type PodHandler interface {
	PodDetail(namespace, podName string) (*pb.PodDetail, error)
}

// NewPodHandler creates a new PodHandler.
func NewPodHandler(statusCli *client.AppRuntimeSyncClient) PodHandler {
	return &PodAction{
		statusCli: statusCli,
	}
}
