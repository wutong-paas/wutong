// WUTONG, Application Management Platform
// Copyright (C) 2014-2017 Wutong Co., Ltd.

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

package l4

import (
	"fmt"

	"github.com/wutong-paas/wutong/gateway/annotations/parser"
	"github.com/wutong-paas/wutong/gateway/annotations/resolver"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Config -
type Config struct {
	L4Enable         bool
	L4Host           string
	L4Port           int
	KeepaliveEnabled bool
	KeepaliveIdle    string // default 30m
	KeepaliveIntvl   string // deafult 75s
	KeepaliveCnt     string // default 9
}

type l4 struct {
	r resolver.Resolver
}

// NewParser -
func NewParser(r resolver.Resolver) parser.IngressAnnotation {
	return l4{r}
}

func (l l4) Parse(meta *metav1.ObjectMeta) (interface{}, error) {
	l4Enable, _ := parser.GetBoolAnnotation("l4-enable", meta)
	l4Host, _ := parser.GetStringAnnotation("l4-host", meta)
	if l4Host == "" {
		l4Host = "0.0.0.0"
	}

	l4Port, _ := parser.GetIntAnnotation("l4-port", meta)
	if l4Enable && (l4Port <= 0 || l4Port > 65535) {
		return nil, fmt.Errorf("error l4Port: %d", l4Port)
	}

	keepaliveEnabled, _ := parser.GetBoolAnnotation("keepalive-enabled", meta)
	keepaliveIdle, _ := parser.GetStringAnnotation("keepalive-idle", meta)
	keepaliveIntvl, _ := parser.GetStringAnnotation("keepalive-intvl", meta)
	keepaliveCnt, _ := parser.GetStringAnnotation("keepalive-cnt", meta)

	return &Config{
		L4Enable:         l4Enable,
		L4Host:           l4Host,
		L4Port:           l4Port,
		KeepaliveEnabled: keepaliveEnabled,
		KeepaliveIdle:    keepaliveIdle,
		KeepaliveIntvl:   keepaliveIntvl,
		KeepaliveCnt:     keepaliveCnt,
	}, nil
}
