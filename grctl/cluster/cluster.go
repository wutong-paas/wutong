// WUTONG, Application Management Platform
// Copyright (C) 2014-2021 Wutong Co., Ltd.

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

package cluster

import (
	"context"
	"fmt"
	"strings"

	"github.com/docker/distribution/reference"
	"github.com/oam-dev/kubevela/pkg/utils/apply"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/wutong-paas/wutong-operator/api/v1alpha1"
	"github.com/wutong-paas/wutong/grctl/clients"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/yaml"
)

// Cluster represents a wutong cluster.
type Cluster struct {
	wutongCluster *v1alpha1.WutongCluster
	namespace     string
	newVersion    string
}

// NewCluster creates new cluster.
func NewCluster(namespace, newVersion string) (*Cluster, error) {
	wutongCluster, err := getWutongCluster(namespace)
	if err != nil {
		return nil, err
	}
	return &Cluster{
		wutongCluster: wutongCluster,
		namespace:     namespace,
		newVersion:    newVersion,
	}, nil
}

// Upgrade upgrade cluster.
func (c *Cluster) Upgrade() error {
	logrus.Infof("upgrade cluster from %s to %s", c.wutongCluster.Spec.InstallVersion, c.newVersion)

	if errs := c.createCrds(); len(errs) > 0 {
		return errors.New(strings.Join(errs, ","))
	}

	if errs := c.updateWutongComponents(); len(errs) > 0 {
		return fmt.Errorf("update wutong components: %s", strings.Join(errs, ","))
	}

	if err := c.updateCluster(); err != nil {
		return err
	}

	return nil
}

func (c *Cluster) createCrds() []string {
	crds := c.getCrds()
	if crds == nil {
		return nil
	}
	logrus.Info("start creating crds")
	var errs []string
	for _, crd := range crds {
		if err := c.createCrd(crd); err != nil {
			errs = append(errs, err.Error())
		}
	}
	logrus.Info("crds applyed")
	return errs
}

func (c *Cluster) createCrd(crdStr string) error {
	var crd apiextensionsv1beta1.CustomResourceDefinition
	if err := yaml.Unmarshal([]byte(crdStr), &crd); err != nil {
		return fmt.Errorf("unmarshal crd: %v", err)
	}
	applyer := apply.NewAPIApplicator(clients.WutongKubeClient)
	if err := applyer.Apply(context.Background(), &crd); err != nil {
		return fmt.Errorf("apply crd: %v", err)
	}
	return nil
}

func (c *Cluster) getCrds() []string {
	for v, versionConfig := range versions {
		if strings.Contains(c.newVersion, v) {
			return versionConfig.CRDs
		}
	}
	return nil
}

func (c *Cluster) updateWutongComponents() []string {
	componentNames := []string{
		"wt-api",
		"wt-chaos",
		"wt-mq",
		"wt-eventlog",
		"wt-gateway",
		"wt-node",
		"wt-resource-proxy",
		"wt-webcli",
		"wt-worker",
		"wt-monitor",
	}
	var errs []string
	for _, name := range componentNames {
		err := c.updateWutongComponent(name)
		if err != nil {
			errs = append(errs, err.Error())
		}
	}
	return errs
}

func (c *Cluster) updateWutongComponent(name string) error {
	var cpt v1alpha1.WutongComponent
	err := clients.WutongKubeClient.Get(context.Background(),
		types.NamespacedName{Namespace: c.namespace, Name: name}, &cpt)
	if err != nil {
		return fmt.Errorf("get wtcomponent %s: %v", name, err)
	}

	ref, err := reference.Parse(cpt.Spec.Image)
	if err != nil {
		return fmt.Errorf("parse image %s: %v", cpt.Spec.Image, err)
	}
	repo := ref.(reference.Named)
	newImage := repo.Name() + ":" + c.newVersion

	oldImageName := cpt.Spec.Image
	cpt.Spec.Image = newImage
	if err := clients.WutongKubeClient.Update(context.Background(), &cpt); err != nil {
		return fmt.Errorf("update wtcomponent %s: %v", name, err)
	}

	logrus.Infof("update wtcomponent %s \nfrom %s \nto   %s", name, oldImageName, newImage)
	return nil
}

func (c *Cluster) updateCluster() error {
	c.wutongCluster.Spec.InstallVersion = c.newVersion
	if err := clients.WutongKubeClient.Update(context.Background(), c.wutongCluster); err != nil {
		return fmt.Errorf("update wutong cluster: %v", err)
	}
	logrus.Infof("update wutong cluster to %s", c.newVersion)
	return nil
}

func getWutongCluster(namespace string) (*v1alpha1.WutongCluster, error) {
	var cluster v1alpha1.WutongCluster
	err := clients.WutongKubeClient.Get(context.Background(),
		types.NamespacedName{Namespace: namespace, Name: "wutongcluster"}, &cluster)
	if err != nil {
		return nil, fmt.Errorf("get wutong cluster: %v", err)
	}
	return &cluster, nil
}
