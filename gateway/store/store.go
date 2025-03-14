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

package store

import (
	"bytes"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync"

	k8sutil "github.com/wutong-paas/wutong/util/k8s"

	"github.com/eapache/channels"
	"github.com/sirupsen/logrus"
	"github.com/wutong-paas/wutong/cmd/gateway/option"
	"github.com/wutong-paas/wutong/gateway/annotations"
	"github.com/wutong-paas/wutong/gateway/annotations/l4"
	"github.com/wutong-paas/wutong/gateway/annotations/parser"
	"github.com/wutong-paas/wutong/gateway/annotations/rewrite"
	"github.com/wutong-paas/wutong/gateway/cluster"
	"github.com/wutong-paas/wutong/gateway/controller/config"
	"github.com/wutong-paas/wutong/gateway/defaults"
	"github.com/wutong-paas/wutong/gateway/util"
	v1 "github.com/wutong-paas/wutong/gateway/v1"
	coreutil "github.com/wutong-paas/wutong/util"
	istroe "github.com/wutong-paas/wutong/util/ingress-nginx/ingress/controller/store"
	ik8s "github.com/wutong-paas/wutong/util/ingress-nginx/k8s"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	betav1 "k8s.io/api/networking/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

// EventType -
type EventType string

const (
	// CreateEvent event associated with new objects in an informer
	CreateEvent EventType = "CREATE"
	// UpdateEvent event associated with an object update in an informer
	UpdateEvent EventType = "UPDATE"
	// DeleteEvent event associated when an object is removed from an informer
	DeleteEvent EventType = "DELETE"
	// CertificatePath is the default path of certificate file
	CertificatePath = "/run/nginx/conf/certificate"
	// DefVirSrvName is the default virtual service name
	DefVirSrvName = "_"
)

var l7PoolMap = make(map[string]struct{})

// l7PoolBackendMap is the mapping between backend and pool
var l7PoolBackendMap map[string][]backend

var l4PoolMap = make(map[string]struct{})

// l4PoolBackendMap is the mapping between backend and pool
var l4PoolBackendMap map[string][]backend

// Storer is the interface that wraps the required methods to gather information
type Storer interface {
	// list endpoints pool
	ListPool() ([]*v1.Pool, []*v1.Pool)

	// list virtual service
	ListVirtualService() ([]*v1.VirtualService, []*v1.VirtualService)

	ListIngresses() interface{}

	// Run initiates the synchronization of the controllers
	Run(stopCh chan struct{})

	// GetDefaultBackend returns the default backend configuration
	GetDefaultBackend() defaults.Backend
}

type backend struct {
	name              string
	weight            int
	hashBy            string
	loadBalancingType string
}

// Event holds the context of an event.
type Event struct {
	Type EventType
	Obj  interface{}
}

// Lister contains object listers (stores).
type Lister struct {
	Ingress           istroe.IngressLister
	Service           istroe.ServiceLister
	Endpoint          istroe.EndpointLister
	Secret            istroe.SecretLister
	IngressAnnotation IngressAnnotationsLister
}

type k8sStore struct {
	conf   *option.Config
	client kubernetes.Interface
	// informer contains the cache Informers
	informers      *Informer
	sharedInformer informers.SharedInformerFactory
	// Lister contains object listers (stores).
	listers          *Lister
	secretIngressMap *secretIngressMap
	// sslStore local store of SSL certificates (certificates used in ingress)
	// this is required because the certificates must be present in the
	// container filesystem
	sslStore    *SSLCertTracker
	annotations annotations.Extractor

	// backendConfig contains the running configuration from the configmap
	// this is required because this rarely changes but is a very expensive
	// operation to execute in each OnUpdate invocation
	backendConfig config.Configuration

	// backendConfigMu protects against simultaneous read/write of backendConfig
	backendConfigMu *sync.RWMutex
	// Node controller to get the available IP address of the current node
	node     *cluster.NodeManager
	updateCh *channels.RingChannel
}

// New creates a new Storer
func New(client kubernetes.Interface,
	updateCh *channels.RingChannel,
	conf *option.Config, node *cluster.NodeManager) Storer {
	store := &k8sStore{
		client:    client,
		informers: &Informer{},
		listers:   &Lister{},
		secretIngressMap: &secretIngressMap{
			make(map[string][]string),
		},
		sslStore:        NewSSLCertTracker(),
		conf:            conf,
		backendConfigMu: &sync.RWMutex{},
		backendConfig:   config.NewDefault(),
		node:            node,
		updateCh:        updateCh,
	}

	store.annotations = annotations.NewAnnotationExtractor(store)
	store.listers.IngressAnnotation.Store = cache.NewStore(cache.DeletionHandlingMetaNamespaceKeyFunc)

	// create informers factory, enable and assign required informers
	store.sharedInformer = informers.NewFilteredSharedInformerFactory(client, conf.ResyncPeriod, corev1.NamespaceAll,
		func(options *metav1.ListOptions) {
			options.LabelSelector = "creator=Wutong"
		})

	if k8sutil.IsHighVersion() {
		store.informers.Ingress = store.sharedInformer.Networking().V1().Ingresses().Informer()
	} else {
		store.informers.Ingress = store.sharedInformer.Networking().V1beta1().Ingresses().Informer()
	}
	store.listers.Ingress.Store = store.informers.Ingress.GetStore()

	store.informers.Service = store.sharedInformer.Core().V1().Services().Informer()
	store.listers.Service.Store = store.informers.Service.GetStore()

	store.informers.Endpoint = store.sharedInformer.Core().V1().Endpoints().Informer()
	store.listers.Endpoint.Store = store.informers.Endpoint.GetStore()

	store.informers.Secret = store.sharedInformer.Core().V1().Secrets().Informer()
	store.listers.Secret.Store = store.informers.Secret.GetStore()

	ingEventHandler := cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			store.extractAnnotations(obj)
			store.secretIngressMap.update(obj)
			store.syncSecrets(obj)

			updateCh.In() <- Event{
				Type: CreateEvent,
				Obj:  obj,
			}
		},
		DeleteFunc: func(obj interface{}) {
			updateCh.In() <- Event{
				Type: DeleteEvent,
				Obj:  obj,
			}
		},
		UpdateFunc: func(old, cur interface{}) {
			var (
				ingress interface{}
			)
			if k8sutil.IsHighVersion() {
				oldIng := old.(*networkingv1.Ingress)
				curIng := cur.(*networkingv1.Ingress)
				// ignore the same secret as the old one
				if oldIng.ResourceVersion == curIng.ResourceVersion || reflect.DeepEqual(oldIng, curIng) {
					return
				}
				ingress = curIng

			} else {
				oldIng := old.(*betav1.Ingress)
				curIng := cur.(*betav1.Ingress)
				// ignore the same secret as the old one
				if oldIng.ResourceVersion == curIng.ResourceVersion || reflect.DeepEqual(oldIng, curIng) {
					return
				}
				ingress = curIng
			}
			store.extractAnnotations(ingress)
			store.secretIngressMap.update(ingress)
			store.syncSecrets(ingress)
			updateCh.In() <- Event{
				Type: UpdateEvent,
				Obj:  cur,
			}
		},
	}

	secEventHandler := cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			sec := obj.(*corev1.Secret)
			key := ik8s.MetaNamespaceKey(sec)

			// find references in ingresses and update local ssl certs
			if ings := store.secretIngressMap.getSecretKeys(key); len(ings) > 0 {
				logrus.Infof("secret %v was added and it is used in ingress annotations. Parsing...", key)
				for _, ingKey := range ings {
					ingress, err := store.GetIngress(ingKey)
					if err != nil {
						logrus.Errorf("could not find Ingress %v in local store", ingKey)
						continue
					}
					store.syncSecrets(ingress)
				}
				updateCh.In() <- Event{
					Type: CreateEvent,
					Obj:  obj,
				}
			}
		},
		UpdateFunc: func(old, cur interface{}) {
			if !reflect.DeepEqual(old, cur) {
				curSec := cur.(*corev1.Secret)
				oldSec := old.(*corev1.Secret)
				// ignore the same secret as the old one
				if oldSec.ResourceVersion == curSec.ResourceVersion || reflect.DeepEqual(curSec, oldSec) {
					return
				}
				key := ik8s.MetaNamespaceKey(curSec)

				// find references in ingresses and update local ssl certs
				if ings := store.secretIngressMap.getSecretKeys(key); len(ings) > 0 {
					logrus.Infof("secret %v was updated and it is used in ingress annotations. Parsing...", key)
					for _, ingKey := range ings {
						ing, err := store.GetIngress(ingKey)
						if err != nil {
							logrus.Errorf("could not find Ingress %v in local store", ingKey)
							continue
						}
						store.syncSecrets(ing)
					}
					updateCh.In() <- Event{
						Type: UpdateEvent,
						Obj:  cur,
					}
				}
			}
		},
		DeleteFunc: func(obj interface{}) {
			sec, ok := obj.(*corev1.Secret)
			if !ok {
				// If we reached here it means the secret was deleted but its final state is unrecorded.
				tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
				if !ok {
					logrus.Errorf("couldn't get object from tombstone %#v", obj)
					return
				}
				sec, ok = tombstone.Obj.(*corev1.Secret)
				if !ok {
					logrus.Errorf("Tombstone contained object that is not a Secret: %#v", obj)
					return
				}
			}

			store.sslStore.Delete(ik8s.MetaNamespaceKey(sec))

			key := ik8s.MetaNamespaceKey(sec)

			// find references in ingresses
			if ings := store.secretIngressMap.getSecretKeys(key); len(ings) > 0 {
				logrus.Infof("secret %v was deleted and it is used in ingress annotations. Parsing...", key)
				updateCh.In() <- Event{
					Type: DeleteEvent,
					Obj:  obj,
				}
			}
		},
	}

	// Endpoint Event Handler
	epEventHandler := cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			updateCh.In() <- Event{
				Type: CreateEvent,
				Obj:  obj,
			}
		},
		DeleteFunc: func(obj interface{}) {
			updateCh.In() <- Event{
				Type: DeleteEvent,
				Obj:  obj,
			}
		},
		UpdateFunc: func(old, cur interface{}) {
			oep := old.(*corev1.Endpoints)
			cep := cur.(*corev1.Endpoints)
			if cep.ResourceVersion != oep.ResourceVersion && !reflect.DeepEqual(cep.Subsets, oep.Subsets) {
				updateCh.In() <- Event{
					Type: UpdateEvent,
					Obj:  cur,
				}
			}
		},
	}

	store.informers.Ingress.AddEventHandler(ingEventHandler)
	store.informers.Secret.AddEventHandler(secEventHandler)
	store.informers.Endpoint.AddEventHandler(epEventHandler)
	store.informers.Service.AddEventHandler(cache.ResourceEventHandlerFuncs{})

	return store
}

