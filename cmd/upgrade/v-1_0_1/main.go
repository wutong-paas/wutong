package main

import (
	"context"
	"sync"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
)

type TaskInfo struct {
	API     string
	Total   int
	Succeed int
	Failed  int
}

func newTaskInfo(api string) *TaskInfo {
	return &TaskInfo{
		API: api,
	}
}

var tasks = []*TaskInfo{
	newTaskInfo("deployments"),
	newTaskInfo("statefulsets"),
	newTaskInfo("pods"),
	newTaskInfo("configmaps"),
	newTaskInfo("secrets"),
	newTaskInfo("services"),
	newTaskInfo("ingresses"),
	newTaskInfo("horizontalpodautoscalers"),
	newTaskInfo("persistentvolumeclaims"),
}

var clientset *kubernetes.Clientset
var wg sync.WaitGroup

func init() {
	clientset = kubernetes.NewForConfigOrDie(ctrl.GetConfigOrDie())
}

func main() {

	wg.Add(len(tasks))
	for _, task := range tasks {
		switch task.API {
		case "deployments":
			go DoDeploymentTask(task)
		case "statefulsets":
			go DoStatefuleSetTask(task)
		case "pods":
			go DoPodTask(task)
		case "configmaps":
			go DoConfigMapTask(task)
		case "secrets":
			go DoSecretTask(task)
		case "services":
			go DoServiceTask(task)
		case "ingresses":
			go DoIngressTask(task)
		case "horizontalpodautoscalers":
			go DoHPATask(task)
		case "persistentvolumeclaims":
			go DoPVCTask(task)
		}
	}
}

func DoDeploymentTask(task *TaskInfo) {
	defer wg.Done()
	objs, err := clientset.AppsV1().Deployments(metav1.NamespaceAll).List(context.Background(), metav1.ListOptions{
		LabelSelector: "creator=Wutong",
	})
	if err != nil {
		return
	}
	for _, obj := range objs.Items {
		// 已经有了 tenant_env_id 标签，跳过
		if _, ok := obj.GetLabels()["tenant_env_id"]; ok {
			continue
		}

		// 有 tenant_id 标签：创建 tenant_env_id 标签，赋相同值
		// 没有 tenant_id 标签，跳过
		if v, ok := obj.GetLabels()["tenant_id"]; ok {
			obj.SetLabels(map[string]string{"tenant_env_id": v})
		} else {
			continue
		}

		// 有 tenant_name 标签：创建 tenant_env_name 标签，赋相同值
		// 没有 tenant_name 标签，跳过
		if v, ok := obj.GetLabels()["tenant_name"]; ok {
			obj.SetLabels(map[string]string{"tenant_env_name": v})
		} else {
			continue
		}

		task.Total = task.Total + 1
		_, err := clientset.AppsV1().Deployments(metav1.NamespaceAll).Update(context.Background(), &obj, metav1.UpdateOptions{})
		if err != nil {
			task.Failed = task.Failed + 1
		} else {
			task.Succeed = task.Succeed + 1
		}
	}
}

func DoStatefuleSetTask(task *TaskInfo) {
	defer wg.Done()
	objs, err := clientset.AppsV1().StatefulSets(metav1.NamespaceAll).List(context.Background(), metav1.ListOptions{
		LabelSelector: "creator=Wutong",
	})
	if err != nil {
		return
	}
	for _, obj := range objs.Items {
		// 已经有了 tenant_env_id 标签，跳过
		if _, ok := obj.GetLabels()["tenant_env_id"]; ok {
			continue
		}

		// 有 tenant_id 标签：创建 tenant_env_id 标签，赋相同值
		// 没有 tenant_id 标签，跳过
		if v, ok := obj.GetLabels()["tenant_id"]; ok {
			obj.SetLabels(map[string]string{"tenant_env_id": v})
		} else {
			continue
		}

		// 有 tenant_name 标签：创建 tenant_env_name 标签，赋相同值
		// 没有 tenant_name 标签，跳过
		if v, ok := obj.GetLabels()["tenant_name"]; ok {
			obj.SetLabels(map[string]string{"tenant_env_name": v})
		} else {
			continue
		}

		task.Total = task.Total + 1
		_, err := clientset.AppsV1().StatefulSets(metav1.NamespaceAll).Update(context.Background(), &obj, metav1.UpdateOptions{})
		if err != nil {
			task.Failed = task.Failed + 1
		} else {
			task.Succeed = task.Succeed + 1
		}
	}
}

