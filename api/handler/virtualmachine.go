// Copyright (C) 2014-2018 Wutong Co., Ltd.
// WUTONG, Application Management Platform

// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version. For any non-GPL usage of Wutong,
// one or multiple Commercial Licenses authorized by Wutong Co., Ltd.
// must be obtained first.

// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.

// You should have received a copy of the GNU General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package handler

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cast"
	api_model "github.com/wutong-paas/wutong/api/model"
	"github.com/wutong-paas/wutong/chaos"
	"github.com/wutong-paas/wutong/db"
	dbmodel "github.com/wutong-paas/wutong/db/model"
	"github.com/wutong-paas/wutong/pkg/kube"
	"github.com/wutong-paas/wutong/util"
	"golang.org/x/crypto/bcrypt"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"
	kubevirtcorev1 "kubevirt.io/api/core/v1"
	cdicorev1beta1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"
)

var defailtOSDiskSize int64 = 40

var (
	bootDiskName            = "bootdisk"
	containerDiskName       = "containerdisk"
	cloudInitDiskName       = "cloudinitdisk"
	virtioContainerDiskName = "virtiocontainerdisk"
)

// CreateVM 创建 kubevirt 虚拟机
func (s *ServiceAction) CreateVM(tenantEnv *dbmodel.TenantEnvs, req *api_model.CreateVMRequest) (*api_model.CreateVMResponse, error) {
	if req.OSDiskSize == 0 {
		req.OSDiskSize = defailtOSDiskSize
	}

	if !strings.HasSuffix(req.OSSourceURL, ".iso") {
		req.User = strings.TrimSpace(req.User)
		if req.User == "" {
			return nil, fmt.Errorf("虚拟机初始用户名称不能为空！")
		}
		if req.User == "root" {
			return nil, fmt.Errorf("虚拟机初始用户名称不能为root！")
		}

		req.Password = strings.TrimSpace(req.Password)
		if req.Password == "" {
			return nil, fmt.Errorf("虚拟机初始用户密码不能为空！")
		}

		if ok, err := validatePassword(req.Password); !ok {
			return nil, err
		}
	}

	if err := CheckTenantEnvResource(context.Background(), tenantEnv, int(req.RequestMemory)*1024); err == ErrTenantEnvLackOfMemory {
		return nil, fmt.Errorf("虚拟机申请内存 %dGi 超过当前环境内存限额，无法创建！", req.RequestMemory)
	}

	vmlist, err := kube.ListKubeVirtVMs(s.dynamicClient, tenantEnv.Namespace)
	if err != nil {
		return nil, fmt.Errorf("环境校验失败！")
	}

	for _, v := range vmlist {
		if v.Name == req.Name {
			return nil, fmt.Errorf("虚拟机标识 %s 被占用，尝试使用其他标识创建虚拟机！", req.Name)
		}

		if v.Annotations["wutong.io/display-name"] == req.DisplayName {
			return nil, fmt.Errorf("虚拟机名称 %s 被占用，尝试使用其他名称创建虚拟机！", req.DisplayName)
		}
	}

	wutongLabels := labelsFromTenantEnv(tenantEnv)
	wutongLabels = labels.Merge(wutongLabels, map[string]string{
		"wutong.io/vm-id": req.Name,
	})

	var installType api_model.VMInstallType
	if req.OSSourceFrom == api_model.OSSourceFromHTTP && strings.HasSuffix(req.OSSourceURL, ".iso") {
		installType = api_model.VMInstallTypeISO
	}

	vm := buildVMBase(req, tenantEnv.Namespace, wutongLabels)

	// 根据安装类型设置虚拟机配置，例如：ISO 安装时需要准备启动引导盘和数据盘
	switch installType {
	case api_model.VMInstallTypeISO:
		// 创建数据盘 pvc
		err := createContainerDiskPVC(req, tenantEnv, wutongLabels, s)
		if err != nil {
			return nil, err
		}

		// 设置 datavolume templates
		dvName := req.Name + "-iso"
		vm.Spec.DataVolumeTemplates = buildVMDataVolumeTemplates(dvName, req.OSSourceFrom, req.OSSourceURL, 10)

		// 设置 disks
		vm.Spec.Template.Spec.Domain.Devices.Disks = []kubevirtcorev1.Disk{
			{
				Name: containerDiskName,
				DiskDevice: kubevirtcorev1.DiskDevice{
					Disk: &kubevirtcorev1.DiskTarget{
						Bus: kubevirtcorev1.DiskBusVirtio,
					},
				},
				BootOrder: util.Ptr(uint(2)),
			},
			{
				Name: bootDiskName,
				DiskDevice: kubevirtcorev1.DiskDevice{
					CDRom: &kubevirtcorev1.CDRomTarget{
						Bus: util.If(runtime.GOARCH == "arm64", kubevirtcorev1.DiskBusSCSI, kubevirtcorev1.DiskBusSATA),
					},
				},
				BootOrder: util.Ptr(uint(1)),
			},
		}

		// 设置 volumes
		vm.Spec.Template.Spec.Volumes = []kubevirtcorev1.Volume{
			{
				Name: containerDiskName,
				VolumeSource: kubevirtcorev1.VolumeSource{
					PersistentVolumeClaim: &kubevirtcorev1.PersistentVolumeClaimVolumeSource{
						PersistentVolumeClaimVolumeSource: corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: pvcName(req.Name, "data"),
						},
					},
				},
			}, {
				Name: bootDiskName,
				VolumeSource: kubevirtcorev1.VolumeSource{
					DataVolume: &kubevirtcorev1.DataVolumeSource{
						Name: dvName,
					},
				},
			},
		}
	default:
		// 设置 datavolume templates
		dvName := req.Name + "-data"
		vm.Spec.DataVolumeTemplates = buildVMDataVolumeTemplates(dvName, req.OSSourceFrom, req.OSSourceURL, req.OSDiskSize)

		// 设置 disks
		vm.Spec.Template.Spec.Domain.Devices.Disks = []kubevirtcorev1.Disk{
			{
				Name: containerDiskName,
				DiskDevice: kubevirtcorev1.DiskDevice{
					Disk: &kubevirtcorev1.DiskTarget{
						Bus: kubevirtcorev1.DiskBusVirtio,
					},
				},
			},
			{
				Name: cloudInitDiskName,
				DiskDevice: kubevirtcorev1.DiskDevice{
					Disk: &kubevirtcorev1.DiskTarget{
						Bus: kubevirtcorev1.DiskBusVirtio,
					},
				},
			},
		}

		// 设置 cloudinit
		vmUserData, err := buildVMUserData(s.kubeClient, req.User, req.Password)
		if err != nil {
			return nil, err
		}
		// 设置 volumes
		vm.Spec.Template.Spec.Volumes = []kubevirtcorev1.Volume{
			{
				Name: containerDiskName,
				VolumeSource: kubevirtcorev1.VolumeSource{
					DataVolume: &kubevirtcorev1.DataVolumeSource{
						Name: dvName,
					},
				},
			},
			{
				Name: cloudInitDiskName,
				VolumeSource: kubevirtcorev1.VolumeSource{
					CloudInitNoCloud: &kubevirtcorev1.CloudInitNoCloudSource{
						UserData: vmUserData,
					},
				},
			},
		}
	}

	// 该功能待集成，安装 virtio 驱动（windows），arm64 环境下待验证。
	if req.LoadVirtioDriver {
		vm.Spec.Template.Spec.Domain.Devices.Disks = append(vm.Spec.Template.Spec.Domain.Devices.Disks, kubevirtcorev1.Disk{
			Name: virtioContainerDiskName,
			DiskDevice: kubevirtcorev1.DiskDevice{
				CDRom: &kubevirtcorev1.CDRomTarget{
					Bus: util.If(runtime.GOARCH == "arm64", kubevirtcorev1.DiskBusSCSI, kubevirtcorev1.DiskBusSATA),
				},
			},
		})

		vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, kubevirtcorev1.Volume{
			Name: virtioContainerDiskName,
			VolumeSource: kubevirtcorev1.VolumeSource{
				ContainerDisk: &kubevirtcorev1.ContainerDiskSource{
					Image: chaos.VIRTIOCONTAINERDISKIMAGENAME,
				},
			},
		})
	}

	var result = &api_model.CreateVMResponse{
		VMProfile: vmProfileFromKubeVirtVM(vm, nil),
	}

	created, err := kube.CreateKubevirtVM(s.dynamicClient, vm)
	if err != nil {
		if k8sErrors.IsAlreadyExists(err) {
			return result, fmt.Errorf("虚拟机标识 %s 被占用，尝试使用其他标识创建虚拟机！", req.Name)
		}
		logrus.Errorf("create vm failed, error: %s", err.Error())
		return result, fmt.Errorf("创建虚拟机 %s 失败！", req.Name)
	}

	// 用户使用 iso 自行安装虚拟机时，不需要自动添加额外的端口
	if installType != api_model.VMInstallTypeISO {
		// 创建 ssh 端口
		s.AddVMPort(tenantEnv, req.Name, &api_model.AddVMPortRequest{
			VMPort:   22,
			Protocol: api_model.VMPortProtocolSSH,
		})

		// 创建 file-browser 端口
		s.AddVMPort(tenantEnv, req.Name, &api_model.AddVMPortRequest{
			VMPort:   6173,
			Protocol: api_model.VMPortProtocolHTTP,
		})
	}

	result.Status = string(created.Status.PrintableStatus)
	return result, nil
}

func (s *ServiceAction) GetVM(tenantEnv *dbmodel.TenantEnvs, vmID string) (*api_model.GetVMResponse, error) {
	vm, err := kube.GetKubeVirtVM(s.dynamicClient, tenantEnv.Namespace, vmID)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			return nil, fmt.Errorf("虚拟机 %s 不存在！", vmID)
		}
		logrus.Errorf("get vm failed, error: %s", err.Error())
		return nil, errors.New("获取虚拟机失败！")
	}
	if vm == nil {
		return nil, fmt.Errorf("获取虚拟机 %s 信息失败！", vmID)
	}

	vmi, _ := kube.GetKubeVirtVMI(s.dynamicClient, tenantEnv.Namespace, vmID)

	vmProfile := vmProfileFromKubeVirtVM(vm, vmi)

	return &api_model.GetVMResponse{
		VMProfile: vmProfile,
	}, nil
}

