// Copyright (C) 2014-2018 Wutong Co., Ltd.
// WUTONG, Application Management Platform

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

package controller

import (
	"net/http"

	"github.com/go-chi/chi"
	"github.com/sirupsen/logrus"
	"github.com/wutong-paas/wutong/api/handler"
	"github.com/wutong-paas/wutong/api/model"

	httputil "github.com/wutong-paas/wutong/util/http"
)

// NodeController -
type NodeController struct {
}

func (t *NodeController) ListNodes(w http.ResponseWriter, r *http.Request) {
	nodes, err := handler.GetNodeHandler().ListNodes()
	if err != nil {
		logrus.Errorf("list nodes: %v", err)
		httputil.ReturnError(r, w, 500, err.Error())
		return
	}

	httputil.ReturnSuccess(r, w, nodes)
}

func (t *NodeController) GetNode(w http.ResponseWriter, r *http.Request) {
	nodeName := chi.URLParam(r, "node_name")
	if nodeName == "" {
		httputil.ReturnError(r, w, 400, "node name are required")
		return
	}
	node, err := handler.GetNodeHandler().GetNode(nodeName)
	if err != nil {
		logrus.Errorf("get node: %v", err)
		httputil.ReturnError(r, w, 500, err.Error())
		return
	}
	httputil.ReturnSuccess(r, w, node)
}

func (t *NodeController) SetNodeLabel(w http.ResponseWriter, r *http.Request) {
	nodeName := chi.URLParam(r, "node_name")
	if nodeName == "" {
		httputil.ReturnError(r, w, 400, "node name is required")
		return
	}
	var req model.SetNodeLabelRequest
	ok := httputil.ValidatorRequestStructAndErrorResponse(r, w, &req, nil)
	if !ok {
		logrus.Errorf("start operation validate request body failure")
		return
	}
	err := handler.GetNodeHandler().SetNodeLabel(nodeName, &req)
	if err != nil {
		logrus.Errorf("set label: %v", err)
		httputil.ReturnError(r, w, 500, err.Error())
		return
	}

	httputil.ReturnSuccess(r, w, nil)
}

func (t *NodeController) DeleteNodeLabel(w http.ResponseWriter, r *http.Request) {
	nodeName := chi.URLParam(r, "node_name")
	if nodeName == "" {
		httputil.ReturnError(r, w, 400, "node name is required")
		return
	}
	var req model.DeleteNodeLabelRequest
	ok := httputil.ValidatorRequestStructAndErrorResponse(r, w, &req, nil)
	if !ok {
		logrus.Errorf("start operation validate request body failure")
		return
	}
	err := handler.GetNodeHandler().DeleteNodeLabel(nodeName, &req)
	if err != nil {
		logrus.Errorf("delete label: %v", err)
		httputil.ReturnError(r, w, 500, err.Error())
		return
	}

	httputil.ReturnSuccess(r, w, nil)
}

func (t *NodeController) SetNodeAnnotation(w http.ResponseWriter, r *http.Request) {
	nodeName := chi.URLParam(r, "node_name")
	if nodeName == "" {
		httputil.ReturnError(r, w, 400, "node name is required")
		return
	}
	var req model.SetNodeAnnotationRequest
	ok := httputil.ValidatorRequestStructAndErrorResponse(r, w, &req, nil)
	if !ok {
		logrus.Errorf("start operation validate request body failure")
		return
	}
	err := handler.GetNodeHandler().SetNodeAnnotation(nodeName, &req)
	if err != nil {
		logrus.Errorf("set annotation: %v", err)
		httputil.ReturnError(r, w, 500, err.Error())
		return
	}

	httputil.ReturnSuccess(r, w, nil)
}

func (t *NodeController) DeleteNodeAnnotation(w http.ResponseWriter, r *http.Request) {
	nodeName := chi.URLParam(r, "node_name")
	if nodeName == "" {
		httputil.ReturnError(r, w, 400, "node name is required")
		return
	}
	var req model.DeleteNodeAnnotationRequest
	ok := httputil.ValidatorRequestStructAndErrorResponse(r, w, &req, nil)
	if !ok {
		logrus.Errorf("start operation validate request body failure")
		return
	}
	err := handler.GetNodeHandler().DeleteNodeAnnotation(nodeName, &req)
	if err != nil {
		logrus.Errorf("delete annotation: %v", err)
		httputil.ReturnError(r, w, 500, err.Error())
		return
	}

	httputil.ReturnSuccess(r, w, nil)
}

