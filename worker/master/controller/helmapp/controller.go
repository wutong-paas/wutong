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

package helmapp

import (
	"context"

	"github.com/sirupsen/logrus"
	"github.com/wutong-paas/wutong/pkg/generated/clientset/versioned"
	"github.com/wutong-paas/wutong/pkg/generated/listers/wutong/v1alpha1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

// Controller -
type Controller struct {
	storer      Storer
	stopCh      chan struct{}
	controlLoop *ControlLoop
	finalizer   *Finalizer
}

// NewController creates a new helm app controller.
func NewController(ctx context.Context,
	stopCh chan struct{},
	kubeClient clientset.Interface,
	clientset versioned.Interface,
	informer cache.SharedIndexInformer,
	lister v1alpha1.HelmAppLister,
	repoFile, repoCache, chartCache string) *Controller {
	workQueue := workqueue.New()
	finalizerQueue := workqueue.New()
	storer := NewStorer(informer, lister, workQueue, finalizerQueue)

	controlLoop := NewControlLoop(ctx, kubeClient, clientset, storer, workQueue, repoFile, repoCache, chartCache)
	finalizer := NewFinalizer(ctx, kubeClient, clientset, finalizerQueue, repoFile, repoCache, chartCache)

	return &Controller{
		storer:      storer,
		stopCh:      stopCh,
		controlLoop: controlLoop,
		finalizer:   finalizer,
	}
}

// Start starts the controller.
func (c *Controller) Start() {
	logrus.Info("start helm app controller")
	c.storer.Run(c.stopCh)
	go c.controlLoop.Run()
	c.finalizer.Run()
}

// Stop stops the controller.
func (c *Controller) Stop() {
	c.controlLoop.Stop()
	c.finalizer.Stop()
}
