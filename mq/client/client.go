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

package client

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/wutong-paas/wutong/mq/api/grpc/pb"
	context "golang.org/x/net/context"
	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// BuilderTopic builder for linux
var BuilderTopic = "builder"

// WindowsBuilderTopic builder for windows
var WindowsBuilderTopic = "windows_builder"

// WorkerTopic worker topic
var WorkerTopic = "worker"

// MQClient mq  client
type MQClient interface {
	pb.TaskQueueClient
	Close()
	SendBuilderTopic(t TaskStruct) error
}

type mqClient struct {
	pb.TaskQueueClient
	ctx    context.Context
	cancel context.CancelFunc
}

// NewMqClient new a mq client
func NewMqClient(mqAddr string) (MQClient, error) {
	ctx, cancel := context.WithCancel(context.Background())
	// conn, err := grpc.NewClient(mqAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	conn, err := grpc.DialContext(ctx, mqAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	cli := pb.NewTaskQueueClient(conn)
	client := &mqClient{
		ctx:    ctx,
		cancel: cancel,
	}
	client.TaskQueueClient = cli
	return client, nil
}

// Close mq grpc client must be closed after uesd
func (m *mqClient) Close() {
	m.cancel()
}

// TaskStruct task struct
type TaskStruct struct {
	Topic    string
	TaskType string
	Operator string
	TaskBody interface{}
}

// buildTask build task
func buildTask(t TaskStruct) (*pb.EnqueueRequest, error) {
	var er pb.EnqueueRequest
	taskJSON, err := json.Marshal(t.TaskBody)
	if err != nil {
		logrus.Errorf("tran task json error")
		return &er, err
	}
	er.Topic = t.Topic
	er.Message = &pb.TaskMessage{
		TaskType:   t.TaskType,
		CreateTime: time.Now().Format(time.RFC3339),
		TaskBody:   taskJSON,
		User:       t.Operator,
	}
	return &er, nil
}

func (m *mqClient) SendBuilderTopic(t TaskStruct) error {
	request, err := buildTask(t)
	if err != nil {
		return fmt.Errorf("create task body error %s", err.Error())
	}
	ctx, cancel := context.WithTimeout(m.ctx, time.Second*5)
	defer cancel()
	_, err = m.TaskQueueClient.Enqueue(ctx, request)
	if err != nil {
		return fmt.Errorf("send enqueue request error %s", err.Error())
	}
	return nil
}
