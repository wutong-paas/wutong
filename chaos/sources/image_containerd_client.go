package sources

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"text/tabwriter"
	"time"

	"github.com/containerd/containerd"
	ctrcontent "github.com/containerd/containerd/cmd/ctr/commands/content"
	"github.com/containerd/containerd/content"
	"github.com/containerd/containerd/errdefs"
	"github.com/containerd/containerd/images"
	"github.com/containerd/containerd/images/archive"
	"github.com/containerd/containerd/namespaces"
	criconfig "github.com/containerd/containerd/pkg/cri/config"
	"github.com/containerd/containerd/pkg/progress"
	"github.com/containerd/containerd/remotes"
	"github.com/containerd/containerd/remotes/docker"
	"github.com/containerd/containerd/remotes/docker/config"
	"github.com/containerd/platforms"
	"github.com/distribution/reference"
	dockercli "github.com/docker/docker/client"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/pelletier/go-toml"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/wutong-paas/wutong/chaos"
	"github.com/wutong-paas/wutong/event"
	"github.com/wutong-paas/wutong/util/criutil"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"
)

type containerdImageCliFactory struct{}

func (f containerdImageCliFactory) NewClient(endpoint string, timeout time.Duration) (ImageClient, error) {
	var (
		containerdCli *containerd.Client
		imageClient   runtimeapi.ImageServiceClient
		grpcConn      *grpc.ClientConn
		err           error
	)
	imageClient, grpcConn, err = criutil.GetImageClient(context.Background(), endpoint, time.Second*3)
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
	return &containerdImageCliImpl{
		client:      containerdCli,
		conn:        grpcConn,
		imageClient: imageClient,
	}, nil
}

type containerdImageCliImpl struct {
	client      *containerd.Client
	conn        *grpc.ClientConn
	imageClient runtimeapi.ImageServiceClient
}

func (c *containerdImageCliImpl) CheckIfImageExists(imageName string) (imageRef string, exists bool, err error) {
	named, err := reference.ParseDockerRef(imageName)
	if err != nil {
		return "", false, fmt.Errorf("parse image %s: %v", imageName, err)
	}
	imageFullName := named.String()
	ctx := namespaces.WithNamespace(context.Background(), Namespace)
	containers, err := c.imageClient.ListImages(ctx, &runtimeapi.ListImagesRequest{})
	if err != nil {
		return imageFullName, false, err
	}
	if len(containers.GetImages()) > 0 {
		for _, image := range containers.GetImages() {
			for _, repoTag := range image.GetRepoTags() {
				if repoTag == imageFullName {
					return imageFullName, true, nil
				}
			}
		}
	}
	return imageFullName, false, nil
}

func (c *containerdImageCliImpl) GetContainerdClient() *containerd.Client {
	return c.client
}

func (c *containerdImageCliImpl) GetDockerClient() *dockercli.Client {
	return nil
}

