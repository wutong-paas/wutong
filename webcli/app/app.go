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

package app

import (
	"context"
	"crypto/md5"

	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"net/http/pprof"
	"strings"
	"text/template"

	"github.com/go-chi/chi"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/barnettZQG/gotty/server"
	"github.com/barnettZQG/gotty/webtty"
	"github.com/gorilla/websocket"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/wutong-paas/wutong/util"
	httputil "github.com/wutong-paas/wutong/util/http"
	k8sutil "github.com/wutong-paas/wutong/util/k8s"
	"github.com/yudai/umutex"
	api "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	kubevirtcorev1 "kubevirt.io/api/core/v1"
)

// ExecuteCommandTotal metric
var ExecuteCommandTotal float64

// ExecuteCommandFailed metric
var ExecuteCommandFailed float64

// App -
type App struct {
	options *Options

	upgrader *websocket.Upgrader

	titleTemplate *template.Template

	onceMutex     *umutex.UnblockingMutex
	restClient    *rest.RESTClient
	dynamicClient dynamic.Interface
	coreClient    *kubernetes.Clientset
	config        *rest.Config
}

// Options options
type Options struct {
	Address     string `hcl:"address"`
	Port        string `hcl:"port"`
	PermitWrite bool   `hcl:"permit_write"`
	IndexFile   string `hcl:"index_file"`
	//titile format by golang templete
	TitleFormat     string                 `hcl:"title_format"`
	EnableReconnect bool                   `hcl:"enable_reconnect"`
	ReconnectTime   int                    `hcl:"reconnect_time"`
	PermitArguments bool                   `hcl:"permit_arguments"`
	CloseSignal     int                    `hcl:"close_signal"`
	RawPreferences  map[string]interface{} `hcl:"preferences"`
	SessionKey      string                 `hcl:"session_key"`
	K8SConfPath     string
}

// Version -
var Version = "0.0.2"

// DefaultOptions -
var DefaultOptions = Options{
	Address:         "",
	Port:            "8080",
	PermitWrite:     true,
	IndexFile:       "",
	TitleFormat:     "GRTTY Command",
	EnableReconnect: true,
	ReconnectTime:   10,
	CloseSignal:     1, // syscall.SIGHUP
	SessionKey:      "_auth_user_id",
}

// InitMessage -
type InitMessage struct {
	// TenantEnvID   string `json:"T_id"`
	// ServiceID     string `json:"S_id"`
	PodName       string `json:"C_id"`
	ContainerName string `json:"containerName"`
	Md5           string `json:"Md5"`
	Namespace     string `json:"namespace"`
	// NodeName      string `json:"nodeName"`
}

type InitVMMessage struct {
	// TenantEnvID   string `json:"T_id"`
	Md5         string `json:"Md5"`
	VMID        string `json:"vmID"`
	VMNamespace string `json:"vmNamespace"`
	// VMIP        string `json:"vmIP"`
	VMPort     string `json:"vmPort"`
	VMUser     string `json:"vmUser"`
	VMPassword string `json:"vmPassword"`
}

func checkSameOrigin(r *http.Request) bool {
	return true
}

// New -
func New(options *Options) (*App, error) {
	titleTemplate, _ := template.New("title").Parse(options.TitleFormat)
	app := &App{
		options: options,
		upgrader: &websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			Subprotocols:    []string{"webtty"},
			CheckOrigin:     checkSameOrigin,
		},
		titleTemplate: titleTemplate,
		onceMutex:     umutex.New(),
	}
	//create kube client and config
	if err := app.createKubeClient(); err != nil {
		return nil, err
	}
	return app, nil
}

