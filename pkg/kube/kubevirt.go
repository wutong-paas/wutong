package kube

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	rt "runtime"

	"github.com/sirupsen/logrus"
	"github.com/wutong-paas/wutong/chaos"
	"github.com/wutong-paas/wutong/util"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/retry"
	kubevirtcorev1 "kubevirt.io/api/core/v1"
)

var isKubevirtInstalled *bool

var (
	vmres = schema.GroupVersionResource{
		Group:    kubevirtcorev1.GroupVersion.Group,
		Version:  kubevirtcorev1.GroupVersion.Version,
		Resource: "virtualmachines",
	}
	vmires = schema.GroupVersionResource{
		Group:    kubevirtcorev1.GroupVersion.Group,
		Version:  kubevirtcorev1.GroupVersion.Version,
		Resource: "virtualmachineinstances",
	}
)

func IsKubevirtInstalled(kubeClient kubernetes.Interface, apiextClient apiextclient.Interface) bool {
	if isKubevirtInstalled == nil {
		_, err := apiextClient.ApiextensionsV1().CustomResourceDefinitions().Get(context.Background(), "virtualmachines.kubevirt.io", metav1.GetOptions{})
		if err != nil {
			log.Println("not found kubevirt crd: virtualmachines.kubevirt.io")
			isKubevirtInstalled = util.Ptr(false)
		} else {
			isKubevirtInstalled = util.Ptr(true)
			go registerKubevirtEventHandlers()
		}
	}

	return *isKubevirtInstalled
}

func GetWTChannelSSHPubKey(kubeClient kubernetes.Interface) (string, error) {
	secret, err := GetCachedResources(kubeClient).SecretLister.Secrets("wt-system").Get("wt-channel")
	if err != nil {
		return "", err
	}

	return string(secret.Data["id_rsa.pub"]), nil
}

// registerKubevirtEventHandlers 注册 kubevirt 资源事件处理程序
func registerKubevirtEventHandlers() {
	dynamicInformers := dynamicinformer.NewDynamicSharedInformerFactory(DynamicClient(), time.Minute*10)

	// informer
	vmInformer := dynamicInformers.ForResource(vmres)
	vmiInformer := dynamicInformers.ForResource(vmires)

	// shared informers
	vmSharedInformer := vmInformer.Informer()
	vmSharedInformer.AddEventHandlerWithResyncPeriod(vmEventHandler(), time.Minute)
	vmiSharedInformer := vmiInformer.Informer()
	vmiSharedInformer.AddEventHandlerWithResyncPeriod(vmiEventHandler(), time.Minute*30)

	informers := map[string]cache.SharedInformer{
		"vmSharedInformer":  vmSharedInformer,
		"vmiSharedInformer": vmiSharedInformer,
	}

	var wg sync.WaitGroup
	wg.Add(len(informers))
	for k, v := range informers {
		go func(name string, informer cache.SharedInformer) {
			if !cache.WaitForCacheSync(wait.NeverStop, informer.HasSynced) {
				logrus.Warningln("wait for cached synced failed:", name)
			}
			wg.Done()
		}(k, v)
	}

	dynamicInformers.Start(wait.NeverStop)
	dynamicInformers.WaitForCacheSync(wait.NeverStop)
}

// convertToVirtualMachine 对象转换为虚拟机
func convertToVirtualMachine(obj interface{}) (*kubevirtcorev1.VirtualMachine, error) {
	unstructuredObj, ok := obj.(*unstructured.Unstructured)
	if !ok {
		return nil, fmt.Errorf("cannot cast obj as unstructured: %v", obj)
	}

	vm := &kubevirtcorev1.VirtualMachine{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObj.Object, vm); err != nil {
		return nil, fmt.Errorf("error converting to VirtualMachine: %v", err)
	}

	return vm, nil
}

// convertToVirtualMachineInstance 对象转换为虚拟机实例
func convertToVirtualMachineInstance(obj interface{}) (*kubevirtcorev1.VirtualMachineInstance, error) {
	unstructuredObj, ok := obj.(*unstructured.Unstructured)
	if !ok {
		return nil, fmt.Errorf("cannot cast obj as unstructured: %v", obj)
	}

	vmi := &kubevirtcorev1.VirtualMachineInstance{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObj.Object, vmi); err != nil {
		return nil, fmt.Errorf("error converting to VirtualMachineInstance: %v", err)
	}

	return vmi, nil
}

