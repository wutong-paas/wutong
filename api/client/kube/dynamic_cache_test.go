package kube

import (
	"testing"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/dynamic"
	kubevirtcorev1 "kubevirt.io/api/core/v1"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

func TestGetKubeVirtResource(t *testing.T) {
	config := controllerruntime.GetConfigOrDie()
	dynamicClient := dynamic.NewForConfigOrDie(config)
	dcr := GetDynamicCachedResources(dynamicClient)

	ret, err := dcr.VMLister.ByNamespace("default").List(labels.Everything())
	if err != nil {
		t.Fatal(err)
	}
	// for _, v := range ret {
	// 	// unstructedVM := v.(*unstructured.Unstructured)
	// 	// vm := new(kubevirtcorev1.VirtualMachine)
	// 	// runtime.DefaultUnstructuredConverter.FromUnstructured(unstructedVM.Object, vm)
	// 	// t.Log(vm.Name)

	// 	vm, err := UnDynamicObject[*kubevirtcorev1.VirtualMachine](v)
	// 	if err != nil {
	// 		t.Fatal(err)
	// 	}
	// 	t.Log(vm.Name)
	// }

	vms, err := UnDynamicObjectList[*kubevirtcorev1.VirtualMachine](ret)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range vms {
		t.Log(v.Name)
	}
}