func (c *containerdImageCliImpl) ImagePull(image string, username, password string, logger event.Logger, timeout int) (*ocispec.ImageConfig, error) {
	printLog(logger, "info", fmt.Sprintf("开始拉取镜像：%s", image), map[string]string{"step": "pullimage"})
	named, err := reference.ParseDockerRef(image)
	if err != nil {
		return nil, err
	}
	ref := named.String()
	ongoing := ctrcontent.NewJobs(ref)
	ctx := namespaces.WithNamespace(context.Background(), Namespace)
	pctx, stopProgress := context.WithCancel(ctx)
	progress := make(chan struct{})

	writer := logger.GetWriter("builder", "info")
	go func() {
		ctrcontent.ShowProgress(pctx, ongoing, c.client.ContentStore(), writer)
		close(progress)
	}()
	h := images.HandlerFunc(func(ctx context.Context, desc ocispec.Descriptor) ([]ocispec.Descriptor, error) {
		if desc.MediaType != images.MediaTypeDockerSchema1Manifest {
			ongoing.Add(desc)
		}
		return nil, nil
	})

	registry := reference.Domain(named)
	registryInfo := getRegistryInfo(registry)

	hostOpt := config.HostOptions{
		DefaultTLS: &tls.Config{
			InsecureSkipVerify: true,
		},
		DefaultScheme: registryInfo.schema,
	}

	if username == "" && password == "" {
		// try to set credentials from config.toml
		username = registryInfo.username
		password = registryInfo.password
	}
	hostOpt.Credentials = func(host string) (string, string, error) {
		return username, password, nil
	}

	hosts := getContianerdHosts()
	for _, host := range hosts {
		if host == registry {
			hostOpt.HostDir = func(s string) (string, error) {
				return "/etc/containerd/certs.d/" + registry, nil
			}
		}
	}

	Tracker := docker.NewInMemoryTracker()
	options := docker.ResolverOptions{
		Tracker: Tracker,
		Hosts:   config.ConfigureHosts(pctx, hostOpt),
	}

	platformMC := platforms.Ordered([]ocispec.Platform{platforms.DefaultSpec()}...)
	opts := []containerd.RemoteOpt{
		containerd.WithImageHandler(h),
		containerd.WithSchema1Conversion,
		containerd.WithPlatformMatcher(platformMC),
		containerd.WithResolver(docker.NewResolver(options)),
	}
	var img containerd.Image
	img, err = c.client.Pull(pctx, ref, opts...)
	stopProgress()
	if err != nil {
		return nil, err
	}
	<-progress
	printLog(logger, "info", fmt.Sprintf("成功拉取镜像：%s", ref), map[string]string{"step": "pullimage"})
	return getImageConfig(ctx, img)
}

func getContianerdHosts() []string {
	hosts := []string{}
	// 获取目录下的子目录
	files, err := os.ReadDir("/etc/containerd/certs.d/")
	if err != nil {
		return hosts
	}
	for _, file := range files {
		if file.IsDir() {
			hosts = append(hosts, file.Name())
		}
	}

	return hosts
}

const defaultHttpSchema = "https"

type registryInfo struct {
	schema   string
	username string
	password string
}

func getRegistryInfo(registry string) *registryInfo {
	var result = &registryInfo{
		schema: defaultHttpSchema,
	}

	// 获取目录下的子目录
	data, err := toml.LoadFile("/etc/containerd/config.toml")
	if err != nil {
		return result
	}
	var config criconfig.PluginConfig
	registryData, ok := data.Get("plugins.cri").(*toml.Tree)
	if !ok {
		return result
	}
	err = registryData.Unmarshal(&config)
	if err != nil {
		return result
	}

	if len(config.Registry.Mirrors[registry].Endpoints) > 0 {
		ep := config.Registry.Mirrors[registry].Endpoints[0]
		u, _ := url.Parse(ep)
		if u != nil && u.Scheme != "" {
			result.schema = u.Scheme
		}
	}

	if auth := config.Registry.Configs[registry].Auth; auth != nil {
		result.username = auth.Username
		result.password = auth.Password
	}

	return result
}

func getImageConfig(ctx context.Context, image containerd.Image) (*ocispec.ImageConfig, error) {
	desc, err := image.Config(ctx)
	if err != nil {
		return nil, err
	}
	switch desc.MediaType {
	case ocispec.MediaTypeImageConfig, images.MediaTypeDockerSchema2Config:
		var ocispecImage ocispec.Image
		b, err := content.ReadBlob(ctx, image.ContentStore(), desc)
		if err != nil {
			return nil, err
		}

		if err := json.Unmarshal(b, &ocispecImage); err != nil {
			return nil, err
		}
		return &ocispecImage.Config, nil
	default:
		return nil, fmt.Errorf("unknown media type %q", desc.MediaType)
	}
}

