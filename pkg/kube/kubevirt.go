package kube

import (
	"context"
	"fmt"
	"log"

	"github.com/wutong-paas/wutong/util"
	apiextclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"
	kubevirtcorev1 "kubevirt.io/api/core/v1"
)

var isKubevirtInstalled *bool

func IsKubevirtInstalled(kubeClient kubernetes.Interface, apiextClient apiextclient.Interface) bool {
	if isKubevirtInstalled == nil {
		_, err := apiextClient.ApiextensionsV1().CustomResourceDefinitions().Get(context.Background(), "virtualmachines.kubevirt.io", metav1.GetOptions{})
		if err != nil {
			log.Println("not found kubevirt crd: virtualmachines.kubevirt.io")
			isKubevirtInstalled = util.Ptr(false)
		} else {
			isKubevirtInstalled = util.Ptr(true)
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

func CreateKubevirtVM(dynamicClient dynamic.Interface, vm *kubevirtcorev1.VirtualMachine) (*kubevirtcorev1.VirtualMachine, error) {
	if vm == nil {
		return vm, fmt.Errorf("vm is nil")
	}
	unstructuredVM, err := runtime.DefaultUnstructuredConverter.ToUnstructured(vm)
	if err != nil {
		return vm, err
	}

	unstructuredVMObj := &unstructured.Unstructured{Object: unstructuredVM}

	createdObj, err := dynamicClient.Resource(vmres).Namespace(vm.Namespace).Create(context.Background(), unstructuredVMObj, metav1.CreateOptions{})
	if err != nil {
		return vm, err
	}
	createdVM := &kubevirtcorev1.VirtualMachine{}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(createdObj.Object, vm)
	return createdVM, err
}

func GetKubeVirtVM(dynamicClient dynamic.Interface, namespace, name string) (*kubevirtcorev1.VirtualMachine, error) {
	getObj, err := GetDynamicCachedResources(dynamicClient).VMLister.ByNamespace(namespace).Get(name)
	if err != nil {
		return nil, err
	}

	return UnDynamicObject[*kubevirtcorev1.VirtualMachine](getObj)
}

func UpdateKubeVirtVM(dynamicClient dynamic.Interface, vm *kubevirtcorev1.VirtualMachine) (*kubevirtcorev1.VirtualMachine, error) {
	updatedVM := &kubevirtcorev1.VirtualMachine{}
	err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		if vm == nil {
			return fmt.Errorf("vm is nil")
		}
		unstructuredVM, err := runtime.DefaultUnstructuredConverter.ToUnstructured(vm)
		if err != nil {
			return err
		}
		unstructuredVMObj := &unstructured.Unstructured{Object: unstructuredVM}
		updatedObj, err := dynamicClient.Resource(vmres).Namespace(vm.Namespace).Update(context.Background(), unstructuredVMObj, metav1.UpdateOptions{})
		if err != nil {
			gotObj, err := GetDynamicCachedResources(dynamicClient).VMLister.ByNamespace(vm.Namespace).Get(vm.Name)
			if err != nil {
				return err
			}
			got, _ := UnDynamicObject[*kubevirtcorev1.VirtualMachine](gotObj)
			if got != nil {
				vm.SetResourceVersion(got.GetResourceVersion())
			}
			return err
		}

		// update success, returrn vm
		return runtime.DefaultUnstructuredConverter.FromUnstructured(updatedObj.Object, updatedVM)
	})
	return updatedVM, err
}

func ListKubeVirtVMs(dynamicClient dynamic.Interface, namespace string) ([]*kubevirtcorev1.VirtualMachine, error) {
	listObjs, err := GetDynamicCachedResources(dynamicClient).VMLister.ByNamespace(namespace).List(labels.Everything())
	if err != nil {
		return nil, err
	}

	return UnDynamicObjectList[*kubevirtcorev1.VirtualMachine](listObjs)
}

func DeleteKubeVirtVM(dynamicClient dynamic.Interface, namespace, name string) error {
	err := dynamicClient.Resource(vmres).Namespace(namespace).Delete(context.Background(), name, metav1.DeleteOptions{})
	if err != nil && k8sErrors.IsNotFound(err) {
		return nil
	}
	return err
}

func GetKubeVirtVMI(dynamicClient dynamic.Interface, namespace, name string) (*kubevirtcorev1.VirtualMachineInstance, error) {
	getObj, err := GetDynamicCachedResources(dynamicClient).VMILister.ByNamespace(namespace).Get(name)
	if err != nil {
		return nil, err
	}

	return UnDynamicObject[*kubevirtcorev1.VirtualMachineInstance](getObj)
}

func DeleteKubeVirtVMI(dynamicClient dynamic.Interface, namespace, name string) error {
	err := dynamicClient.Resource(vmires).Namespace(namespace).Delete(context.Background(), name, metav1.DeleteOptions{})
	if err != nil && k8sErrors.IsNotFound(err) {
		return nil
	}
	return err
}
