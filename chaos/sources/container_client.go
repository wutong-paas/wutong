package sources

import (
	"context"
	"fmt"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	dockercli "github.com/docker/docker/client"
	"github.com/sirupsen/logrus"
	"github.com/wutong-paas/wutong/util/containerutil"
	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"
)

const (
	// CONTAINER_ACTION_START is start container event action
	CONTAINER_ACTION_START = "start"

	// CONTAINER_ACTION_STOP is stop container event action
	CONTAINER_ACTION_STOP = "stop"

	// CONTAINER_ACTION_CREATE is create container event action
	CONTAINER_ACTION_CREATE = "create"

	// CONTAINER_ACTION_DESTROY is destroy container event action
	CONTAINER_ACTION_DESTROY = "destroy"

	// CONTAINER_ACTION_DIE is die container event action
	CONTAINER_ACTION_DIE = "die"
)

type ContainerDesc struct {
	ContainerRuntime string
	// Info is extra information of the Container. The key could be arbitrary string, and
	// value should be in json format. The information could include anything useful for
	// debug, e.g. pid for linux container based container runtime.
	// It should only be returned non-empty when Verbose is true.
	Info map[string]string
	*runtimeapi.ContainerStatus
	// Docker container json
	*types.ContainerJSON
}

func (c *ContainerDesc) GetLogPath() string {
	if c.ContainerRuntime == containerutil.ContainerRuntimeDocker {
		return c.ContainerJSON.LogPath
	}
	return c.ContainerStatus.GetLogPath()
}

func (c *ContainerDesc) GetId() string {
	if c.ContainerRuntime == containerutil.ContainerRuntimeDocker {
		return c.ContainerJSON.ID
	}
	return c.ContainerStatus.GetId()
}

// ContainerImageCli container image client
type ContainerImageCli interface {
	ListContainers() ([]*runtimeapi.Container, error)
	InspectContainer(containerID string) (*ContainerDesc, error)
	WatchContainers(ctx context.Context, cchan chan ContainerEvent) error
	GetRuntimeClient() (*runtimeapi.RuntimeServiceClient, error)
	GetDockerClient() (*dockercli.Client, error)
}

// ClientFactory client factory
type ClientFactory interface {
	NewClient(endpoint string, timeout time.Duration) (ContainerImageCli, error)
}

// NewContainerImageClient new container image client
func NewContainerImageClient(containerRuntime, endpoint string, timeout time.Duration) (c ContainerImageCli, err error) {
	logrus.Infof("create container client runtime %s endpoint %s", containerRuntime, endpoint)
	switch containerRuntime {
	case containerutil.ContainerRuntimeDocker:
		factory := &dockerClientFactory{}
		c, err = factory.NewClient(
			endpoint, timeout,
		)
	case containerutil.ContainerRuntimeContainerd:
		factory := &containerdClientFactory{}
		c, err = factory.NewClient(
			endpoint, timeout,
		)
		return
	default:
		err = fmt.Errorf("unknown runtime %s", containerRuntime)
		return
	}
	return
}

// ContainerEvent container event
type ContainerEvent struct {
	Action    events.Action
	Container *ContainerDesc
}

func CacheContainer(cchan chan ContainerEvent, cs ...ContainerEvent) {
	for _, container := range cs {
		logrus.Debugf("found a container %s %s", container.Container.GetMetadata().GetName(), container.Action)
		cchan <- container
	}
}
