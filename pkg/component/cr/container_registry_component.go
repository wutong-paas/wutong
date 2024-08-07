package cr

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	wutongv1alpha1 "github.com/wutong-paas/wutong-operator/api/v1alpha1"
	"github.com/wutong-paas/wutong/chaos/sources/registry"
	"github.com/wutong-paas/wutong/config/configs"
	"github.com/wutong-paas/wutong/pkg/component/k8s"
	"github.com/wutong-paas/wutong/pkg/gogo"
	"github.com/wutong-paas/wutong/wtctl/clients"
	"k8s.io/apimachinery/pkg/types"
)

var defaultRegistryComponent *RegistryComponent

// RegistryComponent -
type RegistryComponent struct {
	RegistryCli *registry.Registry
}

// HubRegistry -
func HubRegistry() *RegistryComponent {
	defaultRegistryComponent = &RegistryComponent{}
	return defaultRegistryComponent
}

// Start -
func (r *RegistryComponent) Start(ctx context.Context, cfg *configs.Config) error {
	logrus.Infof("init hub registry...")
	var cluster wutongv1alpha1.WutongCluster

	err := clients.K8SClientInitClient(k8s.Default().Clientset, k8s.Default().RestConfig)
	if err != nil {
		logrus.Errorf("k8s client init wutongClient failure: %v", err)
		return err
	}
	if err := clients.WutongKubeClient.Get(context.Background(), types.NamespacedName{Namespace: "wt-system", Name: "wutongcluster"}, &cluster); err != nil {
		return errors.Wrap(err, "get configuration from wutong cluster")
	}

	registryConfig := cluster.Spec.ImageHub
	if registryConfig.Domain == "wutong.me" {
		registryConfig.Domain = cfg.APIConfig.WtHub
	}

	gogo.Go(func(ctx context.Context) error {
		var err error
		for {
			r.RegistryCli, err = registry.NewInsecure(registryConfig.Domain, registryConfig.Username, registryConfig.Password)
			if err == nil {
				logrus.Infof("create hub client success")
				return nil
			}
			logrus.Errorf("create hub client failed, try time is %d,%s", 10, err.Error())
			time.Sleep(10 * time.Second)
		}
	})
	return nil
}

// CloseHandle -
func (r *RegistryComponent) CloseHandle() {

}

// Default -
func Default() *RegistryComponent {
	return defaultRegistryComponent
}