func (s *ServiceAction) GetVMConditions(tenantEnv *dbmodel.TenantEnvs, vmID string) (*api_model.GetVMConditionsResponse, error) {
	vm, err := kube.GetKubeVirtVM(s.dynamicClient, tenantEnv.Namespace, vmID)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			return nil, fmt.Errorf("虚拟机 %s 不存在！", vmID)
		}
		logrus.Errorf("get vm failed, error: %s", err.Error())
		return nil, errors.New("获取虚拟机失败！")
	}
	if vm == nil {
		return nil, fmt.Errorf("获取虚拟机 %s 信息失败！", vmID)
	}

	return &api_model.GetVMConditionsResponse{
		Conditions: vmConditions(vm),
	}, nil
}

func (s *ServiceAction) UpdateVM(tenantEnv *dbmodel.TenantEnvs, vmID string, req *api_model.UpdateVMRequest) (*api_model.UpdateVMResponse, error) {
	vm, err := kube.GetKubeVirtVM(s.dynamicClient, tenantEnv.Namespace, vmID)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			return nil, fmt.Errorf("虚拟机 %s 不存在！", vmID)
		}
		logrus.Errorf("get vm failed, error: %s", err.Error())
		return nil, errors.New("获取虚拟机失败！")
	}
	if vm == nil {
		return nil, fmt.Errorf("获取虚拟机 %s 信息失败！", vmID)
	}

	if req.DisplayName != "" {
		vm.Annotations["wutong.io/display-name"] = req.DisplayName
	}
	if req.Desc != "" {
		vm.Annotations["wutong.io/desc"] = req.Desc
	}
	if req.RequestCPU > 0 {
		vm.Annotations["wutong.io/vm-request-cpu"] = fmt.Sprintf("%d", req.RequestCPU)
		if vm.Spec.Template.Spec.Domain.Resources.Requests == nil {
			vm.Spec.Template.Spec.Domain.Resources.Requests = corev1.ResourceList{}
		}
		vm.Spec.Template.Spec.Domain.Resources.Requests[corev1.ResourceCPU] = resource.MustParse(fmt.Sprintf("%dm", req.RequestCPU))
		if vm.Spec.Template.Spec.Domain.Resources.Limits == nil {
			vm.Spec.Template.Spec.Domain.Resources.Limits = corev1.ResourceList{}
		}
		vm.Spec.Template.Spec.Domain.Resources.Limits[corev1.ResourceCPU] = resource.MustParse(fmt.Sprintf("%dm", req.RequestCPU))
	}
	if req.RequestMemory > 0 {
		vm.Annotations["wutong.io/vm-request-memory"] = fmt.Sprintf("%d", req.RequestMemory)
		if vm.Spec.Template.Spec.Domain.Memory == nil {
			vm.Spec.Template.Spec.Domain.Memory = &kubevirtcorev1.Memory{}
		}
		if vm.Spec.Template.Spec.Domain.Memory.Guest == nil {
			vm.Spec.Template.Spec.Domain.Memory.Guest = &resource.Quantity{}
		}
		vm.Spec.Template.Spec.Domain.Memory.Guest = util.Ptr(resource.MustParse(fmt.Sprintf("%dGi", req.RequestMemory)))
		if vm.Spec.Template.Spec.Domain.Resources.Requests == nil {
			vm.Spec.Template.Spec.Domain.Resources.Requests = corev1.ResourceList{}
		}
		vm.Spec.Template.Spec.Domain.Resources.Requests[corev1.ResourceMemory] = resource.MustParse(fmt.Sprintf("%dGi", req.RequestMemory))
		if vm.Spec.Template.Spec.Domain.Resources.Limits == nil {
			vm.Spec.Template.Spec.Domain.Resources.Limits = corev1.ResourceList{}
		}
		vm.Spec.Template.Spec.Domain.Resources.Limits[corev1.ResourceMemory] = resource.MustParse(fmt.Sprintf("%dGi", req.RequestMemory))
	}
	vm.Annotations["wutong.io/last-modifier"] = req.Operator
	vm.Annotations["wutong.io/last-modification-timestamp"] = metav1.Now().UTC().Format(time.RFC3339)

	if req.DefaultLoginUser != "" {
		vm.Annotations["wutong.io/vm-default-login-user"] = req.DefaultLoginUser
	}

	var nodeSelector = map[string]string{
		"wutong.io/vm-schedulable": "true",
	}
	for _, labelKey := range req.NodeSelectorLabels {
		var labelVal string
		kv := strings.Split(labelKey, "=")
		if len(kv) > 1 {
			labelVal = kv[1]
		}

		nodeSelector["vm-scheduling-label.wutong.io/"+kv[0]] = labelVal
	}
	vm.Spec.Template.Spec.NodeSelector = nodeSelector

	updated, err := kube.UpdateKubeVirtVM(s.dynamicClient, vm)
	if err != nil {
		logrus.Errorf("update vm failed, error: %s", err.Error())
		return nil, fmt.Errorf("启动虚拟机 %s 失败！", vmID)
	}

	vmProfile := vmProfileFromKubeVirtVM(updated, nil)

	return &api_model.UpdateVMResponse{
		VMProfile: vmProfile,
	}, nil
}

func (s *ServiceAction) StartVM(tenantEnv *dbmodel.TenantEnvs, vmID string) (*api_model.StartVMResponse, error) {
	vm, err := kube.GetKubeVirtVM(s.dynamicClient, tenantEnv.Namespace, vmID)
	if err != nil {
		return nil, fmt.Errorf("获取虚拟机 %s 信息失败！", vmID)
	}

	memory, err := cast.ToIntE(vm.Annotations["wutong.io/vm-request-memory"])
	if err != nil {
		return nil, fmt.Errorf("无法启动虚拟机 %s，获取虚拟机申请内存失败！", vmID)
	}

	if err := CheckTenantEnvResource(context.Background(), tenantEnv, memory*1024); err == ErrTenantEnvLackOfMemory {
		return nil, fmt.Errorf("虚拟机申请内存 %dGi 超过当前环境内存限额，无法启动！", memory)
	}

	vm.Spec.Running = util.Ptr(true)
	updated, err := kube.UpdateKubeVirtVM(s.dynamicClient, vm)
	if err != nil {
		logrus.Errorf("update vm failed, error: %s", err.Error())
		return nil, fmt.Errorf("启动虚拟机 %s 失败！", vmID)
	}

	vmProfile := vmProfileFromKubeVirtVM(updated, nil)

	return &api_model.StartVMResponse{
		VMProfile: vmProfile,
	}, nil
}

func (s *ServiceAction) StopVM(tenantEnv *dbmodel.TenantEnvs, vmID string) (*api_model.StopVMResponse, error) {
	vm, err := kube.GetKubeVirtVM(s.dynamicClient, tenantEnv.Namespace, vmID)
	if err != nil {
		return nil, fmt.Errorf("获取虚拟机 %s 信息失败！", vmID)
	}

	vm.Spec.Running = util.Ptr(false)
	updated, err := kube.UpdateKubeVirtVM(s.dynamicClient, vm)
	if err != nil {
		logrus.Errorf("update vm failed, error: %s", err.Error())
		return nil, fmt.Errorf("停止虚拟机 %s 失败！", vmID)
	}

	vmProfile := vmProfileFromKubeVirtVM(updated, nil)

	return &api_model.StopVMResponse{
		VMProfile: vmProfile,
	}, nil
}

func (s *ServiceAction) RestartVM(tenantEnv *dbmodel.TenantEnvs, vmID string) (*api_model.RestartVMResponse, error) {
	err := kube.DeleteKubeVirtVMI(s.dynamicClient, tenantEnv.Namespace, vmID)
	if err != nil {
		logrus.Errorf("delete vmi failed, error: %s", err.Error())
		return nil, fmt.Errorf("重启虚拟机 %s 失败！", vmID)
	}

	got, err := kube.GetKubeVirtVM(s.dynamicClient, tenantEnv.Namespace, vmID)
	if err != nil {
		logrus.Errorf("update vm failed, error: %s", err.Error())
		return nil, fmt.Errorf("重启虚拟机 %s 失败！", vmID)
	}

	memory, err := cast.ToIntE(got.Annotations["wutong.io/vm-request-memory"])
	if err != nil {
		return nil, fmt.Errorf("无法启动虚拟机 %s，获取虚拟机申请内存失败！", vmID)
	}

	if err := CheckTenantEnvResource(context.Background(), tenantEnv, memory*1024); err == ErrTenantEnvLackOfMemory {
		return nil, fmt.Errorf("虚拟机申请内存 %dGi 超过当前环境内存限额，无法启动！", memory)
	}

	vmProfile := vmProfileFromKubeVirtVM(got, nil)

	return &api_model.RestartVMResponse{
		VMProfile: vmProfile,
	}, nil
}

func (s *ServiceAction) AddVMPort(tenantEnv *dbmodel.TenantEnvs, vmID string, req *api_model.AddVMPortRequest) error {
	svcName := serviceName(vmID, req.VMPort, req.Protocol)

	wutongLabels := labelsFromTenantEnv(tenantEnv)
	wutongLabels = labels.Merge(wutongLabels, map[string]string{
		"wutong.io/vm-id":            vmID,
		"wutong.io/vm-port-enabled":  "false",
		"wutong.io/vm-port":          fmt.Sprintf("%d", req.VMPort),
		"wutong.io/vm-port-protocol": req.Protocol,
	})

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      svcName,
			Namespace: tenantEnv.Namespace,
			Labels:    wutongLabels,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"wutong.io/vm-id":     vmID,
				"vm.kubevirt.io/name": vmID,
			},
			Ports: []corev1.ServicePort{
				{
					Name:       fmt.Sprintf("%s-%d", req.Protocol, req.VMPort),
					Port:       int32(req.VMPort),
					Protocol:   portProtocol(req.Protocol),
					TargetPort: intstr.FromInt(req.VMPort),
				},
			},
		},
	}

	_, err := s.kubeClient.CoreV1().Services(tenantEnv.Namespace).Create(context.Background(), svc, metav1.CreateOptions{})
	if err != nil {
		if k8sErrors.IsAlreadyExists(err) {
			return fmt.Errorf("虚拟机 %s 端口 %d(%s) 已存在！", vmID, req.VMPort, req.Protocol)
		}
		logrus.Errorf("create service failed, error: %s", err.Error())
		return fmt.Errorf("创建虚拟机 %s 端口 %d(%s) 失败！", vmID, req.VMPort, req.Protocol)
	}

	return nil
}

