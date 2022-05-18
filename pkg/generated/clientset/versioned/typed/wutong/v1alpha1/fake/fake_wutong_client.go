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

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	v1alpha1 "github.com/wutong-paas/wutong/pkg/generated/clientset/versioned/typed/wutong/v1alpha1"
	rest "k8s.io/client-go/rest"
	testing "k8s.io/client-go/testing"
)

type FakeWutongV1alpha1 struct {
	*testing.Fake
}

func (c *FakeWutongV1alpha1) ComponentDefinitions(namespace string) v1alpha1.ComponentDefinitionInterface {
	return &FakeComponentDefinitions{c, namespace}
}

func (c *FakeWutongV1alpha1) HelmApps(namespace string) v1alpha1.HelmAppInterface {
	return &FakeHelmApps{c, namespace}
}

func (c *FakeWutongV1alpha1) ThirdComponents(namespace string) v1alpha1.ThirdComponentInterface {
	return &FakeThirdComponents{c, namespace}
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *FakeWutongV1alpha1) RESTClient() rest.Interface {
	var ret *rest.RESTClient
	return ret
}