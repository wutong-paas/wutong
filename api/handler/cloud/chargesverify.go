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

package cloud

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/pquerna/ffjson/ffjson"
	"github.com/sirupsen/logrus"
	"github.com/wutong-paas/wutong/api/handler"
	"github.com/wutong-paas/wutong/api/util"
	"github.com/wutong-paas/wutong/db"
	"github.com/wutong-paas/wutong/db/model"
)

// PubChargeSverify service Charge Sverify
func PubChargeSverify(tenant *model.Tenants, quantity int, reason string) *util.APIHandleError {
	cloudAPI := os.Getenv("CLOUD_API")
	if cloudAPI == "" {
		cloudAPI = "http://api.wutong-paas.com"
	}
	regionName := os.Getenv("REGION_NAME")
	if regionName == "" {
		return util.CreateAPIHandleError(500, fmt.Errorf("region name must define in api by env REGION_NAME"))
	}
	reason = strings.Replace(reason, " ", "%20", -1)
	api := fmt.Sprintf("%s/openapi/console/v1/enterprises/%s/memory-apply?quantity=%d&tid=%s&reason=%s&region=%s", cloudAPI, tenant.EID, quantity, tenant.UUID, reason, regionName)
	req, err := http.NewRequest("GET", api, nil)
	if err != nil {
		logrus.Error("create request cloud api error", err.Error())
		return util.CreateAPIHandleError(400, fmt.Errorf("create request cloud api error"))
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		logrus.Error("create request cloud api error", err.Error())
		return util.CreateAPIHandleError(400, fmt.Errorf("create request cloud api error"))
	}
	if res.Body != nil {
		defer res.Body.Close()
		rebody, _ := ioutil.ReadAll(res.Body)
		logrus.Debugf("read memory-apply response (%s)", string(rebody))
		var re = make(map[string]interface{})
		if err := ffjson.Unmarshal(rebody, &re); err == nil {
			if msg, ok := re["msg"]; ok {
				return util.CreateAPIHandleError(res.StatusCode, fmt.Errorf("%s", msg))
			}
		}
	}
	return util.CreateAPIHandleError(res.StatusCode, fmt.Errorf("none"))
}

// PriChargeSverify verifies that the resources requested in the private cloud are legal
func PriChargeSverify(ctx context.Context, tenant *model.Tenants, quantity int) *util.APIHandleError {
	t, err := db.GetManager().TenantDao().GetTenantByUUID(tenant.UUID)
	if err != nil {
		logrus.Errorf("error getting tenant: %v", err)
		return util.CreateAPIHandleError(500, fmt.Errorf("error getting tenant: %v", err))
	}
	if t.LimitMemory == 0 {
		clusterStats, err := handler.GetTenantManager().GetAllocatableResources(ctx)
		if err != nil {
			logrus.Errorf("error getting allocatable resources: %v", err)
			return util.CreateAPIHandleError(500, fmt.Errorf("error getting allocatable resources: %v", err))
		}
		availMem := clusterStats.AllMemory - clusterStats.RequestMemory
		if availMem >= int64(quantity) {
			return util.CreateAPIHandleError(200, fmt.Errorf("success"))
		}
		return util.CreateAPIHandleError(200, fmt.Errorf("cluster_lack_of_memory"))
	}
	tenantStas, _ := handler.GetTenantManager().GetTenantResource(tenant.UUID)
	// TODO: it should be limit, not request
	availMem := int64(t.LimitMemory) - (tenantStas.MemoryRequest + tenantStas.UnscdMemoryReq)
	if availMem >= int64(quantity) {
		return util.CreateAPIHandleError(200, fmt.Errorf("success"))
	}
	return util.CreateAPIHandleError(200, fmt.Errorf("lack_of_memory"))
}
