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

package api

import (
	"net/http"

	"github.com/wutong-paas/wutong/chaos/sources"

	"github.com/wutong-paas/wutong/util"

	"strings"

	"github.com/go-chi/chi"
	"github.com/wutong-paas/wutong/chaos/api/controller"
	httputil "github.com/wutong-paas/wutong/util/http"
)

func APIServer() *chi.Mux {
	r := chi.NewRouter()
	r.Route("/v2/builder", func(r chi.Router) {
		r.Get("/publickey/{tenant_env_id}", func(w http.ResponseWriter, r *http.Request) {
			tenantEnvId := strings.TrimSpace(chi.URLParam(r, "tenant_env_id"))
			key := sources.GetPublicKey(tenantEnvId)
			bean := struct {
				Key string `json:"public_key"`
			}{
				Key: key,
			}
			httputil.ReturnSuccess(r, w, bean)
		})
		r.Route("/codecheck", func(r chi.Router) {
			r.Post("/", controller.AddCodeCheck)
			r.Put("/service/{serviceID}", controller.Update)
			r.Get("/service/{serviceID}", controller.GetCodeCheck)
		})
		r.Route("/version", func(r chi.Router) {
			r.Post("/", controller.UpdateDeliveredPath)
			r.Get("/event/{eventID}", controller.GetVersionByEventID)
			r.Post("/event/{eventID}", controller.UpdateVersionByEventID)
			r.Get("/service/{serviceID}", controller.GetVersionByServiceID)
			r.Delete("/service/{eventID}", controller.DeleteVersionByEventID)
		})
		r.Route("/event", func(r chi.Router) {
			r.Get("/", controller.GetEventsByIds)
		})
		r.Route("/health", func(r chi.Router) {
			r.Get("/", controller.CheckHalth)
		})
	})
	util.ProfilerSetup(r)
	return r
}
