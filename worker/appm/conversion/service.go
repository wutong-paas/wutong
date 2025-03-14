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

package conversion

import (
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/wutong-paas/wutong/api/handler/app_governance_mode/adaptor"

	"github.com/jinzhu/gorm"
	"github.com/wutong-paas/wutong/db"
	dbmodel "github.com/wutong-paas/wutong/db/model"
	"github.com/wutong-paas/wutong/util"
	v1 "github.com/wutong-paas/wutong/worker/appm/types/v1"
	"gopkg.in/yaml.v3"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ServiceSource conv ServiceSource
func ServiceSource(as *v1.AppService, dbmanager db.Manager) error {
	sscs, err := dbmanager.ServiceSourceDao().GetServiceSource(as.ServiceID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil
		}
		return fmt.Errorf("conv service source failure %s", err.Error())
	}
	for _, ssc := range sscs {
		switch ssc.SourceType {
		case "deployment":
			var dm appsv1.Deployment
			if err := decoding(ssc.SourceBody, &dm); err != nil {
				return decodeError(err)
			}
			as.SetDeployment(&dm)
		case "statefulset":
			var ss appsv1.StatefulSet
			if err := decoding(ssc.SourceBody, &ss); err != nil {
				return decodeError(err)
			}
			as.SetStatefulSet(&ss)
		case "configmap":
			var cm corev1.ConfigMap
			if err := decoding(ssc.SourceBody, &cm); err != nil {
				return decodeError(err)
			}
			as.SetConfigMap(&cm)
		}
	}
	return nil
}
func decodeError(err error) error {
	return fmt.Errorf("decode service source failure %s", err.Error())
}
func decoding(source string, target interface{}) error {
	return yaml.Unmarshal([]byte(source), target)
}
func int32Ptr(i int) *int32 {
	j := int32(i)
	return &j
}

// TenantEnvServiceBase conv tenant env service base info
func TenantEnvServiceBase(as *v1.AppService, dbmanager db.Manager) error {
	tenantEnvService, err := dbmanager.TenantEnvServiceDao().GetServiceByID(as.ServiceID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return ErrServiceNotFound
		}
		return fmt.Errorf("error getting service base info by serviceID(%s) %s", as.ServiceID, err.Error())
	}
	as.ServiceKind = dbmodel.ServiceKind(tenantEnvService.Kind)
	tenantEnv, err := dbmanager.TenantEnvDao().GetTenantEnvByUUID(tenantEnvService.TenantEnvID)
	if err != nil {
		return fmt.Errorf("get tenant env info failure %s", err.Error())
	}
	as.TenantEnvID = tenantEnvService.TenantEnvID
	if as.DeployVersion == "" {
		as.DeployVersion = tenantEnvService.DeployVersion
	}
	as.AppID = tenantEnvService.AppID
	as.ServiceAlias = tenantEnvService.ServiceAlias
	as.UpgradeMethod = v1.TypeUpgradeMethod(tenantEnvService.UpgradeMethod)
	if tenantEnvService.K8sComponentName == "" {
		tenantEnvService.K8sComponentName = tenantEnvService.ServiceAlias
	}
	as.K8sComponentName = tenantEnvService.K8sComponentName
	if as.CreaterID == "" {
		as.CreaterID = string(util.NewTimeVersion())
	}
	as.TenantEnvName = tenantEnv.Name
	if err := initTenantEnv(as, tenantEnv); err != nil {
		return fmt.Errorf("conversion tenant env info failure %s", err.Error())
	}
	if tenantEnvService.Kind == dbmodel.ServiceKindThirdParty.String() {
		disCfg, _ := dbmanager.ThirdPartySvcDiscoveryCfgDao().GetByServiceID(as.ServiceID)
		as.SetDiscoveryCfg(disCfg)
		return nil
	}

	if tenantEnvService.Kind == dbmodel.ServiceKindCustom.String() {
		return nil
	}
	label, _ := dbmanager.TenantEnvServiceLabelDao().GetLabelByNodeSelectorKey(as.ServiceID, "windows")
	if label != nil {
		as.IsWindowsService = true
	}

	// component resource config
	as.ContainerRequestCPU = tenantEnvService.ContainerRequestCPU
	as.ContainerCPU = tenantEnvService.ContainerCPU
	as.ContainerGPUType = tenantEnvService.ContainerGPUType
	as.ContainerGPU = tenantEnvService.ContainerGPU
	as.ContainerRequestMemory = tenantEnvService.ContainerRequestMemory
	as.ContainerMemory = tenantEnvService.ContainerMemory
	as.Replicas = tenantEnvService.Replicas
	if !tenantEnvService.IsState() {
		initBaseDeployment(as, tenantEnvService)
		return nil
	}
	if tenantEnvService.IsState() {
		initBaseStatefulSet(as, tenantEnvService)
		return nil
	}
	return fmt.Errorf("kind: %s; do not decision build type for service %s", tenantEnvService.Kind, as.ServiceAlias)
}

