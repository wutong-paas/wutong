package cloudos

import (
	"testing"
)

func TestS3PutObject(t *testing.T) {
	cfg := &Config{
		Endpoint:   "9000.wt4433a2.52bli69h.0196bd.wtapps.cn",
		AccessKey:  "dummy",
		SecretKey:  "dummy",
		BucketName: "my-bucket",
	}

	cs, err := newS3(cfg)
	if err != nil {
		t.Fatalf("create s3 driver: %v", err)
	}

	if err := cs.PutObject("aws-sdk-go-1.25.25.zip", "/Users/abewang/Downloads/aws-sdk-go-1.25.25.zip"); err != nil {
		t.Error(err)
	}
}

func TestS3GetObject(t *testing.T) {
	cfg := &Config{
		Endpoint:   "9000.wt4433a2.52bli69h.0196bd.wtapps.cn",
		AccessKey:  "access_key",
		SecretKey:  "dummy",
		BucketName: "my-bucket",
	}

	cs, err := newS3(cfg)
	if err != nil {
		t.Fatalf("create s3 driver: %v", err)
	}

	if err := cs.GetObject("wutong-logo.png", "wutong-logo2.png"); err != nil {
		t.Error(err)
	}
}

func TestS3DeleteObject(t *testing.T) {
	cfg := &Config{
		Endpoint:   "9000.wt4433a2.52bli69h.0196bd.wtapps.cn",
		AccessKey:  "access_key",
		SecretKey:  "dummy",
		BucketName: "my-bucket",
	}

	cs, err := newS3(cfg)
	if err != nil {
		t.Fatalf("create s3 driver: %v", err)
	}

	if err := cs.DeleteObject("wutong-logo.png"); err != nil {
		t.Error(err)
	}
}
