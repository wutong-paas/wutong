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

package annotations

import (
	"dario.cat/mergo"
	"github.com/sirupsen/logrus"
	"github.com/wutong-paas/wutong/gateway/annotations/cookie"
	"github.com/wutong-paas/wutong/gateway/annotations/header"
	"github.com/wutong-paas/wutong/gateway/annotations/l4"
	"github.com/wutong-paas/wutong/gateway/annotations/lbtype"
	"github.com/wutong-paas/wutong/gateway/annotations/parser"
	"github.com/wutong-paas/wutong/gateway/annotations/proxy"
	"github.com/wutong-paas/wutong/gateway/annotations/resolver"
	"github.com/wutong-paas/wutong/gateway/annotations/rewrite"
	"github.com/wutong-paas/wutong/gateway/annotations/upstreamhashby"
	weight "github.com/wutong-paas/wutong/gateway/annotations/wight"
	"github.com/wutong-paas/wutong/util/ingress-nginx/ingress/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DeniedKeyName name of the key that contains the reason to deny a location
const DeniedKeyName = "Denied"

// Ingress defines the valid annotations present in one NGINX Ingress rule
type Ingress struct {
	metav1.ObjectMeta
	Header            header.Config
	Cookie            cookie.Config
	Weight            weight.Config
	Rewrite           rewrite.Config
	L4                l4.Config
	UpstreamHashBy    string
	LoadBalancingType string
	Proxy             proxy.Config
}

// Extractor defines the annotation parsers to be used in the extraction of annotations
type Extractor struct {
	annotations map[string]parser.IngressAnnotation
}

// NewAnnotationExtractor creates a new annotations extractor
func NewAnnotationExtractor(cfg resolver.Resolver) Extractor {
	return Extractor{
		map[string]parser.IngressAnnotation{
			"Header":            header.NewParser(cfg),
			"Cookie":            cookie.NewParser(cfg),
			"Weight":            weight.NewParser(cfg),
			"Rewrite":           rewrite.NewParser(cfg),
			"L4":                l4.NewParser(cfg),
			"UpstreamHashBy":    upstreamhashby.NewParser(cfg),
			"LoadBalancingType": lbtype.NewParser(cfg),
			"Proxy":             proxy.NewParser(cfg),
		},
	}
}

// Extract extracts the annotations from an Ingress
func (e Extractor) Extract(meta *metav1.ObjectMeta) *Ingress {
	data := make(map[string]interface{})
	for name, annotationParser := range e.annotations {
		val, err := annotationParser.Parse(meta)
		if err != nil {
			if errors.IsMissingAnnotations(err) {
				continue
			}

			if !errors.IsLocationDenied(err) {
				continue
			}

			_, alreadyDenied := data[DeniedKeyName]
			if !alreadyDenied {
				data[DeniedKeyName] = err
				logrus.Errorf("error reading %v annotation in Ingress %v/%v: %v", name, meta.GetNamespace(), meta.GetName(), err)
				continue
			}

			logrus.Infof("error reading %v annotation in Ingress %v/%v: %v", name, meta.GetNamespace(), meta.GetName(), err)
		}

		if val != nil {
			data[name] = val
		}
	}

	pia := &Ingress{
		ObjectMeta: *meta,
	}
	err := mergo.Map(pia, data)
	if err != nil {
		logrus.Errorf("unexpected error merging extracted annotations: %v", err)
	}

	return pia
}