func (s *ServiceAction) EnableVMPort(tenantEnv *dbmodel.TenantEnvs, vmID string, req *api_model.EnableVMPortRequest) error {
	// 1、端口配置
	svcName := serviceName(vmID, req.VMPort, req.Protocol)
	svc, err := kube.GetCachedResources(s.kubeClient).ServiceLister.Services(tenantEnv.Namespace).Get(svcName)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			return fmt.Errorf("虚拟机 %s 端口 %d(%s) 不存在！", vmID, req.VMPort, req.Protocol)
		}
		logrus.Errorf("get service failed, error: %s", err.Error())
		return fmt.Errorf("获取虚拟机 %s 端口 %d(%s) 失败！", vmID, req.VMPort, req.Protocol)
	}

	err = retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		svc.Labels["wutong.io/vm-port-enabled"] = "true"
		svc.Spec.Selector = map[string]string{
			"wutong.io/vm-id":     vmID,
			"vm.kubevirt.io/name": vmID,
		}
		_, err = s.kubeClient.CoreV1().Services(tenantEnv.Namespace).Update(context.Background(), svc, metav1.UpdateOptions{})
		if err != nil {
			latest, err := s.kubeClient.CoreV1().Services(tenantEnv.Namespace).Get(context.Background(), svcName, metav1.GetOptions{})
			if err != nil {
				return err
			}
			svc.SetResourceVersion(latest.ResourceVersion)
		}
		return err
	})
	if err != nil {
		logrus.Errorf("update service failed, error: %s", err.Error())
		return fmt.Errorf("开启虚拟机 %s 端口 %d(%s) 失败！", vmID, req.VMPort, req.Protocol)
	}

	// 2、网关配置
	gateways, err := kube.GetCachedResources(s.kubeClient).IngressV1Lister.Ingresses(tenantEnv.Namespace).List(labels.SelectorFromSet(labels.Set{
		"wutong.io/vm-id":            vmID,
		"wutong.io/vm-port":          fmt.Sprintf("%d", req.VMPort),
		"wutong.io/vm-port-protocol": req.Protocol,
	}))
	if err != nil {
		logrus.Errorf("list ingress failed, error: %s", err.Error())
		return fmt.Errorf("关闭虚拟机 %s 端口 %d(%s) 下网关失败！", vmID, req.VMPort, req.Protocol)
	}

	if len(gateways) == 0 {
		// 2.1、开启了网关并默认创建第一个网关
		return s.CreateVMPortGateway(tenantEnv, vmID, &api_model.CreateVMPortGatewayRequest{
			VMPort:   req.VMPort,
			Protocol: req.Protocol,
		})
	} else {
		// 2.2、网关添加标签，让 wt-gateway 正确识别
		for _, ing := range gateways {
			ing.Labels["creator"] = "Wutong"
			err = retry.RetryOnConflict(retry.DefaultBackoff, func() error {
				_, err = s.kubeClient.NetworkingV1().Ingresses(tenantEnv.Namespace).Update(context.Background(), ing, metav1.UpdateOptions{})
				if err != nil {
					latest, err := s.kubeClient.NetworkingV1().Ingresses(tenantEnv.Namespace).Get(context.Background(), ing.Name, metav1.GetOptions{})
					if err != nil {
						return err
					}
					ing.SetResourceVersion(latest.ResourceVersion)
				}
				return err
			})
			if err != nil {
				logrus.Errorf("update ingress failed, error: %s", err.Error())
				return fmt.Errorf("开启虚拟机 %s 端口 %d(%s) 下网关失败！", vmID, req.VMPort, req.Protocol)
			}
		}
	}
	return nil
}

func (s *ServiceAction) DisableVMPort(tenantEnv *dbmodel.TenantEnvs, vmID string, req *api_model.DisableVMPortRequest) error {
	// 1、service 添加关闭标签
	svcName := serviceName(vmID, req.VMPort, req.Protocol)
	svc, err := kube.GetCachedResources(s.kubeClient).ServiceLister.Services(tenantEnv.Namespace).Get(svcName)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			return fmt.Errorf("虚拟机 %s 端口 %d(%s) 不存在！", vmID, req.VMPort, req.Protocol)
		}
		logrus.Errorf("get service failed, error: %s", err.Error())
		return fmt.Errorf("获取虚拟机 %s 端口 %d(%s) 失败！", vmID, req.VMPort, req.Protocol)
	}

	err = retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		svc.Labels["wutong.io/vm-port-enabled"] = "false"
		_, err = s.kubeClient.CoreV1().Services(tenantEnv.Namespace).Update(context.Background(), svc, metav1.UpdateOptions{})
		if err != nil {
			latest, err := s.kubeClient.CoreV1().Services(tenantEnv.Namespace).Get(context.Background(), svcName, metav1.GetOptions{})
			if err != nil {
				return err
			}
			svc.SetResourceVersion(latest.ResourceVersion)
		}
		return err
	})
	if err != nil {
		logrus.Errorf("update service failed, error: %s", err.Error())
		return fmt.Errorf("关闭虚拟机 %s 端口 %d(%s) 失败！", vmID, req.VMPort, req.Protocol)
	}

	// 2、网关去除标签，让 wt-gateway 失去识别
	gateways, err := kube.GetCachedResources(s.kubeClient).IngressV1Lister.Ingresses(tenantEnv.Namespace).List(labels.SelectorFromSet(labels.Set{
		"wutong.io/vm-id":            vmID,
		"wutong.io/vm-port":          fmt.Sprintf("%d", req.VMPort),
		"wutong.io/vm-port-protocol": req.Protocol,
	}))
	if err != nil {
		logrus.Errorf("list ingress failed, error: %s", err.Error())
		return fmt.Errorf("关闭虚拟机 %s 端口 %d(%s) 下网关失败！", vmID, req.VMPort, req.Protocol)
	}
	for _, gateway := range gateways {
		err = s.DisableVMPortGateway(tenantEnv, vmID, gateway.Name)
		if err != nil {
			return fmt.Errorf("关闭虚拟机 %s 端口 %d(%s) 下网关失败！", vmID, req.VMPort, req.Protocol)
		}
	}
	return nil
}

func (s *ServiceAction) DisableVMPortGateway(tenantEnv *dbmodel.TenantEnvs, vmID, gatewayID string) error {
	ing, err := kube.GetCachedResources(s.kubeClient).IngressV1Lister.Ingresses(tenantEnv.Namespace).Get(gatewayID)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			return nil
		}
		logrus.Errorf("get ingress failed, error: %s", err.Error())
		return fmt.Errorf("获取虚拟机 %s 网关失败！", vmID)
	}

	err = retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		ing.Labels["creator"] = ""
		_, err = s.kubeClient.NetworkingV1().Ingresses(tenantEnv.Namespace).Update(context.Background(), ing, metav1.UpdateOptions{})
		if err != nil {
			latest, err := s.kubeClient.NetworkingV1().Ingresses(tenantEnv.TenantName).Get(context.Background(), gatewayID, metav1.GetOptions{})
			if err != nil {
				return err
			}
			ing.SetResourceVersion(latest.ResourceVersion)
		}
		return err
	})

	return nil
}

func (s *ServiceAction) GetVMPorts(tenantEnv *dbmodel.TenantEnvs, vmID string) (*api_model.GetVMPortsResponse, error) {
	var result = new(api_model.GetVMPortsResponse)
	svcList, err := kube.GetCachedResources(s.kubeClient).ServiceLister.Services(tenantEnv.Namespace).List(labels.SelectorFromSet(labels.Set{
		"wutong.io/vm-id": vmID,
	}))
	if err != nil {
		logrus.Errorf("list service failed, error: %s", err.Error())
		return nil, fmt.Errorf("获取虚拟机 %s 端口列表失败！", vmID)
	}

	for _, svc := range svcList {
		protocol := svc.Labels["wutong.io/vm-port-protocol"]
		portNumber := cast.ToInt(svc.Labels["wutong.io/vm-port"])
		if slices.Contains(api_model.VMPortProtocols, api_model.VMPortProtocol(protocol)) && portNumber > 0 {
			vmPort := api_model.VMPort{
				VMPort:       cast.ToInt(svc.Labels["wutong.io/vm-port"]),
				Protocol:     api_model.VMPortProtocol(protocol),
				InnerService: fmt.Sprintf("%s.%s", svc.Name, svc.Namespace),
			}
			vmPort.GatewayEnabled = svc.Labels["wutong.io/vm-port-enabled"] == "true"
			ings, _ := kube.GetCachedResources(s.kubeClient).IngressV1Lister.Ingresses(tenantEnv.Namespace).List(labels.SelectorFromSet(labels.Set{
				"wutong.io/vm-id":            vmID,
				"wutong.io/vm-port":          fmt.Sprintf("%d", vmPort.VMPort),
				"wutong.io/vm-port-protocol": protocol,
			}))
			if len(ings) > 0 {
				for _, ing := range ings {
					vpg := api_model.VMPortGateway{
						GatewayID: ing.Name,
					}
					if protocol == api_model.VMPortProtocolHTTP {
						if len(ing.Spec.Rules) > 0 {
							vpg.GatewayHost = ing.Spec.Rules[0].Host
							if ing.Spec.Rules[0].HTTP != nil && len(ing.Spec.Rules[0].HTTP.Paths) > 0 {
								vpg.GatewayPath = ing.Spec.Rules[0].HTTP.Paths[0].Path
							}
						}
					} else {
						vpg.GatewayIP = ing.Annotations["nginx.ingress.kubernetes.io/l4-host"]
						vpg.GatewayPort = cast.ToInt(ing.Annotations["nginx.ingress.kubernetes.io/l4-port"])
					}

					vmPort.Gateways = append(vmPort.Gateways, vpg)
				}
			}
			result.Ports = append(result.Ports, vmPort)
		}
	}

	slices.SortFunc(result.Ports, func(i, j api_model.VMPort) int {
		if i.VMPort > j.VMPort {
			return 1
		} else if i.VMPort < j.VMPort {
			return -1
		}
		return 0
	})

	result.Total = len(result.Ports)
	return result, nil
}

