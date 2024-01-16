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

package v1

import (
	"fmt"
	"strconv"
	"strings"

	monitorv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/sirupsen/logrus"
	dbmodel "github.com/wutong-paas/wutong/db/model"
	"github.com/wutong-paas/wutong/event"
	"github.com/wutong-paas/wutong/util/k8s"
	v1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	betav1 "k8s.io/api/networking/v1beta1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// EventType type of event
type EventType string

const (
	// StartEvent event about to start third-party service
	StartEvent EventType = "START"
	// StopEvent event about to stop third-party service
	StopEvent EventType = "STOP"
)

// Event holds the context of a start event.
type Event struct {
	Type    EventType
	Sid     string // service id
	Port    int
	IsInner bool
}

// AppServiceStatus the status of service, calculate in real time from kubernetes
type AppServiceStatus string

// AppServiceType the deploy type of service.
type AppServiceType string

// TypeStatefulSet statefulset
var TypeStatefulSet AppServiceType = "statefulset"

// TypeDeployment deployment
var TypeDeployment AppServiceType = "deployment"

// TypeReplicationController rc
var TypeReplicationController AppServiceType = "replicationcontroller"

// TypeUpgradeMethod upgrade service method type
type TypeUpgradeMethod string

// Rolling Start the new version before stoping the old version the rolling upgrade
var Rolling TypeUpgradeMethod = "Rolling"

// OnDelete Stop the old version before starting the new version the upgrade
var OnDelete TypeUpgradeMethod = "OnDelete"

// AppServiceBase app service base info
type AppServiceBase struct {
	TenantEnvID            string
	TenantEnvName          string
	AppID                  string
	ServiceID              string
	ServiceAlias           string
	ServiceType            AppServiceType
	ServiceKind            dbmodel.ServiceKind
	discoveryCfg           *dbmodel.ThirdPartySvcDiscoveryCfg
	DeployVersion          string
	ContainerRequestCPU    int
	ContainerCPU           int
	ContainerRequestMemory int
	ContainerMemory        int
	ContainerGPUType       string
	ContainerGPU           int
	UpgradeMethod          TypeUpgradeMethod
	Replicas               int
	NeedProxy              bool
	IsWindowsService       bool
	CreaterID              string
	//depend all service id
	Dependces        []string
	ExtensionSet     map[string]string
	GovernanceMode   string
	K8sApp           string
	K8sComponentName string
}

// GetComponentDefinitionName get component definition name by component kind
func (a AppServiceBase) GetComponentDefinitionName() string {
	if strings.HasPrefix(a.ServiceKind.String(), dbmodel.ServiceKindCustom.String()) {
		return strings.Replace(a.ServiceKind.String(), dbmodel.ServiceKindCustom.String(), "", 1)
	}
	if a.ServiceKind == dbmodel.ServiceKindThirdParty {
		return "core-thirdcomponent"
	}
	return ""
}

// IsCustomComponent -
func (a AppServiceBase) IsCustomComponent() bool {
	if strings.HasPrefix(a.ServiceKind.String(), dbmodel.ServiceKindCustom.String()) {
		return true
	}
	if a.ServiceKind == dbmodel.ServiceKindThirdParty {
		return true
	}
	return false
}

// IsThirdComponent -
func (a AppServiceBase) IsThirdComponent() bool {
	return a.ServiceKind.String() == dbmodel.ServiceKindThirdParty.String()
}

// SetDiscoveryCfg -
func (a *AppServiceBase) SetDiscoveryCfg(discoveryCfg *dbmodel.ThirdPartySvcDiscoveryCfg) {
	a.discoveryCfg = discoveryCfg
}

// SetDiscoveryCfg -
func (a *AppServiceBase) GetK8sWorkloadName() string {
	return fmt.Sprintf("%s-%s", a.K8sApp, a.K8sComponentName)
}

// AppService a service of wutong app state in kubernetes
type AppService struct {
	AppServiceBase
	tenantEnv        *corev1.Namespace
	statefulset      *v1.StatefulSet
	deployment       *v1.Deployment
	workload         client.Object
	hpas             []*autoscalingv1.HorizontalPodAutoscaler
	delHPAs          []*autoscalingv1.HorizontalPodAutoscaler
	replicasets      []*v1.ReplicaSet
	services         []*corev1.Service
	delServices      []*corev1.Service
	endpoints        []*corev1.Endpoints
	configMaps       []*corev1.ConfigMap
	ingresses        []*networkingv1.Ingress
	delIngs          []*networkingv1.Ingress // ingresses which need to be deleted
	betaIngresses    []*betav1.Ingress
	delBetaIngresses []*betav1.Ingress // ingresses which need to be deleted
	secrets          []*corev1.Secret
	delSecrets       []*corev1.Secret // secrets which need to be deleted
	pods             []*corev1.Pod
	claims           []*corev1.PersistentVolumeClaim
	serviceMonitor   []*monitorv1.ServiceMonitor
	// claims that needs to be created manually
	claimsmanual     []*corev1.PersistentVolumeClaim
	podMemoryRequest int64
	podCPURequest    int64
	BootSeqContainer *corev1.Container
	Logger           event.Logger
	storageClasses   []*storagev1.StorageClass
	UpgradePatch     map[string][]byte
	CustomParams     map[string]string
	envVarSecrets    []*corev1.Secret
	// custom componentdefinition output manifests
	manifests []*unstructured.Unstructured
}

// CacheKey app cache key
type CacheKey string

// Equal cache key serviceid and version and createID Equal
func (c CacheKey) Equal(end CacheKey) bool {
	return string(c) == string(end)
}

// GetCacheKeyOnlyServiceID get cache key only service id
func GetCacheKeyOnlyServiceID(serviceID string) CacheKey {
	return CacheKey(serviceID)
}

// GetDeployment get kubernetes deployment model
func (a AppService) GetDeployment() *v1.Deployment {
	return a.deployment
}

// SetDeployment set kubernetes deployment model
func (a *AppService) SetDeployment(d *v1.Deployment) {
	a.deployment = d
	a.workload = d
	if v, ok := d.Spec.Template.Labels["version"]; ok && v != "" {
		a.DeployVersion = v
	}
	a.Replicas = int(*d.Spec.Replicas)
	a.calculateComponentMemoryRequest()
}

// DeleteDeployment delete kubernetes deployment model
func (a *AppService) DeleteDeployment(d *v1.Deployment) {
	a.deployment = nil
}

// GetStatefulSet get kubernetes statefulset model
func (a AppService) GetStatefulSet() *v1.StatefulSet {
	return a.statefulset
}

// SetStatefulSet set kubernetes statefulset model
func (a *AppService) SetStatefulSet(d *v1.StatefulSet) {
	a.statefulset = d
	a.workload = d
	if v, ok := d.Spec.Template.Labels["version"]; ok && v != "" {
		a.DeployVersion = v
	}
	a.Replicas = int(*d.Spec.Replicas)
	a.calculateComponentMemoryRequest()
}

// SetReplicaSets set kubernetes replicaset
func (a *AppService) SetReplicaSets(d *v1.ReplicaSet) {
	if len(a.replicasets) > 0 {
		for i, replicaset := range a.replicasets {
			if replicaset.GetName() == d.GetName() {
				a.replicasets[i] = d
				return
			}
		}
	}
	a.replicasets = append(a.replicasets, d)
}

// DeleteReplicaSet delete replicaset
func (a *AppService) DeleteReplicaSet(d *v1.ReplicaSet) {
	for i, c := range a.replicasets {
		if c.GetName() == d.GetName() {
			a.replicasets = append(a.replicasets[0:i], a.replicasets[i+1:]...)
			return
		}
	}
}

// GetReplicaSets get replicaset
func (a *AppService) GetReplicaSets() []*v1.ReplicaSet {
	return a.replicasets
}

// GetNewestReplicaSet returns the newest replica set.
func (a *AppService) GetNewestReplicaSet() (newest *v1.ReplicaSet) {
	if len(a.replicasets) == 0 {
		return
	}
	newest = a.replicasets[0]
	for _, rs := range a.replicasets {
		if newest.ObjectMeta.CreationTimestamp.Before(&rs.ObjectMeta.CreationTimestamp) {
			newest = rs
		}
	}
	return
}

// GetReplicaSetVersion get rs version
func GetReplicaSetVersion(rs *v1.ReplicaSet) int {
	if version, ok := rs.Annotations["deployment.kubernetes.io/revision"]; ok {
		v, _ := strconv.Atoi(version)
		return v
	}
	return 0
}

// GetCurrentReplicaSet get current replicaset
func (a *AppService) GetCurrentReplicaSet() *v1.ReplicaSet {
	if a.deployment != nil {
		revision, ok := a.deployment.Annotations["deployment.kubernetes.io/revision"]
		if ok {
			for _, rs := range a.replicasets {
				if rs.Annotations["deployment.kubernetes.io/revision"] == revision {
					return rs
				}
			}
		}
	}
	return nil
}

// DeleteStatefulSet set kubernetes statefulset model
func (a *AppService) DeleteStatefulSet(d *v1.StatefulSet) {
	a.statefulset = nil
}

// SetConfigMap set kubernetes configmap model
func (a *AppService) SetConfigMap(d *corev1.ConfigMap) {
	if len(a.configMaps) > 0 {
		for i, configMap := range a.configMaps {
			if configMap.GetName() == d.GetName() {
				a.configMaps[i] = d
				return
			}
		}
	}
	a.configMaps = append(a.configMaps, d)
}

// GetConfigMaps get configmaps
func (a *AppService) GetConfigMaps() []*corev1.ConfigMap {
	if len(a.configMaps) > 0 {
		return a.configMaps
	}
	return nil
}

// DeleteConfigMaps delete configmaps
func (a *AppService) DeleteConfigMaps(config *corev1.ConfigMap) {
	for i, c := range a.configMaps {
		if c.GetName() == config.GetName() {
			a.configMaps = append(a.configMaps[0:i], a.configMaps[i+1:]...)
			return
		}
	}
}

// SetService set kubernetes service model
func (a *AppService) SetService(d *corev1.Service) {
	if len(a.services) > 0 {
		for i, service := range a.services {
			if service.GetName() == d.GetName() {
				a.services[i] = d
				return
			}
		}
	}
	a.services = append(a.services, d)
}

// SetServices set set k8s service model list
func (a *AppService) SetServices(svcs []*corev1.Service) {
	a.services = svcs
}

// GetServices get services
func (a *AppService) GetServices(canCopy bool) []*corev1.Service {
	if canCopy {
		return append(a.services[:0:0], a.services...)
	}
	return a.services
}

// GetDelServices returns services that need to be deleted.
func (a *AppService) GetDelServices() []*corev1.Service {
	return a.delServices
}

// DeleteServices delete service
func (a *AppService) DeleteServices(service *corev1.Service) {
	for i, c := range a.services {
		if c.GetName() == service.GetName() {
			a.services = append(a.services[0:i], a.services[i+1:]...)
			return
		}
	}
}

// AddEndpoints adds k8s endpoints to receiver *AppService.
func (a *AppService) AddEndpoints(ep *corev1.Endpoints) {
	if len(a.endpoints) > 0 {
		for i, e := range a.endpoints {
			if e.GetName() == ep.GetName() {
				a.endpoints[i] = ep
				return
			}
		}
	}
	a.endpoints = append(a.endpoints, ep)
}

// GetEndpoints returns endpoints in AppService
func (a *AppService) GetEndpoints(canCopy bool) []*corev1.Endpoints {
	if canCopy {
		return append(a.endpoints[:0:0], a.endpoints...)
	}
	return a.endpoints
}

// GetEndpointsByName returns endpoints in AppService
func (a *AppService) GetEndpointsByName(name string) *corev1.Endpoints {
	for _, ep := range a.endpoints {
		if ep.GetName() == name {
			return ep
		}
	}
	return nil
}

// DelEndpoints deletes *corev1.Endpoints
func (a *AppService) DelEndpoints(ep *corev1.Endpoints) {
	for i, c := range a.endpoints {
		if c.GetName() == ep.GetName() {
			a.endpoints = append(a.endpoints[0:i], a.endpoints[i+1:]...)
			return
		}
	}
}

// GetIngress get ingress
func (a *AppService) GetIngress(canCopy bool) ([]*networkingv1.Ingress, []*betav1.Ingress) {
	if k8s.IsHighVersion() {
		if canCopy {
			cr := make([]*networkingv1.Ingress, len(a.ingresses))
			copy(cr, a.ingresses[0:])
			return cr, nil
		}
		return a.ingresses, nil
	}
	if canCopy {
		cr := make([]*betav1.Ingress, len(a.betaIngresses))
		copy(cr, a.betaIngresses[0:])
		return nil, cr
	}
	return nil, a.betaIngresses

}

// GetDelIngs gets delIngs which need to be deleted
func (a *AppService) GetDelIngs() ([]*networkingv1.Ingress, []*betav1.Ingress) {
	return a.delIngs, a.delBetaIngresses
}

// SetIngress set kubernetes ingress model
func (a *AppService) SetIngress(d interface{}) {
	nwkIngress, ok := d.(*networkingv1.Ingress)
	if ok {
		if len(a.ingresses) > 0 {
			for i, ingress := range a.ingresses {
				if ingress.GetName() == nwkIngress.GetName() {
					a.ingresses[i] = ingress
					return
				}
			}
		}
		a.ingresses = append(a.ingresses, nwkIngress)
	}
	betaIngress, ok := d.(*betav1.Ingress)
	if ok {
		if len(a.betaIngresses) > 0 {
			for i, ingress := range a.betaIngresses {
				if ingress.GetName() == betaIngress.GetName() {
					a.betaIngresses[i] = ingress
					return
				}
			}
		}
		a.betaIngresses = append(a.betaIngresses, betaIngress)
	}
}

// SetIngresses sets k8s ingress list
func (a *AppService) SetIngresses(i []*networkingv1.Ingress) {
	a.ingresses = i
}

// DeleteIngress delete kubernetes ingress model
func (a *AppService) DeleteIngress(d *networkingv1.Ingress) {
	for i, c := range a.ingresses {
		if c.GetName() == d.GetName() {
			a.ingresses = append(a.ingresses[0:i], a.ingresses[i+1:]...)
			return
		}
	}
}

// DeleteBetaIngress delete kubernetes networking v1beta1 ingress model
func (a *AppService) DeleteBetaIngress(d *betav1.Ingress) {
	for i, c := range a.betaIngresses {
		if c.GetName() == d.GetName() {
			a.betaIngresses = append(a.betaIngresses[0:i], a.betaIngresses[i+1:]...)
			return
		}
	}
}

func (a *AppService) calculateComponentMemoryRequest() {
	var memoryRequest int64
	var cpuRequest int64
	if a.statefulset != nil {
		for _, c := range a.statefulset.Spec.Template.Spec.Containers {
			memoryRequest += c.Resources.Requests.Memory().Value() / 1024 / 1024
			cpuRequest += c.Resources.Requests.Cpu().MilliValue()
		}
	}
	if a.deployment != nil {
		for _, c := range a.deployment.Spec.Template.Spec.Containers {
			memoryRequest += c.Resources.Requests.Memory().Value() / 1024 / 1024
			cpuRequest += c.Resources.Requests.Cpu().MilliValue()
		}
	}
	a.podMemoryRequest = memoryRequest
	a.podCPURequest = cpuRequest
}

// SetPodTemplate set pod template spec
func (a *AppService) SetPodTemplate(d corev1.PodTemplateSpec) {
	if a.statefulset != nil {
		a.statefulset.Spec.Template = d
	}
	if a.deployment != nil {
		a.deployment.Spec.Template = d
	}
	a.calculateComponentMemoryRequest()
}

// GetPodTemplate get pod template
func (a *AppService) GetPodTemplate() *corev1.PodTemplateSpec {
	if a.statefulset != nil {
		return &a.statefulset.Spec.Template
	}
	if a.deployment != nil {
		return &a.deployment.Spec.Template
	}
	return nil
}

// SetSecret set srcrets
func (a *AppService) SetSecret(d *corev1.Secret) {
	if d == nil {
		return
	}
	if len(a.secrets) > 0 {
		for i, secret := range a.secrets {
			if secret.GetName() == d.GetName() {
				a.secrets[i] = d
				return
			}
		}
	}
	a.secrets = append(a.secrets, d)
}

// SetSecrets sets k8s secret list
func (a *AppService) SetSecrets(s []*corev1.Secret) {
	a.secrets = s
}

// SetAllSecrets sets secrets
func (a *AppService) SetAllSecrets(secrets []*corev1.Secret) {
	a.secrets = secrets
}

// DeleteSecrets set secrets
func (a *AppService) DeleteSecrets(d *corev1.Secret) {
	for i, c := range a.secrets {
		if c.GetName() == d.GetName() {
			a.secrets = append(a.secrets[0:i], a.secrets[i+1:]...)
			return
		}
	}
}

// GetSecrets get secrets
func (a *AppService) GetSecrets(canCopy bool) []*corev1.Secret {
	if canCopy {
		return append(a.secrets[:0:0], a.secrets...)
	}
	return a.secrets
}

// GetDelSecrets get delSecrets which need to be deleted
func (a *AppService) GetDelSecrets() []*corev1.Secret {
	return a.delSecrets
}

// SetEnvVarSecrets -
func (a *AppService) SetEnvVarSecrets(secrets []*corev1.Secret) {
	a.envVarSecrets = secrets
}

// GetEnvVarSecrets -
func (a *AppService) GetEnvVarSecrets(canCopy bool) []*corev1.Secret {
	if canCopy {
		return append(a.envVarSecrets[:0:0], a.envVarSecrets...)
	}
	return a.envVarSecrets
}

// SetPods set pod
func (a *AppService) SetPods(d *corev1.Pod) {
	if len(a.pods) > 0 {
		for i, pod := range a.pods {
			if pod.GetName() == d.GetName() {
				a.pods[i] = d
				return
			}
		}
	}
	a.pods = append(a.pods, d)
}

// DeletePods delete pod
func (a *AppService) DeletePods(d *corev1.Pod) {
	for i, c := range a.pods {
		if c.GetName() == d.GetName() {
			a.pods = append(a.pods[0:i], a.pods[i+1:]...)
			return
		}
	}
}

// GetPods get pods
func (a *AppService) GetPods(canCopy bool) []*corev1.Pod {
	if canCopy {
		return append(a.pods[:0:0], a.pods...)
	}
	return a.pods
}

// GetPodsByName returns the pod based on podname.
func (a *AppService) GetPodsByName(podname string) *corev1.Pod {
	for _, pod := range a.pods {
		if pod.ObjectMeta.Name == podname {
			return pod
		}
	}
	return nil
}

// SetTenantEnv set tenantEnv
func (a *AppService) SetTenantEnv(d *corev1.Namespace) {
	a.tenantEnv = d
}

// GetTenantEnv get tenant env namespace
func (a *AppService) GetTenantEnv() *corev1.Namespace {
	return a.tenantEnv
}

// GetNamespace get tenant env namespace name
func (a *AppService) GetNamespace() string {
	return a.tenantEnv.Name
}

// SetDeletedResources sets the resources that need to be deleted
func (a *AppService) SetDeletedResources(old *AppService) {
	if old == nil {
		logrus.Debugf("empty old app service.")
		return
	}
	oldNwkIngresses, oldBetaIngresses := old.GetIngress(true)
	nwkIngresses, betaIngresses := a.GetIngress(true)
	if oldNwkIngresses != nil && nwkIngresses != nil {
		for _, o := range oldNwkIngresses {
			del := true
			for _, n := range nwkIngresses {
				// if service_id is not same, can not delete it
				if o.Name == n.Name {
					del = false
					break
				}
			}
			if del {
				a.delIngs = append(a.delIngs, o)
			}
		}

	} else if oldBetaIngresses != nil && betaIngresses != nil {
		for _, o := range oldBetaIngresses {
			del := true
			for _, n := range betaIngresses {
				// if service_id is not same, can not delete it
				if o.Name == n.Name {
					del = false
					break
				}
			}
			if del {
				a.delBetaIngresses = append(a.delBetaIngresses, o)
			}
		}
	}

	for _, o := range old.GetSecrets(true) {
		del := true
		for _, n := range a.GetSecrets(true) {
			if o.Name == n.Name {
				del = false
				break
			}
		}
		if del {
			a.delSecrets = append(a.delSecrets, o)
		}
	}
	for _, o := range old.GetServices(true) {
		del := true
		for _, n := range a.GetServices(true) {
			if o.Name == n.Name {
				del = false
				break
			}
		}
		if del {
			a.delServices = append(a.delServices, o)
		}
	}
	for _, o := range old.GetHPAs() {
		del := true
		for _, n := range a.GetHPAs() {
			if o.Name == n.Name {
				del = false
				break
			}
		}
		if del {
			a.delHPAs = append(a.delHPAs, o)
		}
	}
}

// DistinguishPod uses replica set to distinguish between old and new pods
// true: new pod; false: old pod.
func (a *AppService) DistinguishPod(pod *corev1.Pod) bool {
	rss := a.GetNewestReplicaSet()
	if rss == nil {
		return true
	}
	return !pod.ObjectMeta.CreationTimestamp.Before(&rss.ObjectMeta.CreationTimestamp)
}

// GetClaims get claims
func (a *AppService) GetClaims() []*corev1.PersistentVolumeClaim {
	return a.claims
}

// GetClaimsManually get claims
func (a *AppService) GetClaimsManually() []*corev1.PersistentVolumeClaim {
	return a.claimsmanual
}

// SetClaim set claim
func (a *AppService) SetClaim(claim *corev1.PersistentVolumeClaim) {
	claim.Namespace = a.GetNamespace()
	if len(a.claims) > 0 {
		for i, c := range a.claims {
			if c.GetName() == claim.GetName() {
				a.claims[i] = claim
				return
			}
		}
	}
	a.claims = append(a.claims, claim)
}

// SetClaimManually sets claim that needs to be created manually.
func (a *AppService) SetClaimManually(claim *corev1.PersistentVolumeClaim) {
	claim.Namespace = a.GetNamespace()
	if len(a.claimsmanual) > 0 {
		for i, c := range a.claimsmanual {
			if c.GetName() == claim.GetName() {
				a.claimsmanual[i] = claim
				return
			}
		}
	}
	a.claimsmanual = append(a.claimsmanual, claim)
}

// DeleteClaim delete claim
func (a *AppService) DeleteClaim(claim *corev1.PersistentVolumeClaim) {
	if len(a.claims) == 0 {
		return
	}
	for i, c := range a.claims {
		if c.GetName() == claim.GetName() {
			a.claims = append(a.claims[0:i], a.claims[i+1:]...)
			return
		}
	}
}

// SetHPAs -
func (a *AppService) SetHPAs(hpas []*autoscalingv1.HorizontalPodAutoscaler) {
	a.hpas = hpas
}

// SetHPA -
func (a *AppService) SetHPA(hpa *autoscalingv1.HorizontalPodAutoscaler) {
	if len(a.hpas) > 0 {
		for i, old := range a.hpas {
			if old.GetName() == hpa.GetName() {
				a.hpas[i] = hpa
				return
			}
		}
	}
	a.hpas = append(a.hpas, hpa)
}

// SetServiceMonitor -
func (a *AppService) SetServiceMonitor(sm *monitorv1.ServiceMonitor) {
	for i, s := range a.serviceMonitor {
		if s.Name == sm.Name {
			a.serviceMonitor[i] = sm
			return
		}
	}
	a.serviceMonitor = append(a.serviceMonitor, sm)
}

// DeleteServiceMonitor delete service monitor
func (a *AppService) DeleteServiceMonitor(sm *monitorv1.ServiceMonitor) {
	if len(a.serviceMonitor) == 0 {
		return
	}
	for i, old := range a.serviceMonitor {
		if old.GetName() == sm.GetName() {
			a.serviceMonitor = append(a.serviceMonitor[0:i], a.serviceMonitor[i+1:]...)
			return
		}
	}
}

// GetServiceMonitors -
func (a *AppService) GetServiceMonitors(canCopy bool) []*monitorv1.ServiceMonitor {
	if canCopy {
		return append(a.serviceMonitor[:0:0], a.serviceMonitor...)
	}
	return a.serviceMonitor
}

// GetHPAs -
func (a *AppService) GetHPAs() []*autoscalingv1.HorizontalPodAutoscaler {
	return a.hpas
}

// GetDelHPAs -
func (a *AppService) GetDelHPAs() []*autoscalingv1.HorizontalPodAutoscaler {
	return a.delHPAs
}

// DelHPA -
func (a *AppService) DelHPA(hpa *autoscalingv1.HorizontalPodAutoscaler) {
	if len(a.hpas) == 0 {
		return
	}
	for i, old := range a.hpas {
		if old.GetName() == hpa.GetName() {
			a.hpas = append(a.hpas[0:i], a.hpas[i+1:]...)
			return
		}
	}
}

// SetStorageClass set storageclass
func (a *AppService) SetStorageClass(sc *storagev1.StorageClass) {
	if len(a.storageClasses) > 0 {
		for i, old := range a.storageClasses {
			if old.Name == sc.GetName() {
				a.storageClasses[i] = sc
			}
		}
	}
	a.storageClasses = append(a.storageClasses, sc)
}

// DeleteStorageClass deelete storageclass
func (a *AppService) DeleteStorageClass(sc *storagev1.StorageClass) {
	if len(a.storageClasses) == 0 {
		return
	}
	for i, old := range a.storageClasses {
		if old.Name == sc.GetName() {
			a.storageClasses = append(a.storageClasses[0:i], a.storageClasses[i+1:]...)
			return
		}
	}
}

// GetMemoryRequest get component memory request
func (a *AppService) GetMemoryRequest() (res int64) {
	for _, pod := range a.pods {
		res += CalculatePodResource(pod).MemoryRequest / 1024 / 1024
	}
	return
}

// GetCPURequest get component cpu request
func (a *AppService) GetCPURequest() (res int64) {
	for _, pod := range a.pods {
		res += CalculatePodResource(pod).CPURequest
	}
	return
}

// GetManifests get component custom manifest
func (a *AppService) GetManifests() []*unstructured.Unstructured {
	return a.manifests
}

// SetManifests get component custom manifest
func (a *AppService) SetManifests(manifests []*unstructured.Unstructured) {
	a.manifests = manifests
}

// SetWorkload set component workload
func (a *AppService) SetWorkload(workload client.Object) {
	a.workload = workload
}

// GetWorkload get component workload
func (a *AppService) GetWorkload() client.Object {
	return a.workload
}

// DeleteWorkload delete component workload
func (a *AppService) DeleteWorkload(workload client.Object) {
	a.workload = nil
}

func (a *AppService) String() string {
	var ingresses string
	for _, i := range a.ingresses {
		ingresses += i.Name + ","
	}
	for _, i := range a.betaIngresses {
		ingresses += i.Name + ","
	}
	return fmt.Sprintf(`
	-----------------------------------------------------
	App:%s
	DeployVersion:%s
	Statefulset %+v
	Deployment %+v
	Pod %d
	ingresses %s
	service %s
	endpoints %+v
	-----------------------------------------------------
	`,
		a.ServiceAlias,
		a.DeployVersion,
		a.statefulset,
		a.deployment,
		len(a.pods),
		ingresses,
		func(ing []*corev1.Service) string {
			result := ""
			for _, i := range ing {
				result += i.Name + ","
			}
			return result
		}(a.services),
		a.endpoints,
	)
}

// TenantEnvResource tenant env resource statistical models
type TenantEnvResource struct {
	TenantEnvID      string `json:"tenant_env_id,omitempty"`
	CPURequest       int64  `json:"cpu_request,omitempty"`
	CPULimit         int64  `json:"cpu_limit,omitempty"`
	MemoryRequest    int64  `json:"memory_request,omitempty"`
	MemoryLimit      int64  `json:"memory_limit,omitempty"`
	UnscdCPUReq      int64  `json:"unscd_cpu_req,omitempty"`
	UnscdCPULimit    int64  `json:"unscd_cpu_limit,omitempty"`
	UnscdMemoryReq   int64  `json:"unscd_memory_req,omitempty"`
	UnscdMemoryLimit int64  `json:"unscd_memory_limit,omitempty"`
}

// K8sResources holds kubernetes resources(svc, sercert, ep, ing).
type K8sResources struct {
	Services  []*corev1.Service
	Secrets   []*corev1.Secret
	Ingresses []interface{}
}

// CalculatePodResource calculate pod resource
func CalculatePodResource(pod *corev1.Pod) *PodResource {
	for _, con := range pod.Status.Conditions {
		if con.Type == corev1.PodScheduled && con.Status == corev1.ConditionFalse {
			return &PodResource{}
		}
	}
	var pr PodResource
	for _, con := range pod.Spec.Containers {
		pr.MemoryRequest += con.Resources.Requests.Memory().Value()
		pr.CPURequest += con.Resources.Requests.Cpu().MilliValue()
		pr.MemoryLimit += con.Resources.Limits.Memory().Value()
		pr.CPULimit += con.Resources.Limits.Cpu().MilliValue()
	}
	pr.NodeName = pod.Spec.NodeName
	return &pr
}

// PodResource resource struct
type PodResource struct {
	MemoryRequest int64
	MemoryLimit   int64
	CPURequest    int64
	CPULimit      int64
	NodeName      string
}
