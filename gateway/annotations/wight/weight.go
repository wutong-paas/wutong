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

package weight

import (
	"strconv"

	"github.com/sirupsen/logrus"
	"github.com/wutong-paas/wutong/gateway/annotations/parser"
	"github.com/wutong-paas/wutong/gateway/annotations/resolver"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Config contains weight or router
type Config struct {
	Weight int
}

type weight struct {
	r resolver.Resolver
}

// NewParser creates a new parser
func NewParser(r resolver.Resolver) parser.IngressAnnotation {
	return weight{r}
}

func (c weight) Parse(meta *metav1.ObjectMeta) (interface{}, error) {
	wstr, err := parser.GetStringAnnotation("weight", meta)
	var w int
	if err != nil || wstr == "" {
		w = 1
	} else {
		w, err = strconv.Atoi(wstr)
		if err != nil {
			logrus.Warnf("Unexpected error occurred when convert string(%s) to int: %v", wstr, err)
			w = 1
		}
	}
	return &Config{
		Weight: w,
	}, nil
}