func (s *ServiceAction) CreateVMPortGateway(tenantEnv *dbmodel.TenantEnvs, vmID string, req *api_model.CreateVMPortGatewayRequest) error {
	svcName := serviceName(vmID, req.VMPort, req.Protocol)

	svc, err := s.kubeClient.CoreV1().Services(tenantEnv.Namespace).Get(context.Background(), svcName, metav1.GetOptions{})
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			return fmt.Errorf("虚拟机 %s 端口 %d(%s) 不存在！", vmID, req.VMPort, req.Protocol)
		}
		logrus.Errorf("get service failed, error: %s", err.Error())
		return fmt.Errorf("获取虚拟机 %s 端口 %d(%s) 失败！", vmID, req.VMPort, req.Protocol)
	}

	protocol, ok := svc.Labels["wutong.io/vm-port-protocol"]
	if !ok {
		return fmt.Errorf("虚拟机 %s 端口 %d 协议未知！", vmID, req.VMPort)
	}

	wutongLabels := labelsFromTenantEnv(tenantEnv)
	wutongLabels = labels.Merge(wutongLabels, map[string]string{
		"wutong.io/vm-id":            vmID,
		"wutong.io/vm-port":          fmt.Sprintf("%d", req.VMPort),
		"wutong.io/vm-port-protocol": protocol,
	})

	if svc.Labels["wutong.io/vm-port-enabled"] != "true" {
		// 如果端口未开启网关，那么创建的网关应该去除 creator=Wutong 标签，让 wt-gateway 失去识别
		wutongLabels["creator"] = ""
	}

	gatewayID := util.NewUUID() // 生成网关 Ingres 名称, 作为唯一标识

	ing := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      gatewayID,
			Namespace: tenantEnv.Namespace,
			Labels:    wutongLabels,
		},
		Spec: networkingv1.IngressSpec{
			DefaultBackend: &networkingv1.IngressBackend{
				Service: &networkingv1.IngressServiceBackend{
					Name: svcName,
					Port: networkingv1.ServiceBackendPort{
						Number: int32(req.VMPort),
					},
				},
			},
		},
	}

	if protocol == api_model.VMPortProtocolHTTP {
		// http mode ingerss
		if req.GatewayPath == "" {
			req.GatewayPath = "/"
		}

		if req.GatewayHost == "" {
			req.GatewayHost = generateGatewayHost(tenantEnv.Namespace, vmID, req.VMPort)
		}

		// 验证是否已被占用
		h, _ := db.GetManager().HTTPRuleDao().GetHTTPRuleByDomainAndHost(req.GatewayHost, req.GatewayPath)
		if len(h) > 0 && h[0].UUID != gatewayID {
			return fmt.Errorf("网关域名 %s%s 已被占用！", req.GatewayHost, req.GatewayPath)
		}

		ing.Spec.Rules = []networkingv1.IngressRule{
			{
				Host: req.GatewayHost,
				IngressRuleValue: networkingv1.IngressRuleValue{
					HTTP: &networkingv1.HTTPIngressRuleValue{
						Paths: []networkingv1.HTTPIngressPath{
							{
								Path:     req.GatewayPath,
								PathType: util.Ptr(networkingv1.PathTypePrefix),
								Backend: networkingv1.IngressBackend{
									Service: &networkingv1.IngressServiceBackend{
										Name: svcName,
										Port: networkingv1.ServiceBackendPort{
											Number: int32(req.VMPort),
										},
									},
								},
							},
						},
					},
				},
			},
		}
	} else {
		// tcp mode ingress
		if req.GatewayIP == "" {
			req.GatewayIP = "0.0.0.0"
		}
		if req.GatewayPort > 0 {
			if !GetGatewayHandler().IsPortAvailable(req.GatewayIP, req.GatewayPort) {
				return fmt.Errorf("网关端口 %d 被占用或超出范围！", req.GatewayPort)
			}
		} else {
			avaliablePort, err := GetGatewayHandler().GetAvailablePort(req.GatewayIP, false)
			if err != nil {
				return fmt.Errorf("获取可用端口失败！")
			}
			req.GatewayPort = avaliablePort
		}
		ing.Annotations = map[string]string{
			"nginx.ingress.kubernetes.io/l4-enable": "true",
			"nginx.ingress.kubernetes.io/l4-host":   req.GatewayIP,
			"nginx.ingress.kubernetes.io/l4-port":   cast.ToString(req.GatewayPort),
		}
	}

	_, err = s.kubeClient.NetworkingV1().Ingresses(ing.Namespace).Create(context.Background(), ing, metav1.CreateOptions{})
	if err != nil {
		if k8sErrors.IsAlreadyExists(err) {
			return fmt.Errorf("虚拟机 %s 端口 %d(%s) 对外网关已存在！", vmID, req.VMPort, req.Protocol)
		}
		logrus.Errorf("create ingress failed, error: %s", err.Error())
		return fmt.Errorf("创建虚拟机 %s 网关失败！", vmID)
	}

	if protocol == api_model.VMPortProtocolHTTP {
		// register http domain and host
		err := db.GetManager().HTTPRuleDao().CreateOrUpdateHTTPRuleInBatch([]*dbmodel.HTTPRule{
			{
				UUID:          gatewayID,
				VMID:          vmID,
				ContainerPort: req.VMPort,
				Domain:        req.GatewayHost,
				Path:          req.GatewayPath,
			},
		})
		if err != nil {
			s.kubeClient.NetworkingV1().Ingresses(ing.Namespace).Delete(context.Background(), ing.Name, metav1.DeleteOptions{})
			logrus.Errorf("register tcp port failed, error: %s", err.Error())
			return fmt.Errorf("虚拟机注册 %s TCP 网关 %d -> %s:%d 失败！", vmID, req.VMPort, req.GatewayIP, req.GatewayPort)
		}
	} else {
		// register tcp port
		err := db.GetManager().TCPRuleDao().CreateOrUpdateTCPRuleInBatch([]*dbmodel.TCPRule{
			{
				UUID:          gatewayID,
				VMID:          vmID,
				ContainerPort: req.VMPort,
				IP:            req.GatewayIP,
				Port:          req.GatewayPort,
			},
		})
		if err != nil {
			s.kubeClient.NetworkingV1().Ingresses(ing.Namespace).Delete(context.Background(), ing.Name, metav1.DeleteOptions{})
			logrus.Errorf("register tcp port failed, error: %s", err.Error())
			return fmt.Errorf("虚拟机 %s 注册 HTTP 网关 %d -> %s%s 失败！", vmID, req.VMPort, req.GatewayHost, req.GatewayPath)
		}
	}

	return nil
}

