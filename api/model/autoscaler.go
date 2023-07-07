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

package model

import dbmodel "github.com/wutong-paas/wutong/db/model"

// AutoscalerRuleReq -
type AutoscalerRuleReq struct {
	RuleID      string `json:"rule_id" validate:"rule_id|required"`
	ServiceID   string
	Enable      bool   `json:"enable" validate:"enable|required"`
	XPAType     string `json:"xpa_type" validate:"xpa_type|required"`
	MinReplicas int    `json:"min_replicas" validate:"min_replicas|required"`
	MaxReplicas int    `json:"max_replicas" validate:"min_replicas|required"`
	Metrics     []struct {
		MetricsType       string `json:"metric_type"`
		MetricsName       string `json:"metric_name"`
		MetricTargetType  string `json:"metric_target_type"`
		MetricTargetValue int    `json:"metric_target_value"`
	} `json:"metrics"`
}

// AutoscalerRuleResp -
type AutoscalerRuleResp struct {
	RuleID      string `json:"rule_id"`
	ServiceID   string `json:"service_id"`
	Enable      bool   `json:"enable"`
	XPAType     string `json:"xpa_type"`
	MinReplicas int    `json:"min_replicas"`
	MaxReplicas int    `json:"max_replicas"`
	Metrics     []struct {
		MetricsType       string `json:"metric_type"`
		MetricsName       string `json:"metric_name"`
		MetricTargetType  string `json:"metric_target_type"`
		MetricTargetValue int    `json:"metric_target_value"`
	} `json:"metrics"`
}

// AutoScalerRule -
type AutoScalerRule struct {
	RuleID      string       `json:"rule_id"`
	Enable      bool         `json:"enable"`
	XPAType     string       `json:"xpa_type"`
	MinReplicas int          `json:"min_replicas"`
	MaxReplicas int          `json:"max_replicas"`
	RuleMetrics []RuleMetric `json:"metrics"`
}

// DbModel return database model
func (a AutoScalerRule) DbModel(componentID string) *dbmodel.TenantEnvServiceAutoscalerRules {
	return &dbmodel.TenantEnvServiceAutoscalerRules{
		RuleID:      a.RuleID,
		ServiceID:   componentID,
		MinReplicas: a.MinReplicas,
		MaxReplicas: a.MaxReplicas,
		Enable:      a.Enable,
		XPAType:     a.XPAType,
	}
}

// RuleMetric -
type RuleMetric struct {
	MetricsType       string `json:"metric_type"`
	MetricsName       string `json:"metric_name"`
	MetricTargetType  string `json:"metric_target_type"`
	MetricTargetValue int    `json:"metric_target_value"`
}

// DbModel return database model
func (r RuleMetric) DbModel(ruleID string) *dbmodel.TenantEnvServiceAutoscalerRuleMetrics {
	return &dbmodel.TenantEnvServiceAutoscalerRuleMetrics{
		RuleID:            ruleID,
		MetricsType:       r.MetricsType,
		MetricsName:       r.MetricsName,
		MetricTargetType:  r.MetricTargetType,
		MetricTargetValue: r.MetricTargetValue,
	}
}