// checkIngress checks whether the given ing is valid.
func (s *k8sStore) checkIngress(meta *metav1.ObjectMeta) bool {
	i, err := l4.NewParser(s).Parse(meta)
	if err != nil {
		logrus.Warningf("Uxpected error with ingress: %v", err)
		return false
	}

	cfg := i.(*l4.Config)
	if cfg.L4Enable {
		_, err := net.Dial("tcp", fmt.Sprintf("%s:%d", cfg.L4Host, cfg.L4Port))
		if err == nil {
			logrus.Warningf("%s, in Ingress(%v), is already in use.",
				fmt.Sprintf("%s:%d", cfg.L4Host, cfg.L4Port), meta)
			return false
		}
		return true
	}

	return true
}

// extractAnnotations parses ingress annotations converting the value of the
// annotation to a go struct and also information about the referenced secrets
func (s *k8sStore) extractAnnotations(ingress interface{}) {
	key := ik8s.MetaNamespaceKey(ingress)
	logrus.Debugf("updating annotations information for ingress %v", key)

	var anns *annotations.Ingress
	if k8sutil.IsHighVersion() {
		nwkIngress := ingress.(*networkingv1.Ingress)
		anns = s.annotations.Extract(&nwkIngress.ObjectMeta)
	} else {
		betaIngress := ingress.(*betav1.Ingress)
		anns = s.annotations.Extract(&betaIngress.ObjectMeta)
	}

	err := s.listers.IngressAnnotation.Update(anns)
	if err != nil {
		logrus.Error(err)
	}
}