// Run Run
func (app *App) Run() error {

	endpoint := net.JoinHostPort(app.options.Address, app.options.Port)

	wsHandler := http.HandlerFunc(app.handleWS)
	virtctlConsoleChannelWSHandler := http.HandlerFunc(app.handleVirtctlConsoleChannelWS)
	virtualMachineSSHChannelWSHandler := http.HandlerFunc(app.handleVirtualMachineSSHChannelWS)
	health := http.HandlerFunc(app.healthCheck)

	var siteMux = http.NewServeMux()

	siteHandler := http.Handler(siteMux)

	siteHandler = wrapHeaders(siteHandler)

	exporter := NewExporter()
	prometheus.MustRegister(exporter)

	wsMux := http.NewServeMux()
	wsMux.Handle("/", siteHandler)
	wsMux.Handle("/docker_console", wsHandler)
	wsMux.Handle("/docker_virtctl_console", virtctlConsoleChannelWSHandler)
	wsMux.Handle("/docker_vm_ssh", virtualMachineSSHChannelWSHandler)
	wsMux.Handle("/health", health)
	wsMux.Handle("/metrics", promhttp.Handler())
	wsMux.HandleFunc("/debug/pprof/", pprof.Index)

	siteHandler = (http.Handler(wsMux))

	siteHandler = wrapLogger(siteHandler)

	server, err := app.makeServer(endpoint, &siteHandler)
	if err != nil {
		return errors.New("Failed to build server: " + err.Error())
	}
	go func() {
		logrus.Printf("webcli listen %s", endpoint)
		logrus.Fatal(server.ListenAndServe())
		logrus.Printf("Exiting...")
	}()
	return nil
}

func (app *App) makeServer(addr string, handler *http.Handler) (*http.Server, error) {
	server := &http.Server{
		Addr:    addr,
		Handler: *handler,
	}

	return server, nil
}

func (app *App) healthCheck(w http.ResponseWriter, r *http.Request) {
	httputil.ReturnSuccess(r, w, map[string]string{"status": "health", "info": "webcli service health"})
}

func (app *App) handleWS(w http.ResponseWriter, r *http.Request) {
	logrus.Printf("New client connected: %s", r.RemoteAddr)

	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	conn, err := app.upgrader.Upgrade(w, r, nil)
	if err != nil {
		logrus.Print("Failed to upgrade connection: " + err.Error())
		return
	}

	_, stream, err := conn.ReadMessage()
	if err != nil {
		logrus.Print("Failed to authenticate websocket connection " + err.Error())
		conn.Close()
		return
	}

	message := string(stream)
	logrus.Print("message=", message)

	var init InitMessage

	json.Unmarshal(stream, &init)

	//todo auth
	if init.PodName == "" {
		logrus.Print("Parameter is error, pod name is empty")
		conn.WriteMessage(websocket.TextMessage, []byte("pod name can not be empty"))
		conn.Close()
		return
	}
	// key := init.TenantEnvID + "_" + init.ServiceID + "_" + init.PodName
	// md5 := md5Func(key)
	// if md5 != init.Md5 {
	// 	logrus.Print("Auth is not allowed!")
	// 	conn.WriteMessage(websocket.TextMessage, []byte("Auth is not allowed!"))
	// 	conn.Close()
	// 	return
	// }
	// base kubernetes api create exec slave
	// if init.Namespace == "" {
	// 	init.Namespace = init.TenantEnvID
	// }
	containerName, ip, args, err := app.GetContainerArgs(init.Namespace, init.PodName, init.ContainerName)
	if err != nil {
		logrus.Errorf("get default container failure %s", err.Error())
		conn.WriteMessage(websocket.TextMessage, []byte("Get default container name failure!"))
		ExecuteCommandFailed++
		return
	}
	slave, err := app.tryExecRequest(init.Namespace, init.PodName, containerName, args)
	// request := app.NewRequest(init.PodName, init.Namespace, containerName, args)
	// var slave server.Slave
	// slave, err = NewExecContext(request, app.config)
	if err != nil {
		logrus.Errorf("open exec context failure %s", err.Error())
		conn.WriteMessage(websocket.TextMessage, []byte("open tty failure!"))
		ExecuteCommandFailed++
		return
	}
	defer slave.Close()
	opts := []webtty.Option{
		webtty.WithWindowTitle([]byte(ip)),
		webtty.WithReconnect(10),
		webtty.WithPermitWrite(),
	}
	// create web tty and run
	tty, err := webtty.New(&WsWrapper{conn}, slave, opts...)
	if err != nil {
		logrus.Errorf("open web tty context failure %s", err.Error())
		conn.WriteMessage(websocket.TextMessage, []byte("open tty failure!"))
		ExecuteCommandFailed++
		return
	}
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	err = tty.Run(ctx)
	if err != nil {
		if strings.Contains(err.Error(), "master closed") {
			logrus.Infof("client close connection")
			return
		}
		logrus.Errorf("run web tty failure %s", err.Error())
		conn.WriteMessage(websocket.TextMessage, []byte("run tty failure!"))
		ExecuteCommandFailed++
		return
	}
}