func (s *ServiceAction) UpdateVMPortGateway(tenantEnv *dbmodel.TenantEnvs, vmID, gatewayID string, req *api_model.UpdateVMPortGatewayRequest) error {
	ingBeforUpdate, err := kube.GetCachedResources(s.kubeClient).IngressV1Lister.Ingresses(tenantEnv.Namespace).Get(gatewayID)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			return fmt.Errorf("虚拟机 %s 网关不存在！", vmID)
		}
		logrus.Errorf("get ingress failed, error: %s", err.Error())
		return fmt.Errorf("获取虚拟机 %s 网关失败！", vmID)
	}

	port := cast.ToInt(ingBeforUpdate.Labels["wutong.io/vm-port"])
	if port == 0 {
		return fmt.Errorf("虚拟机 %s 端口号未知！", vmID)
	}

	protocol := ingBeforUpdate.Labels["wutong.io/vm-port-protocol"]
	if !slices.Contains(api_model.VMPortProtocols, api_model.VMPortProtocol(protocol)) {
		return fmt.Errorf("虚拟机 %s 端口协议未知！", vmID)
	}

	svcName := serviceName(vmID, port, protocol)

	ingToUpdate := ingBeforUpdate.DeepCopy()

	if protocol == api_model.VMPortProtocolHTTP {
		// http mode ingerss
		if req.GatewayPath == "" {
			req.GatewayPath = "/"
		}
		// 更新时 Host 不允许为空
		if req.GatewayHost == "" {
			return fmt.Errorf("虚拟机 %s 端口协议为 %s 时，更新网关域名不能为空！", vmID, protocol)
		} else {
			// 验证是否已被占用
			h, _ := db.GetManager().HTTPRuleDao().GetHTTPRuleByDomainAndHost(req.GatewayHost, req.GatewayPath)
			if len(h) > 0 && h[0].UUID != gatewayID {
				return fmt.Errorf("网关域名 %s%s 已被占用！", req.GatewayHost, req.GatewayPath)
			}
		}
		ingToUpdate.Spec.Rules = []networkingv1.IngressRule{
			{
				Host: req.GatewayHost,
				IngressRuleValue: networkingv1.IngressRuleValue{
					HTTP: &networkingv1.HTTPIngressRuleValue{
						Paths: []networkingv1.HTTPIngressPath{
							{
								Path:     req.GatewayPath,
								PathType: util.Ptr(networkingv1.PathTypePrefix),
								Backend: networkingv1.IngressBackend{
									Service: &networkingv1.IngressServiceBackend{
										Name: svcName,
										Port: networkingv1.ServiceBackendPort{
											Number: int32(port),
										},
									},
								},
							},
						},
					},
				},
			},
		}
	} else {
		// tcp mode ingress
		if req.GatewayIP == "" {
			req.GatewayIP = "0.0.0.0"
		}
		// 更新时 GatewayPort 不允许为空
		if req.GatewayPort == 0 {
			return fmt.Errorf("虚拟机 %s 端口协议为 %s 时，更新网关对外端口不能为空！", vmID, protocol)
		}
		if req.GatewayPort > 0 {
			if !GetGatewayHandler().IsPortAvailable(req.GatewayIP, req.GatewayPort) {
				return fmt.Errorf("网关端口 %d 被占用或超出范围！", req.GatewayPort)
			}
		}
		ingToUpdate.Annotations = map[string]string{
			"nginx.ingress.kubernetes.io/l4-host": req.GatewayIP,
			"nginx.ingress.kubernetes.io/l4-port": cast.ToString(req.GatewayPort),
		}
	}

	var updateVMPortGatewayFunc = func(ing *networkingv1.Ingress) error {
		return retry.RetryOnConflict(retry.DefaultBackoff, func() error {
			_, err = s.kubeClient.NetworkingV1().Ingresses(ing.Namespace).Update(context.Background(), ing, metav1.UpdateOptions{})
			if err != nil {
				latest, err := s.kubeClient.NetworkingV1().Ingresses(ing.Namespace).Get(context.Background(), gatewayID, metav1.GetOptions{})
				if err != nil {
					return err
				}
				ing.SetResourceVersion(latest.ResourceVersion)
			}
			return err
		})
	}

	err = updateVMPortGatewayFunc(ingToUpdate)
	if err != nil {
		logrus.Errorf("update ingress failed, error: %s", err.Error())
		return fmt.Errorf("更新虚拟机 %s 网关 %d -> %s%s 失败！", vmID, port, req.GatewayHost, req.GatewayPath)
	}

	if protocol == api_model.VMPortProtocolHTTP {
		// update registered http domain and host
		httpRule, err := db.GetManager().HTTPRuleDao().GetHTTPRuleByID(gatewayID)
		if err != nil {
			logrus.Errorf("get tcp rule failed, error: %s", err.Error())
			return fmt.Errorf("获取虚拟机 %s 注册 HTTP 网关 %d -> %s%s 失败！", vmID, port, req.GatewayHost, req.GatewayPath)
		}

		if httpRule == nil {
			httpRule = &dbmodel.HTTPRule{
				UUID: gatewayID,
			}
		}
		httpRule.VMID = vmID
		httpRule.Domain = req.GatewayHost
		httpRule.Path = req.GatewayPath
		err = db.GetManager().HTTPRuleDao().CreateOrUpdateHTTPRuleInBatch([]*dbmodel.HTTPRule{httpRule})
		if err != nil {
			logrus.Errorf("register http domain and path failed, error: %s", err.Error())
			updateVMPortGatewayFunc(ingBeforUpdate)
			return fmt.Errorf("虚拟机 %s 注册 HTTP 网关 %d -> %s%s 失败！", vmID, port, req.GatewayHost, req.GatewayPath)
		}
	} else {
		// update registered tcp port
		tcpRule, err := db.GetManager().TCPRuleDao().GetTCPRuleByID(gatewayID)
		if err != nil {
			logrus.Errorf("get tcp rule failed, error: %s", err.Error())
			return fmt.Errorf("获取虚拟机 %s 注册 TCP 网关 %d -> %s:%d 失败！", vmID, port, req.GatewayIP, req.GatewayPort)
		}

		if tcpRule == nil {
			tcpRule = &dbmodel.TCPRule{
				UUID: gatewayID,
			}
		}
		tcpRule.VMID = vmID
		tcpRule.ContainerPort = port
		tcpRule.IP = req.GatewayIP
		tcpRule.Port = req.GatewayPort
		err = db.GetManager().TCPRuleDao().CreateOrUpdateTCPRuleInBatch([]*dbmodel.TCPRule{tcpRule})
		if err != nil {
			logrus.Errorf("register tcp port failed, error: %s", err.Error())
			updateVMPortGatewayFunc(ingBeforUpdate)
			return fmt.Errorf("注册虚拟机 %s TCP 网关 %d -> %s:%d 失败！", vmID, port, req.GatewayIP, req.GatewayPort)
		}
	}

	return nil
}

func (s *ServiceAction) DeleteVMPortGateway(tenantEnv *dbmodel.TenantEnvs, vmID, gatewayID string) error {
	ing, err := s.kubeClient.NetworkingV1().Ingresses(tenantEnv.Namespace).Get(context.Background(), gatewayID, metav1.GetOptions{})
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			return nil
		}
		logrus.Errorf("get ingress failed, error: %s", err.Error())
		return fmt.Errorf("获取虚拟机 %s 网关失败！", vmID)
	}

	protocol := ing.Labels["wutong.io/vm-port-protocol"]
	if !slices.Contains(api_model.VMPortProtocols, api_model.VMPortProtocol(protocol)) {
		return fmt.Errorf("虚拟机 %s 端口协议未知！", vmID)
	}

	if protocol == api_model.VMPortProtocolHTTP {
		err = db.GetManager().HTTPRuleDao().DeleteHTTPRuleByID(gatewayID)
		if err != nil {
			logrus.Errorf("delete http rule failed, error: %s", err.Error())
			// return fmt.Errorf("删除虚拟机 %s HTTP 网关失败！", vmID)
		}
	} else {
		err = db.GetManager().TCPRuleDao().DeleteByID(gatewayID)
		if err != nil {
			logrus.Errorf("delete tcp rule failed, error: %s", err.Error())
			// return fmt.Errorf("删除虚拟机 %s TCP 网关失败！", vmID)
		}
	}

	err = s.kubeClient.NetworkingV1().Ingresses(tenantEnv.Namespace).Delete(context.Background(), gatewayID, metav1.DeleteOptions{})
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			return nil
		}
		logrus.Errorf("delete ingress failed, error: %s", err.Error())
		return fmt.Errorf("删除虚拟机 %s 网关失败！", vmID)
	}

	return nil
}

func (s *ServiceAction) DeleteVMPort(tenantEnv *dbmodel.TenantEnvs, vmID string, req *api_model.DeleteVMPortRequest) error {
	// 1、删除虚拟机端口服务下所有已开通的网关
	gateways, err := kube.GetCachedResources(s.kubeClient).IngressV1Lister.Ingresses(tenantEnv.Namespace).List(labels.SelectorFromSet(labels.Set{
		"wutong.io/vm-id":            vmID,
		"wutong.io/vm-port":          fmt.Sprintf("%d", req.VMPort),
		"wutong.io/vm-port-protocol": req.Protocol,
	}))
	if err != nil {
		logrus.Errorf("list ingress failed, error: %s", err.Error())
		return fmt.Errorf("删除虚拟机 %s 端口 %d(%s) 下网关失败！", vmID, req.VMPort, req.Protocol)
	}
	for _, gateway := range gateways {
		err = s.DeleteVMPortGateway(tenantEnv, vmID, gateway.Name)
		if err != nil {
			return fmt.Errorf("删除虚拟机 %s 端口 %d(%s) 下网关失败！", vmID, req.VMPort, req.Protocol)
		}
	}

	// 2、删除虚拟机端口服务
	svcName := serviceName(vmID, req.VMPort, req.Protocol)
	err = s.kubeClient.CoreV1().Services(tenantEnv.Namespace).Delete(context.Background(), svcName, metav1.DeleteOptions{})
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			return nil
		}
		logrus.Errorf("delete service failed, error: %s", err.Error())
		return fmt.Errorf("删除虚拟机 %s 端口 %d(%s) 失败！", vmID, req.VMPort, req.Protocol)
	}

	return nil
}

func (s *ServiceAction) DeleteVM(tenantEnv *dbmodel.TenantEnvs, vmID string) error {
	vm, err := kube.GetKubeVirtVM(s.dynamicClient, tenantEnv.Namespace, vmID)
	if err != nil && !k8sErrors.IsNotFound(err) {
		logrus.Errorf("get vm failed, error: %s", err.Error())
		return fmt.Errorf("获取虚拟机 %s 失败！", vmID)
	}

	// 0、关闭虚拟机，如果虚拟机还处于运行状态
	if _, err := s.StopVM(tenantEnv, vmID); err != nil {
		logrus.Errorf("stop vm failed, error: %s", err.Error())
		return fmt.Errorf("关闭虚拟机 %s 失败！", vmID)
	}

	// 1、删除虚拟机端口服务下所有已开通的网关
	services, err := kube.GetCachedResources(s.kubeClient).ServiceLister.Services(tenantEnv.Namespace).List(labels.SelectorFromSet(labels.Set{
		"wutong.io/vm-id": vmID,
	}))
	if err != nil {
		logrus.Errorf("list service failed, error: %s", err.Error())
		return fmt.Errorf("删除虚拟机 %s 下端口服务失败！", vmID)
	}
	for _, svc := range services {
		err = s.DeleteVMPort(tenantEnv, vmID, &api_model.DeleteVMPortRequest{
			VMPort:   cast.ToInt(svc.Labels["wutong.io/vm-port"]),
			Protocol: api_model.VMPortProtocol(svc.Labels["wutong.io/vm-port-protocol"]),
		})
		if err != nil {
			return fmt.Errorf("删除虚拟机 %s 下端口服务 %s 失败！", vmID, svc.Name)
		}
	}

	// 2、删除虚拟机存储卷
	pvcs, err := kube.GetCachedResources(s.kubeClient).PersistentVolumeClaimLister.PersistentVolumeClaims(tenantEnv.Namespace).List(labels.SelectorFromSet(labels.Set{
		"wutong.io/vm-id": vmID,
	}))
	if err != nil {
		logrus.Errorf("list pvc failed, error: %s", err.Error())
		return fmt.Errorf("删除虚拟机 %s 下存储卷失败！", vmID)
	}
	for _, pvc := range pvcs {
		err = s.DeleteVMVolume(tenantEnv, vmID, pvc.Labels["wutong.io/vm-volume"])
		if err != nil {
			return fmt.Errorf("删除虚拟机 %s 下存储卷 %s 失败！", vmID, pvc.Labels["wutong.io/vm-volume"])
		}
	}

	// 3、删除虚拟机
	err = kube.DeleteKubeVirtVM(s.dynamicClient, tenantEnv.Namespace, vmID)
	if err != nil {
		logrus.Errorf("delete vm failed, error: %s", err.Error())
		return fmt.Errorf("删除虚拟机 %s 失败！", vmID)
	}

	// 4、如果使用 .iso 系统源创建的虚拟机，需要额外回收数据盘 pvc
	if vm.Labels["wutong.io/vm-os-source-from"] == api_model.OSSourceFromHTTP && strings.HasSuffix(vm.Labels["wutong.io/vm-os-source-url"], ".iso") {
		dataPVCName := pvcName(vmID, "data")
		err = s.kubeClient.CoreV1().PersistentVolumeClaims(vm.Namespace).Delete(context.Background(), dataPVCName, metav1.DeleteOptions{})
		if err != nil {
			if k8sErrors.IsNotFound(err) {
				return nil
			}
			logrus.Errorf("delete vm data pvc failed, error: %s", err.Error())
			return fmt.Errorf("删除虚拟机 %s 数据盘失败！", vmID)
		}
	}

	return nil
}

