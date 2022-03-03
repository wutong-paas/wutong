package thirdcomponent

import (
	"context"

	"github.com/sirupsen/logrus"
	"github.com/wutong-paas/wutong/pkg/apis/wutong/v1alpha1"
	dis "github.com/wutong-paas/wutong/worker/master/controller/thirdcomponent/discover"
	"github.com/wutong-paas/wutong/worker/master/controller/thirdcomponent/prober"
)

// Worker -
type Worker struct {
	discover   dis.Discover
	cancel     context.CancelFunc
	ctx        context.Context
	updateChan chan *v1alpha1.ThirdComponent
	stoped     bool

	proberManager prober.Manager
}

// Start -
func (w *Worker) Start() {
	defer func() {
		logrus.Infof("discover endpoint list worker %s/%s stoed", w.discover.GetComponent().Namespace, w.discover.GetComponent().Name)
		w.stoped = true
		if w.proberManager != nil {
			w.proberManager.Stop()
		}
	}()
	w.stoped = false
	logrus.Infof("discover endpoint list worker %s/%s  started", w.discover.GetComponent().Namespace, w.discover.GetComponent().Name)
	w.discover.Discover(w.ctx, w.updateChan)
}

// UpdateDiscover -
func (w *Worker) UpdateDiscover(discover dis.Discover) {
	component := discover.GetComponent()
	if component.Spec.IsStaticEndpoints() {
		w.proberManager.AddThirdComponent(discover.GetComponent())
		discover.SetProberManager(w.proberManager)
	}
	w.discover = discover
}

// Stop -
func (w *Worker) Stop() {
	w.cancel()
	if w.proberManager != nil {
		w.proberManager.Stop()
	}
}

// IsStop -
func (w *Worker) IsStop() bool {
	return w.stoped
}