func (c *containerdImageCliImpl) ImagePush(image, user, pass string, logger event.Logger, timeout int) error {
	printLog(logger, "info", fmt.Sprintf("开始推送镜像：%s", image), map[string]string{"step": "pushimage"})
	named, err := reference.ParseDockerRef(image)
	if err != nil {
		return err
	}
	reference := named.String()
	ctx := namespaces.WithNamespace(context.Background(), Namespace)
	img, err := c.client.ImageService().Get(ctx, reference)
	if err != nil {
		return errors.Wrap(err, "unable to resolve image to manifest")
	}
	desc := img.Target
	cs := c.client.ContentStore()
	if manifests, err := images.Children(ctx, cs, desc); err == nil && len(manifests) > 0 {
		matcher := platforms.NewMatcher(platforms.DefaultSpec())
		for _, manifest := range manifests {
			if manifest.Platform != nil && matcher.Match(*manifest.Platform) {
				if _, err := images.Children(ctx, cs, manifest); err != nil {
					return errors.Wrap(err, "no matching manifest")
				}
				desc = manifest
				break
			}
		}
	}
	NewTracker := docker.NewInMemoryTracker()
	options := docker.ResolverOptions{
		Tracker: NewTracker,
	}
	hostOptions := config.HostOptions{
		DefaultTLS: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
	hostOptions.Credentials = func(host string) (string, string, error) {
		return user, pass, nil
	}
	options.Hosts = config.ConfigureHosts(ctx, hostOptions)
	resolver := docker.NewResolver(options)
	ongoing := newPushJobs(NewTracker)

	eg, ctx := errgroup.WithContext(ctx)
	// used to notify the progress writer
	doneCh := make(chan struct{})
	eg.Go(func() error {
		defer close(doneCh)
		jobHandler := images.HandlerFunc(func(ctx context.Context, desc ocispec.Descriptor) ([]ocispec.Descriptor, error) {
			ongoing.add(remotes.MakeRefKey(ctx, desc))
			return nil, nil
		})

		ropts := []containerd.RemoteOpt{
			containerd.WithResolver(resolver),
			containerd.WithImageHandler(jobHandler),
		}
		return c.client.Push(ctx, reference, desc, ropts...)
	})
	writer := logger.GetWriter("builder", "info")
	eg.Go(func() error {
		var (
			ticker = time.NewTicker(100 * time.Millisecond)
			fw     = progress.NewWriter(writer)
			start  = time.Now()
			done   bool
		)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				fw.Flush()
				tw := tabwriter.NewWriter(fw, 1, 8, 1, ' ', 0)
				ctrcontent.Display(tw, ongoing.status(), start)
				tw.Flush()
				if done {
					fw.Flush()
					return nil
				}
			case <-doneCh:
				done = true
			case <-ctx.Done():
				done = true // allow ui to update once more
			}
		}
	})
	// wait all goroutines
	waitErr := eg.Wait()
	if waitErr != nil {
		printLog(logger, "error", fmt.Sprintf("推送镜像 %s 失败，错误信息：%v", reference, waitErr), map[string]string{"step": "pushimage"})
		return waitErr
	}
	// create a container
	printLog(logger, "info", fmt.Sprintf("成功推送镜像：%s", reference), map[string]string{"step": "pushimage"})
	return nil
}

// ImageTag change docker image tag
func (c *containerdImageCliImpl) ImageTag(source, target string, logger event.Logger, timeout int) error {
	srcNamed, err := reference.ParseDockerRef(source)
	if err != nil {
		return err
	}
	srcImage := srcNamed.String()
	targetNamed, err := reference.ParseDockerRef(target)
	if err != nil {
		return err
	}
	targetImage := targetNamed.String()
	logrus.Infof(fmt.Sprintf("change image tag: %s -> %s", srcImage, targetImage))
	printLog(logger, "info", fmt.Sprintf("修改镜像 Tag：%s -> %s", source, target), map[string]string{"step": "changetag"})
	ctx := namespaces.WithNamespace(context.Background(), Namespace)
	imageService := c.client.ImageService()
	image, err := imageService.Get(ctx, srcImage)
	if err != nil {
		logrus.Errorf("imagetag imageService Get error: %s", err.Error())
		return err
	}
	image.Name = targetImage
	if _, err = imageService.Create(ctx, image); err != nil {
		if errdefs.IsAlreadyExists(err) {
			if err = imageService.Delete(ctx, image.Name); err != nil {
				logrus.Errorf("imagetag imageService Delete error: %s", err.Error())
				return err
			}
			if _, err = imageService.Create(ctx, image); err != nil {
				logrus.Errorf("imageService Create error: %s", err.Error())
				return err
			}
		} else {
			logrus.Errorf("imagetag imageService Create error: %s", err.Error())
			return err
		}
	}
	logrus.Info("change image tag success")
	printLog(logger, "info", "成功修改镜像 Tag", map[string]string{"step": "changetag"})
	return nil
}