func (s *ServiceAction) ListVMs(tenantEnv *dbmodel.TenantEnvs) (*api_model.ListVMsResponse, error) {
	vms, err := kube.ListKubeVirtVMs(s.dynamicClient, tenantEnv.Namespace)
	if err != nil {
		logrus.Errorf("list vm failed, error: %s", err.Error())
		return nil, errors.New("获取虚拟机列表失败！")
	}

	sort.Slice(vms, func(i, j int) bool {
		return vms[i].CreationTimestamp.After(vms[j].CreationTimestamp.Time)
	})

	var result = new(api_model.ListVMsResponse)
	for _, vm := range vms {
		vp := vmProfileFromKubeVirtVM(vm, nil)
		result.VMs = append(result.VMs, vp)
	}
	result.Total = len(result.VMs)

	return result, nil
}

func (s *ServiceAction) ListVMVolumes(tenantEnv *dbmodel.TenantEnvs, vmID string) (*api_model.ListVMVolumesResponse, error) {
	pvcs, err := kube.GetCachedResources(s.kubeClient).PersistentVolumeClaimLister.PersistentVolumeClaims(tenantEnv.Namespace).List(labels.SelectorFromSet(labels.Set{
		"wutong.io/vm-id": vmID,
	}))
	if err != nil {
		logrus.Errorf("list pvc failed, error: %s", err.Error())
		return nil, fmt.Errorf("获取虚拟机 %s 存储卷列表失败！", vmID)
	}

	sort.Slice(pvcs, func(i, j int) bool {
		return pvcs[i].CreationTimestamp.After(pvcs[j].CreationTimestamp.Time)
	})

	podPvcVolumes := vmPodVolumes(vmPod(s.kubeClient, tenantEnv, vmID))

	volumes := make([]api_model.VMVolume, 0)
	for _, pvc := range pvcs {
		if pvc.Labels["wutong.io/vm-volume"] == "" {
			continue
		}
		volumes = append(volumes, api_model.VMVolume{
			VolumeName: pvc.Labels["wutong.io/vm-volume"],
			VolumeSize: cast.ToInt64(pvc.Annotations["wutong.io/vm-volume-size"]),
			StorageClass: func() string {
				if pvc.Spec.StorageClassName != nil {
					return *pvc.Spec.StorageClassName
				}
				return ""
			}(),
			Status: volumeStatus(podPvcVolumes, pvc),
		})
	}
	return &api_model.ListVMVolumesResponse{
		VMVolumes: volumes,
		Total:     len(volumes),
	}, nil
}

func (s *ServiceAction) AddVMVolume(tenantEnv *dbmodel.TenantEnvs, vmID string, req *api_model.AddVMVolumeRequest) error {
	vm, err := kube.GetKubeVirtVM(s.dynamicClient, tenantEnv.Namespace, vmID)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			return fmt.Errorf("虚拟机 %s 不存在！", vmID)
		}
		logrus.Errorf("get vm failed, error: %s", err.Error())
		return fmt.Errorf("获取虚拟机 %s 失败！", vmID)
	}

	pvcName := pvcName(vmID, req.VolumeName)

	wutongLabels := labelsFromTenantEnv(tenantEnv)
	wutongLabels = labels.Merge(wutongLabels, map[string]string{
		"wutong.io/vm-id":     vmID,
		"wutong.io/vm-volume": req.VolumeName,
	})

	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pvcName,
			Namespace: tenantEnv.Namespace,
			Labels:    wutongLabels,
			Annotations: map[string]string{
				"wutong.io/vm-volume-size": fmt.Sprintf("%d", req.VolumeSize),
			},
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
			},
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse(fmt.Sprintf("%dGi", req.VolumeSize)),
				},
			},
			StorageClassName: util.Ptr(req.StorageClass),
			VolumeMode:       util.Ptr(corev1.PersistentVolumeFilesystem),
		},
	}

	_, err = s.kubeClient.CoreV1().PersistentVolumeClaims(tenantEnv.Namespace).Create(context.Background(), pvc, metav1.CreateOptions{})
	if err != nil {
		if k8sErrors.IsAlreadyExists(err) {
			return fmt.Errorf("虚拟机 %s 存储卷名称 %s 已存在！", vmID, req.VolumeName)
		}
		logrus.Errorf("create pvc failed, error: %s", err.Error())
		return fmt.Errorf("创建虚拟机 %s 存储卷 %s 失败！", vmID, req.VolumeName)
	}

	for _, fs := range vm.Spec.Template.Spec.Domain.Devices.Filesystems {
		if fs.Name == req.VolumeName {
			return fmt.Errorf("虚拟机 %s 存储卷名称 %s 已存在！", vmID, req.VolumeName)
		}
	}

	for _, volume := range vm.Spec.Template.Spec.Volumes {
		if volume.Name == req.VolumeName {
			return fmt.Errorf("虚拟机 %s 存储卷名称 %s 已存在！", vmID, req.VolumeName)
		}
	}

	vm.Spec.Template.Spec.Domain.Devices.Filesystems = append(vm.Spec.Template.Spec.Domain.Devices.Filesystems, kubevirtcorev1.Filesystem{
		Name:     req.VolumeName,
		Virtiofs: &kubevirtcorev1.FilesystemVirtiofs{},
	})

	vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, kubevirtcorev1.Volume{
		Name: req.VolumeName,
		VolumeSource: kubevirtcorev1.VolumeSource{
			PersistentVolumeClaim: &kubevirtcorev1.PersistentVolumeClaimVolumeSource{
				PersistentVolumeClaimVolumeSource: corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: pvcName,
				},
			},
		},
	})

	_, err = kube.UpdateKubeVirtVM(s.dynamicClient, vm)
	if err != nil {
		logrus.Errorf("add vm volume failed, error: %s", err.Error())
		return fmt.Errorf("虚拟机 %s 添加存储卷 %s 失败！", vmID, req.VolumeName)
	}

	return nil
}

func (s *ServiceAction) DeleteVMVolume(tenantEnv *dbmodel.TenantEnvs, vmID, volumeName string) error {
	vm, err := kube.GetKubeVirtVM(s.dynamicClient, tenantEnv.Namespace, vmID)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			return fmt.Errorf("虚拟机 %s 不存在！", vmID)
		}
		logrus.Errorf("get vm failed, error: %s", err.Error())
		return fmt.Errorf("获取虚拟机 %s 失败！", vmID)
	}

	if vm.Spec.Running != nil && *vm.Spec.Running {
		return fmt.Errorf("虚拟机 %s 正在运行，无法删除存储卷！", vmID)
	}

	for idx, fs := range vm.Spec.Template.Spec.Domain.Devices.Filesystems {
		if fs.Name == volumeName {
			vm.Spec.Template.Spec.Domain.Devices.Filesystems = append(vm.Spec.Template.Spec.Domain.Devices.Filesystems[:idx], vm.Spec.Template.Spec.Domain.Devices.Filesystems[idx+1:]...)
			break
		}
	}

	for idx, volume := range vm.Spec.Template.Spec.Volumes {
		if volume.Name == volumeName {
			vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes[:idx], vm.Spec.Template.Spec.Volumes[idx+1:]...)
			break
		}
	}

	_, err = kube.UpdateKubeVirtVM(s.dynamicClient, vm)
	if err != nil {
		logrus.Errorf("delete vm volume failed, error: %s", err.Error())
		return fmt.Errorf("虚拟机 %s 删除存储卷 %s 失败！", vmID, volumeName)
	}

	pvcName := pvcName(vmID, volumeName)
	err = s.kubeClient.CoreV1().PersistentVolumeClaims(tenantEnv.Namespace).Delete(context.Background(), pvcName, metav1.DeleteOptions{})
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			return nil
		}
		logrus.Errorf("delete pvc failed, error: %s", err.Error())
		return fmt.Errorf("删除虚拟机 %s 存储卷 %s 失败！", vmID, volumeName)
	}

	return nil
}

// RemoveBootDisk 取消虚拟机启动盘设置
// 并未直接删除，因为可能后续需要依赖该启动盘重装系统等操作
func (s *ServiceAction) RemoveBootDisk(tenantEnv *dbmodel.TenantEnvs, vmID string) error {
	vm, err := kube.GetKubeVirtVM(s.dynamicClient, tenantEnv.Namespace, vmID)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			return fmt.Errorf("虚拟机 %s 不存在！", vmID)
		}
		logrus.Errorf("get vm failed, error: %s", err.Error())
		return fmt.Errorf("获取虚拟机 %s 失败！", vmID)
	}

	for i := range vm.Spec.Template.Spec.Domain.Devices.Disks {
		// 取消 bootdisk disk 启动顺序设置
		if vm.Spec.Template.Spec.Domain.Devices.Disks[i].Name == bootDiskName {
			vm.Spec.Template.Spec.Domain.Devices.Disks[i].BootOrder = nil
		}

		// 设置 containerdisk 为启动盘
		if vm.Spec.Template.Spec.Domain.Devices.Disks[i].Name == containerDiskName &&
			vm.Spec.Template.Spec.Domain.Devices.Disks[i].BootOrder != nil &&
			*vm.Spec.Template.Spec.Domain.Devices.Disks[i].BootOrder == 2 {
			vm.Spec.Template.Spec.Domain.Devices.Disks[i].BootOrder = util.Ptr[uint](1)
		}
	}

	// 更新虚拟机
	_, err = kube.UpdateKubeVirtVM(s.dynamicClient, vm)
	if err != nil {
		logrus.Errorf("update vm failed, error: %s", err.Error())
		return fmt.Errorf("更新虚拟机 %s 启动顺序失败！", vmID)
	}

	// 重启
	_, err = s.RestartVM(tenantEnv, vmID)
	if err != nil {
		logrus.Errorf("restart vm failed, error: %s", err.Error())
		return fmt.Errorf("重启虚拟机 %s 失败！", vmID)
	}
	return nil
}

