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
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cast"
	"github.com/wutong-paas/wutong/api/client/kube"
	api_model "github.com/wutong-paas/wutong/api/model"
	"github.com/wutong-paas/wutong/db"
	dbmodel "github.com/wutong-paas/wutong/db/model"
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

func (s *ServiceAction) CreateVM(tenantEnv *dbmodel.TenantEnvs, req *api_model.CreateVMRequest) (*api_model.CreateVMResponse, error) {
	ok := kube.IsKubevirtInstalled(s.kubeClient, s.apiextClient)
	if !ok {
		return nil, errors.New("集群中未检测到 Kubevirt 服务，使用该功能请联系管理员安装 Kubevirt 服务！")
	}

	if req.OSDiskSize == 0 {
		req.OSDiskSize = defailtOSDiskSize
	}

	req.User = strings.TrimSpace(req.User)
	if req.User == "" {
		return nil, fmt.Errorf("虚拟机初始用户名称不能为空！")
	}

	req.Password = strings.TrimSpace(req.Password)
	if req.Password == "" {
		return nil, fmt.Errorf("虚拟机初始用户密码不能为空！")
	}

	wutongLabels := labelsFromTenantEnv(tenantEnv)
	wutongLabels = labels.Merge(wutongLabels, map[string]string{
		"wutong.io/vm-id": req.Name,
	})

	var nodeSelector = map[string]string{
		"wutong.io/vm-schedulable": "true",
	}

	for _, labelKey := range req.NodeSelectorLabels {
		nodeSelector["vm-node-selector.wutong.io/"+labelKey] = ""
	}

	var source cdicorev1beta1.DataVolumeSource
	var sourceUrl string
	switch req.OSSourceFrom {
	case api_model.OSSourceFromHTTP:
		sourceUrl = req.OSSourceURL
		source = cdicorev1beta1.DataVolumeSource{
			HTTP: &cdicorev1beta1.DataVolumeSourceHTTP{
				URL: sourceUrl,
			},
		}
	case api_model.OSSourceFromRegistry:
		sourceUrl = "docker://" + req.OSSourceURL
		source = cdicorev1beta1.DataVolumeSource{
			Registry: &cdicorev1beta1.DataVolumeSourceRegistry{
				URL: util.Ptr(sourceUrl),
			},
		}
	}

	vmUserData := vmUserData(s.kubeClient, req.User, req.Password)

	vm := &kubevirtcorev1.VirtualMachine{
		TypeMeta: metav1.TypeMeta{
			Kind:       "VirtualMachine",
			APIVersion: kubevirtcorev1.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      req.Name,
			Namespace: tenantEnv.Namespace,
			Labels:    wutongLabels,
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
				"wutong.io/vm-os-source-from":           string(req.OSSourceFrom),
				"wutong.io/vm-os-source-url":            req.OSSourceURL,
				"wutong.io/vm-default-login-user":       req.User,
				"wutong.io/last-modification-timestamp": metav1.Now().UTC().Format(time.RFC3339),
			},
		},
		Spec: kubevirtcorev1.VirtualMachineSpec{
			DataVolumeTemplates: []kubevirtcorev1.DataVolumeTemplateSpec{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:   req.Name + "-dv",
						Labels: wutongLabels,
						Annotations: map[string]string{
							"cdi.kubevirt.io/storage.import.source":   string(req.OSSourceFrom),
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
									corev1.ResourceStorage: *resource.NewQuantity(req.OSDiskSize*1024*1024*1024, resource.BinarySI),
								},
							},
							StorageClassName: util.Ptr(kube.GetDefaultStorageClass(s.kubeClient)),
						},
						Source: &source,
					},
				},
			},
			Running: util.Ptr(req.Running), // 默认不启动
			Template: &kubevirtcorev1.VirtualMachineInstanceTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: wutongLabels,
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
								{
									Name: "cloudinitdisk",
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
								"cpu":    resource.MustParse(fmt.Sprintf("%dm", req.RequestCPU)),
								"memory": resource.MustParse(fmt.Sprintf("%dGi", req.RequestMemory)),
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
					Volumes: []kubevirtcorev1.Volume{
						{
							Name: "containerdisk",
							VolumeSource: kubevirtcorev1.VolumeSource{
								DataVolume: &kubevirtcorev1.DataVolumeSource{
									Name: req.Name + "-dv",
								},
							},
						},
					},
				},
			},
		},
	}

	var result = &api_model.CreateVMResponse{
		VMProfile: vmProfileFromKubeVirtVM(vm, nil),
	}

	bcrypt.GenerateFromPassword([]byte("ubuntu"), bcrypt.DefaultCost)
	bcryptedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		logrus.Errorf("bcrypt password failed, error: %s", err.Error())
		return nil, fmt.Errorf("虚拟机初始用户密码加密失败！")
	}

	vmUserData += fmt.Sprintf(`runcmd:
%s`, filebrowserRunCmd(req.User, string(bcryptedPassword)))

	// set cloudinit
	vm.Spec.Template.Spec.Volumes = append(vm.Spec.Template.Spec.Volumes, kubevirtcorev1.Volume{
		Name: "cloudinitdisk",
		VolumeSource: kubevirtcorev1.VolumeSource{
			CloudInitNoCloud: &kubevirtcorev1.CloudInitNoCloudSource{
				UserData: vmUserData,
			},
		},
	})

	created, err := kube.CreateKubevirtVM(s.dynamicClient, vm)
	if err != nil {
		if k8sErrors.IsAlreadyExists(err) {
			return result, fmt.Errorf("虚拟机 %s 名称被占用！", req.Name)
		}
		logrus.Errorf("create vm failed, error: %s", err.Error())
		return result, fmt.Errorf("创建虚拟机 %s 失败！", req.Name)
	}

	// create ssh, filebrowser port
	s.AddVMPort(tenantEnv, req.Name, &api_model.AddVMPortRequest{
		VMPort:   22,
		Protocol: api_model.VMPortProtocolSSH,
	})

	s.AddVMPort(tenantEnv, req.Name, &api_model.AddVMPortRequest{
		VMPort:   6173,
		Protocol: api_model.VMPortProtocolHTTP,
	})

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
		vm.Spec.Template.Spec.Domain.Resources.Requests["cpu"] = resource.MustParse(fmt.Sprintf("%dm", req.RequestCPU))
	}
	if req.RequestMemory > 0 {
		vm.Annotations["wutong.io/vm-request-memory"] = fmt.Sprintf("%d", req.RequestMemory)
		vm.Spec.Template.Spec.Domain.Resources.Requests["memory"] = resource.MustParse(fmt.Sprintf("%dGi", req.RequestMemory))
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
		nodeSelector["vm-node-selector.wutong.io/"+labelKey] = ""
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

	vmProfile := vmProfileFromKubeVirtVM(got, nil)

	return &api_model.RestartVMResponse{
		VMProfile: vmProfile,
	}, nil
}

