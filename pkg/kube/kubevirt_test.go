package kube

import (
	"context"
	"testing"

	"github.com/wutong-paas/wutong/util"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubevirtcorev1 "kubevirt.io/api/core/v1"
)

func TestCreateKubevirtVM(t *testing.T) {
	vm := &kubevirtcorev1.VirtualMachine{
		TypeMeta: metav1.TypeMeta{
			Kind:       "VirtualMachine",
			APIVersion: "kubevirt.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "hellovm",
			Namespace: metav1.NamespaceDefault,
		},
		Spec: kubevirtcorev1.VirtualMachineSpec{
			Running: util.Ptr(true),
			Template: &kubevirtcorev1.VirtualMachineInstanceTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"kubevirt.io/size":   "small",
						"kubevirt.io/domain": "hellovm",
					},
				},
				Spec: kubevirtcorev1.VirtualMachineInstanceSpec{
					Domain: kubevirtcorev1.DomainSpec{
						Devices: kubevirtcorev1.Devices{
							Disks: []kubevirtcorev1.Disk{
								{
									Name: "containerdisk",
									DiskDevice: kubevirtcorev1.DiskDevice{
										Disk: &kubevirtcorev1.DiskTarget{
											Bus: "virtio",
										},
									},
								},
								{Name: "cloudinitdisk",
									DiskDevice: kubevirtcorev1.DiskDevice{
										Disk: &kubevirtcorev1.DiskTarget{
											Bus: "virtio",
										},
									},
								},
							},
							Interfaces: []kubevirtcorev1.Interface{
								{
									Name: "default",
									InterfaceBindingMethod: kubevirtcorev1.InterfaceBindingMethod{
										Masquerade: &kubevirtcorev1.InterfaceMasquerade{},
									},
								},
							},
						},
						Resources: kubevirtcorev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								"memory": resource.MustParse("64M"),
							},
						},
					},
					Networks: []kubevirtcorev1.Network{
						{
							Name: "default",
							NetworkSource: kubevirtcorev1.NetworkSource{
								Pod: &kubevirtcorev1.PodNetwork{},
							},
						},
					},
					Volumes: []kubevirtcorev1.Volume{
						{
							Name: "containerdisk",
							VolumeSource: kubevirtcorev1.VolumeSource{
								ContainerDisk: &kubevirtcorev1.ContainerDiskSource{
									Image: "quay.io/kubevirt/cirros-container-disk-demo",
								},
							},
						}, {
							Name: "cloudinitdisk",
							VolumeSource: kubevirtcorev1.VolumeSource{
								CloudInitNoCloud: &kubevirtcorev1.CloudInitNoCloudSource{
									UserData: "Hello Kubevirt!",
								},
							},
						},
					},
				},
			},
		},
	}
	created, err := KubevirtClient().VirtualMachine(vm.Namespace).Create(context.Background(), vm, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("create vm failed: %v", err)
	}
	t.Logf("created vm: %s/%s", created.GetNamespace(), created.GetName())
}

func TestListKubeVirtVMs(t *testing.T) {
	vms, err := KubevirtClient().VirtualMachine(metav1.NamespaceDefault).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		t.Fatalf("list vm failed: %v", err)
	}
	for _, vm := range vms.Items {
		t.Logf("vm: %s/%s", vm.GetNamespace(), vm.GetName())
	}
}