// -------------------------- 私有函数 ------------------------------------

// buildVMBase 构建虚拟机基础结构实例
func buildVMBase(req *api_model.CreateVMRequest, namespace string, labels map[string]string) *kubevirtcorev1.VirtualMachine {
	var nodeSelector = map[string]string{
		"wutong.io/vm-schedulable": "true",
	}

	for _, labelKey := range req.NodeSelectorLabels {
		var labelVal string
		kv := strings.Split(labelKey, "=")
		if len(kv) > 1 {
			labelVal = kv[1]
		}

		nodeSelector["vm-scheduling-label.wutong.io/"+kv[0]] = labelVal
	}

	vm := &kubevirtcorev1.VirtualMachine{
		TypeMeta: metav1.TypeMeta{
			Kind:       "VirtualMachine",
			APIVersion: kubevirtcorev1.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      req.Name,
			Namespace: namespace,
			Labels:    labels,
			Annotations: map[string]string{
				"wutong.io/display-name":                req.DisplayName,
				"wutong.io/desc":                        req.Desc,
				"wutong.io/creator":                     req.Operator,
				"wutong.io/last-modifier":               req.Operator,
				"wutong.io/vm-disk-size":                fmt.Sprintf("%d", req.OSDiskSize),
				"wutong.io/vm-request-cpu":              fmt.Sprintf("%d", req.RequestCPU),
				"wutong.io/vm-request-memory":           fmt.Sprintf("%d", req.RequestMemory),
				"wutong.io/vm-os-name":                  req.OSName,
				"wutong.io/vm-os-version":               req.OSVersion,
				"wutong.io/vm-os-source-from":           req.OSSourceFrom,
				"wutong.io/vm-os-source-url":            req.OSSourceURL,
				"wutong.io/vm-default-login-user":       req.User,
				"wutong.io/last-modification-timestamp": metav1.Now().UTC().Format(time.RFC3339),
			},
		},
		Spec: kubevirtcorev1.VirtualMachineSpec{
			Running: util.Ptr(req.Running),
			Template: &kubevirtcorev1.VirtualMachineInstanceTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: kubevirtcorev1.VirtualMachineInstanceSpec{
					Domain: kubevirtcorev1.DomainSpec{
						Clock: &kubevirtcorev1.Clock{
							ClockOffset: kubevirtcorev1.ClockOffset{
								Timezone: util.Ptr(kubevirtcorev1.ClockOffsetTimezone("Asia/Shanghai")), // default timezone
							},
						},
						Devices: kubevirtcorev1.Devices{
							Interfaces: []kubevirtcorev1.Interface{
								{
									Name: "default",
									InterfaceBindingMethod: kubevirtcorev1.InterfaceBindingMethod{
										Masquerade: &kubevirtcorev1.InterfaceMasquerade{},
									},
									MacAddress: util.GenerateMACAddress(), // 自动生成 mac 地址，避免冲突导致网络异常
								},
							},
						},
						Memory: &kubevirtcorev1.Memory{
							Guest: util.Ptr(resource.MustParse(fmt.Sprintf("%dGi", req.RequestMemory))),
						},
						Resources: kubevirtcorev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse(fmt.Sprintf("%dm", req.RequestCPU)),
								corev1.ResourceMemory: resource.MustParse(fmt.Sprintf("%dGi", req.RequestMemory)),
							},
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse(fmt.Sprintf("%dm", req.RequestCPU)),
								corev1.ResourceMemory: resource.MustParse(fmt.Sprintf("%dGi", req.RequestMemory)),
							},
						},
					},
					NodeSelector: nodeSelector,
					Networks: []kubevirtcorev1.Network{
						{
							Name: "default",
							NetworkSource: kubevirtcorev1.NetworkSource{
								Pod: &kubevirtcorev1.PodNetwork{},
							},
						},
					},
				},
			},
		},
	}

	return vm
}

// createContainerDiskPVC 创建虚拟机数据盘 PVC
func createContainerDiskPVC(req *api_model.CreateVMRequest, tenantEnv *dbmodel.TenantEnvs, wutongLabels map[string]string, s *ServiceAction) error {
	pvc := corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pvcName(req.Name, "data"),
			Namespace: tenantEnv.Namespace,
			Labels:    wutongLabels,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
			},
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: *resource.NewQuantity(req.OSDiskSize*1024*1024*1024, resource.BinarySI),
				},
			},
		},
	}

	_, err := s.kubeClient.CoreV1().PersistentVolumeClaims(tenantEnv.Namespace).Create(context.Background(), &pvc, metav1.CreateOptions{})
	if err != nil {
		logrus.Errorf("create pvc failed, error: %s", err.Error())
		return fmt.Errorf("创建虚拟机 %s 安装磁盘失败！", req.Name)
	}
	return nil
}

// buildVMDataVolumeTemplates 构建虚拟机数据盘模板
func buildVMDataVolumeTemplates(name string, sourceFrom api_model.OSSourceFrom, sourceUrl string, size int64) []kubevirtcorev1.DataVolumeTemplateSpec {
	var source *cdicorev1beta1.DataVolumeSource
	switch sourceFrom {
	case api_model.OSSourceFromHTTP:
		source = &cdicorev1beta1.DataVolumeSource{
			HTTP: &cdicorev1beta1.DataVolumeSourceHTTP{
				URL: sourceUrl,
			},
		}
	case api_model.OSSourceFromRegistry:
		sourceUrl = "docker://" + sourceUrl
		source = &cdicorev1beta1.DataVolumeSource{
			Registry: &cdicorev1beta1.DataVolumeSourceRegistry{
				URL: util.Ptr(sourceUrl),
			},
		}
	}
	result := []kubevirtcorev1.DataVolumeTemplateSpec{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
				Annotations: map[string]string{
					"cdi.kubevirt.io/storage.import.source":   sourceFrom,
					"cdi.kubevirt.io/storage.import.endpoint": sourceUrl,
				},
			},
			Spec: cdicorev1beta1.DataVolumeSpec{
				PVC: &corev1.PersistentVolumeClaimSpec{
					AccessModes: []corev1.PersistentVolumeAccessMode{
						corev1.ReadWriteOnce,
					},
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: *resource.NewQuantity(size*1024*1024*1024, resource.BinarySI),
						},
					},
				},
				Source: source,
			},
		},
	}
	return result
}

// buildVMUserData 构建虚拟机初始化用户数据，对支持 cloud-init 的虚拟机有效
// 1. 初始用户账号密码
// 2. 初始用户 ssh key
// 3. 默认安装 file-browser 服务，供用户管理虚拟机文件
func buildVMUserData(kubeClient kubernetes.Interface, username, password string) (string, error) {
	vmSSHPubKey, _ := kube.GetWTChannelSSHPubKey(kubeClient)
	vmUserData := fmt.Sprintf(`#cloud-config
disable_root: false
chpasswd:
  expire: False
groups:
  - %s-group
users:
  - default
  - name: %s
    lock_passwd: false
    gecos: %s
    primary-group: %s-group
    groups: [sudo]
    sudo: ["ALL=(ALL) NOPASSWD:ALL"]
    plain_text_passwd: %s
    ssh_authorized_keys:
      - %s
password: %s
ssh_pwauth: True
ssh_authorized_keys:
  - %s\n`, username, username, username, username, password, vmSSHPubKey, password, vmSSHPubKey)

	bcryptedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		logrus.Errorf("bcrypt password failed, error: %s", err.Error())
		return "", fmt.Errorf("虚拟机初始用户密码加密失败！")
	}

	vmUserData += fmt.Sprintf(`bootcmd:
  - sudo cp -r /home/%s/.ssh /root/
`, username)

	vmUserData += fmt.Sprintf(`runcmd:
  - sudo dhclient
%s`, filebrowserRunCmd(username, string(bcryptedPassword)))

	return vmUserData, nil
}

// filebrowserRunCmd 安装 file-browser 服务命令
func filebrowserRunCmd(username, bcryptedPassword string) string {
	// `sudo sh -c 'cat <<\EOF` cat 命令使用单引号， `\EOF` 前添加 \ 是为了防止文本转义，bcryptedPassword 中一般包含 $ 符号
	return fmt.Sprintf(`  - sudo wget -O /usr/local/bin/filebrowser https://wutong-paas.obs.cn-east-3.myhuaweicloud.com/linux/$(uname -m)/filebrowser
  - sudo chmod +x /usr/local/bin/filebrowser
  - |
    sudo sh -c 'cat <<\EOF > /etc/systemd/system/filebrowser.service
    [Unit]
    Description=FileBrowser Service
    After=network.target
    
    [Service]
    ExecStartPre=mkdir -p /filebrowser
    ExecStart=/usr/local/bin/filebrowser -a 0.0.0.0 -r /filebrowser -p 6173
    Restart=always
    User=root
    Group=root
    Environment=FB_USERNAME=%s
    Environment=FB_PASSWORD=%s
    
    [Install]
    WantedBy=multi-user.target
    EOF'
  - sudo systemctl daemon-reload
  - sudo systemctl enable filebrowser
  - sudo systemctl start filebrowser`, username, bcryptedPassword)
}

