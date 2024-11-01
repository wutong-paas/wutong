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

package handler

import (
	"context"
	"fmt"
	"time"

	api_model "github.com/wutong-paas/wutong/api/model"
	clientv3 "go.etcd.io/etcd/client/v3"

	"testing"
)

func TestStoreETCD(t *testing.T) {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{"127.0.0.1:2379"},
		DialTimeout: 10 * time.Second,
	})
	if err != nil {
		t.Error(err)
	}
	nra := &NetRulesAction{
		etcdCli: cli,
	}
	rules := &api_model.NetDownStreamRules{
		Limit: 1024,
		//Header: "E1:V1,E2:V2",
		//Domain: "test.redis.com",
		//Prefix: "/redis",
	}

	srs := &api_model.SetNetDownStreamRuleStruct{
		TenantEnvName: "123",
		ServiceAlias:  "wttest12",
	}
	srs.Body.DestService = "redis"
	srs.Body.DestServiceAlias = "wttest34"
	srs.Body.Port = 6379
	srs.Body.Protocol = "tcp"
	srs.Body.Rules = rules

	tenantEnvID := "tenantEnvid1b50sfadfadfafadfadfadf"

	if err := nra.CreateDownStreamNetRules(tenantEnvID, srs); err != nil {
		t.Error(err)
	}

	k := fmt.Sprintf("/netRules/%s/%s/downstream/%s/%v",
		tenantEnvID, srs.ServiceAlias, srs.Body.DestServiceAlias, srs.Body.Port)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	v, err := cli.Get(ctx, k)
	cancel()
	if err != nil {
		t.Error(err)
	}
	fmt.Printf("v is %v\n", string(v.Kvs[0].Value))
}
