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

package region

import (
	"path"

	"github.com/mitchellh/mapstructure"
	"github.com/sirupsen/logrus"
	"github.com/wutong-paas/wutong/api/util"
	dbmodel "github.com/wutong-paas/wutong/db/model"
	utilhttp "github.com/wutong-paas/wutong/util/http"
)

type tenantEnv struct {
	regionImpl
	tenantEnvName string
	prefix        string
}

// TenantEnvInterface TenantEnvInterface
type TenantEnvInterface interface {
	Get() (*dbmodel.TenantEnvs, *util.APIHandleError)
	List() ([]*dbmodel.TenantEnvs, *util.APIHandleError)
	Delete() *util.APIHandleError
	Services(serviceAlias string) ServiceInterface
	// DefineSources(ss *api_model.SourceSpec) DefineSourcesInterface
	// DefineCloudAuth(gt *api_model.GetUserToken) DefineCloudAuthInterface
}

func (t *tenantEnv) Get() (*dbmodel.TenantEnvs, *util.APIHandleError) {
	var decode utilhttp.ResponseBody
	var tenantEnv dbmodel.TenantEnvs
	decode.Bean = &tenantEnv
	code, err := t.DoRequest(t.prefix, "GET", nil, &decode)
	if err != nil {
		return nil, util.CreateAPIHandleError(code, err)
	}
	return &tenantEnv, nil
}
func (t *tenantEnv) List() ([]*dbmodel.TenantEnvs, *util.APIHandleError) {
	if t.tenantEnvName != "" {
		return nil, util.CreateAPIHandleErrorf(400, "tenant env name must be empty in this api")
	}
	var decode utilhttp.ResponseBody
	code, err := t.DoRequest(t.prefix, "GET", nil, &decode)
	if err != nil {
		return nil, util.CreateAPIHandleError(code, err)
	}
	if decode.Bean == nil {
		return nil, nil
	}
	bean, ok := decode.Bean.(map[string]interface{})
	if !ok {
		logrus.Warningf("list tenantEnvs; wrong data: %v", decode.Bean)
		return nil, nil
	}
	list, ok := bean["list"]
	if !ok {
		return nil, nil
	}
	var tenantEnvs []*dbmodel.TenantEnvs
	if err := mapstructure.Decode(list, &tenantEnvs); err != nil {
		logrus.Errorf("map: %+v; error decoding to map to []*dbmodel.TenantEnvs: %v", list, err)
		return nil, util.CreateAPIHandleError(500, err)
	}
	return tenantEnvs, nil
}
func (t *tenantEnv) Delete() *util.APIHandleError {
	return nil
}
func (t *tenantEnv) Services(serviceAlias string) ServiceInterface {
	return &services{
		prefix:    path.Join(t.prefix, "services", serviceAlias),
		tenantEnv: *t,
	}
}
