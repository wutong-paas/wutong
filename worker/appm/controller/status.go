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
func WaitReady(store store.Storer, a *v1.AppService, timeout time.Duration, logger event.Logger, cancel chan struct{}) (err error) {
	if timeout < 80 {
		timeout = time.Second * 80
	}
	logger.Info(fmt.Sprintf("等待应用组件就绪超时时间：%ds", int(timeout.Seconds())), map[string]string{"step": "appruntime", "status": "running"})
	ticker := time.NewTicker(5 * time.Second)
	timer := time.NewTimer(timeout)
	defer func() {
		if err != nil {
			printAbnormalPods(a, logger)
		}
	}()
	defer timer.Stop()
	defer ticker.Stop()
	var i int
	for {
		if i > 2 {
			a = store.UpdateGetAppService(a.ServiceID)
		}

		select {
		case <-cancel:
			err = ErrWaitCancel
			return
		case <-timer.C:
			err = ErrWaitTimeOut
			return
		case <-ticker.C:
		}
		i++

		switch checkPodStatus(a, logger) {
		case waiting:
			continue
		case running:
			return nil
		case abnormal:
			err = ErrPodStatus
			return
		}

		if a.Ready() {
			return
		}
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

		switch checkPodStatus(a, logger) {
		case waiting:
		case running:
			return nil
		case abnormal:
			printAbnormalPods(a, logger)
			return ErrPodStatus
		}

		if a.IsClosed() {
			return nil
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
func WaitUpgradeReady(store store.Storer, a *v1.AppService, timeout time.Duration, logger event.Logger, cancel chan struct{}) (err error) {
	if a == nil {
		return
	}
	if timeout < 40 {
		timeout = time.Second * 40
	}
	logger.Info(fmt.Sprintf("等待应用组件更新完成超时时间：%ds", int(timeout.Seconds())), map[string]string{"step": "appruntime", "status": "upgrading"})
	ticker := time.NewTicker(5 * time.Second)
	timer := time.NewTimer(timeout)
	defer func() {
		if err != nil {
			printAbnormalPods(a, logger)
		}
	}()
	defer timer.Stop()
	defer ticker.Stop()
	var i int
	for {
		if i > 2 {
			a = store.UpdateGetAppService(a.ServiceID)
		}
		select {
		case <-cancel:
			err = ErrWaitCancel
			return
		case <-timer.C:
			err = ErrWaitTimeOut
			return
		case <-ticker.C:
		}
		i++

		switch checkPodStatus(a, logger) {
		case waiting:
			continue
		case running:
			return
		case abnormal:
			err = ErrPodStatus
			return
		}

		if a.UpgradeComlete() {
			return
		}
	}
}

// podCheckStatus，pod 状态检测结果，枚举值：waiting、running、abnormal
type podCheckStatus string

const (
	// 等待中状态，如果检测到该状态，则继续检测
	waiting podCheckStatus = "waiting"
	// 运行中状态，如果检测到该状态，则判定任务已经完成，也可以结束等待
	running podCheckStatus = "running"
	// 非正常状态，如果检测到该状态，则结束任务等待，并打印错误信息
	abnormal podCheckStatus = "abnormal" // 如果检测
)

// checkPodStatus 检测并打印 pod 状态，如果是确定的错误状态则返回 true
// 那么等待可以直接结束
func checkPodStatus(a *v1.AppService, logger event.Logger) podCheckStatus {
	podCheckStatus := waiting
	var newAvailableReplicas int32

	pods := a.GetPods(false)

	var podStatusMessage string
	for _, pod := range pods {
		if !isNewPod(pod, a) {
			continue
		}

		if pod.Status.Phase == corev1.PodRunning {
			newAvailableReplicas++
		}

		for _, con := range pod.Status.Conditions {
			if con.Status == corev1.ConditionFalse {
				switch con.Type {
				case corev1.PodInitialized:
					podStatusMessage = fmt.Sprintf("等待组件实例 %s 初始化容器完成...", pod.Name)
				case corev1.PodScheduled:
					podStatusMessage = fmt.Sprintf("等待组件实例 %s 调度完成...", pod.Name)
				case corev1.PodReady, corev1.ContainersReady:
					podStatusMessage = fmt.Sprintf("等待组件实例 %s 运行就绪...", pod.Name)
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
							podCheckStatus = abnormal
						case "CrashLoopBackOff":
							podCheckStatus = abnormal
						case "CreateContainerConfigError":
							podCheckStatus = abnormal
						}
					}
				}
			}
		}
	}

	logger.Info(fmt.Sprintf("待更新应用组件运行实例：[%d/%d]", newAvailableReplicas, a.Replicas), map[string]string{"step": "appruntime", "status": "running"})
	if podStatusMessage != "" {
		logger.Info(podStatusMessage, map[string]string{"step": "appruntime", "status": "notready"})
	}

	if a.Replicas == int(newAvailableReplicas) && podCheckStatus != abnormal {
		podCheckStatus = running
	}
	return podCheckStatus
}

// printAbnormalPods 打印非正常状态 Pod 信息
func printAbnormalPods(a *v1.AppService, logger event.Logger) {
	pods := a.GetPods(false)
	for _, pod := range pods {
		if !isNewPod(pod, a) {
			continue
		}

		// 通过查看 State，知道明确的错误状态，然后打印错误信息
		var stateFailed bool
		for _, cs := range pod.Status.ContainerStatuses {
			if !cs.Ready {
				if cs.State.Waiting != nil {
					var printDetail bool
					switch cs.State.Waiting.Reason {
					case "ImagePullBackOff", "ErrImagePull":
						logger.Info(fmt.Sprintf("组件实例 %s 镜像拉取失败，请检查应用组件镜像源是否正确或镜像是否存在。", pod.Name), map[string]string{"step": "appruntime", "status": "notready"})
						printDetail = true
						stateFailed = true
					case "CrashLoopBackOff":
						logger.Info(fmt.Sprintf("组件实例 %s 容器运行失败并不断重启，请点击运行实例查看实例详情信息或查看组件实时日志。", pod.Name), map[string]string{"step": "appruntime", "status": "notready"})
						printDetail = true
						stateFailed = true
					case "CreateContainerConfigError":
						logger.Info(fmt.Sprintf("组件实例 %s 容器配置错误，请检查组件容器存储是否成功挂载。", pod.Name), map[string]string{"step": "appruntime", "status": "notready"})
						printDetail = true
						stateFailed = true
					}
					if printDetail && cs.State.Waiting.Message != "" {
						logger.Info("::: 错误信息："+cs.State.Waiting.Message, map[string]string{"step": "appruntime", "status": "notready"})
					}
				}
			}
		}
		if stateFailed {
			continue
		}

		// 非明确的错误状态，通过查看 Conditions，打印提示信息
		for _, con := range pod.Status.Conditions {
			if con.Status == corev1.ConditionFalse {
				var printDetail bool
				switch con.Type {
				case corev1.PodInitialized:
					logger.Info(fmt.Sprintf("组件实例 %s 初始化容器未完成，请检查初始化插件配置是否正确。", pod.Name), map[string]string{"step": "appruntime", "status": "notready"})
					printDetail = true
				case corev1.PodScheduled:
					logger.Info(fmt.Sprintf("组件实例 %s 未完成调度，请检查集群资源是否充足或适当向下调整组件资源配置。", pod.Name), map[string]string{"step": "appruntime", "status": "notready"})
					printDetail = true
				case corev1.PodReady, corev1.ContainersReady:
					logger.Info(fmt.Sprintf("组件实例 %s 未完全就绪，请点击运行实例查看实例详情信息或查看组件实时日志。", pod.Name), map[string]string{"step": "appruntime", "status": "notready"})
					printDetail = true
				}
				if printDetail && con.Reason != "" {
					logger.Info("::: 详细信息："+con.Message, map[string]string{"step": "appruntime", "status": "notready"})
				}
				break
			}
		}
	}
}

// isNewPod 是否是更新的一批 Pod
func isNewPod(pod *corev1.Pod, a *v1.AppService) bool {
	if pod.Labels["version"] != a.DeployVersion {
		return false
	}

	if a.GetNewestReplicaSet() != nil {
		if pod.Labels["pod-template-hash"] != a.GetNewestReplicaSet().Labels["pod-template-hash"] {
			return false
		}
	}

	if a.GetStatefulSet() != nil {
		if pod.Labels["controller-revision-hash"] != a.GetStatefulSet().Status.UpdateRevision {
			return false
		}
	}

	return true
}