// ListPool returns the list of Pools
func (s *k8sStore) ListPool() ([]*v1.Pool, []*v1.Pool) {
	var httpPools []*v1.Pool
	var tcpPools []*v1.Pool
	l7Pools := make(map[string]*v1.Pool)
	l4Pools := make(map[string]*v1.Pool)
	for _, item := range s.listers.Endpoint.List() {
		ep := item.(*corev1.Endpoints)
		if ep.Subsets != nil || len(ep.Subsets) != 0 {
			epn := ep.ObjectMeta.Name
			// l7
			backends := l7PoolBackendMap[ep.ObjectMeta.Name]
			for _, backend := range backends {
				pool := l7Pools[backend.name]
				if pool == nil {
					pool = &v1.Pool{
						Nodes: []*v1.Node{},
					}
					pool.Name = backend.name
					// TODO: The tenant env isolation
					pool.Namespace = "default"
					pool.UpstreamHashBy = backend.hashBy
					pool.LoadBalancingType = v1.GetLoadBalancingType(backend.loadBalancingType)
					l7Pools[backend.name] = pool
				}
				for _, ss := range ep.Subsets {
					for _, port := range ss.Ports {
						for _, address := range ss.Addresses {
							if _, ok := l7PoolMap[epn]; ok { // l7
								pool.Nodes = append(pool.Nodes, &v1.Node{
									Host:   address.IP,
									Port:   port.Port,
									Weight: backend.weight,
								})
							}
						}
					}
				}
			}
			// l4
			backends = l4PoolBackendMap[ep.ObjectMeta.Name]
			for _, backend := range backends {
				pool := l4Pools[backend.name]
				if pool == nil {
					pool = &v1.Pool{
						Nodes: []*v1.Node{},
					}
					// TODO: The tenant env isolation
					pool.Namespace = "default"
					pool.Name = backend.name
					l4Pools[backend.name] = pool
				}
				for _, ss := range ep.Subsets {
					for _, port := range ss.Ports {
						for _, address := range ss.Addresses {
							if _, ok := l4PoolMap[epn]; ok { // l7
								pool.Nodes = append(pool.Nodes, &v1.Node{
									Host:   address.IP,
									Port:   port.Port,
									Weight: backend.weight,
								})
							}
						}
					}
				}
			}
		}
	}
	// change map to slice TODO: use map directly
	for _, pool := range l7Pools {
		httpPools = append(httpPools, pool)
	}
	for _, pool := range l4Pools {
		tcpPools = append(tcpPools, pool)
	}
	return httpPools, tcpPools
}