func (app *App) handleVirtctlConsoleChannelWS(w http.ResponseWriter, r *http.Request) {
	logrus.Printf("New client connected: %s", r.RemoteAddr)

	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	conn, err := app.upgrader.Upgrade(w, r, nil)
	if err != nil {
		logrus.Print("Failed to upgrade connection: " + err.Error())
		return
	}

	_, stream, err := conn.ReadMessage()
	if err != nil {
		logrus.Print("Failed to authenticate websocket connection " + err.Error())
		conn.Close()
		return
	}

	message := string(stream)
	logrus.Print("message=", message)

	var init InitVMMessage

	json.Unmarshal(stream, &init)

	containerName, podName, args, err := app.GetVirtctlConsoleChannelArgs(init.VMNamespace, init.VMID)
	if err != nil {
		logrus.Errorf("get default container failure %s", err.Error())
		conn.WriteMessage(websocket.TextMessage, []byte("Get default container name failure!"))
		ExecuteCommandFailed++
		return
	}

	slave, err := app.tryExecRequest("wt-system", podName, containerName, args)
	if err != nil {
		logrus.Errorf("open exec context failure %s", err.Error())
		conn.WriteMessage(websocket.TextMessage, []byte("open tty failure!"))
		ExecuteCommandFailed++
		return
	}
	defer slave.Close()
	opts := []webtty.Option{
		webtty.WithWindowTitle([]byte(podName)),
		webtty.WithReconnect(10),
		webtty.WithPermitWrite(),
	}
	// create web tty and run
	tty, err := webtty.New(&WsWrapper{conn}, slave, opts...)
	if err != nil {
		logrus.Errorf("open web tty context failure %s", err.Error())
		conn.WriteMessage(websocket.TextMessage, []byte("open tty failure!"))
		ExecuteCommandFailed++
		return
	}
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	err = tty.Run(ctx)
	if err != nil {
		if strings.Contains(err.Error(), "master closed") {
			logrus.Infof("client close connection")
			return
		}
		logrus.Errorf("run web tty failure %s", err.Error())
		conn.WriteMessage(websocket.TextMessage, []byte("run tty failure!"))
		ExecuteCommandFailed++
		return
	}
}

func (app *App) handleVirtualMachineSSHChannelWS(w http.ResponseWriter, r *http.Request) {
	logrus.Printf("New client connected: %s", r.RemoteAddr)

	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	conn, err := app.upgrader.Upgrade(w, r, nil)
	if err != nil {
		logrus.Print("Failed to upgrade connection: " + err.Error())
		return
	}

	_, stream, err := conn.ReadMessage()
	if err != nil {
		logrus.Print("Failed to authenticate websocket connection " + err.Error())
		conn.Close()
		return
	}

	chi.URLParam(r, "vmID")

	message := string(stream)
	logrus.Print("message=", message)

	var init InitVMMessage

	json.Unmarshal(stream, &init)

	containerName, podName, args, err := app.GetVirtualMachineSSHChannelArgs(init.VMNamespace, init.VMID, init.VMPort, init.VMUser)
	if err != nil {
		logrus.Errorf("get default container failure %s", err.Error())
		conn.WriteMessage(websocket.TextMessage, []byte("Get default container name failure!"))
		ExecuteCommandFailed++
		return
	}

	slave, err := app.tryExecRequest("wt-system", podName, containerName, args)
	if err != nil {
		logrus.Errorf("open exec context failure %s", err.Error())
		conn.WriteMessage(websocket.TextMessage, []byte("open tty failure!"))
		ExecuteCommandFailed++
		return
	}
	defer slave.Close()
	opts := []webtty.Option{
		webtty.WithWindowTitle([]byte(podName)),
		webtty.WithReconnect(10),
		webtty.WithPermitWrite(),
	}
	// create web tty and run
	tty, err := webtty.New(&WsWrapper{conn}, slave, opts...)
	if err != nil {
		logrus.Errorf("open web tty context failure %s", err.Error())
		conn.WriteMessage(websocket.TextMessage, []byte("open tty failure!"))
		ExecuteCommandFailed++
		return
	}
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	err = tty.Run(ctx)
	if err != nil {
		if strings.Contains(err.Error(), "master closed") {
			logrus.Infof("client close connection")
			return
		}
		logrus.Errorf("run web tty failure %s", err.Error())
		conn.WriteMessage(websocket.TextMessage, []byte("run tty failure!"))
		ExecuteCommandFailed++
		return
	}
}

