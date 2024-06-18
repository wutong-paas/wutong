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

	"github.com/wutong-paas/wutong/util"

	"github.com/go-chi/chi"
	"github.com/wutong-paas/wutong/monitor/api/controller"
	httputil "github.com/wutong-paas/wutong/util/http"
)

// Server api server
func Server(c *controller.RuleControllerManager) *chi.Mux {
	r := chi.NewRouter()
	r.Route("/monitor", func(r chi.Router) {
		r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
			bean := map[string]string{"status": "health", "info": "monitor service health"}
			httputil.ReturnSuccess(r, w, bean)
		})
	})
	// r.Route("/v2/rules", func(r chi.Router) {
	// 	r.Post("/", c.AddRules)
	// 	r.Put("/{rules_name}", c.RegRules)
	// 	r.Delete("/{rules_name}", c.DelRules)
	// 	r.Get("/{rules_name}", c.GetRules)
	// 	r.Get("/all", c.GetAllRules)
	// })
	util.ProfilerSetup(r)
	return r
}