// ListVirtualService list l7 virtual service and l4 virtual service
func (s *k8sStore) ListVirtualService() (l7vs []*v1.VirtualService, l4vs []*v1.VirtualService) {
	l7PoolBackendMap = make(map[string][]backend)
	l4PoolBackendMap = make(map[string][]backend)
	l7vsMap := make(map[string]*v1.VirtualService)
	l4vsMap := make(map[string]*v1.VirtualService)
	// ServerName-LocationPath -> location
	srvLocMap := make(map[string]*v1.Location)

	for _, item := range s.listers.Ingress.List() {
		if !s.ingressIsValid(item) {
			continue
		}
		var ingName, ingNamespace, ingServiceName string
		isBetaIngress := false
		var anns *annotations.Ingress
		if ing, ok := item.(*networkingv1.Ingress); ok {
			ingName = ing.Name
			ingNamespace = ing.Namespace
			if ing.Spec.DefaultBackend == nil && len(ing.Spec.Rules) > 0 {
				paths := ing.Spec.Rules[0].IngressRuleValue.HTTP.Paths
				if len(paths) == 0 {
					logrus.Info("[ListVirtualService] not ingress rule value")
					continue
				}
				ingServiceName = paths[0].Backend.Service.Name
			} else {
				ingServiceName = ing.Spec.DefaultBackend.Service.Name
			}
			anns = s.annotations.Extract(&ing.ObjectMeta)
		} else {
			if ing, ok := item.(*betav1.Ingress); ok {
				ingName = ing.Name
				ingNamespace = ing.Namespace
				isBetaIngress = true
				if ing.Spec.Backend == nil && len(ing.Spec.Rules) > 0 {
					paths := ing.Spec.Rules[0].IngressRuleValue.HTTP.Paths
					if len(paths) == 0 {
						logrus.Info("[ListVirtualService] not ingress rule value")
						continue
					}
					ingServiceName = paths[0].Backend.ServiceName
				} else {
					ingServiceName = ing.Spec.Backend.ServiceName
				}
				anns = s.annotations.Extract(&ing.ObjectMeta)
			}
		}
		if anns.L4.L4Enable && anns.L4.L4Port != 0 {
			// region l4
			host := strings.Replace(anns.L4.L4Host, " ", "", -1)
			if host == "" || host == "0.0.0.0" {
				host = "0.0.0.0"
			} else {
				//Determines whether the current node is in effect
				ip := net.ParseIP(host)
				if ip == nil {
					logrus.Warningf("ingress %s (Namespace:%s) l4 host config is invalid", ingName, ingNamespace)
					continue
				}
				if !s.node.IPManager().IPInCurrentHost(ip) {
					logrus.Debugf("ingress %s (Namespace:%s) l4 host %s does not belong to the current node", ingName, ingNamespace, host)
					continue
				}
			}

			var svcKey string
			var protocol corev1.Protocol
			if isBetaIngress {
				svcKey = fmt.Sprintf("%v/%v", item.(*betav1.Ingress).Namespace, item.(*betav1.Ingress).Spec.Backend.ServiceName)
				protocol = s.GetServiceProtocol(svcKey, item.(*betav1.Ingress).Spec.Backend.ServicePort.IntVal)
			} else {
				svcKey = fmt.Sprintf("%v/%v", ingNamespace, item.(*networkingv1.Ingress).Spec.DefaultBackend.Service.Name)
				protocol = s.GetServiceProtocol(svcKey, item.(*networkingv1.Ingress).Spec.DefaultBackend.Service.Port.Number)
			}

			listening := fmt.Sprintf("%s:%v", host, anns.L4.L4Port)
			if string(protocol) == string(v1.ProtocolUDP) {
				listening = fmt.Sprintf("%s %s", listening, "udp")
			}
			//Detect if the IP and port are in conflict
			//This is based on the order of the judgment, if the conflict, the latter discarded
			if anns.L4.L4Port == s.conf.ListenPorts.HTTP || anns.L4.L4Port == s.conf.ListenPorts.HTTPS ||
				anns.L4.L4Port == s.conf.ListenPorts.Health || anns.L4.L4Port == s.conf.ListenPorts.Status {
				logrus.Warningf("ingress %s (Namespace:%s) l4 host repeat listening will be ignored", ingName, ingNamespace)
				continue
			}
			conflictkey := []string{
				listening, strings.Replace(listening, host, "0.0.0.0", 1),
			}
			for _, key := range conflictkey {
				vs := l4vsMap[key]
				if vs != nil {
					logrus.Warningf("ingress %s (Namespace:%s) l4 host repeat listening will be ignored", ingName, ingNamespace)
					continue
				}
			}
			backendName := util.BackendName(listening, ingNamespace)
			vs := &v1.VirtualService{
				Listening: []string{listening},
				PoolName:  backendName,
				Protocol:  protocol,
			}
			vs.Namespace = anns.Namespace
			vs.ServiceID = anns.Labels["service_id"]
			vs.TCPKeepaliveEnabled = anns.L4.KeepaliveEnabled
			vs.TCPKeepaliveIdle = anns.L4.KeepaliveIdle
			vs.TCPKeepaliveIntvl = anns.L4.KeepaliveIntvl
			vs.TCPKeepaliveCnt = anns.L4.KeepaliveCnt
			vs.ProxyStreamTimeout = anns.L4.ProxyStreamTimeout
			vs.ProxyStreamNextUpstreamTimeout = anns.L4.ProxyStreamNextUpstreamTimeout
			l4PoolMap[ingServiceName] = struct{}{}
			l4vsMap[listening] = vs
			l4vs = append(l4vs, vs)
			backend := backend{name: backendName, weight: anns.Weight.Weight}
			l4PoolBackendMap[ingServiceName] = append(l4PoolBackendMap[ingServiceName], backend)
			// endregion
		} else {
			// region l7
			// parse TLS into a map
			hostSSLMap := make(map[string]*v1.SSLCert)
			if isBetaIngress {
				ing := item.(*betav1.Ingress)
				for _, tls := range ing.Spec.TLS {
					secrKey := fmt.Sprintf("%s/%s", ingNamespace, tls.SecretName)
					item, exists := s.sslStore.Get(secrKey)
					if !exists {
						logrus.Warnf("Secret named %s does not exist", secrKey)
						continue
					}
					sslCert := item.(*v1.SSLCert)
					for _, host := range tls.Hosts {
						hostSSLMap[host] = sslCert
					}
					// make the first SSLCert as default
					if _, exists := hostSSLMap[DefVirSrvName]; !exists {
						hostSSLMap[DefVirSrvName] = sslCert
					}
				}

				for _, rule := range ing.Spec.Rules {
					var vs *v1.VirtualService
					// virtual service name
					virSrvName := strings.Replace(rule.Host, " ", "", -1)
					if virSrvName == "" {
						virSrvName = DefVirSrvName
					}
					if len(hostSSLMap) != 0 {
						virSrvName = fmt.Sprintf("tls%s", virSrvName)
					}

					vs = l7vsMap[virSrvName]
					if vs == nil {
						vs = &v1.VirtualService{
							Listening:    []string{strconv.Itoa(s.conf.ListenPorts.HTTP)},
							ServerName:   virSrvName,
							Locations:    []*v1.Location{},
							SSlProtocols: "TLSv1.2 TLSv1.3",
						}
						sslProtocols := os.Getenv("SSL_PROTOCOLS")
						if sslProtocols != "" {
							vs.SSlProtocols = sslProtocols
						}

						vs.Namespace = ing.Namespace
						vs.ServiceID = anns.Labels["service_id"]
						if len(hostSSLMap) != 0 {
							vs.Listening = []string{strconv.Itoa(s.conf.ListenPorts.HTTPS), "ssl"}
							if hostSSLMap[virSrvName] != nil {
								vs.SSLCert = hostSSLMap[virSrvName]
							} else { // TODO: if there is necessary to provide a default virtual service name
								vs.SSLCert = hostSSLMap[DefVirSrvName]
							}
						}

						l7vsMap[virSrvName] = vs
						l7vs = append(l7vs, vs)
					}

					for _, path := range rule.IngressRuleValue.HTTP.Paths {
						locKey := fmt.Sprintf("%s_%s", virSrvName, path.Path)
						location := srvLocMap[locKey]
						l7PoolMap[path.Backend.ServiceName] = struct{}{}
						// if location do not exists, then creates a new one
						if location == nil {
							location = &v1.Location{
								Path:          path.Path,
								NameCondition: map[string]*v1.Condition{},
							}
							i, err := rewrite.NewParser(s).Parse(&ing.ObjectMeta)
							if err == nil {
								if cfg, ok := i.(*rewrite.Config); ok {
									location.Rewrite.Rewrites = cfg.Rewrites
								}
							}
							pathRewrite, _ := parser.GetBoolAnnotation("path-rewrite", &ing.ObjectMeta)
							if pathRewrite {
								location.PathRewrite = true
							}
							srvLocMap[locKey] = location
							vs.Locations = append(vs.Locations, location)
							// the first ingress proxy takes effect
							location.Proxy = anns.Proxy
						}
						// If their ServiceName is the same, then the new one will overwrite the old one.
						nameCondition := &v1.Condition{}
						var backendName string
						if anns.Header.Header != nil {
							nameCondition.Type = v1.HeaderType
							nameCondition.Value = anns.Header.Header
							backendName = fmt.Sprintf("%s_%s", locKey, v1.HeaderType)
						} else if anns.Cookie.Cookie != nil {
							nameCondition.Type = v1.CookieType
							nameCondition.Value = anns.Cookie.Cookie
							backendName = fmt.Sprintf("%s_%s", locKey, v1.CookieType)
						} else {
							nameCondition.Type = v1.DefaultType
							nameCondition.Value = map[string]string{"1": "1"}
							backendName = fmt.Sprintf("%s_%s", locKey, v1.DefaultType)
						}
						backendName = util.BackendName(backendName, ing.Namespace)
						location.NameCondition[backendName] = nameCondition
						backend := backend{
							name:              backendName,
							weight:            anns.Weight.Weight,
							loadBalancingType: anns.LoadBalancingType,
						}
						if anns.UpstreamHashBy != "" {
							backend.hashBy = anns.UpstreamHashBy
						}
						l7PoolBackendMap[path.Backend.ServiceName] = append(l7PoolBackendMap[path.Backend.ServiceName], backend)
					}
				}
			} else {
				ing := item.(*networkingv1.Ingress)
				for _, tls := range ing.Spec.TLS {
					secrKey := fmt.Sprintf("%s/%s", ingNamespace, tls.SecretName)
					item, exists := s.sslStore.Get(secrKey)
					if !exists {
						logrus.Warnf("Secret named %s does not exist", secrKey)
						continue
					}
					sslCert := item.(*v1.SSLCert)
					for _, host := range tls.Hosts {
						hostSSLMap[host] = sslCert
					}
					// make the first SSLCert as default
					if _, exists := hostSSLMap[DefVirSrvName]; !exists {
						hostSSLMap[DefVirSrvName] = sslCert
					}
				}

				for _, rule := range ing.Spec.Rules {
					var vs *v1.VirtualService
					// virtual service name
					virSrvName := strings.Replace(rule.Host, " ", "", -1)
					if virSrvName == "" {
						virSrvName = DefVirSrvName
					}
					if len(hostSSLMap) != 0 {
						virSrvName = fmt.Sprintf("tls%s", virSrvName)
					}

					vs = l7vsMap[virSrvName]
					if vs == nil {
						vs = &v1.VirtualService{
							Listening:    []string{strconv.Itoa(s.conf.ListenPorts.HTTP)},
							ServerName:   virSrvName,
							Locations:    []*v1.Location{},
							SSlProtocols: "TLSv1.2 TLSv1.3",
						}
						sslProtocols := os.Getenv("SSL_PROTOCOLS")
						if sslProtocols != "" {
							vs.SSlProtocols = sslProtocols
						}

						vs.Namespace = ing.Namespace
						vs.ServiceID = anns.Labels["service_id"]
						if len(hostSSLMap) != 0 {
							vs.Listening = []string{strconv.Itoa(s.conf.ListenPorts.HTTPS), "ssl"}
							if hostSSLMap[virSrvName] != nil {
								vs.SSLCert = hostSSLMap[virSrvName]
							} else { // TODO: if there is necessary to provide a default virtual service name
								vs.SSLCert = hostSSLMap[DefVirSrvName]
							}
						}

						l7vsMap[virSrvName] = vs
						l7vs = append(l7vs, vs)
					}

					for _, path := range rule.IngressRuleValue.HTTP.Paths {
						locKey := fmt.Sprintf("%s_%s", virSrvName, path.Path)
						location := srvLocMap[locKey]
						l7PoolMap[path.Backend.Service.Name] = struct{}{}
						// if location do not exists, then creates a new one
						if location == nil {
							location = &v1.Location{
								Path:          path.Path,
								NameCondition: map[string]*v1.Condition{},
								Rewrite:       rewrite.Config{},
							}
							i, err := rewrite.NewParser(s).Parse(&ing.ObjectMeta)
							if err == nil {
								if cfg, ok := i.(*rewrite.Config); ok {
									location.Rewrite.Rewrites = cfg.Rewrites
								}
							}
							pathRewrite, _ := parser.GetBoolAnnotation("path-rewrite", &ing.ObjectMeta)
							if pathRewrite {
								location.PathRewrite = true
							}
							srvLocMap[locKey] = location
							vs.Locations = append(vs.Locations, location)
							// the first ingress proxy takes effect
							location.Proxy = anns.Proxy
						}
						// If their ServiceName is the same, then the new one will overwrite the old one.
						nameCondition := &v1.Condition{}
						var backendName string
						if anns.Header.Header != nil {
							nameCondition.Type = v1.HeaderType
							nameCondition.Value = anns.Header.Header
							backendName = fmt.Sprintf("%s_%s", locKey, v1.HeaderType)
						} else if anns.Cookie.Cookie != nil {
							nameCondition.Type = v1.CookieType
							nameCondition.Value = anns.Cookie.Cookie
							backendName = fmt.Sprintf("%s_%s", locKey, v1.CookieType)
						} else {
							nameCondition.Type = v1.DefaultType
							nameCondition.Value = map[string]string{"1": "1"}
							backendName = fmt.Sprintf("%s_%s", locKey, v1.DefaultType)
						}
						backendName = util.BackendName(backendName, ing.Namespace)
						location.NameCondition[backendName] = nameCondition
						backend := backend{
							name:              backendName,
							weight:            anns.Weight.Weight,
							loadBalancingType: anns.LoadBalancingType,
						}
						if anns.UpstreamHashBy != "" {
							backend.hashBy = anns.UpstreamHashBy
						}
						l7PoolBackendMap[path.Backend.Service.Name] = append(l7PoolBackendMap[path.Backend.Service.Name], backend)
					}
				}
			}
			// endregion
		}
	}

	for _, item := range s.listers.Ingress.List() {
		if !s.ingressIsValid(item) {
			continue
		}

		isBetaIngress := false
		var anns *annotations.Ingress
		if ing, ok := item.(*networkingv1.Ingress); ok {
			anns = s.annotations.Extract(&ing.ObjectMeta)
		}
		if ing, ok := item.(*betav1.Ingress); ok {
			isBetaIngress = true
			anns = s.annotations.Extract(&ing.ObjectMeta)
		}

		if !anns.Rewrite.ForceSSLRedirect {
			continue
		}

		if !anns.L4.L4Enable || anns.L4.L4Port == 0 {
			if isBetaIngress {
				ing := item.(*betav1.Ingress)
				for _, rule := range ing.Spec.Rules {
					var vs *v1.VirtualService
					virSrvName := strings.TrimSpace(rule.Host)
					vs = l7vsMap[virSrvName]
					if vs == nil {
						vs = &v1.VirtualService{
							Listening:  []string{strconv.Itoa(s.conf.ListenPorts.HTTP)},
							ServerName: virSrvName,
							Locations:  []*v1.Location{},
						}
						l7vsMap[virSrvName] = vs
						l7vs = append(l7vs, vs)
					}

					for _, path := range rule.IngressRuleValue.HTTP.Paths {
						locKey := fmt.Sprintf("%s_%s", virSrvName, path.Path)
						location := srvLocMap[locKey]
						if location != nil {
							// If location != nil, the http policy for path is already set.
							// In this case, ForceSSLRedirect should be ignored.
							continue
						}
						location = &v1.Location{
							Path:             path.Path,
							DisableProxyPass: true,
							Rewrite: rewrite.Config{
								Rewrites: []*rewrite.Rewrite{
									{
										Regex:       "^",
										Replacement: "https://$http_host$request_uri?",
										Flag:        "permanent",
									},
								},
							},
						}
						location.Proxy = anns.Proxy
						vs.Locations = append(vs.Locations, location)
					}
				}
			} else {
				ing := item.(*networkingv1.Ingress)
				for _, rule := range ing.Spec.Rules {
					var vs *v1.VirtualService
					virSrvName := strings.TrimSpace(rule.Host)
					vs = l7vsMap[virSrvName]
					if vs == nil {
						vs = &v1.VirtualService{
							Listening:  []string{strconv.Itoa(s.conf.ListenPorts.HTTP)},
							ServerName: virSrvName,
							Locations:  []*v1.Location{},
						}
						l7vsMap[virSrvName] = vs
						l7vs = append(l7vs, vs)
					}

					for _, path := range rule.IngressRuleValue.HTTP.Paths {
						locKey := fmt.Sprintf("%s_%s", virSrvName, path.Path)
						location := srvLocMap[locKey]
						if location != nil {
							// If location != nil, the http policy for path is already set.
							// In this case, ForceSSLRedirect should be ignored.
							continue
						}
						location = &v1.Location{
							Path:             path.Path,
							DisableProxyPass: true,
							Rewrite: rewrite.Config{
								Rewrites: []*rewrite.Rewrite{
									{
										Regex:       "^",
										Replacement: "https://$http_host$request_uri?",
										Flag:        "permanent",
									},
								},
							},
						}
						location.Proxy = anns.Proxy
						vs.Locations = append(vs.Locations, location)
					}
				}
			}
			// endregion
		}
	}
	return l7vs, l4vs
}

