package helm

import (
	"encoding/json"
	"fmt"
	"testing"

	"k8s.io/client-go/tools/clientcmd"
)

func TestResources(t *testing.T) {
	resources, err := Resources("wutong-operator", "wt-system")
	if err != nil {
		fmt.Println(err.Error())
	}
	for _, resource := range resources {
		fmt.Println(resource.Kind, resource.Namespace, resource.Name)
	}
}

func TestAllResources(t *testing.T) {
	c, err := clientcmd.BuildConfigFromFlags("", "/root/.kube/config")
	if err != nil {
		t.Error(err)
	}
	kubeConfig = c
	resources, err := AllResources("wutong", "wt-system")
	if err != nil {
		fmt.Println(err.Error())
	}
	for _, resource := range resources {
		fmt.Println(resource.Info.APIVersion, resource.Info.Kind, resource.Info.Namespace, resource.Info.Name)
		// r, err := resource.ApiResource.MarshalJSON()
		// if err != nil {
		// 	fmt.Println(err.Error())
		// } else {
		// 	fmt.Println(string(r))
		// }
		fmt.Println(resource.ApiResource.GetAPIVersion(), resource.ApiResource.GetKind(), resource.ApiResource.GetNamespace(), resource.ApiResource.GetName(), resource.ApiResource.GetAnnotations()["meta.helm.sh/release-name"], resource.ApiResource.GetAnnotations()["meta.helm.sh/release-namespace"])
	}
}
func TestAllReleases(t *testing.T) {
	c, err := clientcmd.BuildConfigFromFlags("", "/root/.kube/config")
	if err != nil {
		t.Error(err)
	}
	kubeConfig = c
	releases, err := AllReleases("default")
	if err != nil {
		fmt.Println(err.Error())
	}
	for _, release := range releases {
		j, e := json.Marshal(release)
		fmt.Println(string(j), e)
	}
}
