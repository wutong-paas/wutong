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

package entry

import (
	"net"
	"time"

	"github.com/wutong-paas/wutong/eventlog/conf"
	"github.com/wutong-paas/wutong/eventlog/store"
	"github.com/wutong-paas/wutong/eventlog/util"

	"golang.org/x/net/context"

	"fmt"

	"sync"

	zmq4 "github.com/pebbe/zmq4"
	"github.com/sirupsen/logrus"
)

// DockerLogServer 日志接受服务
type DockerLogServer struct {
	conf               conf.DockerLogServerConf
	log                *logrus.Entry
	cancel             func()
	context            context.Context
	server             *zmq4.Socket
	storemanager       store.Manager
	messageChan        chan []byte
	listenErr          chan error
	serverLock         sync.Mutex
	stopReceiveMessage bool
	bufferServer       *util.Server
	listen             *net.TCPListener
}

// NewDockerLogServer 创建zmq server服务端
func NewDockerLogServer(conf conf.DockerLogServerConf, log *logrus.Entry, storeManager store.Manager) (*DockerLogServer, error) {
	ctx, cancel := context.WithCancel(context.Background())
	s := &DockerLogServer{
		conf:         conf,
		log:          log,
		cancel:       cancel,
		context:      ctx,
		storemanager: storeManager,
		listenErr:    make(chan error),
	}
	s.log.Info("receive docker container log server start.")
	tcpAddr, err := net.ResolveTCPAddr("tcp4", fmt.Sprintf("%s:%d", s.conf.BindIP, s.conf.BindPort))
	if err != nil {
		s.log.Error("create stream log server address error.", err.Error())
		return nil, err
	}
	listener, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		s.log.Error("create stream log server listener error.", err.Error())
		return nil, err
	}
	s.listen = listener
	// creates a server
	config := &util.Config{
		PacketSendChanLimit:    10,
		PacketReceiveChanLimit: 5000,
	}
	s.bufferServer = util.NewServer(config, s, s.context)
	s.log.Infof("Docker container log server listen %s", tcpAddr)

	s.messageChan = s.storemanager.DockerLogMessageChan()
	return s, nil
}

// Serve 执行
func (s *DockerLogServer) Serve() {
	s.bufferServer.Start(s.listen, 3*time.Second)
}

// OnConnect is called when the connection was accepted,
// If the return value of false is closed
func (s *DockerLogServer) OnConnect(c *util.Conn) bool {
	s.log.Debugf("receive a log client connect.")
	return true
}

// OnMessage is called when the connection receives a packet,
// If the return value of false is closed
func (s *DockerLogServer) OnMessage(p util.Packet) bool {
	var msg = p.Serialize()
	if len(msg) > 0 {
		select {
		// eventlog receive message here
		case s.messageChan <- msg:
			return true
		default:
			//TODO: return false and receive exist
			return true
		}
	} else {
		logrus.Error("receive a null message")
	}
	return true
}

// OnClose is called when the connection closed
func (s *DockerLogServer) OnClose(*util.Conn) {
	s.log.Debugf("a log client closed.")
}

// Stop 停止
func (s *DockerLogServer) Stop() {
	s.cancel()
	if s.bufferServer != nil {
		s.bufferServer.Stop()
	}
	s.log.Info("receive event message server stop")
}

// ListenError listen error chan
func (s *DockerLogServer) ListenError() chan error {
	return s.listenErr
}
