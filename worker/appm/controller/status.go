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
	"errors"
	"fmt"
	"time"

	"github.com/wutong-paas/wutong/event"
	"github.com/wutong-paas/wutong/worker/appm/store"
	v1 "github.com/wutong-paas/wutong/worker/appm/types/v1"
	corev1 "k8s.io/api/core/v1"
)

// ErrWaitTimeOut wait time out
var ErrWaitTimeOut = errors.New("wait time out")

// ErrWaitCancel wait cancel
var ErrWaitCancel = errors.New("wait cancel")

// ErrPodStatus pod status error
var ErrPodStatus = errors.New("pod status error")

// WaitReady wait ready
func WaitReady(store store.Storer, a *v1.AppService, timeout time.Duration, logger event.Logger, cancel chan struct{}) error {
	if timeout < 80 {
		timeout = time.Second * 80
	}
	logger.Info(fmt.Sprintf("等待应用组件就绪超时时间：%ds", int(timeout.Seconds())), map[string]string{"step": "appruntime", "status": "running"})
	ticker := time.NewTicker(5 * time.Second)
	timer := time.NewTimer(timeout)
	defer printUnnormalPods(a, logger)
	defer timer.Stop()
	defer ticker.Stop()
	var i int
	for {
		if i > 2 {
			a = store.UpdateGetAppService(a.ServiceID)
		}
		if a.Ready() {
			return nil
		}

		if checkPodStatusOrBreak(a, logger) {
			return ErrPodStatus
		}
		select {
		case <-cancel:
			return ErrWaitCancel
		case <-timer.C:
			return ErrWaitTimeOut
		case <-ticker.C:
		}
		i++
	}
}

// WaitStop wait service stop complete
func WaitStop(store store.Storer, a *v1.AppService, timeout time.Duration, logger event.Logger, cancel chan struct{}) error {
	if a == nil {
		return nil
	}
	if timeout < 80 {
		timeout = time.Second * 80
	}
	logger.Info(fmt.Sprintf("等待应用组件关闭超时时间：%ds", int(timeout.Seconds())), map[string]string{"step": "appruntime", "status": "stopping"})
	ticker := time.NewTicker(timeout / 10)
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	defer ticker.Stop()
	var i int
	for {
		i++
		if i > 2 {
			a = store.UpdateGetAppService(a.ServiceID)
		}
		if a.IsClosed() {
			return nil
		}
		if checkPodStatusOrBreak(a, logger) {
			printUnnormalPods(a, logger)
			return ErrPodStatus
		}
		select {
		case <-cancel:
			return ErrWaitCancel
		case <-timer.C:
			return ErrWaitTimeOut
		case <-ticker.C:
		}
	}
}

// WaitUpgradeReady wait upgrade success
func WaitUpgradeReady(store store.Storer, a *v1.AppService, timeout time.Duration, logger event.Logger, cancel chan struct{}) error {
	if a == nil {
		return nil
	}
	if timeout < 40 {
		timeout = time.Second * 40
	}
	logger.Info(fmt.Sprintf("等待应用组件更新完成超时时间：%ds", int(timeout.Seconds())), map[string]string{"step": "appruntime", "status": "upgrading"})
	ticker := time.NewTicker(5 * time.Second)
	timer := time.NewTimer(timeout)
	defer printUnnormalPods(a, logger)
	defer timer.Stop()
	defer ticker.Stop()
	for {
		if a.UpgradeComlete() {
			return nil
		}
		if checkPodStatusOrBreak(a, logger) {
			return ErrPodStatus
		}
		select {
		case <-cancel:
			return ErrWaitCancel
		case <-timer.C:
			return ErrWaitTimeOut
		case <-ticker.C:
		}
	}
}