// ingressIsValid checks if the specified ingress is valid
func (s *k8sStore) ingressIsValid(ingress interface{}) bool {
	endpointKey := getEndpointKey(ingress)
	if endpointKey == "" {
		return false
	}
	item, exists, err := s.listers.Endpoint.GetByKey(endpointKey)
	if err != nil {
		logrus.Errorf("Can not get endpoint by key(%s): %v", endpointKey, err)
		return false
	}
	if !exists {
		logrus.Debugf("Endpoint %s does not exist.", endpointKey)
		return false
	}
	endpoint, ok := item.(*corev1.Endpoints)
	if !ok {
		logrus.Errorf("Cant not convert %v to %v", reflect.TypeOf(item), reflect.TypeOf(endpoint))
		return false
	}
	if len(endpoint.Subsets) == 0 {
		logrus.Debugf("Endpoints(%s) is empty, ignore it", endpointKey)
		return false
	}
	if !hasReadyAddresses(endpoint) {
		logrus.Debugf("Endpoints(%s) is empty, ignore it", endpointKey)
		return false
	}

	return true
}

func getEndpointKey(ingress interface{}) string {
	var endpointKey string
	ntwIngress, ok := ingress.(*networkingv1.Ingress)
	if ok {
		if ntwIngress.Spec.DefaultBackend != nil { // stream
			endpointKey = fmt.Sprintf("%s/%s", ntwIngress.Namespace, ntwIngress.Spec.DefaultBackend.Service.Name)
		} else { // http
		NtwLoop:
			for _, rule := range ntwIngress.Spec.Rules {
				for _, path := range rule.IngressRuleValue.HTTP.Paths {
					endpointKey = fmt.Sprintf("%s/%s", ntwIngress.Namespace, path.Backend.Service.Name)
					if endpointKey != "" {
						break NtwLoop
					}
				}
			}
		}
	} else {
		betaIngress, ok := ingress.(*betav1.Ingress)
		if !ok {
			return ""
		}
		if betaIngress.Spec.Backend != nil { // stream
			endpointKey = fmt.Sprintf("%s/%s", betaIngress.Namespace, betaIngress.Spec.Backend.ServiceName)
		} else { // http
		Loop:
			for _, rule := range betaIngress.Spec.Rules {
				for _, path := range rule.IngressRuleValue.HTTP.Paths {
					endpointKey = fmt.Sprintf("%s/%s", betaIngress.Namespace, path.Backend.ServiceName)
					if endpointKey != "" {
						break Loop
					}
				}
			}
		}
	}
	return endpointKey
}

