package controller

import (
	"net/http"

	"github.com/sirupsen/logrus"
	"github.com/wutong-paas/wutong/api/handler"
	httputil "github.com/wutong-paas/wutong/util/http"
)

type SchedulingController struct{}

func (t *NodeController) ListSchedulingNodes(w http.ResponseWriter, r *http.Request) {
	nodes, err := handler.GetSchedulingHandler().ListSchedulingNodes()
	if err != nil {
		logrus.Errorf("list nodes: %v", err)
		httputil.ReturnError(r, w, 500, err.Error())
		return
	}

	httputil.ReturnSuccess(r, w, nodes)
}

func (t *NodeController) ListSchedulingTaints(w http.ResponseWriter, r *http.Request) {
	nodes, err := handler.GetSchedulingHandler().ListSchedulingTaints()
	if err != nil {
		logrus.Errorf("list taints: %v", err)
		httputil.ReturnError(r, w, 500, err.Error())
		return
	}

	httputil.ReturnSuccess(r, w, nodes)
}

func (t *NodeController) ListVMSchedulingLabels(w http.ResponseWriter, r *http.Request) {
	labels, err := handler.GetSchedulingHandler().ListVMSchedulingLabels()
	if err != nil {
		logrus.Errorf("get vm node scheduling labels: %v", err)
		httputil.ReturnError(r, w, 500, err.Error())
		return
	}

	httputil.ReturnSuccess(r, w, labels)
}

func (t *NodeController) ListSchedulingLabels(w http.ResponseWriter, r *http.Request) {
	labels, err := handler.GetSchedulingHandler().ListSchedulingLabels()
	if err != nil {
		logrus.Errorf("get service node scheduling labels: %v", err)
		httputil.ReturnError(r, w, 500, err.Error())
		return
	}

	httputil.ReturnSuccess(r, w, labels)
}
