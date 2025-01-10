package containerutil

import (
	"context"
	"os"

	dockercli "github.com/docker/docker/client"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/wutong-paas/wutong-oam/pkg/util/docker"
)

type dockerClient struct {
	client *dockercli.Client
}

func newDockerClient() (ContainerImageClient, error) {
	if os.Getenv("DOCKER_API_VERSION") == "" {
		os.Setenv("DOCKER_API_VERSION", "1.40")
	}
	cli, err := dockercli.NewClientWithOpts(dockercli.FromEnv)
	if err != nil {
		return nil, err
	}
	return &dockerClient{client: cli}, nil
}

func (d *dockerClient) ImageSave(destination string, images []string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	return docker.MultiImageSave(ctx, d.client, destination, images...)
}

func (d *dockerClient) ImagePull(image string, username, password string, timeout int) (*ocispec.ImageConfig, error) {
	img, err := docker.ImagePull(d.client, image, username, password, timeout)
	if err != nil {
		return nil, err
	}
	exportPorts := make(map[string]struct{})
	for port := range img.Config.ExposedPorts {
		exportPorts[string(port)] = struct{}{}
	}
	return &ocispec.ImageConfig{
		User:         img.Config.User,
		ExposedPorts: exportPorts,
		Env:          img.Config.Env,
		Entrypoint:   img.Config.Entrypoint,
		Cmd:          img.Config.Cmd,
		Volumes:      img.Config.Volumes,
		WorkingDir:   img.Config.WorkingDir,
		Labels:       img.Config.Labels,
		StopSignal:   img.Config.StopSignal,
	}, nil
}

func (d *dockerClient) ImageLoad(tarFile string) error {
	return docker.ImageLoad(d.client, tarFile)
}

func (d *dockerClient) ImagePush(image, user, pass string, timeout int) error {
	return docker.ImagePush(d.client, image, user, pass, timeout)
}

func (d *dockerClient) ImageTag(source, target string, timeout int) error {
	return docker.ImageTag(d.client, source, target, timeout)
}