func hasReadyAddresses(endpoints *corev1.Endpoints) bool {
	for _, ep := range endpoints.Subsets {
		if len(ep.Addresses) > 0 {
			return true
		}
	}
	return false
}

// GetIngress returns the Ingress matching key.
func (s *k8sStore) GetIngress(key string) (interface{}, error) {
	return s.listers.Ingress.ByKey(key)
}

// ListIngresses returns the list of Ingresses
func (s *k8sStore) ListIngresses() interface{} {
	// filter ingress rules
	var ingresses []*networkingv1.Ingress
	for _, item := range s.listers.Ingress.List() {
		ing := item.(*networkingv1.Ingress)

		ingresses = append(ingresses, ing)
	}

	return ingresses
}

// GetServiceProtocol returns the Service matching key and port.
func (s *k8sStore) GetServiceProtocol(key string, port int32) corev1.Protocol {
	svcs, err := s.listers.Service.ByKey(key)
	if err != nil {
		return corev1.ProtocolTCP
	}
	for _, p := range svcs.Spec.Ports {
		if p.Port == port {
			return p.Protocol
		}
	}

	return corev1.ProtocolTCP
}

// Run initiates the synchronization of the informers.
func (s *k8sStore) Run(stopCh chan struct{}) {
	// start informers
	s.informers.Run(stopCh)
	go s.loopUpdateIngress()
}

