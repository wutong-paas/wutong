package controller

import (
	"net/http"

	"github.com/go-chi/chi"

	"github.com/wutong-paas/wutong/api/handler"

	httputil "github.com/wutong-paas/wutong/util/http"
)

//GetRunningServices list all running service ids
func GetRunningServices(w http.ResponseWriter, r *http.Request) {
	enterpriseID := chi.URLParam(r, "enterprise_id")
	runningList, err := handler.GetServiceManager().GetEnterpriseRunningServices(enterpriseID)
	if err != nil {
		err.Handle(r, w)
		return
	}
	httputil.ReturnNoFomart(r, w, 200, map[string]interface{}{"service_ids": runningList})
}
