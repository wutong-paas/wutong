package main

import (
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
	k8sutil "github.com/wutong-paas/wutong/util/k8s"
	"github.com/wutong-paas/wutong/webcli/app"
	restclient "k8s.io/client-go/rest"
)

func main() {
	option := app.DefaultOptions
	option.K8SConfPath = "/root/.kube/config"
	config, err := k8sutil.NewRestConfig(option.K8SConfPath)
	if err != nil {
		logrus.Error(err)
	}
	config.UserAgent = "wutong/webcli"
	app.SetConfigDefaults(config)
	restClient, err := restclient.RESTClientFor(config)
	if err != nil {
		logrus.Error(err)
	}
	namespace := os.Getenv("WT_NAMESPACE")
	if namespace == "" {
		namespace = "wt-system"
	}
	commands := []string{"sh"}
	req := restClient.Post().
		Resource("pods").
		Name("wutong-operator-0").
		Namespace(namespace).
		SubResource("exec").
		Param("container", "operator").
		Param("stdin", "true").
		Param("stdout", "true").
		Param("tty", "true")
	for _, c := range commands {
		req.Param("command", c)
	}

	slave, err := app.NewExecContext(req, config)
	if err != nil {
		logrus.Error(err)
		return
	}
	slave.ResizeTerminal(100, 60)
	defer slave.Close()
	for {
		buffer := make([]byte, 1024)
		n, err := slave.Read(buffer)
		if err != nil {
			logrus.Error(err)
			return
		}
		fmt.Print(string(buffer[:n]))
	}
}
