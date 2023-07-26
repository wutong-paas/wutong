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

package controller

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/wutong-paas/wutong/event"
	"github.com/wutong-paas/wutong/util"
	"github.com/wutong-paas/wutong/worker/appm/store"
	v1 "github.com/wutong-paas/wutong/worker/appm/types/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type stopController struct {
	stopChan     chan struct{}
	controllerID string
	appService   []v1.AppService
	manager      *Manager
	waiting      time.Duration
	ctx          context.Context
}

func (s *stopController) Begin() {
	var wait sync.WaitGroup
	for _, service := range s.appService {
		wait.Add(1)
		go func(service v1.AppService) {
			defer wait.Done()
			service.Logger.Info("运行时正在准备关闭应用组件程序："+service.K8sComponentName, event.GetLoggerOption("starting"))
			if err := s.stopOne(service); err != nil {
				if err != ErrWaitTimeOut {
					service.Logger.Error(util.Translation("stop service error"), event.GetCallbackLoggerOption())
					logrus.Errorf("stop service %s failure %s", service.K8sComponentName, err.Error())
				} else {
					service.Logger.Error(util.Translation("stop service timeout"), event.GetTimeoutLoggerOption())
				}
			} else {
				service.Logger.Info(fmt.Sprintf("应用组件 %s 关闭成功！", service.K8sComponentName), event.GetLastLoggerOption())
			}
		}(service)
	}
	wait.Wait()
	s.manager.callback(s.controllerID, nil)
}
func (s *stopController) stopOne(app v1.AppService) error {

	var zero int64
	//step 1: delete services
	if services := app.GetServices(true); services != nil {
		for _, service := range services {
			if service != nil && service.Name != "" {
				err := s.manager.client.CoreV1().Services(app.GetNamespace()).Delete(s.ctx, service.Name, metav1.DeleteOptions{
					GracePeriodSeconds: &zero,
				})
				if err != nil && !errors.IsNotFound(err) {
					return fmt.Errorf("删除 Service 资源错误：%s", err.Error())
				}
			}
		}
	}
	//step 2: delete secrets
	if secrets := app.GetSecrets(true); secrets != nil {
		for _, secret := range secrets {
			if secret != nil && secret.Name != "" {
				err := s.manager.client.CoreV1().Secrets(app.GetNamespace()).Delete(s.ctx, secret.Name, metav1.DeleteOptions{
					GracePeriodSeconds: &zero,
				})
				if err != nil && !errors.IsNotFound(err) {
					return fmt.Errorf("删除 Secret 资源错误：%s", err.Error())
				}
			}
		}
	}
	//step 3: delete ingress
	if ingresses, betaIngresses := app.GetIngress(true); ingresses != nil || betaIngresses != nil {
		if ingresses != nil {
			for _, ingress := range ingresses {
				if ingress != nil && ingress.Name != "" {
					err := s.deleteIngress(app.GetNamespace(), ingress.Name, zero)
					if err != nil {
						return fmt.Errorf("删除 Ingress 资源错误：%s", err.Error())
					}
				}
			}
		} else {
			for _, ingress := range betaIngresses {
				if ingress != nil && ingress.Name != "" {
					err := s.deleteBetaIngress(app.GetNamespace(), ingress.Name, zero)
					if err != nil {
						return fmt.Errorf("删除 Ingress 资源错误：%s", err.Error())
					}
				}
			}
		}

	}
	//step 4: delete configmap
	if configs := app.GetConfigMaps(); configs != nil {
		for _, config := range configs {
			if config != nil && config.Name != "" {
				err := s.manager.client.CoreV1().ConfigMaps(app.GetNamespace()).Delete(s.ctx, config.Name, metav1.DeleteOptions{
					GracePeriodSeconds: &zero,
				})
				if err != nil && !errors.IsNotFound(err) {
					return fmt.Errorf("删除 ConfigMap 资源错误：%s", err.Error())
				}
			}
		}
	}
	// for custom component
	if len(app.GetManifests()) > 0 {
		for _, manifest := range app.GetManifests() {
			if err := s.manager.runtimeClient.Delete(s.ctx, manifest); err != nil && !errors.IsNotFound(err) {
				logrus.Errorf("删除自定义组件 %s/%s 资源错误：%s", manifest.GetKind(), manifest.GetName(), err.Error())
			}
		}
	}
	// for workload
	if workload := app.GetWorkload(); workload != nil {
		if err := s.manager.runtimeClient.Delete(s.ctx, workload); err != nil && !errors.IsNotFound(err) {
			ma := meta.NewAccessor()
			name, _ := ma.Name(workload)
			kind, _ := ma.Kind(workload)
			logrus.Errorf("删除工作负载 %s/%s 资源错误：%s", kind, name, err.Error())
		}
	}
	//step 5: delete statefulset or deployment
	if statefulset := app.GetStatefulSet(); statefulset != nil {
		err := s.manager.client.AppsV1().StatefulSets(app.GetNamespace()).Delete(s.ctx, statefulset.Name, metav1.DeleteOptions{})
		if err != nil && !errors.IsNotFound(err) {
			return fmt.Errorf("删除 StatefulSet 资源错误：%s", err.Error())
		}
		s.manager.store.OnDeletes(statefulset)
	}
	if deployment := app.GetDeployment(); deployment != nil && deployment.Name != "" {
		err := s.manager.client.AppsV1().Deployments(app.GetNamespace()).Delete(s.ctx, deployment.Name, metav1.DeleteOptions{})
		if err != nil && !errors.IsNotFound(err) {
			return fmt.Errorf("删除 Deployment 资源错误：%s", err.Error())
		}
		s.manager.store.OnDeletes(deployment)
	}
	//step 6: delete all pod
	var gracePeriodSeconds int64
	if pods := app.GetPods(true); pods != nil {
		for _, pod := range pods {
			if pod != nil && pod.Name != "" {
				err := s.manager.client.CoreV1().Pods(app.GetNamespace()).Delete(s.ctx, pod.Name, metav1.DeleteOptions{
					GracePeriodSeconds: &gracePeriodSeconds,
				})
				if err != nil && !errors.IsNotFound(err) {
					return fmt.Errorf("删除 Pod  资源错误：%s", err.Error())
				}
			}
		}
	}
	//step 7: deleta all hpa
	if hpas := app.GetHPAs(); len(hpas) != 0 {
		for _, hpa := range hpas {
			err := s.manager.client.AutoscalingV2beta2().HorizontalPodAutoscalers(hpa.GetNamespace()).Delete(s.ctx, hpa.GetName(), metav1.DeleteOptions{})
			if err != nil && !errors.IsNotFound(err) {
				return fmt.Errorf("删除 HPA 资源错误：%v", err)
			}
		}
	}

	//step 8: delete CR resource
	if crd, _ := s.manager.store.GetCrd(store.ServiceMonitor); crd != nil {
		if sms := app.GetServiceMonitors(true); len(sms) > 0 {
			smClient, err := s.manager.store.GetServiceMonitorClient()
			if err != nil {
				logrus.Errorf("create service monitor client failure %s", err.Error())
			}
			if smClient != nil {
				for _, sm := range sms {
					err := smClient.MonitoringV1().ServiceMonitors(sm.GetNamespace()).Delete(s.ctx, sm.GetName(), metav1.DeleteOptions{})
					if err != nil && !errors.IsNotFound(err) {
						logrus.Errorf("delete service monitor failure: %s", err.Error())
					}
				}
			}
		}
	}

	//step 9: waiting endpoint ready
	app.Logger.Info("组件模型相关资源清理成功，等待应用组件关闭...", event.GetLoggerOption("stopping"))
	return s.WaitingReady(app)
}
func (s *stopController) Stop() error {
	close(s.stopChan)
	return nil
}