func DoPodTask(task *TaskInfo) {
	defer wg.Done()
	objs, err := clientset.CoreV1().Pods(metav1.NamespaceAll).List(context.Background(), metav1.ListOptions{
		LabelSelector: "creator=Wutong",
	})
	if err != nil {
		return
	}
	for _, obj := range objs.Items {
		// 已经有了 tenant_env_id 标签，跳过
		if _, ok := obj.GetLabels()["tenant_env_id"]; ok {
			continue
		}

		// 有 tenant_id 标签：创建 tenant_env_id 标签，赋相同值
		// 没有 tenant_id 标签，跳过
		if v, ok := obj.GetLabels()["tenant_id"]; ok {
			obj.SetLabels(map[string]string{"tenant_env_id": v})
		} else {
			continue
		}

		// 有 tenant_name 标签：创建 tenant_env_name 标签，赋相同值
		// 没有 tenant_name 标签，跳过
		if v, ok := obj.GetLabels()["tenant_name"]; ok {
			obj.SetLabels(map[string]string{"tenant_env_name": v})
		} else {
			continue
		}

		task.Total = task.Total + 1
		_, err := clientset.CoreV1().Pods(metav1.NamespaceAll).Update(context.Background(), &obj, metav1.UpdateOptions{})
		if err != nil {
			task.Failed = task.Failed + 1
		} else {
			task.Succeed = task.Succeed + 1
		}
	}
}

func DoConfigMapTask(task *TaskInfo) {
	defer wg.Done()
	objs, err := clientset.CoreV1().ConfigMaps(metav1.NamespaceAll).List(context.Background(), metav1.ListOptions{
		LabelSelector: "creator=Wutong",
	})
	if err != nil {
		return
	}
	for _, obj := range objs.Items {
		// 已经有了 tenant_env_id 标签，跳过
		if _, ok := obj.GetLabels()["tenant_env_id"]; ok {
			continue
		}

		// 有 tenant_id 标签：创建 tenant_env_id 标签，赋相同值
		// 没有 tenant_id 标签，跳过
		if v, ok := obj.GetLabels()["tenant_id"]; ok {
			obj.SetLabels(map[string]string{"tenant_env_id": v})
		} else {
			continue
		}

		// 有 tenant_name 标签：创建 tenant_env_name 标签，赋相同值
		// 没有 tenant_name 标签，跳过
		if v, ok := obj.GetLabels()["tenant_name"]; ok {
			obj.SetLabels(map[string]string{"tenant_env_name": v})
		} else {
			continue
		}

		task.Total = task.Total + 1
		_, err := clientset.CoreV1().ConfigMaps(metav1.NamespaceAll).Update(context.Background(), &obj, metav1.UpdateOptions{})
		if err != nil {
			task.Failed = task.Failed + 1
		} else {
			task.Succeed = task.Succeed + 1
		}
	}
}

func DoSecretTask(task *TaskInfo) {
	defer wg.Done()
	objs, err := clientset.CoreV1().Secrets(metav1.NamespaceAll).List(context.Background(), metav1.ListOptions{
		LabelSelector: "creator=Wutong",
	})
	if err != nil {
		return
	}
	for _, obj := range objs.Items {
		// 已经有了 tenant_env_id 标签，跳过
		if _, ok := obj.GetLabels()["tenant_env_id"]; ok {
			continue
		}

		// 有 tenant_id 标签：创建 tenant_env_id 标签，赋相同值
		// 没有 tenant_id 标签，跳过
		if v, ok := obj.GetLabels()["tenant_id"]; ok {
			obj.SetLabels(map[string]string{"tenant_env_id": v})
		} else {
			continue
		}

		// 有 tenant_name 标签：创建 tenant_env_name 标签，赋相同值
		// 没有 tenant_name 标签，跳过
		if v, ok := obj.GetLabels()["tenant_name"]; ok {
			obj.SetLabels(map[string]string{"tenant_env_name": v})
		} else {
			continue
		}

		task.Total = task.Total + 1
		_, err := clientset.CoreV1().Secrets(metav1.NamespaceAll).Update(context.Background(), &obj, metav1.UpdateOptions{})
		if err != nil {
			task.Failed = task.Failed + 1
		} else {
			task.Succeed = task.Succeed + 1
		}
	}
}

func DoServiceTask(task *TaskInfo) {
	defer wg.Done()
	objs, err := clientset.CoreV1().Services(metav1.NamespaceAll).List(context.Background(), metav1.ListOptions{
		LabelSelector: "creator=Wutong",
	})
	if err != nil {
		return
	}
	for _, obj := range objs.Items {
		// 已经有了 tenant_env_id 标签，跳过
		if _, ok := obj.GetLabels()["tenant_env_id"]; ok {
			continue
		}

		// 有 tenant_id 标签：创建 tenant_env_id 标签，赋相同值
		// 没有 tenant_id 标签，跳过
		if v, ok := obj.GetLabels()["tenant_id"]; ok {
			obj.SetLabels(map[string]string{"tenant_env_id": v})
		} else {
			continue
		}

		// 有 tenant_name 标签：创建 tenant_env_name 标签，赋相同值
		// 没有 tenant_name 标签，跳过
		if v, ok := obj.GetLabels()["tenant_name"]; ok {
			obj.SetLabels(map[string]string{"tenant_env_name": v})
		} else {
			continue
		}

		task.Total = task.Total + 1
		_, err := clientset.CoreV1().Services(metav1.NamespaceAll).Update(context.Background(), &obj, metav1.UpdateOptions{})
		if err != nil {
			task.Failed = task.Failed + 1
		} else {
			task.Succeed = task.Succeed + 1
		}
	}
}

