package sources

import (
	"context"
	"errors"
	"os"
	"strings"
	"time"

	"github.com/containerd/containerd"
	containerdEventstypes "github.com/containerd/containerd/api/events"
	"github.com/containerd/containerd/events"
	"github.com/containerd/typeurl/v2"
	dockercli "github.com/docker/docker/client"
	"github.com/sirupsen/logrus"
	"github.com/wutong-paas/wutong/util/containerutil"
	"github.com/wutong-paas/wutong/util/criutil"
	"google.golang.org/grpc"
	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"
)

type containerdClientFactory struct{}

func (f containerdClientFactory) NewClient(endpoint string, timeout time.Duration) (ContainerImageCli, error) {
	var (
		containerdCli *containerd.Client
		runtimeClient runtimeapi.RuntimeServiceClient
		grpcConn      *grpc.ClientConn
		err           error
	)
	runtimeClient, grpcConn, err = criutil.GetRuntimeClient(context.Background(), endpoint, time.Second*3)
	if err != nil {
		return nil, err
	}
	if os.Getenv("CONTAINERD_SOCK") != "" {
		endpoint = os.Getenv("CONTAINERD_SOCK")
	}
	containerdCli, err = containerd.New(endpoint, containerd.WithTimeout(timeout))
	if err != nil {
		return nil, err
	}
	return &containerdClientImpl{
		client:        containerdCli,
		conn:          grpcConn,
		runtimeClient: runtimeClient,
	}, nil
}

type containerdClientImpl struct {
	client        *containerd.Client
	conn          *grpc.ClientConn
	runtimeClient runtimeapi.RuntimeServiceClient
}

func (c *containerdClientImpl) ListContainers() ([]*runtimeapi.Container, error) {
	containers, err := c.runtimeClient.ListContainers(context.Background(), &runtimeapi.ListContainersRequest{})
	if err != nil {
		return nil, err
	}
	return containers.GetContainers(), nil
}

func (c *containerdClientImpl) InspectContainer(containerID string) (*ContainerDesc, error) {
	containerStatus, err := c.runtimeClient.ContainerStatus(context.Background(), &runtimeapi.ContainerStatusRequest{
		ContainerId: containerID,
		Verbose:     true,
	})
	if err != nil {
		return nil, err
	}
	return &ContainerDesc{
		ContainerRuntime: containerutil.ContainerRuntimeContainerd,
		ContainerStatus:  containerStatus.GetStatus(),
		Info:             containerStatus.GetInfo(),
	}, nil
}

func (c *containerdClientImpl) WatchContainers(ctx context.Context, cchan chan ContainerEvent) error {
	eventsClient := c.client.EventService()
	eventsCh, errCh := eventsClient.Subscribe(ctx)
	var err error
	for {
		var e *events.Envelope
		select {
		case <-ctx.Done():
			return ctx.Err()
		case e = <-eventsCh:
		case err = <-errCh:
			return err
		}
		if e != nil {
			if e.Event != nil {
				ev, err := typeurl.UnmarshalAny(e.Event)
				if err != nil {
					logrus.Warn("cannot unmarshal an event from Any")
					continue
				}
				switch ev := ev.(type) {
				case *containerdEventstypes.TaskStart:
					// PATCH: if it's start event of pause container
					// we would skip it.
					// QUESTION: what if someone's container ID equals the other Sandbox ID?
					targetContainerID := ev.ContainerID
					resp, _ := c.runtimeClient.ListPodSandbox(context.Background(),
						&runtimeapi.ListPodSandboxRequest{
							Filter: &runtimeapi.PodSandboxFilter{
								Id: targetContainerID,
							},
						})
					if resp != nil && len(resp.Items) == 1 {
						// it's sandbox container! skip this one!
						logrus.Infof("skipped start event of container %s since it's sandbox container", targetContainerID)
						continue
					}
					container, err := c.InspectContainer(targetContainerID)
					if err != nil {
						if !strings.Contains(err.Error(), "No such container") {
							logrus.Errorf("get container detail info failure %s", err.Error())
						}
						break
					}
					CacheContainer(cchan, ContainerEvent{Action: CONTAINER_ACTION_START, Container: container})
				case containerdEventstypes.TaskExit, containerdEventstypes.TaskDelete:
					var targetContainerID string
					evVal, ok := ev.(*containerdEventstypes.TaskExit)
					if ok {
						targetContainerID = evVal.ContainerID
					} else {
						targetContainerID = ev.(*containerdEventstypes.TaskDelete).ContainerID
					}
					container, err := c.InspectContainer(targetContainerID)
					if err != nil {
						if !strings.Contains(err.Error(), "No such container") {
							logrus.Errorf("get container detail info failure %s", err.Error())
						}
						break
					}
					CacheContainer(cchan, ContainerEvent{Action: CONTAINER_ACTION_STOP, Container: container})
				}
			}
		}
	}
}

func (c *containerdClientImpl) GetRuntimeClient() (*runtimeapi.RuntimeServiceClient, error) {
	return &c.runtimeClient, nil
}

func (d *containerdClientImpl) GetDockerClient() (*dockercli.Client, error) {
	return nil, errors.New("not support")
}
