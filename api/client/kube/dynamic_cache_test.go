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

	vms, err := UnDynamicObjectList[*kubevirtcorev1.VirtualMachine](ret)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range vms {
		t.Log(v.Name)
	}
}