func (app *App) tryExecRequest(ns, pod, c string, args []string) (server.Slave, error) {
	request := app.NewRequest(pod, ns, c, args)
	var slave server.Slave
	slave, err := NewExecContext(request, app.config)
	if err != nil {
		// 如果是 /bin/bash 失败了，那么使用 /bin/sh 重试
		if args[0] == "/bin/bash" {
			args[0] = "/bin/sh"
			return app.tryExecRequest(ns, pod, c, args)
		}
	}
	return slave, err
}

// Exit -
func (app *App) Exit() (firstCall bool) {
	return true
}

func (app *App) createKubeClient() error {
	config, err := k8sutil.NewRestConfig(app.options.K8SConfPath)
	if err != nil {
		return err
	}
	config.UserAgent = "wutong/webcli"
	coreAPI, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}
	SetConfigDefaults(config)
	app.config = config
	restClient, err := rest.RESTClientFor(config)
	if err != nil {
		return err
	}
	app.restClient = restClient
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return err
	}
	app.dynamicClient = dynamicClient
	app.coreClient = coreAPI
	return nil
}

// SetConfigDefaults -
func SetConfigDefaults(config *rest.Config) error {
	if config.APIPath == "" {
		config.APIPath = "/api"
	}
	config.GroupVersion = &schema.GroupVersion{Group: "", Version: "v1"}
	config.NegotiatedSerializer = serializer.NewCodecFactory(runtime.NewScheme())
	if config.UserAgent == "" {
		config.UserAgent = rest.DefaultKubernetesUserAgent()
	}
	return nil
}

// GetContainerArgs get default container name
func (app *App) GetContainerArgs(namespace, podname, containerName string) (string, string, []string, error) {
	var args = []string{"/bin/bash"}
	pod, err := app.coreClient.CoreV1().Pods(namespace).Get(context.Background(), podname, metav1.GetOptions{})
	if err != nil {
		return "", "", args, err
	}

	if pod.Status.Phase == api.PodSucceeded || pod.Status.Phase == api.PodFailed {
		return "", "", args, fmt.Errorf("cannot exec into a container in a completed pod; current phase is %s", pod.Status.Phase)
	}
	for i, container := range pod.Spec.Containers {
		if container.Name == containerName || (containerName == "" && i == 0) {
			for _, env := range container.Env {
				if env.Name == "ES_DEFAULT_EXEC_ARGS" {
					args = strings.Split(env.Value, " ")
				}
			}
			return container.Name, pod.Status.PodIP, args, nil
		}
	}
	return "", "", args, fmt.Errorf("not have container in pod %s/%s", namespace, podname)
}

// GetVirtctlConsoleChannelArgs return containerName, podName, args, error
func (app *App) GetVirtctlConsoleChannelArgs(vmNamespace, vmID string) (string, string, []string, error) {
	// var args = []string{"/bin/bash", "-c", "ssh root@[vm-ssh-22][namespace]"}
	var args = []string{"virtctl", "console", "-n", vmNamespace, vmID}

	virtctlSts, err := app.coreClient.AppsV1().StatefulSets("wt-system").Get(context.Background(), "wt-channel", metav1.GetOptions{})
	if err != nil {
		return "wt-channel", "", args, fmt.Errorf("wt-channel is not ready")
	}

	randPodNo := rand.Int31n(util.Value(virtctlSts.Spec.Replicas))
	podName := fmt.Sprintf("wt-channel-%d", randPodNo)

	pod, err := app.coreClient.CoreV1().Pods("wt-system").Get(context.Background(), podName, metav1.GetOptions{})
	if err != nil || pod.Status.Phase == api.PodRunning {
		return "wt-channel", podName, args, fmt.Errorf("wt-channel is not ready")
	}

	return "wt-channel", podName, args, nil
}