func DoIngressTask(task *TaskInfo) {
	defer wg.Done()
	objs, err := clientset.NetworkingV1().Ingresses(metav1.NamespaceAll).List(context.Background(), metav1.ListOptions{
		LabelSelector: "creator=Wutong",
	})
	if err != nil {
		return
	}
	for _, obj := range objs.Items {
		// 已经有了 tenant_env_id 标签，跳过
		if _, ok := obj.GetLabels()["tenant_env_id"]; ok {
			continue
		}

		// 有 tenant_id 标签：创建 tenant_env_id 标签，赋相同值
		// 没有 tenant_id 标签，跳过
		if v, ok := obj.GetLabels()["tenant_id"]; ok {
			obj.SetLabels(map[string]string{"tenant_env_id": v})
		} else {
			continue
		}

		// 有 tenant_name 标签：创建 tenant_env_name 标签，赋相同值
		// 没有 tenant_name 标签，跳过
		if v, ok := obj.GetLabels()["tenant_name"]; ok {
			obj.SetLabels(map[string]string{"tenant_env_name": v})
		} else {
			continue
		}

		task.Total = task.Total + 1
		_, err := clientset.NetworkingV1().Ingresses(metav1.NamespaceAll).Update(context.Background(), &obj, metav1.UpdateOptions{})
		if err != nil {
			task.Failed = task.Failed + 1
		} else {
			task.Succeed = task.Succeed + 1
		}
	}
}

func DoHPATask(task *TaskInfo) {
	defer wg.Done()
	objs, err := clientset.AutoscalingV1().HorizontalPodAutoscalers(metav1.NamespaceAll).List(context.Background(), metav1.ListOptions{
		LabelSelector: "creator=Wutong",
	})
	if err != nil {
		return
	}
	for _, obj := range objs.Items {
		// 已经有了 tenant_env_id 标签，跳过
		if _, ok := obj.GetLabels()["tenant_env_id"]; ok {
			continue
		}

		// 有 tenant_id 标签：创建 tenant_env_id 标签，赋相同值
		// 没有 tenant_id 标签，跳过
		if v, ok := obj.GetLabels()["tenant_id"]; ok {
			obj.SetLabels(map[string]string{"tenant_env_id": v})
		} else {
			continue
		}

		// 有 tenant_name 标签：创建 tenant_env_name 标签，赋相同值
		// 没有 tenant_name 标签，跳过
		if v, ok := obj.GetLabels()["tenant_name"]; ok {
			obj.SetLabels(map[string]string{"tenant_env_name": v})
		} else {
			continue
		}

		task.Total = task.Total + 1
		_, err := clientset.AutoscalingV1().HorizontalPodAutoscalers(metav1.NamespaceAll).Update(context.Background(), &obj, metav1.UpdateOptions{})
		if err != nil {
			task.Failed = task.Failed + 1
		} else {
			task.Succeed = task.Succeed + 1
		}
	}
}

func DoPVCTask(task *TaskInfo) {
	defer wg.Done()
	objs, err := clientset.CoreV1().PersistentVolumeClaims(metav1.NamespaceAll).List(context.Background(), metav1.ListOptions{
		LabelSelector: "creator=Wutong",
	})
	if err != nil {
		return
	}
	for _, obj := range objs.Items {
		// 已经有了 tenant_env_id 标签，跳过
		if _, ok := obj.GetLabels()["tenant_env_id"]; ok {
			continue
		}

		// 有 tenant_id 标签：创建 tenant_env_id 标签，赋相同值
		// 没有 tenant_id 标签，跳过
		if v, ok := obj.GetLabels()["tenant_id"]; ok {
			obj.SetLabels(map[string]string{"tenant_env_id": v})
		} else {
			continue
		}

		// 有 tenant_name 标签：创建 tenant_env_name 标签，赋相同值
		// 没有 tenant_name 标签，跳过
		if v, ok := obj.GetLabels()["tenant_name"]; ok {
			obj.SetLabels(map[string]string{"tenant_env_name": v})
		} else {
			continue
		}

		task.Total = task.Total + 1
		_, err := clientset.CoreV1().PersistentVolumeClaims(metav1.NamespaceAll).Update(context.Background(), &obj, metav1.UpdateOptions{})
		if err != nil {
			task.Failed = task.Failed + 1
		} else {
			task.Succeed = task.Succeed + 1
		}
	}
}
