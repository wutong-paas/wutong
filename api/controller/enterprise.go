package controller

import (
	"net/http"

	"github.com/wutong-paas/wutong/api/handler"

	httputil "github.com/wutong-paas/wutong/util/http"
)

// GetRunningServices list all running service ids
func GetRunningServices(w http.ResponseWriter, r *http.Request) {
	runningList, err := handler.GetServiceManager().GetAllRunningServices()
	if err != nil {
		err.Handle(r, w)
		return
	}
	httputil.ReturnNoFomart(r, w, 200, map[string]interface{}{"service_ids": runningList})
}

func GetServicesStatus(w http.ResponseWriter, r *http.Request) {
	servicesStatus, err := handler.GetServiceManager().GetAllServicesStatus()
	if err != nil {
		err.Handle(r, w)
		return
	}
	httputil.ReturnNoFomart(r, w, 200, servicesStatus)
}

func GetServicesStatusWithFormat(w http.ResponseWriter, r *http.Request) {
	servicesStatus, err := handler.GetServiceManager().GetAllServicesStatus()
	if err != nil {
		err.Handle(r, w)
		return
	}
	httputil.ReturnSuccess(r, w, servicesStatus)
}