// labelsFromTenantEnv 从租户环境信息组成基础标签信息
func labelsFromTenantEnv(te *dbmodel.TenantEnvs) map[string]string {
	return map[string]string{
		"creator":         "Wutong",
		"tenant_env_id":   te.UUID,
		"tenant_env_name": te.Name,
		"tenant_name":     te.TenantName,
		"tenant_id":       te.TenantID,
	}
}

func serviceName(vmID string, port int, protocol string) string {
	return fmt.Sprintf("%s-%d-%s", vmID, port, protocol)
}

func pvcName(vmID, volumeName string) string {
	return fmt.Sprintf("%s-%s", vmID, volumeName)
}

func generateGatewayHost(namespace, vmID string, port int) string {
	exDomain := os.Getenv("EX_DOMAIN")
	if exDomain == "" {
		return ""
	}
	if strings.Contains(exDomain, ":") {
		exDomain = strings.Split(exDomain, ":")[0]
	}
	svcName := serviceName(vmID, port, api_model.VMPortProtocolHTTP)
	return fmt.Sprintf("%s.%s.%s", svcName, namespace, exDomain)
}

func portProtocol(p api_model.VMPortProtocol) corev1.Protocol {
	switch p {
	case api_model.VMPortProtocolUDP:
		return corev1.ProtocolUDP
	case api_model.VMPortProtocolSCTP:
		return corev1.ProtocolSCTP
	}
	return corev1.ProtocolTCP
}

func vmProfileFromKubeVirtVM(vm *kubevirtcorev1.VirtualMachine, vmi *kubevirtcorev1.VirtualMachineInstance) api_model.VMProfile {
	if vm == nil {
		return api_model.VMProfile{}
	}

	result := api_model.VMProfile{
		Name:             vm.Name,
		DisplayName:      vm.Annotations["wutong.io/display-name"],
		Desc:             vm.Annotations["wutong.io/desc"],
		OSSourceFrom:     api_model.OSSourceFrom(vm.Annotations["wutong.io/vm-os-source-from"]),
		OSSourceURL:      vm.Annotations["wutong.io/vm-os-source-url"],
		OSDiskSize:       cast.ToInt64(vm.Annotations["wutong.io/vm-disk-size"]),
		RequestCPU:       cast.ToInt64(vm.Annotations["wutong.io/vm-request-cpu"]),
		RequestMemory:    cast.ToInt64(vm.Annotations["wutong.io/vm-request-memory"]),
		Namespace:        vm.Namespace,
		DefaultLoginUser: vm.Annotations["wutong.io/vm-default-login-user"],
		Status:           string(vm.Status.PrintableStatus),
		StatusMessage:    vmStatusMessageFromKubeVirtVM(vm),
		CreatedBy:        vm.Annotations["wutong.io/creator"],
		LastModifiedBy:   vm.Annotations["wutong.io/last-modifier"],
		CreatedAt:        timeString(vm.CreationTimestamp.Time),
		LastModifiedAt:   timeString(cast.ToTime(vm.Annotations["wutong.io/last-modification-timestamp"])),
		OSInfo: api_model.VMOSInfo{
			Name:    vm.Annotations["wutong.io/vm-os-name"],
			Version: vm.Annotations["wutong.io/vm-os-version"],
			Arch:    vm.Spec.Template.Spec.Architecture,
		},
		InternalDomainName: vm.Name,
	}

	containsBootDisk := func(vm *kubevirtcorev1.VirtualMachine) bool {
		if !vm.Status.Ready {
			return false
		}
		for _, disk := range vm.Spec.Template.Spec.Domain.Devices.Disks {
			if disk.Name == bootDiskName && disk.BootOrder != nil && *disk.BootOrder == 1 {
				return true
			}
		}
		return false
	}

	result.ContainsBootDisk = containsBootDisk(vm)

	result.Conditions = vmConditions(vm)

	for labelKey, labelVal := range vm.Spec.Template.Spec.NodeSelector {
		if label, ok := strings.CutPrefix(labelKey, "vm-scheduling-label.wutong.io/"); ok {
			if labelVal != "" {
				label = fmt.Sprintf("%s=%s", label, labelVal)
			}
			result.NodeSelectorLabels = append(result.NodeSelectorLabels, label)
		}
	}

	if vmi != nil {
		if len(vmi.Status.Interfaces) > 0 {
			result.IP = vmi.Status.Interfaces[0].IP
		}
		result.ScheduleNode = vmi.Status.NodeName
		if vmi.Status.GuestOSInfo.Name != "" {
			result.OSInfo.Name = vmi.Status.GuestOSInfo.Name
		}
		if vmi.Status.GuestOSInfo.Version != "" {
			result.OSInfo.Version = vmi.Status.GuestOSInfo.Version
		}
		result.OSInfo.KernelRelease = vmi.Status.GuestOSInfo.KernelRelease
		result.OSInfo.KernelVersion = vmi.Status.GuestOSInfo.KernelVersion
	}

	return result
}

var statusMessageMap = map[kubevirtcorev1.VirtualMachinePrintableStatus]string{
	// kubevirtcorev1.VirtualMachineStatusStopped:                 "虚拟机已停止",
	// kubevirtcorev1.VirtualMachineStatusProvisioning:            "虚拟机正在创建...",
	// kubevirtcorev1.VirtualMachineStatusStarting:                "虚拟机正在启动...",
	// kubevirtcorev1.VirtualMachineStatusRunning:                 "虚拟机运行中",
	// kubevirtcorev1.VirtualMachineStatusPaused:                  "虚拟机已暂停",
	// kubevirtcorev1.VirtualMachineStatusStopping:                "虚拟机正在停止",
	// kubevirtcorev1.VirtualMachineStatusTerminating:             "虚拟机正在删除",
	kubevirtcorev1.VirtualMachineStatusCrashLoopBackOff: "虚拟机启动失败",
	// kubevirtcorev1.VirtualMachineStatusMigrating:               "虚拟机正在迁移",
	kubevirtcorev1.VirtualMachineStatusUnknown:                 "虚拟机状态未知",
	kubevirtcorev1.VirtualMachineStatusUnschedulable:           "虚拟机调度失败",
	kubevirtcorev1.VirtualMachineStatusErrImagePull:            "虚拟机镜像拉取失败",
	kubevirtcorev1.VirtualMachineStatusImagePullBackOff:        "虚拟机镜像拉取失败",
	kubevirtcorev1.VirtualMachineStatusPvcNotFound:             "虚拟机持久数据卷未找到",
	kubevirtcorev1.VirtualMachineStatusDataVolumeError:         "虚拟机数据卷错误",
	kubevirtcorev1.VirtualMachineStatusWaitingForVolumeBinding: "虚拟机数据卷等待绑定中...",
}

func vmStatusMessageFromKubeVirtVM(vm *kubevirtcorev1.VirtualMachine) string {
	var result = statusMessageMap[vm.Status.PrintableStatus]
	switch vm.Status.PrintableStatus {
	case kubevirtcorev1.VirtualMachineStatusUnschedulable:
		for _, cond := range vm.Status.Conditions {
			if cond.Type == kubevirtcorev1.VirtualMachineConditionType(corev1.PodScheduled) && cond.Status == corev1.ConditionFalse {
				result += fmt.Sprintf("，原因：%s", cond.Message)
			}
		}
	}
	return result
}

const (
	VolumeStatusUnknown          = "未知"
	VolumeStatusPending          = "待分配"
	VolumeStatusBoundButNotInUse = "未使用(重启虚拟机以使用该卷)"
	VolumeStatusBoundAndInUse    = "使用中(需进入虚拟机手动挂载)"
	VolumeStatusLost             = "已丢失"
)

func vmPod(kubeClient kubernetes.Interface, tenantEnv *dbmodel.TenantEnvs, vmID string) *corev1.Pod {
	pods, _ := kube.GetCachedResources(kubeClient).PodLister.Pods(tenantEnv.Namespace).List(labels.SelectorFromSet(labels.Set{
		"wutong.io/vm-id": vmID,
		"kubevirt.io":     "virt-launcher",
	}))
	if len(pods) > 0 {
		return pods[0]
	}
	return nil
}

func vmPodVolumes(vmPod *corev1.Pod) []string {
	if vmPod == nil {
		return nil
	}
	var result []string
	for _, volume := range vmPod.Spec.Volumes {
		if volume.PersistentVolumeClaim != nil {
			result = append(result, volume.PersistentVolumeClaim.ClaimName)
		}
	}
	return result
}

func volumeStatus(vmVolumes []string, pvc *corev1.PersistentVolumeClaim) string {
	if pvc == nil {
		return VolumeStatusUnknown
	}
	switch pvc.Status.Phase {
	case corev1.ClaimPending:
		return VolumeStatusPending
	case corev1.ClaimLost:
		return VolumeStatusLost
	case corev1.ClaimBound:
		if slices.Contains(vmVolumes, pvc.Name) {
			return VolumeStatusBoundAndInUse
		}
		return VolumeStatusBoundButNotInUse
	}
	return VolumeStatusUnknown
}

func validatePassword(password string) (bool, error) {
	if len(password) < 8 {
		return false, fmt.Errorf("密码长度不能小于 8 位！")
	}

	if !strings.ContainsAny(password, "abcdefghijklmnopqrstuvwxyz") || !strings.ContainsAny(password, "ABCDEFGHIJKLMNOPQRSTUVWXYZ") || !strings.ContainsAny(password, "0123456789") {
		return false, fmt.Errorf("密码必须同时包含大小写字母和数字！")
	}

	return true, nil
}

func vmConditions(vm *kubevirtcorev1.VirtualMachine) []api_model.VMCondition {
	var result []api_model.VMCondition

	if vm == nil {
		return result
	}

	for _, cond := range vm.Status.Conditions {
		result = append(result, api_model.VMCondition{
			Type:           string(cond.Type),
			Status:         cond.Status == corev1.ConditionTrue,
			Reason:         cond.Reason,
			Message:        cond.Message,
			LastReportedAt: timeString(cond.LastTransitionTime.Time),
		})
	}
	return result
}

func timeString(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Local().Format("2006-01-02 15:04:05")
}
