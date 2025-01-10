package containerutil

import (
	"os"
	"testing"
)

func TestSaveImage(t *testing.T) {
	cli, err := NewClient("docker", "/var/run/docker.sock")
	if err != nil {
		t.Error(err)
		return
	}

	if _, err := cli.ImagePull("registry.cn-hangzhou.aliyuncs.com/pding/nginx:latest", "", "", 0); err != nil {
		t.Error(err)
		return
	}

	if err := cli.ImageSave("nginx.tar", []string{"registry.cn-hangzhou.aliyuncs.com/pding/nginx:latest"}); err != nil {
		t.Error(err)
	} else {
		t.Log("success")
		os.Remove("nginx.tar")
	}
}
