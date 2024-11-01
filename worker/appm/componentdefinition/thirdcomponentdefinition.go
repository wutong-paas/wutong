// WUTONG, Application Management Platform
// Copyright (C) 2021-2021 Wutong Co., Ltd.

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

package componentdefinition

import (
	"github.com/oam-dev/kubevela/apis/core.oam.dev/common"
	"github.com/wutong-paas/wutong/pkg/apis/wutong/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var cueTemplate = `
output: {
	apiVersion: "wutong.io/v1alpha1"
	kind:       "ThirdComponent"
	metadata: {
		name: context.componentID
		namespace: context.namespace
	}
	spec: {
		endpointSource: {
			if parameter["kubernetes"] != _|_ {
				kubernetesService: {
					namespace: parameter["kubernetes"]["namespace"],
					name: parameter["kubernetes"]["name"]
				}
			}
			if parameter["endpoints"] != _|_ {
				endpoints: parameter["endpoints"]
			}
		}
		if parameter["port"] != _|_ {
			ports: parameter["port"]
		}
		if parameter["probe"] != _|_ {
			probe: parameter["probe"]
		}
	}
}

parameter: {
	kubernetes?: {
		namespace?: string
		name: string
	}
	endpoints?: [...{
		address:       string
		name?:         string
		protocol?:     string
		clientSecret?: string
	}]
	port?: [...{
		name:   string
		port:   >0 & <=65533
		openInner: bool
		openOuter: bool
	}]
	probe?: {
		httpGet?: {
			path?: string
            httpHeaders?: [...{
				name?: string
				vale?: string
			}]
		}
		tcpSocket?:{
		}
		timeoutSeconds?: >0 & <=65533
		periodSeconds?: >0 & <=65533
		successThreshold?: >0 & <=65533
		failureThreshold?: >0 & <=65533
	}
}

context: {
    componentID: string
    namespace: string
}
`
var thirdComponentDefineName = "core-thirdcomponent"
var thirdComponentDefine = v1alpha1.ComponentDefinition{
	TypeMeta: v1.TypeMeta{
		Kind:       "ComponentDefinition",
		APIVersion: "wutong.io/v1alpha1",
	},
	ObjectMeta: v1.ObjectMeta{
		Name: thirdComponentDefineName,
		Annotations: map[string]string{
			"definition.oam.dev/description": "Wutong built-in component type that defines third-party service components.",
			"version":                        "0.3",
		},
	},
	Spec: v1alpha1.ComponentDefinitionSpec{
		Workload: common.WorkloadTypeDescriptor{
			Type: "ThirdComponent",
			Definition: common.WorkloadGVK{
				APIVersion: "wutong.io/v1alpha1",
				Kind:       "ThirdComponent",
			},
		},
		Schematic: &v1alpha1.Schematic{
			CUE: &common.CUE{
				Template: cueTemplate,
			},
		},
	},
}