// vmEventHandler 虚拟机事件处理
func vmEventHandler() cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			vm, err := convertToVirtualMachine(obj)
			// vm, ok := obj.(*kubevirtcorev1.VirtualMachine)
			if err == nil && vm.Labels["creator"] == "Wutong" {
				// if ok && vm.Labels["creator"] == "Wutong" {
				keepVMStatements(vm)
				tryCreateInternalDomainService(vm)
			} else {
				logrus.Infof("vm %s is not created by Wutong", vm.Name)
			}
		},
		UpdateFunc: func(_, obj interface{}) {
			vm, err := convertToVirtualMachine(obj)
			// vm, ok := obj.(*kubevirtcorev1.VirtualMachine)
			if err == nil && vm.Labels["creator"] == "Wutong" {
				// if ok && vm.Labels["creator"] == "Wutong" {
				keepVMStatements(vm)
				tryCreateInternalDomainService(vm)
			}
		},
		DeleteFunc: func(obj interface{}) {
			vm, err := convertToVirtualMachine(obj)
			// vm, ok := obj.(*kubevirtcorev1.VirtualMachine)
			if err == nil && vm.Labels["creator"] == "Wutong" {
				// if ok && vm.Labels["creator"] == "Wutong" {
				tryDeleteInternalDomainService(vm)
			}
		},
	}
}

// vmiEventHandler 虚拟机实例事件处理
func vmiEventHandler() cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			vmi, err := convertToVirtualMachineInstance(obj)
			// vmi, ok := obj.(*kubevirtcorev1.VirtualMachineInstance)
			if err == nil && vmi.Labels["creator"] == "Wutong" {
				// if ok && vmi.Labels["creator"] == "Wutong" {
				keepVirtVNC(vmi)
			}
		},
		UpdateFunc: func(_, obj interface{}) {
			vmi, err := convertToVirtualMachineInstance(obj)
			// vmi, ok := obj.(*kubevirtcorev1.VirtualMachineInstance)
			if err == nil && vmi.Labels["creator"] == "Wutong" {
				// if ok && vmi.Labels["creator"] == "Wutong" {
				keepVirtVNC(vmi)
			}
		},
		DeleteFunc: func(obj interface{}) {
			vmi, err := convertToVirtualMachineInstance(obj)
			// vmi, ok := obj.(*kubevirtcorev1.VirtualMachineInstance)
			if err == nil && vmi.Labels["creator"] == "Wutong" {
				// if ok && vmi.Labels["creator"] == "Wutong" {
				reclaimVirtVNC(vmi)
			}
		},
	}
}

// keepVMStatements 保持虚拟机声明
func keepVMStatements(vm *kubevirtcorev1.VirtualMachine) {
	var changed bool

	if vm.Labels["wutong.io/vm-id"] != vm.Name {
		vm.Labels["wutong.io/vm-id"] = vm.Name
		changed = true
	}
	if vm.Annotations["wutong.io/display-name"] == "" {
		vm.Annotations["wutong.io/display-name"] = vm.Name
		changed = true
	}

	vmClone, err := KubevirtClient().VirtualMachineClone(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
	if err == nil {
		// 说明这个虚拟机是克隆出来的
		vmCreator := vmClone.Annotations["wutong.io/creator"]
		if vm.Annotations["wutong.io/creator"] != vmCreator {
			vm.Annotations["wutong.io/creator"] = vmCreator
			changed = true
		}
		if vm.Annotations["wutong.io/last-modifier"] == "" {
			vm.Annotations["wutong.io/last-modifier"] = vmCreator
			changed = true
		}
		if vm.Annotations["wutong.io/last-modification-timestamp"] == "" {
			vm.Annotations["wutong.io/last-modification-timestamp"] = metav1.Now().UTC().Format(time.RFC3339)
			changed = true
		}
	}

	if changed {
		if err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
			if _, err := KubevirtClient().VirtualMachine(vm.Namespace).Update(context.Background(), vm, metav1.UpdateOptions{}); err != nil {
				if k8sErrors.IsConflict(err) {
					latest, getLatestErr := KubevirtClient().VirtualMachine(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
					if getLatestErr != nil {
						return err
					}
					vm.SetResourceVersion(latest.GetResourceVersion())
					return err
				}
				return err
			}
			return nil
		}); err != nil {
			logrus.Warningf("failed to update vm %s: %v", vm.Name, err)
		}
	}
}

type kubevirtInfo struct {
	Arch            string
	Version         string
	ImageRegistry   string
	ImageTag        string
	ImagePullPolicy corev1.PullPolicy
}

var defaultKubeVirtInfo = &kubevirtInfo{
	Arch:            rt.GOARCH,
	Version:         "v1.3.0",
	ImageRegistry:   "quay.io/kubevirt",
	ImageTag:        "v1.3.0",
	ImagePullPolicy: corev1.PullIfNotPresent,
}

