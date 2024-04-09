// WUTONG, Application Management Platform
// Copyright (C) 2014-2020 Wutong Co., Ltd.

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
	"fmt"
	"io"
	"os"
	"syscall"
	"time"
	"unsafe"

	"github.com/barnettZQG/gotty/server"
	"github.com/kr/pty"
	"github.com/sirupsen/logrus"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
)

type execContext struct {
	tty, pty    *os.File
	kubeRequest *restclient.Request
	config      *restclient.Config
	sizeUpdate  chan remotecommand.TerminalSize
	closed      bool
}

// NewExecContext new exec Context
func NewExecContext(kubeRequest *restclient.Request, config *restclient.Config) (server.Slave, error) {
	pty, tty, err := pty.Open()
	if err != nil {
		logrus.Errorf("open pty failure %s", err.Error())
		return nil, err
	}
	ec := &execContext{
		tty:         tty,
		pty:         pty,
		kubeRequest: kubeRequest,
		config:      config,
		sizeUpdate:  make(chan remotecommand.TerminalSize, 2),
	}
	if err := ec.Run(); err != nil {
		return nil, err
	}
	return ec, nil
}

// NewLogContext new log Context
func NewLogContext(kubeRequest *restclient.Request, config *restclient.Config) (server.Slave, error) {
	pty, tty, err := pty.Open()
	if err != nil {
		logrus.Errorf("open pty failure %s", err.Error())
		return nil, err
	}
	ec := &execContext{
		tty:         tty,
		pty:         pty,
		kubeRequest: kubeRequest,
		config:      config,
		sizeUpdate:  make(chan remotecommand.TerminalSize, 2),
	}
	if err := ec.RunLog(); err != nil {
		return nil, err
	}
	return ec, nil
}

func (e *execContext) WaitingStop() bool {
	return !e.closed
}

func (e *execContext) Run() error {
	exec, err := remotecommand.NewSPDYExecutor(e.config, "POST", e.kubeRequest.URL())
	if err != nil {
		return fmt.Errorf("create executor failure %s", err.Error())
	}

	errCh := make(chan error)

	go func() {
		out := CreateOut(e.tty)
		t := out.SetTTY()
		errCh <- t.Safe(func() error {
			defer e.Close()
			if err := exec.Stream(remotecommand.StreamOptions{
				Stdin:             out.Stdin,
				Stdout:            out.Stdout,
				Stderr:            nil,
				Tty:               true,
				TerminalSizeQueue: e,
			}); err != nil {
				logrus.Errorf("executor stream failure %s", err.Error())
				return err
			}
			return nil
		})
	}()

	// 如果在 200 毫秒内 errCh 有返回，则说明出错了，返回错误
	// 否则认为是正常的，返回 nil
	timeout := time.After(200 * time.Millisecond)
	select {
	case err := <-errCh:
		return err
	case <-timeout:
		return nil
	}
}

func (e *execContext) RunLog() error {
	errCh := make(chan error)

	go func() {
		out := CreateOut(e.tty)
		t := out.SetTTY()
		errCh <- t.Safe(func() error {
			defer e.Close()
			rc, err := e.kubeRequest.Stream(context.Background())
			if err != nil {
				logrus.Errorf("stream failure %s", err.Error())
				return err
			}
			defer rc.Close()
			if _, err := io.Copy(out.Stdout, rc); err != nil {
				logrus.Errorf("copy failure %s", err.Error())
				return err
			}
			return nil
		})
	}()

	// 如果在 200 毫秒内 errCh 有返回，则说明出错了，返回错误
	// 否则认为是正常的，返回 nil
	timeout := time.After(200 * time.Millisecond)
	select {
	case err := <-errCh:
		return err
	case <-timeout:
		return nil
	}
}

func (e *execContext) Read(p []byte) (n int, err error) {
	return e.pty.Read(p)
}

func (e *execContext) Write(p []byte) (n int, err error) {
	return e.pty.Write(p)
}

func (e *execContext) Close() error {
	e1 := e.pty.Close()
	e2 := e.tty.Close()

	if e1 != nil {
		return e1
	}

	return e2
}

func (e *execContext) WindowTitleVariables() map[string]interface{} {
	return map[string]interface{}{}
}

func (e *execContext) Next() *remotecommand.TerminalSize {
	size, ok := <-e.sizeUpdate
	if !ok {
		return nil
	}
	logrus.Infof("width %d height %d", size.Width, size.Height)
	return &size
}

func (e *execContext) ResizeTerminal(width int, height int) error {
	logrus.Infof("set width %d height %d", width, height)
	e.sizeUpdate <- remotecommand.TerminalSize{
		Width:  uint16(width),
		Height: uint16(height),
	}
	window := struct {
		row uint16
		col uint16
		x   uint16
		y   uint16
	}{
		uint16(height),
		uint16(width),
		0,
		0,
	}
	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		e.pty.Fd(),
		syscall.TIOCSWINSZ,
		uintptr(unsafe.Pointer(&window)),
	)
	if errno != 0 {
		return errno
	}
	return nil
}