// syncSecrets synchronizes data from all Secrets referenced by the given
// Ingress with the local store and file system.
func (s *k8sStore) syncSecrets(ingress interface{}) {
	key := ik8s.MetaNamespaceKey(ingress)
	for _, secrKey := range s.secretIngressMap.getSecretKeys(key) {
		s.syncSecret(secrKey)
	}
}

func (s *k8sStore) syncSecret(secrKey string) {
	sslCert, err := s.getCertificatePem(secrKey)
	if err != nil {
		logrus.Errorf("fail to get certificate pem: %v", err)
		return
	}
	old, exists := s.sslStore.Get(secrKey)
	if exists {
		oldSSLCert := old.(*v1.SSLCert)
		if sslCert.Equals(oldSSLCert) {
			logrus.Debugf("no need to update SSLCert named %s", secrKey)
			return
		}
		s.sslStore.Delete(secrKey)
	}

	s.sslStore.Add(secrKey, sslCert)
}

func (s *k8sStore) getCertificatePem(secrKey string) (*v1.SSLCert, error) {
	item, exists, err := s.listers.Secret.GetByKey(secrKey)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("the secret named %s does not exists", secrKey)
	}
	secret := item.(*corev1.Secret)
	crt := secret.Data[corev1.TLSCertKey]
	key := secret.Data[corev1.TLSPrivateKeyKey]

	var buffer bytes.Buffer
	buffer.Write(crt)
	buffer.Write([]byte("\n"))
	buffer.Write(key)

	secrKey = strings.Replace(secrKey, "/", "-", 1)
	filename := fmt.Sprintf("%s/%s.pem", CertificatePath, secrKey)

	if e := os.MkdirAll(CertificatePath, 0777); e != nil {
		return nil, fmt.Errorf("cant not create directory %s: %v", CertificatePath, e)
	}

	if e := os.WriteFile(filename, buffer.Bytes(), 0666); e != nil {
		return nil, fmt.Errorf("cant not write data to %s: %v", filename, e)
	}
	fileContent, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("read certificate file failed: %s", err.Error())
	}
	pemContent, _ := pem.Decode(fileContent)
	if pemContent == nil {
		logrus.Errorf("no pem content found from: %v", fileContent)
		return nil, fmt.Errorf("generate certificate object failed, pemContent is nil")
	}
	certificate, err := x509.ParseCertificate(pemContent.Bytes)
	if err != nil {
		return nil, fmt.Errorf("generate certificate object failed: %s", err.Error())
	}

	return &v1.SSLCert{
		CertificatePem: filename,
		Certificate:    certificate,
		CertificateStr: string(certificate.Raw),
		PrivateKey:     string(key),
		CN:             []string{certificate.Subject.CommonName},
	}, nil
}

