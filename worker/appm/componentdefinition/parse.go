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
	"encoding/json"
	"fmt"
	"strings"
	"unicode"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/build"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/parser"
	"github.com/kubevela/workflow/pkg/cue/model"
	"github.com/pkg/errors"
	v1 "github.com/wutong-paas/wutong/worker/appm/types/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const (
	// ParameterTag is the keyword in CUE template to define users' input
	ParameterTag = "parameter"
	// OutputFieldName is the reference of context base object
	OutputFieldName = "output"
	// OutputsFieldName is the reference of context Auxiliaries
	OutputsFieldName = "outputs"
	// ConfigFieldName is the reference of context config
	ConfigFieldName = "config"
	// ContextName is the name of context
	ContextName = "name"
	// ContextAppName is the appName of context
	ContextAppName = "appName"
	// ContextID is the componentID of context
	ContextID = "componentID"
	// ContextAppID is the appID of context
	ContextAppID = "appID"
	// ContextNamespace is the namespace of the app
	ContextNamespace = "namespace"
)

type TemplateContext struct {
	as            *v1.AppService
	componentName string
	appName       string
	componentID   string
	appID         string
	namespace     string
	template      string
	params        interface{}
}

func NewTemplateContext(as *v1.AppService, template string, params interface{}) *TemplateContext {
	return &TemplateContext{
		as:            as,
		componentName: as.ServiceAlias,
		appName:       as.AppID,
		componentID:   as.ServiceID,
		appID:         as.AppID,
		namespace:     as.GetNamespace(),
		template:      template,
		params:        params,
	}
}

func (c *TemplateContext) GenerateComponentManifests() ([]*unstructured.Unstructured, error) {
	bi := build.NewContext().NewInstance("", nil)
	templateFile, err := parser.ParseFile("-", c.template)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to parse cue template of component %s", c.componentID)
	}
	if err := bi.AddSyntax(templateFile); err != nil {
		return nil, errors.WithMessagef(err, "invalid cue template of component %s", c.componentID)
	}
	var param = "parameter: {}"
	if c.params != nil {
		bt, err := json.Marshal(c.params)
		if err != nil {
			return nil, errors.WithMessagef(err, "marshal parameter of component %s", c.componentID)
		}
		if string(bt) != "null" {
			param = fmt.Sprintf("%s: %s", ParameterTag, string(bt))
		}
	}
	paramFile, err := parser.ParseFile("parameter", param)
	if err != nil {
		return nil, errors.WithMessagef(err, "invalid cue parameter of component %s", c.componentID)
	}
	if err := bi.AddSyntax(paramFile); err != nil {
		return nil, errors.WithMessagef(err, "invalid parameter of component %s", c.componentID)
	}

	context := c.ExtendedContextFile()
	contextFile, err := parser.ParseFile("-", context)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to parse cue context of component %s", c.componentID)
	}

	if err := bi.AddSyntax(contextFile); err != nil {
		return nil, err
	}

	cueValue := cuecontext.New().BuildInstance(bi)
	if err := cueValue.Validate(); err != nil {
		return nil, errors.WithMessagef(err, "invalid cue template of component %s after merge parameter and context", c.componentID)
	}

	output := cueValue.LookupPath(cue.ParsePath(OutputFieldName))

	base, err := model.NewBase(output)
	if err != nil {
		return nil, errors.WithMessagef(err, "invalid output of component %s", c.componentID)
	}
	workload, err := base.Unstructured()
	if err != nil {
		return nil, errors.WithMessagef(err, "invalid output of component %s", c.componentID)
	}

	manifests := []*unstructured.Unstructured{workload}

	outputs := cueValue.LookupPath(cue.ParsePath(OutputsFieldName))
	if !outputs.Exists() {
		return manifests, nil
	}

	fields, err := outputs.Value().Fields()
	if err != nil {
		return nil, errors.WithMessagef(err, "invalid outputs of workload %s", c.componentID)
	}
	for fields.Next() {
		if fields.Selector().IsDefinition() || fields.Selector().PkgPath() != "" || fields.IsOptional() {
			continue
		}
		other, err := model.NewOther(fields.Value())
		if err != nil {
			return nil, errors.WithMessagef(err, "invalid outputs of workload %s", c.componentID)
		}
		othermanifest, err := other.Unstructured()
		if err != nil {
			return nil, errors.WithMessagef(err, "invalid outputs of workload %s", c.componentID)
		}
		manifests = append(manifests, othermanifest)
	}

	return manifests, nil
}

func (c *TemplateContext) SetContextValue(manifests []*unstructured.Unstructured) {
	for i := range manifests {
		manifests[i].SetNamespace(c.namespace)
		manifests[i].SetLabels(c.as.GetCommonLabels(manifests[i].GetLabels()))
	}
}
func (c *TemplateContext) ExtendedContextFile() string {
	var buff string
	buff += fmt.Sprintf(ContextName+": \"%s\"\n", c.componentName)
	buff += fmt.Sprintf(ContextAppName+": \"%s\"\n", c.appName)
	buff += fmt.Sprintf(ContextNamespace+": \"%s\"\n", c.namespace)
	buff += fmt.Sprintf(ContextAppID+": \"%s\"\n", c.appID)
	buff += fmt.Sprintf(ContextID+": \"%s\"\n", c.componentID)

	return fmt.Sprintf("context: %s", structMarshal(buff))
}

func structMarshal(v string) string {
	skip := false
	v = strings.TrimFunc(v, func(r rune) bool {
		if !skip {
			if unicode.IsSpace(r) {
				return true
			}
			skip = true

		}
		return false
	})

	if strings.HasPrefix(v, "{") {
		return v
	}
	return fmt.Sprintf("{%s}", v)
}