func (*NodeController) TaintNode(w http.ResponseWriter, r *http.Request) {
	nodeName := chi.URLParam(r, "node_name")
	if nodeName == "" {
		httputil.ReturnError(r, w, 400, "node name is required")
		return
	}
	var req model.TaintNodeRequest
	ok := httputil.ValidatorRequestStructAndErrorResponse(r, w, &req, nil)
	if !ok {
		logrus.Errorf("start operation validate request body failure")
		return
	}
	err := handler.GetNodeHandler().TaintNode(nodeName, &req)
	if err != nil {
		logrus.Errorf("taint node: %v", err)
		httputil.ReturnError(r, w, 500, err.Error())
		return
	}

	httputil.ReturnSuccess(r, w, nil)
}

func (*NodeController) DeleteTaintNode(w http.ResponseWriter, r *http.Request) {
	nodeName := chi.URLParam(r, "node_name")
	if nodeName == "" {
		httputil.ReturnError(r, w, 400, "node name is required")
		return
	}
	var req model.DeleteTaintNodeRequest
	ok := httputil.ValidatorRequestStructAndErrorResponse(r, w, &req, nil)
	if !ok {
		logrus.Errorf("start operation validate request body failure")
		return
	}
	err := handler.GetNodeHandler().DeleteTaintNode(nodeName, &req)
	if err != nil {
		logrus.Errorf("untaint node: %v", err)
		httputil.ReturnError(r, w, 500, err.Error())
		return
	}

	httputil.ReturnSuccess(r, w, nil)
}

func (*NodeController) CordonNode(w http.ResponseWriter, r *http.Request) {
	nodeName := chi.URLParam(r, "node_name")
	if nodeName == "" {
		httputil.ReturnError(r, w, 400, "node name is required")
		return
	}
	var req model.CordonNodeRequest
	ok := httputil.ValidatorRequestStructAndErrorResponse(r, w, &req, nil)
	if !ok {
		logrus.Errorf("start operation validate request body failure")
		return
	}
	err := handler.GetNodeHandler().CordonNode(nodeName, &req)
	if err != nil {
		logrus.Errorf("cordon node: %v", err)
		httputil.ReturnError(r, w, 500, err.Error())
		return
	}

	httputil.ReturnSuccess(r, w, nil)
}

func (*NodeController) UncordonNode(w http.ResponseWriter, r *http.Request) {
	nodeName := chi.URLParam(r, "node_name")
	if nodeName == "" {
		httputil.ReturnError(r, w, 400, "node name is required")
		return
	}
	err := handler.GetNodeHandler().UncordonNode(nodeName)
	if err != nil {
		logrus.Errorf("uncordon node: %v", err)
		httputil.ReturnError(r, w, 500, err.Error())
		return
	}

	httputil.ReturnSuccess(r, w, nil)
}

func (t *NodeController) SetVMSchedulingLabel(w http.ResponseWriter, r *http.Request) {
	nodeName := chi.URLParam(r, "node_name")
	if nodeName == "" {
		httputil.ReturnError(r, w, 400, "node name is required")
		return
	}
	var req model.SetVMSchedulingLabelRequest
	ok := httputil.ValidatorRequestStructAndErrorResponse(r, w, &req, nil)
	if !ok {
		logrus.Errorf("start operation validate request body failure")
		return
	}
	err := handler.GetNodeHandler().SetVMSchedulingLabel(nodeName, &req)
	if err != nil {
		logrus.Errorf("add vm node scheduling label: %v", err)
		httputil.ReturnError(r, w, 500, err.Error())
		return
	}
	httputil.ReturnSuccess(r, w, nil)
}

func (t *NodeController) DeleteVMSchedulingLabel(w http.ResponseWriter, r *http.Request) {
	nodeName := chi.URLParam(r, "node_name")
	if nodeName == "" {
		httputil.ReturnError(r, w, 400, "node name are required")
		return
	}
	var req model.DeleteVMSchedulingLabelRequest
	ok := httputil.ValidatorRequestStructAndErrorResponse(r, w, &req, nil)
	if !ok {
		logrus.Errorf("start operation validate request body failure")
		return
	}
	err := handler.GetNodeHandler().DeleteVMSchedulingLabel(nodeName, &req)
	if err != nil {
		logrus.Errorf("delete vm node scheduling label: %v", err)
		httputil.ReturnError(r, w, 500, err.Error())
		return
	}
	httputil.ReturnSuccess(r, w, nil)
}
