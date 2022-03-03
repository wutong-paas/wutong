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

package proxy

import "net/http"

//Proxy proxy
type Proxy interface {
	Proxy(w http.ResponseWriter, r *http.Request)
	Do(r *http.Request) (*http.Response, error)
	UpdateEndpoints(endpoints ...string) // format: ["name=>ip:port", ...]
}

//CreateProxy create proxy
func CreateProxy(name string, mode string, endpoints []string) Proxy {
	switch mode {
	case "websocket":
		return createWebSocketProxy(name, endpoints)
	case "http":
		if name == "eventlog" {
			return createHTTPProxy(name, endpoints, NewSelectBalance())
		}
		return createHTTPProxy(name, endpoints, nil)
	default:
		return createHTTPProxy(name, endpoints, nil)
	}
}