var vmGVR = schema.GroupVersionResource{
	Group:    "kubevirt.io",
	Version:  "v1",
	Resource: "virtualmachineinstances",
}

// GetVirtualMachineSSHChannelArgs return containerName, podName, args, error
func (app *App) GetVirtualMachineSSHChannelArgs(vmNamespace, vmID, vmPort, vmUser string) (string, string, []string, error) {
	unstructuredVM, err := app.dynamicClient.Resource(vmGVR).Namespace(vmNamespace).Get(context.Background(), vmID, metav1.GetOptions{})
	if err != nil {
		logrus.Errorf("failed to get vm %s/%s error %s", vmNamespace, vmID, err.Error())
		return "wt-channel", "", nil, fmt.Errorf("failed to get vm %s/%s", vmNamespace, vmID)
	}
	var vm = &kubevirtcorev1.VirtualMachineInstance{}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredVM.Object, vm)
	if err != nil {
		logrus.Errorf("failed to convert unstructured vm %s/%s error %s", vmNamespace, vmID, err.Error())
		return "wt-channel", "", nil, fmt.Errorf("failed to convert unstructured vm %s/%s", vmNamespace, vmID)
	}

	var vmIP string
	if len(vm.Status.Interfaces) > 0 {
		vmIP = vm.Status.Interfaces[0].IP
	}
	if vmIP == "" {
		return "wt-channel", "", nil, fmt.Errorf("vm %s/%s has no an available IP address", vmNamespace, vmID)
	}

	if vmUser == "" {
		vmUser = "root"
	}

	if vmPort == "" {
		vmPort = "22"
	}

	var args = []string{"ssh", fmt.Sprintf("%s@%s", vmUser, vmIP), "-p", vmPort}

	virtctlSts, err := app.coreClient.AppsV1().StatefulSets("wt-system").Get(context.Background(), "wt-channel", metav1.GetOptions{})
	if err != nil {
		return "wt-channel", "", args, fmt.Errorf("wt-channel is not ready")
	}

	randPodNo := rand.Int31n(util.Value(virtctlSts.Spec.Replicas))
	podName := fmt.Sprintf("wt-channel-%d", randPodNo)

	pod, err := app.coreClient.CoreV1().Pods("wt-system").Get(context.Background(), podName, metav1.GetOptions{})
	if err != nil || pod.Status.Phase != api.PodRunning {
		return "wt-channel", podName, args, fmt.Errorf("wt-channel is not ready")
	}

	return "wt-channel", podName, args, nil
}

// NewRequest new exec request
func (app *App) NewRequest(podName, namespace, containerName string, command []string) *rest.Request {
	// TODO: consider abstracting into a client invocation or client helper
	req := app.restClient.Post().
		Resource("pods").
		Name(podName).
		Namespace(namespace).
		SubResource("exec").
		Param("container", containerName).
		Param("stdin", "true").
		Param("stdout", "true").
		Param("stderr", "false").
		Param("tty", "true")
	for _, c := range command {
		req.Param("command", c)
	}
	return req
}

func wrapLogger(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rw := &responseWrapper{w, 200}
		handler.ServeHTTP(rw, r)
		logrus.Printf("%s %d %s %s", r.RemoteAddr, rw.status, r.Method, r.URL.Path)
	})
}

func wrapHeaders(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Server", "GoTTY/"+Version)
		handler.ServeHTTP(w, r)
	})
}

func md5Func(str string) string {
	h := md5.New()
	h.Write([]byte(str))
	cipherStr := h.Sum(nil)
	return hex.EncodeToString(cipherStr)
}