// WaitingReady wait app start or upgrade ready
func (s *stopController) WaitingReady(app v1.AppService) error {
	storeAppService := s.manager.store.GetAppService(app.ServiceID)
	//at least waiting time is 40 second
	var timeout = time.Second * 40
	if storeAppService != nil && storeAppService.Replicas > 0 {
		timeout = time.Duration(storeAppService.Replicas) * timeout
	}
	if s.waiting != 0 {
		timeout = s.waiting
	}
	if err := WaitStop(s.manager.store, storeAppService, timeout, app.Logger, s.stopChan); err != nil {
		return err
	}
	return nil
}

func (s *stopController) deleteIngress(namespace, ingressName string, zero int64) error {
	err := s.manager.client.NetworkingV1().Ingresses(namespace).Delete(s.ctx, ingressName, metav1.DeleteOptions{
		GracePeriodSeconds: &zero,
	})
	if err != nil && !errors.IsNotFound(err) {
		return err
	}
	return nil
}

func (s *stopController) deleteBetaIngress(namespace, ingressName string, zero int64) error {
	err := s.manager.client.ExtensionsV1beta1().Ingresses(namespace).Delete(s.ctx, ingressName, metav1.DeleteOptions{
		GracePeriodSeconds: &zero,
	})
	if err != nil && !errors.IsNotFound(err) {
		return err
	}
	return nil
}