func (s *ServiceAction) AddVMPort(tenantEnv *dbmodel.TenantEnvs, vmID string, req *api_model.AddVMPortRequest) error {
	svcName := serviceName(vmID, req.VMPort, string(req.Protocol))

	wutongLabels := labelsFromTenantEnv(tenantEnv)
	wutongLabels = labels.Merge(wutongLabels, map[string]string{
		"wutong.io/vm-id":            vmID,
		"wutong.io/vm-port-enabled":  "false",
		"wutong.io/vm-port":          fmt.Sprintf("%d", req.VMPort),
		"wutong.io/vm-port-protocol": string(req.Protocol),
	})

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      svcName,
			Namespace: tenantEnv.Namespace,
			Labels:    wutongLabels,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"wutong.io/vm-id": vmID,
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
	svcName := serviceName(vmID, req.VMPort, string(req.Protocol))
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
		"wutong.io/vm-port-protocol": string(req.Protocol),
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
	svcName := serviceName(vmID, req.VMPort, string(req.Protocol))
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
		"wutong.io/vm-port-protocol": string(req.Protocol),
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

	slices.SortFunc(svcList, func(i, j *corev1.Service) int {
		if i.CreationTimestamp.Before(&j.CreationTimestamp) {
			return 1
		} else if i.CreationTimestamp.After(j.CreationTimestamp.Time) {
			return -1
		}
		return 0
	})

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
					if protocol == string(api_model.VMPortProtocolHTTP) {
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

	result.Total = len(result.Ports)
	return result, nil
}

func (s *ServiceAction) CreateVMPortGateway(tenantEnv *dbmodel.TenantEnvs, vmID string, req *api_model.CreateVMPortGatewayRequest) error {
	svcName := serviceName(vmID, req.VMPort, string(req.Protocol))

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

	if protocol == string(api_model.VMPortProtocolHTTP) {
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

	if protocol == string(api_model.VMPortProtocolHTTP) {
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

	if protocol == string(api_model.VMPortProtocolHTTP) {
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

	if protocol == string(api_model.VMPortProtocolHTTP) {
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

	if protocol == string(api_model.VMPortProtocolHTTP) {
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
		"wutong.io/vm-port-protocol": string(req.Protocol),
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
	svcName := serviceName(vmID, req.VMPort, string(req.Protocol))
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

	// 2、删除虚拟机
	err = kube.DeleteKubeVirtVM(s.dynamicClient, tenantEnv.Namespace, vmID)
	if err != nil {
		logrus.Errorf("delete vm failed, error: %s", err.Error())
		return fmt.Errorf("删除虚拟机 %s 失败！", vmID)
	}
	return nil
}

func (s *ServiceAction) ListVMs(tenantEnv *dbmodel.TenantEnvs) (*api_model.ListVMsResponse, error) {

	vms, err := kube.ListKubeVirtVMs(s.dynamicClient, tenantEnv.Namespace)
	if err != nil {
		logrus.Errorf("list vm failed, error: %s", err.Error())
		return nil, errors.New("获取虚拟机列表失败！")
	}

	var result = new(api_model.ListVMsResponse)
	for _, vm := range vms {
		vp := vmProfileFromKubeVirtVM(vm, nil)
		result.VMs = append(result.VMs, vp)
	}
	result.Total = len(result.VMs)

	sort.Slice(result.VMs, func(i, j int) bool {
		return result.VMs[i].CreatedAt.After(result.VMs[j].CreatedAt)
	})
	return result, nil
}

func vmUserData(kubeClient kubernetes.Interface, username, password string) string {
	vmUserData := fmt.Sprintf("#cloud-config\nchpasswd: { expire: False }\nuser: %s\npassword: %s\nssh_pwauth: True\n", username, password)
	vmSSHPubKey, _ := kube.GetWTChannelSSHPubKey(kubeClient)
	if len(vmSSHPubKey) > 0 {
		vmUserData += fmt.Sprintf("ssh_authorized_keys:\n  - %s\n", string(vmSSHPubKey))
	}
	return vmUserData
}

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

func generateGatewayHost(namespace, vmID string, port int) string {
	exDomain := os.Getenv("EX_DOMAIN")
	if exDomain == "" {
		return ""
	}
	if strings.Contains(exDomain, ":") {
		exDomain = strings.Split(exDomain, ":")[0]
	}
	svcName := serviceName(vmID, port, string(api_model.VMPortProtocolHTTP))
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
		CreatedBy:        vm.Annotations["wutong.io/creator"],
		LastModifiedBy:   vm.Annotations["wutong.io/last-modifier"],
		CreatedAt:        vm.CreationTimestamp.Time.Local(),
		LastModifiedAt:   cast.ToTime(vm.Annotations["wutong.io/last-modification-timestamp"]).Local(),
		OSInfo: api_model.VMOSInfo{
			Name:    vm.Annotations["wutong.io/vm-os-name"],
			Version: vm.Annotations["wutong.io/vm-os-version"],
			Arch:    vm.Spec.Template.Spec.Architecture,
		},
	}

	for _, cond := range vm.Status.Conditions {
		if cond.Type == kubevirtcorev1.VirtualMachineConditionType(kubevirtcorev1.VirtualMachineReady) {
			result.StatusMessage = cond.Message
		}
	}

	for k := range vm.Spec.Template.Spec.NodeSelector {
		if label, ok := strings.CutPrefix(k, "vm-node-selector.wutong.io/"); ok {
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
