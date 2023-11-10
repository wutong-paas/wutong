package kube

import (
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/tools/cache"
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
	dynamicInformers := dynamicinformer.NewDynamicSharedInformerFactory(dynamicClient, time.Hour*8)

	// informer
	vmInformer := dynamicInformers.ForResource(vmres)
	vmiInformer := dynamicInformers.ForResource(vmires)

	// shared informers
	vmSharedInformer := vmInformer.Informer()
	vmiSharedInformer := vmiInformer.Informer()

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
	_ = corev1.Pod{}
	if obj == nil {
		return result, fmt.Errorf("obj is nil")
	}
	unstructuredObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return result, err
	}

	// unstructedVM := obj.(*unstructured.Unstructured)
	// vm := new(kubevirtcorev1.VirtualMachine)
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObj, &result)
	// if err!=nil{
	// 	return result, err
	// }
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
