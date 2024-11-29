package handler

import (
	"path"
	"runtime"
	"testing"

	"github.com/wutong-paas/wutong/util"
	kubevirtcorev1 "kubevirt.io/api/core/v1"
)

func TestValidatePassword(t *testing.T) {
	testdata := []struct {
		password string
		want     bool
	}{
		{
			password: "123456",
			want:     false,
		},
		{
			password: "12345678",
			want:     false,
		},
		{
			password: "abcdefgh",
			want:     false,
		},
		{
			password: "ABCDEFGH",
			want:     false,
		},
		{
			password: "Abcabc123",
			want:     true,
		}, {
			password: "Abab12@>",
			want:     true,
		},
	}

	for _, v := range testdata {
		got, err := validatePassword(v.password)
		if got != v.want {
			t.Errorf("ValidatePassword(%v) = %v, want %v", v.password, got, v.want)
		}
		if err != nil {
			t.Logf("ValidatePassword(%v) = %v, want %v, err: %v", v.password, got, v.want, err)
		}
	}
}

func TestBootDisk(t *testing.T) {
	var disks = []kubevirtcorev1.Disk{
		{
			Name: "containerdisk",
			DiskDevice: kubevirtcorev1.DiskDevice{
				Disk: &kubevirtcorev1.DiskTarget{
					Bus: "virtio",
				},
			},
			BootOrder: util.Ptr(uint(2)),
		},
		{
			Name: "bootdisk",
			DiskDevice: kubevirtcorev1.DiskDevice{
				CDRom: &kubevirtcorev1.CDRomTarget{
					Bus: util.If(runtime.GOARCH == "arm64", kubevirtcorev1.DiskBusSCSI, kubevirtcorev1.DiskBusSATA),
				},
			},
			BootOrder: util.Ptr(uint(1)),
		},
	}

	containsBootDisk := func(disks []kubevirtcorev1.Disk) bool {
		for _, disk := range disks {
			if disk.Name == "bootdisk" && disk.BootOrder != nil && *disk.BootOrder == 1 {
				return true
			}
		}
		return false
	}

	t.Log(containsBootDisk(disks))
}

func TestUrlPathBase(t *testing.T) {
	testdata := []struct {
		url  string
		want string
	}{
		{
			url:  "http://www.baidu.com/abc/test.pdf",
			want: "test.pdf",
		},
		{
			url:  "http://www.baidu.com/abc/test",
			want: "test",
		},
	}

	for _, v := range testdata {
		got := path.Base(v.url)
		if got != v.want {
			t.Errorf("path.Base(%v) = %v, want %v", v.url, got, v.want)
		}
	}
}
