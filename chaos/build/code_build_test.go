package build

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	jobc "github.com/wutong-paas/wutong/chaos/job"
	"github.com/wutong-paas/wutong/chaos/parser/code"
	"github.com/wutong-paas/wutong/chaos/sources"
	"github.com/wutong-paas/wutong/event"
	k8sutil "github.com/wutong-paas/wutong/util/k8s"
	"k8s.io/client-go/kubernetes"
)

func TestCreateJob(t *testing.T) {
	event.NewLoggerManager()
	restConfig, err := k8sutil.NewRestConfig("/Users/fanyangyang/Documents/company/wutong/remote/192.168.2.206/admin.kubeconfig")
	if err != nil {
		t.Fatal(err)
	}
	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		t.Fatal(err)
	}
	// dockerClient, err := client.NewEnvClient()
	// if err != nil {
	// 	t.Fatal("new docker error: ", err.Error())
	// }
	logger := event.GetLogger("0000")
	req := Request{
		ServerType: "git",
		// DockerClient:  dockerClient,
		KubeClient:    clientset,
		ServiceID:     "d9b8d718510dc53118af1e1219e36d3a",
		DeployVersion: "123",
		TenantEnvID:   "7c89455140284fd7b263038b44dc65bc",
		Lang:          code.JavaMaven,
		Runtime:       "1.8",
		Logger:        logger,
	}
	req.BuildEnvs = map[string]string{
		"PROCFILE": "web: java $JAVA_OPTS -jar target/java-maven-demo-0.0.1.jar",
		"PROC_ENV": `{"procfile": "", "dependencies": {}, "language": "Java-maven", "runtimes": "1.8"}`,
		"RUNTIME":  "1.8",
	}
	req.CacheDir = fmt.Sprintf("/cache/build/%s/cache/%s", req.TenantEnvID, req.ServiceID)
	req.TGZDir = fmt.Sprintf("/wtdata/build/tenantEnv/%s/slug/%s", req.TenantEnvID, req.ServiceID)
	req.SourceDir = fmt.Sprintf("/cache/source/build/%s/%s", req.TenantEnvID, req.ServiceID)
	sb := slugBuild{tgzDir: "string"}
	if err := sb.runBuildJob(&req); err != nil {
		t.Fatal(err)
	}
	fmt.Println("create job finished")

}

func Test1(t *testing.T) {
	tarFile := "/opt/wutong/pkg/wutong-pkg-V5.2-dev.tgz"
	srcFile, err := os.Open(tarFile)
	if err != nil {
		t.Fatal(err)
	}
	defer srcFile.Close()
	gr, err := gzip.NewReader(srcFile) //handle gzip feature
	if err != nil {
		t.Fatal(err)
	}
	defer gr.Close()
	tr := tar.NewReader(gr) // tar reader
	now := time.Now()
	for hdr, err := tr.Next(); err != io.EOF; hdr, err = tr.Next() { // next range tar info
		if err != nil {
			t.Fatal(err)
			continue
		}
		// 读取文件信息
		fi := hdr.FileInfo()

		if !strings.HasPrefix(fi.Name(), "._") && strings.HasSuffix(fi.Name(), ".tgz") {
			t.Logf("name: %s, size: %d", fi.Name(), fi.Size())

		}
	}
	t.Logf("cost: %d", time.Since(now))
}

func TestDockerClient(t *testing.T) {
	dockerClient, err := client.NewEnvClient()
	if err != nil {
		t.Fatal("new docker error: ", err.Error())
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	containers, err := dockerClient.ContainerList(ctx, container.ListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	for _, container := range containers {
		t.Log("container id : ", container.ID)
	}
	// images, err := dockerClient.ImageList(ctx, types.ImageListOptions{})
	// for _, image := range images {
	// 	t.Log("image is : ", image.ID)
	// }
}

func TestBuildFromOSS(t *testing.T) {
	restConfig, err := k8sutil.NewRestConfig("/Users/barnett/.kube/config")
	if err != nil {
		t.Fatal(err)
	}
	os.Setenv("IMAGE_PULL_SECRET", "wt-hub-credentials")
	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		t.Fatal(err)
	}
	stop := make(chan struct{})
	if err := jobc.InitJobController("wt-system", stop, clientset); err != nil {
		t.Fatal(err)
	}
	logger := event.GetTestLogger()
	req := &Request{
		ServerType:    "oss",
		RepositoryURL: "http://8081.wt021644.64q1jlfb.17f4cc.wtapps.cn/artifactory/dev/java-war-demo-master.zip",
		CodeSouceInfo: sources.CodeSourceInfo{
			User:     "demo",
			Password: "wt123465!",
		},
		KubeClient:    clientset,
		Ctx:           context.Background(),
		ServiceID:     "d9b8d718510dc53118af1e1219e36d3a",
		DeployVersion: "123asdadsadsasdasd1",
		TenantEnvID:   "7c89455140284fd7b263038b44dc65bc",
		Lang:          code.OSS,
		Logger:        logger,
		WTDataPVCName: "wt-cpt-wtdata",
		CachePVCName:  "wt-chaos-cache",
	}
	build, err := GetBuild(code.OSS)
	if err != nil {
		t.Fatal(err)
	}
	res, err := build.Build(req)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(res.MediumPath)
}