func initTenantEnv(as *v1.AppService, tenantEnv *dbmodel.TenantEnvs) error {
	if tenantEnv == nil || tenantEnv.Namespace == "" {
		return fmt.Errorf("tenant env is invalid")
	}
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   tenantEnv.Namespace,
			Labels: map[string]string{"creator": "Wutong"},
		},
	}
	as.SetTenantEnv(namespace)
	return nil
}
func initSelector(selector *metav1.LabelSelector, service *dbmodel.TenantEnvServices) {
	if selector.MatchLabels == nil {
		selector.MatchLabels = make(map[string]string)
	}
	selector.MatchLabels["name"] = service.ServiceAlias
	selector.MatchLabels["tenant_env_id"] = service.TenantEnvID
	selector.MatchLabels["service_id"] = service.ServiceID
	//selector.MatchLabels["version"] = service.DeployVersion
}
func initBaseStatefulSet(as *v1.AppService, service *dbmodel.TenantEnvServices) {
	as.ServiceType = v1.TypeStatefulSet
	stateful := as.GetStatefulSet()
	if stateful == nil {
		stateful = &appsv1.StatefulSet{}
	}
	stateful.Namespace = as.GetNamespace()
	stateful.Spec.Replicas = int32Ptr(service.Replicas)
	if stateful.Spec.Selector == nil {
		stateful.Spec.Selector = &metav1.LabelSelector{}
	}
	initSelector(stateful.Spec.Selector, service)
	stateful.Name = as.GetK8sWorkloadName()
	stateful.Spec.ServiceName = as.GetK8sWorkloadName()
	stateful.GenerateName = service.ServiceAlias
	injectLabels := getInjectLabels(as)
	stateful.Labels = as.GetCommonLabels(stateful.Labels, map[string]string{
		"name":    service.ServiceAlias,
		"version": service.DeployVersion,
	}, injectLabels)
	stateful.Spec.UpdateStrategy.Type = appsv1.RollingUpdateStatefulSetStrategyType
	if as.UpgradeMethod == v1.OnDelete {
		stateful.Spec.UpdateStrategy.Type = appsv1.OnDeleteStatefulSetStrategyType
	}
	as.SetStatefulSet(stateful)
}

func initBaseDeployment(as *v1.AppService, service *dbmodel.TenantEnvServices) {
	as.ServiceType = v1.TypeDeployment
	deployment := as.GetDeployment()
	if deployment == nil {
		deployment = &appsv1.Deployment{}
	}
	deployment.Namespace = as.GetNamespace()
	deployment.Spec.Replicas = int32Ptr(service.Replicas)
	if deployment.Spec.Selector == nil {
		deployment.Spec.Selector = &metav1.LabelSelector{}
	}
	initSelector(deployment.Spec.Selector, service)
	deployment.Name = as.GetK8sWorkloadName()
	deployment.GenerateName = strings.Replace(service.ServiceAlias, "_", "-", -1)
	injectLabels := getInjectLabels(as)
	deployment.Labels = as.GetCommonLabels(deployment.Labels, map[string]string{
		"name":    service.ServiceAlias,
		"version": service.DeployVersion,
	}, injectLabels)
	deployment.Spec.Strategy.Type = appsv1.RollingUpdateDeploymentStrategyType
	if as.UpgradeMethod == v1.OnDelete {
		deployment.Spec.Strategy.Type = appsv1.RecreateDeploymentStrategyType
	}
	as.SetDeployment(deployment)
}

func getInjectLabels(as *v1.AppService) map[string]string {
	mode, err := adaptor.NewAppGoveranceModeHandler(as.GovernanceMode, nil)
	if err != nil {
		logrus.Warningf("getInjectLabels failed: %v", err)
		return nil
	}
	injectLabels := mode.GetInjectLabels()
	return injectLabels
}
