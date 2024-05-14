package kube

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/wutong-paas/wutong/chaos"
	"github.com/wutong-paas/wutong/util"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/tools/cache"
	kubevirtcorev1 "kubevirt.io/api/core/v1"
)

var dynamicCachedResources *DynamicCachedResources

type DynamicCachedResources struct {
	VMLister  cache.GenericLister
	VMILister cache.GenericLister
}

func GetDynamicCachedResources(dynamicClient dynamic.Interface) *DynamicCachedResources {
	if dynamicCachedResources == nil {
		dynamicCachedResources = initializeDynamicCachedResources(dynamicClient)
	}
	return dynamicCachedResources
}

var (
	vmres = schema.GroupVersionResource{
		Group:    "kubevirt.io",
		Version:  "v1",
		Resource: "virtualmachines",
	}
	vmires = schema.GroupVersionResource{
		Group:    "kubevirt.io",
		Version:  "v1",
		Resource: "virtualmachineinstances",
	}
)

func initializeDynamicCachedResources(dynamicClient dynamic.Interface) *DynamicCachedResources {
	dynamicInformers := dynamicinformer.NewDynamicSharedInformerFactory(dynamicClient, time.Minute*10)

	// informer
	vmInformer := dynamicInformers.ForResource(vmres)
	vmiInformer := dynamicInformers.ForResource(vmires)

	// shared informers
	vmSharedInformer := vmInformer.Informer()
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

	return &DynamicCachedResources{
		VMLister:  vmInformer.Lister(),
		VMILister: vmiInformer.Lister(),
	}
}

func (d *DynamicCachedResources) GetVMLister() cache.GenericLister {
	return d.VMLister
}

func (d *DynamicCachedResources) GetVMILister() cache.GenericLister {
	return d.VMILister
}

func UnDynamicObject[O runtime.Object](obj runtime.Object) (O, error) {
	var result O
	if obj == nil {
		return result, fmt.Errorf("obj is nil")
	}
	unstructuredObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return result, err
	}

	err = runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObj, &result)
	return result, err
}

func UnDynamicObjectList[O runtime.Object](objs []runtime.Object) ([]O, error) {
	var result []O
	for _, obj := range objs {
		o, err := UnDynamicObject[O](obj)
		if err != nil {
			return result, err
		}
		result = append(result, o)
	}
	return result, nil
}

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

func vmiEventHandler() cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			vmi, err := convertToVirtualMachineInstance(obj)
			if err == nil && vmi.Labels["creator"] == "Wutong" {
				keepVirtVNC(vmi)
			}
		},
		UpdateFunc: func(_, obj interface{}) {
			vmi, err := convertToVirtualMachineInstance(obj)
			if err == nil && vmi.Labels["creator"] == "Wutong" {
				keepVirtVNC(vmi)
			}
		},
		DeleteFunc: func(obj interface{}) {
			vmi, err := convertToVirtualMachineInstance(obj)
			if err == nil && vmi.Labels["creator"] == "Wutong" {
				reclaimVirtVNC(vmi)
			}
		},
	}
}

func keepVirtVNC(vmi *kubevirtcorev1.VirtualMachineInstance) {
	namespace := vmi.Namespace
	wutongLabels := vmi.Labels
	vncName := fmt.Sprintf("%s-vnc", vmi.Name)
	selectorLabels := map[string]string{
		"wutong.io/vm-id":    vmi.Name,
		"wutong.io/vnc-name": vncName,
	}
	wutongLabels = labels.Merge(wutongLabels, selectorLabels)
	if _, err := RegionClientset().CoreV1().ServiceAccounts(namespace).Get(context.Background(), vncName, metav1.GetOptions{}); err != nil && k8sErrors.IsNotFound(err) {
		var sa = &corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:      vncName,
				Namespace: namespace,
				Labels:    wutongLabels,
			},
		}
		RegionClientset().CoreV1().ServiceAccounts(namespace).Create(context.Background(), sa, metav1.CreateOptions{})
	}

	if _, err := RegionClientset().RbacV1().Roles(namespace).Get(context.Background(), vncName, metav1.GetOptions{}); err != nil && k8sErrors.IsNotFound(err) {
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
		RegionClientset().RbacV1().Roles(namespace).Create(context.Background(), r, metav1.CreateOptions{})
	}

	if _, err := RegionClientset().RbacV1().RoleBindings(namespace).Get(context.Background(), vncName, metav1.GetOptions{}); err != nil && k8sErrors.IsNotFound(err) {
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
		RegionClientset().RbacV1().RoleBindings(namespace).Create(context.Background(), rb, metav1.CreateOptions{})
	}

	if _, err := RegionClientset().CoreV1().Services(namespace).Get(context.Background(), vncName, metav1.GetOptions{}); err != nil && k8sErrors.IsNotFound(err) {
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
		RegionClientset().CoreV1().Services(namespace).Create(context.Background(), svc, metav1.CreateOptions{})
	}

	if _, err := GetCachedResources(RegionClientset()).DeploymentLister.Deployments(namespace).Get(vncName); err != nil && k8sErrors.IsNotFound(err) {
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
		RegionClientset().AppsV1().Deployments(namespace).Create(context.Background(), dep, metav1.CreateOptions{})
	}
}

func reclaimVirtVNC(vmi *kubevirtcorev1.VirtualMachineInstance) {
	namespace := vmi.Namespace
	vncName := fmt.Sprintf("%s-vnc", vmi.Name)
	if err := RegionClientset().CoreV1().ServiceAccounts(namespace).Delete(context.Background(), vncName, metav1.DeleteOptions{}); err != nil && !k8sErrors.IsNotFound(err) {
		logrus.Warningf("delete service account %s failed: %v", vncName, err)
	}

	if err := RegionClientset().RbacV1().Roles(namespace).Delete(context.Background(), vncName, metav1.DeleteOptions{}); err != nil && !k8sErrors.IsNotFound(err) {
		logrus.Warningf("delete role %s failed: %v", vncName, err)
	}

	if err := RegionClientset().RbacV1().RoleBindings(namespace).Delete(context.Background(), vncName, metav1.DeleteOptions{}); err != nil && !k8sErrors.IsNotFound(err) {
		logrus.Warningf("delete role binding %s failed: %v", vncName, err)
	}

	if err := RegionClientset().CoreV1().Services(namespace).Delete(context.Background(), vncName, metav1.DeleteOptions{}); err != nil && !k8sErrors.IsNotFound(err) {
		logrus.Warningf("delete service %s failed: %v", vncName, err)
	}

	if err := RegionClientset().AppsV1().Deployments(namespace).Delete(context.Background(), vncName, metav1.DeleteOptions{}); err != nil && !k8sErrors.IsNotFound(err) {
		logrus.Warningf("delete deployment %s failed: %v", vncName, err)
	}
}