// checkPodStatusOrBreak 检测并打印 pod 状态，如果是确定的错误状态则返回 true
// 那么等待可以直接结束
func checkPodStatusOrBreak(a *v1.AppService, logger event.Logger) bool {
	canBreak := false
	var ready int32
	if a.GetStatefulSet() != nil {
		ready = a.GetStatefulSet().Status.ReadyReplicas
	}
	if a.GetDeployment() != nil {
		ready = a.GetDeployment().Status.ReadyReplicas
	}
	unready := int32(len(a.GetPods(false))) - ready
	logger.Info(fmt.Sprintf("检测应用组件运行实例 -> 当前实例总数：%d，已就绪：%d，未就绪：%d", len(a.GetPods(false)), ready, unready), map[string]string{"step": "appruntime", "status": "running"})

	pods := a.GetPods(false)
	for _, pod := range pods {
		for _, con := range pod.Status.Conditions {
			if con.Status == corev1.ConditionFalse {
				switch con.Type {
				case corev1.PodInitialized:
					logger.Info(fmt.Sprintf("等待组件实例 %s 初始化容器完成...", pod.Name), map[string]string{"step": "appruntime", "status": "notready"})
				case corev1.PodScheduled:
					logger.Info(fmt.Sprintf("等待组件实例 %s 调度完成...", pod.Name), map[string]string{"step": "appruntime", "status": "notready"})
				case corev1.PodReady, corev1.ContainersReady:
					logger.Info(fmt.Sprintf("等待组件实例 %s 运行就绪...", pod.Name), map[string]string{"step": "appruntime", "status": "notready"})
				}
				break
			}
		}

		if pod.Labels["version"] == a.DeployVersion {
			for _, cs := range pod.Status.ContainerStatuses {
				if !cs.Ready {
					if cs.State.Waiting != nil {
						switch cs.State.Waiting.Reason {
						case "ImagePullBackOff", "ErrImagePull":
							canBreak = true
						case "CrashLoopBackOff":
							canBreak = true
						case "CreateContainerConfigError":
							canBreak = true
						}
					}
				}
			}
		}
	}
	return canBreak
}

func printUnnormalPods(a *v1.AppService, logger event.Logger) {
	pods := a.GetPods(false)
	for _, pod := range pods {
		for _, con := range pod.Status.Conditions {
			if con.Status == corev1.ConditionFalse {
				switch con.Type {
				case corev1.PodInitialized:
					logger.Info(fmt.Sprintf("组件实例 %s 初始化容器未完成，请检查初始化插件配置是否正确。详细信息：", pod.Name), map[string]string{"step": "appruntime", "status": "notready"})
					logger.Info("	--- "+con.Message, map[string]string{"step": "appruntime", "status": "notready"})
				case corev1.PodScheduled:
					logger.Info(fmt.Sprintf("组件实例 %s 未调度成功，请检查集群资源是否充足或适当向下调整组件资源配置。详细信息：", pod.Name), map[string]string{"step": "appruntime", "status": "notready"})
					logger.Info("	--- "+con.Message, map[string]string{"step": "appruntime", "status": "notready"})
				case corev1.PodReady, corev1.ContainersReady:
					logger.Info(fmt.Sprintf("组件实例 %s 未完全就绪，请点击运行实例查看实例详情信息或查看组件实时日志。详细信息：", pod.Name), map[string]string{"step": "appruntime", "status": "notready"})
					logger.Info("	--- "+con.Message, map[string]string{"step": "appruntime", "status": "notready"})
				}
				break
			}
		}

		for _, cs := range pod.Status.ContainerStatuses {
			if !cs.Ready {
				if cs.State.Waiting != nil {
					switch cs.State.Waiting.Reason {
					case "ImagePullBackOff", "ErrImagePull":
						logger.Info(fmt.Sprintf("组件实例 %s 镜像拉取失败，请检查应用组件镜像源是否正确或镜像是否存在。错误信息：", pod.Name), map[string]string{"step": "appruntime", "status": "notready"})
						logger.Info("	--- "+cs.State.Waiting.Message, map[string]string{"step": "appruntime", "status": "notready"})
					case "CrashLoopBackOff":
						logger.Info(fmt.Sprintf("组件实例 %s 容器启动失败，请点击运行实例查看实例详情信息或查看组件实时日志。错误信息：", pod.Name), map[string]string{"step": "appruntime", "status": "notready"})
						logger.Info("	--- "+cs.State.Waiting.Message, map[string]string{"step": "appruntime", "status": "notready"})
					case "CreateContainerConfigError":
						logger.Info(fmt.Sprintf("组件实例 %s 容器配置错误，请检查组件容器存储是否成功挂载。错误信息：", pod.Name), map[string]string{"step": "appruntime", "status": "notready"})
						logger.Info("	--- "+cs.State.Waiting.Message, map[string]string{"step": "appruntime", "status": "notready"})
					}
				}
			}
		}
	}
}