// GetDefaultBackend returns the default backend
func (s *k8sStore) GetDefaultBackend() defaults.Backend {
	return s.GetBackendConfiguration().Backend
}

func (s *k8sStore) GetBackendConfiguration() config.Configuration {
	s.backendConfigMu.RLock()
	defer s.backendConfigMu.RUnlock()

	return s.backendConfig
}

func (s *k8sStore) loopUpdateIngress() {
	for ipevent := range s.node.IPManager().NeedUpdateGatewayPolicy() {
		ingress := s.listers.Ingress.List()
		for i := range ingress {
			var meta *metav1.ObjectMeta
			var superIngress interface{}
			netIngress, ok := ingress[i].(*networkingv1.Ingress)
			if ok && netIngress != nil {
				superIngress = netIngress
				meta = &netIngress.ObjectMeta
			} else {
				betaIngress, ok := ingress[i].(*betav1.Ingress)
				if !ok || betaIngress == nil {
					continue
				}
				superIngress = betaIngress
				meta = &betaIngress.ObjectMeta
			}

			if s.annotations.Extract(meta).L4.L4Host == ipevent.IP.String() {
				s.extractAnnotations(superIngress)
				s.secretIngressMap.update(superIngress)
				s.syncSecrets(superIngress)

				s.updateCh.In() <- Event{
					Type: func() EventType {
						switch ipevent.Type {
						case coreutil.ADD:
							return CreateEvent
						case coreutil.UPDATE:
							return UpdateEvent
						case coreutil.DEL:
							return DeleteEvent
						default:
							return UpdateEvent
						}
					}(),
					Obj: ingress[i],
				}
			}
		}
	}
}
