package containerutil

import (
	"fmt"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/sirupsen/logrus"
)

const (
	ContainerRuntimeDocker     = "docker"
	ContainerRuntimeContainerd = "containerd"
	DefaultDockerSock          = "/var/run/dockershim.sock"
	DefaultContainerdSock      = "/run/containerd/containerd.sock"
)

type ContainerImageClient interface {
	ImageSave(destination string, images []string) error
	ImageLoad(tarFile string) error
	ImagePull(image string, username, password string, timeout int) (*ocispec.ImageConfig, error)
	ImagePush(image, user, pass string, timeout int) error
	ImageTag(source, target string, timeout int) error
}

var containerImageClientInstance ContainerImageClient

func NewClient(containerRuntime, endpoint string) (ContainerImageClient, error) {
	// lazy init
	if containerImageClientInstance != nil {
		return containerImageClientInstance, nil
	}

	logrus.Infof("create container client runtime %s endpoint %s", containerRuntime, endpoint)
	var err error
	switch containerRuntime {
	case ContainerRuntimeDocker:
		containerImageClientInstance, err = newDockerClient()
	case ContainerRuntimeContainerd:
		containerImageClientInstance, err = newContainerdClient(endpoint)
	default:
		err = fmt.Errorf("unknown container runtime %s", containerRuntime)
	}

	return containerImageClientInstance, err
}
