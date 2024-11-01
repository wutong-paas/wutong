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

package server

import (
	"context"
	"fmt"
	"io"
	"net"

	"github.com/wutong-paas/wutong/eventlog/conf"
	"github.com/wutong-paas/wutong/eventlog/store"

	"github.com/wutong-paas/wutong/eventlog/entry/grpc/pb"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type EventLogRPCServer struct {
	conf         conf.EventLogServerConf
	log          *logrus.Entry
	cancel       func()
	context      context.Context
	storemanager store.Manager
	messageChan  chan []byte
	listenErr    chan error
	lis          net.Listener
	pb.UnimplementedEventLogServer
}

// NewServer server
func NewServer(conf conf.EventLogServerConf, log *logrus.Entry, storeManager store.Manager, listenErr chan error) *EventLogRPCServer {
	ctx, cancel := context.WithCancel(context.Background())
	return &EventLogRPCServer{
		conf:         conf,
		log:          log,
		storemanager: storeManager,
		context:      ctx,
		cancel:       cancel,
		messageChan:  storeManager.ReceiveMessageChan(),
		listenErr:    listenErr,
	}
}

// Start start grpc server
func (s *EventLogRPCServer) Start() error {
	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", s.conf.BindIP, s.conf.BindPort))
	if err != nil {
		logrus.Errorf("failed to listen: %v", err)
		return err
	}
	s.lis = lis
	server := grpc.NewServer()
	pb.RegisterEventLogServer(server, s)
	// Register reflection service on gRPC server.
	reflection.Register(server)
	s.log.Infof("event message grpc server listen %s:%d", s.conf.BindIP, s.conf.BindPort)
	if err := server.Serve(lis); err != nil {
		s.log.Error("event log api grpc listen error.", err.Error())
		s.listenErr <- err
	}
	return nil
}

// Stop stop
func (s *EventLogRPCServer) Stop() {
	s.cancel()
	// if s.lis != nil {
	// 	s.lis.Close()
	// }
}

// Log impl EventLogServerServer
func (s *EventLogRPCServer) Log(stream pb.EventLog_LogServer) error {
	for {
		select {
		case <-s.context.Done():
			if err := stream.SendAndClose(&pb.Reply{Status: "success", Message: "server closed"}); err != nil {
				return err
			}
			return nil
		default:
		}
		log, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				s.log.Error("receive log error:", err.Error())
				if err := stream.SendAndClose(&pb.Reply{Status: "success"}); err != nil {
					return err
				}
				return nil
			}
			return err
		}
		select {
		case s.messageChan <- log.Log:
		default:
		}
	}
}
