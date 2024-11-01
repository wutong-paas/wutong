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
	"errors"
	"net"
	"time"

	"github.com/wutong-paas/wutong/eventlog/conf"
	"github.com/wutong-paas/wutong/eventlog/store"

	"golang.org/x/net/context"

	"sync"

	"github.com/sirupsen/logrus"
)

// NMonitorMessageServer 新性能分析实时数据接受服务
type NMonitorMessageServer struct {
	conf               conf.NewMonitorMessageServerConf
	log                *logrus.Entry
	cancel             func()
	context            context.Context
	storemanager       store.Manager
	messageChan        chan []byte
	listenErr          chan error
	serverLock         sync.Mutex
	stopReceiveMessage bool
	listener           *net.UDPConn
}

// NewNMonitorMessageServer 创建UDP服务端
func NewNMonitorMessageServer(conf conf.NewMonitorMessageServerConf, log *logrus.Entry, storeManager store.Manager) (*NMonitorMessageServer, error) {
	ctx, cancel := context.WithCancel(context.Background())
	s := &NMonitorMessageServer{
		conf:         conf,
		log:          log,
		cancel:       cancel,
		context:      ctx,
		storemanager: storeManager,
		listenErr:    make(chan error),
	}
	listener, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP(conf.ListenerHost), Port: conf.ListenerPort})
	if err != nil {
		logrus.Errorf("failed to listen on udp %s:%d, %s", conf.ListenerHost, conf.ListenerPort, err.Error())
		return nil, err
	}
	log.Infof("UDP Server Listener: %s", listener.LocalAddr().String())
	s.listener = listener
	s.messageChan = s.storemanager.NewMonitorMessageChan()
	if s.messageChan == nil {
		return nil, errors.New("receive monitor message server can not get store message chan ")
	}
	return s, nil
}

// Serve 执行
func (s *NMonitorMessageServer) Serve() {
	s.handleMessage()
}

// Stop 停止
func (s *NMonitorMessageServer) Stop() {
	s.cancel()
	s.log.Info("receive new monitor message server stop")
}

func (s *NMonitorMessageServer) handleMessage() {
	buf := make([]byte, 65535)
	defer s.listener.Close()
	s.log.Infoln("start receive monitor message by udp")
	for {
		n, _, err := s.listener.ReadFromUDP(buf)
		if err != nil {
			logrus.Errorf("read new monitor message from udp error,%s", err.Error())
			time.Sleep(time.Second * 2)
			continue
		}
		// fix issues https://github.com/golang/go/issues/35725
		message := make([]byte, n)
		copy(message, buf[0:n])
		s.messageChan <- message
	}
}

// ListenError listen error chan
func (s *NMonitorMessageServer) ListenError() chan error {
	return s.listenErr
}