// ImagesPullAndPush Used to process mirroring of non local components, example: builder, runner, /wt-mesh-data-panel
func (c *containerdImageCliImpl) ImagesPullAndPush(sourceImage, targetImage, username, password string, logger event.Logger) error {
	sourceImage, exists, err := c.CheckIfImageExists(sourceImage)
	if err != nil {
		logrus.Errorf("failed to check whether the builder mirror exists: %s", err.Error())
		return err
	}
	logrus.Infof("source image %v, targetImage %v, exists %v", sourceImage, targetImage, exists)
	if !exists {
		hubUser, hubPass := chaos.GetImageUserInfoV2(sourceImage, username, password)
		if _, err := c.ImagePull(targetImage, hubUser, hubPass, logger, 15); err != nil {
			printLog(logger, "error", fmt.Sprintf("拉取镜像 %s 失败，错误信息： %v", targetImage, err), map[string]string{"step": "builder-exector", "status": "failure"})
			return err
		}
		if err := c.ImageTag(targetImage, sourceImage, logger, 15); err != nil {
			printLog(logger, "error", fmt.Sprintf("修改镜像 Tag： %s -> %s 失败", targetImage, sourceImage), map[string]string{"step": "builder-exector", "status": "failure"})
			return err
		}
		if err := c.ImagePush(sourceImage, hubUser, hubPass, logger, 15); err != nil {
			printLog(logger, "error", fmt.Sprintf("推送镜像 %s 失败，错误信息：%v", sourceImage, err), map[string]string{"step": "builder-exector", "status": "failure"})
			return err
		}
	}
	return nil
}

// ImageRemove remove image
func (c *containerdImageCliImpl) ImageRemove(image string) error {
	named, err := reference.ParseDockerRef(image)
	if err != nil {
		return err
	}
	reference := named.String()
	ctx := namespaces.WithNamespace(context.Background(), Namespace)
	imageStore := c.client.ImageService()
	err = imageStore.Delete(ctx, reference)
	if err != nil && !errdefs.IsNotFound(err) {
		logrus.Errorf("failed to remove image %s: %v", reference, err)
		return nil
	}
	return err
}

// ImageSave save image to tar file
// destination destination file name eg. /tmp/xxx.tar
func (c *containerdImageCliImpl) ImageSave(image, destination string) error {
	named, err := reference.ParseDockerRef(image)
	if err != nil {
		return err
	}
	ctx := namespaces.WithNamespace(context.Background(), Namespace)
	var exportOpts = []archive.ExportOpt{archive.WithImage(c.client.ImageService(), named.String())}
	w, err := os.Create(destination)
	if err != nil {
		return err
	}
	defer w.Close()
	return c.client.Export(ctx, w, exportOpts...)
}

// TrustedImagePush push image to trusted registry
func (c *containerdImageCliImpl) TrustedImagePush(image, user, pass string, logger event.Logger, timeout int) error {
	if err := CheckTrustedRepositories(image, user, pass); err != nil {
		return err
	}
	return c.ImagePush(image, user, pass, logger, timeout)
}

// ImageLoad load image from  tar file
// destination destination file name eg. /tmp/xxx.tar
func (c *containerdImageCliImpl) ImageLoad(tarFile string, logger event.Logger) error {
	ctx := namespaces.WithNamespace(context.Background(), Namespace)
	reader, err := os.OpenFile(tarFile, os.O_RDONLY, 0644)
	if err != nil {
		return err
	}
	defer reader.Close()
	if _, err = c.client.Import(ctx, reader); err != nil {
		return err
	}
	return nil
}