func KubeVirtInfo() *kubevirtInfo {
	kubevirt, err := KubevirtClient().KubeVirt("kubevirt").Get(context.Background(), "kubevirt", metav1.GetOptions{})
	if err != nil {
		logrus.Warningf("failed to get kubevirt: %v", err)
		return defaultKubeVirtInfo
	}

	return &kubevirtInfo{
		Arch:            util.If(kubevirt.Status.DefaultArchitecture != "", kubevirt.Status.DefaultArchitecture, rt.GOARCH),
		Version:         util.If(kubevirt.Status.TargetKubeVirtVersion != "", kubevirt.Status.TargetKubeVirtVersion, "v1.3.0"),
		ImageRegistry:   util.If(kubevirt.Status.TargetKubeVirtRegistry != "", kubevirt.Status.TargetKubeVirtRegistry, "quay.io/kubevirt"),
		ImageTag:        util.If(kubevirt.Status.TargetKubeVirtVersion != "", kubevirt.Status.TargetKubeVirtVersion, "v1.3.0"),
		ImagePullPolicy: util.If(kubevirt.Spec.ImagePullPolicy != "", kubevirt.Spec.ImagePullPolicy, corev1.PullIfNotPresent),
	}
}

// tryCreateInternalDomainService 尝试创建虚拟机内部域名服务
func tryCreateInternalDomainService(vm *kubevirtcorev1.VirtualMachine) {
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      vm.Name,
			Namespace: vm.Namespace,
			Labels:    vm.Labels,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"wutong.io/vm-id":     vm.Name,
				"vm.kubevirt.io/name": vm.Name,
			},
			ClusterIP: corev1.ClusterIPNone,
		},
	}
	if _, err := KubeClient().CoreV1().Services(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{}); err != nil && k8sErrors.IsNotFound(err) {
		if _, err := KubeClient().CoreV1().Services(vm.Namespace).Create(context.Background(), svc, metav1.CreateOptions{}); err != nil {
			logrus.Warningf("create vnc service %s failed: %v", vm.Name, err)
		}
	}
}

// tryDeleteInternalDomainService 尝试删除虚拟机内部域名服务
func tryDeleteInternalDomainService(vm *kubevirtcorev1.VirtualMachine) {
	if _, err := KubeClient().CoreV1().Services(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{}); err != nil && !k8sErrors.IsNotFound(err) {
		if err := KubeClient().CoreV1().Services(vm.Namespace).Delete(context.Background(), vm.Name, metav1.DeleteOptions{}); err != nil && !k8sErrors.IsNotFound(err) {
			logrus.Warningf("delete service %s failed: %v", vm.Name, err)
		}
	}
}

