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

	"github.com/pquerna/ffjson/ffjson"
	clientv3 "go.etcd.io/etcd/client/v3"

	"github.com/sirupsen/logrus"
	api_model "github.com/wutong-paas/wutong/api/model"
	"github.com/wutong-paas/wutong/api/util"
)

// NetRulesAction  rules action struct
type NetRulesAction struct {
	etcdCli *clientv3.Client
}

// CreateNetRulesManager get net rules manager
func CreateNetRulesManager(etcdCli *clientv3.Client) *NetRulesAction {
	return &NetRulesAction{
		etcdCli: etcdCli,
	}
}

// CreateDownStreamNetRules CreateDownStreamNetRules
func (n *NetRulesAction) CreateDownStreamNetRules(
	tenantEnvID string,
	rs *api_model.SetNetDownStreamRuleStruct) *util.APIHandleError {
	k := fmt.Sprintf("/netRules/%s/%s/downstream/%s/%v",
		tenantEnvID, rs.ServiceAlias, rs.Body.DestServiceAlias, rs.Body.Port)
	sb := &api_model.NetRulesDownStreamBody{
		DestService:      rs.Body.DestService,
		DestServiceAlias: rs.Body.DestServiceAlias,
		Port:             rs.Body.Port,
		Protocol:         rs.Body.Protocol,
		Rules:            rs.Body.Rules,
	}
	v, err := ffjson.Marshal(sb)
	if err != nil {
		logrus.Errorf("mashal etcd value error, %v", err)
		return util.CreateAPIHandleError(500, err)
	}
	_, err = n.etcdCli.Put(context.TODO(), k, string(v))
	if err != nil {
		logrus.Errorf("put k %s into etcd error, %v", k, err)
		return util.CreateAPIHandleError(500, err)
	}
	//TODO: store mysql
	return nil
}

// GetDownStreamNetRule GetDownStreamNetRule
func (n *NetRulesAction) GetDownStreamNetRule(
	tenantEnvID,
	serviceAlias,
	destServiceAlias,
	port string) (*api_model.NetRulesDownStreamBody, *util.APIHandleError) {
	k := fmt.Sprintf(
		"/netRules/%s/%s/downstream/%s/%v",
		tenantEnvID,
		serviceAlias,
		destServiceAlias,
		port)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	resp, err := n.etcdCli.Get(ctx, k)
	cancel()
	if err != nil {
		logrus.Errorf("get etcd value error, %v", err)
		return nil, util.CreateAPIHandleError(500, err)
	}
	if resp.Count != 0 {
		v := resp.Kvs[0].Value
		var sb api_model.NetRulesDownStreamBody
		if err := ffjson.Unmarshal(v, &sb); err != nil {
			logrus.Errorf("unmashal etcd v error, %v", err)
			return nil, util.CreateAPIHandleError(500, err)
		}
		return &sb, nil
	}
	//TODO: query mysql
	//TODO: create etcd record
	return nil, nil
}

// UpdateDownStreamNetRule UpdateDownStreamNetRule
func (n *NetRulesAction) UpdateDownStreamNetRule(
	tenantEnvID string,
	urs *api_model.UpdateNetDownStreamRuleStruct) *util.APIHandleError {

	srs := &api_model.SetNetDownStreamRuleStruct{
		TenantEnvName: urs.TenantEnvName,
		ServiceAlias:  urs.ServiceAlias,
	}
	srs.Body.DestService = urs.Body.DestService
	srs.Body.DestServiceAlias = urs.DestServiceAlias
	srs.Body.Port = urs.Port
	srs.Body.Protocol = urs.Body.Protocol
	srs.Body.Rules = urs.Body.Rules

	//TODO: update mysql transaction
	if err := n.CreateDownStreamNetRules(tenantEnvID, srs); err != nil {
		return err
	}
	return nil
}