// keepVirtVNC 保持虚拟机VNC服务
func keepVirtVNC(vmi *kubevirtcorev1.VirtualMachineInstance) {
	namespace := vmi.Namespace
	wutongLabels := vmi.Labels
	vncName := fmt.Sprintf("%s-vnc", vmi.Name)
	selectorLabels := map[string]string{
		"wutong.io/vm-id":    vmi.Name,
		"wutong.io/vnc-name": vncName,
	}
	wutongLabels = labels.Merge(wutongLabels, selectorLabels)
	if _, err := KubeClient().CoreV1().ServiceAccounts(namespace).Get(context.Background(), vncName, metav1.GetOptions{}); err != nil && k8sErrors.IsNotFound(err) {
		var sa = &corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:      vncName,
				Namespace: namespace,
				Labels:    wutongLabels,
			},
		}
		KubeClient().CoreV1().ServiceAccounts(namespace).Create(context.Background(), sa, metav1.CreateOptions{})
	}

	if _, err := KubeClient().RbacV1().Roles(namespace).Get(context.Background(), vncName, metav1.GetOptions{}); err != nil && k8sErrors.IsNotFound(err) {
		var r = &rbacv1.Role{
			ObjectMeta: metav1.ObjectMeta{
				Name:   vncName,
				Labels: wutongLabels,
			},
			Rules: []rbacv1.PolicyRule{
				{
					APIGroups:     []string{"subresources.kubevirt.io"},
					Resources:     []string{"virtualmachineinstances/console", "virtualmachineinstances/vnc"},
					Verbs:         []string{"get"},
					ResourceNames: []string{vmi.Name},
				},
			},
		}
		KubeClient().RbacV1().Roles(namespace).Create(context.Background(), r, metav1.CreateOptions{})
	}

	if _, err := KubeClient().RbacV1().RoleBindings(namespace).Get(context.Background(), vncName, metav1.GetOptions{}); err != nil && k8sErrors.IsNotFound(err) {
		var rb = &rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:   vncName,
				Labels: wutongLabels,
			},
			RoleRef: rbacv1.RoleRef{
				APIGroup: rbacv1.GroupName,
				Kind:     "Role",
				Name:     vncName,
			},
			Subjects: []rbacv1.Subject{
				{
					Kind:      "ServiceAccount",
					Name:      vncName,
					Namespace: namespace,
				},
			},
		}
		KubeClient().RbacV1().RoleBindings(namespace).Create(context.Background(), rb, metav1.CreateOptions{})
	}

	if _, err := KubeClient().CoreV1().Services(namespace).Get(context.Background(), vncName, metav1.GetOptions{}); err != nil && k8sErrors.IsNotFound(err) {
		var svc = &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      vncName,
				Namespace: namespace,
				Labels:    wutongLabels,
			},
			Spec: corev1.ServiceSpec{
				Selector: selectorLabels,
				Ports: []corev1.ServicePort{
					{
						Name:       "http",
						Protocol:   corev1.ProtocolTCP,
						Port:       80,
						TargetPort: intstr.FromInt(8001),
					},
				},
			},
		}
		KubeClient().CoreV1().Services(namespace).Create(context.Background(), svc, metav1.CreateOptions{})
	}

	if _, err := GetCachedResources(KubeClient()).DeploymentLister.Deployments(namespace).Get(vncName); err != nil && k8sErrors.IsNotFound(err) {
		var dep = &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      vncName,
				Namespace: namespace,
				Labels:    wutongLabels,
			},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: selectorLabels,
				},
				Replicas: util.Ptr(int32(1)),
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: wutongLabels,
					},
					Spec: corev1.PodSpec{
						ServiceAccountName: vncName,
						Containers: []corev1.Container{
							{
								Name:  vncName,
								Image: chaos.VIRTVNCIMAGENAME,
								Env: []corev1.EnvVar{
									{
										Name:  "VM_NAMESPACE",
										Value: namespace,
									},
									{
										Name:  "VM_NAME",
										Value: vmi.Name,
									},
									{
										Name:  "VNC_PATH_PREFIX",
										Value: fmt.Sprintf("/console/virt-vnc/%s/%s/k8s", namespace, vmi.Name),
									},
								},
								LivenessProbe: &corev1.Probe{
									ProbeHandler: corev1.ProbeHandler{
										HTTPGet: &corev1.HTTPGetAction{
											Path:   "/",
											Port:   intstr.FromInt(8001),
											Scheme: corev1.URISchemeHTTP,
										},
									},
									FailureThreshold:    30,
									InitialDelaySeconds: 30,
									PeriodSeconds:       10,
									SuccessThreshold:    1,
									TimeoutSeconds:      5,
								},
								Ports: []corev1.ContainerPort{
									{
										ContainerPort: 8001,
										Name:          "http",
									},
								},
								ImagePullPolicy: corev1.PullAlways,
							},
						},
						Tolerations: []corev1.Toleration{
							// tolerate master or control-plane nodes
							{
								Key:      "node-role.kubernetes.io/master",
								Operator: corev1.TolerationOpEqual,
								Effect:   corev1.TaintEffectNoSchedule,
							},
							{
								Key:      "node-role.kubernetes.io/control-plane",
								Operator: corev1.TolerationOpEqual,
								Effect:   corev1.TaintEffectNoSchedule,
							},
						},
					},
				},
			},
		}
		KubeClient().AppsV1().Deployments(namespace).Create(context.Background(), dep, metav1.CreateOptions{})
	}
}

// reclaimVirtVNC 回收虚拟机VNC服务
func reclaimVirtVNC(vmi *kubevirtcorev1.VirtualMachineInstance) {
	namespace := vmi.Namespace
	vncName := fmt.Sprintf("%s-vnc", vmi.Name)
	if err := KubeClient().CoreV1().ServiceAccounts(namespace).Delete(context.Background(), vncName, metav1.DeleteOptions{}); err != nil && !k8sErrors.IsNotFound(err) {
		logrus.Warningf("delete service account %s failed: %v", vncName, err)
	}

	if err := KubeClient().RbacV1().Roles(namespace).Delete(context.Background(), vncName, metav1.DeleteOptions{}); err != nil && !k8sErrors.IsNotFound(err) {
		logrus.Warningf("delete role %s failed: %v", vncName, err)
	}

	if err := KubeClient().RbacV1().RoleBindings(namespace).Delete(context.Background(), vncName, metav1.DeleteOptions{}); err != nil && !k8sErrors.IsNotFound(err) {
		logrus.Warningf("delete role binding %s failed: %v", vncName, err)
	}

	if err := KubeClient().CoreV1().Services(namespace).Delete(context.Background(), vncName, metav1.DeleteOptions{}); err != nil && !k8sErrors.IsNotFound(err) {
		logrus.Warningf("delete service %s failed: %v", vncName, err)
	}

	if err := KubeClient().AppsV1().Deployments(namespace).Delete(context.Background(), vncName, metav1.DeleteOptions{}); err != nil && !k8sErrors.IsNotFound(err) {
		logrus.Warningf("delete deployment %s failed: %v", vncName, err)
	}
}
