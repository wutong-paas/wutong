package probe

import (
	"bytes"
	"context"
	"os/exec"
	"strings"
	"time"

	"github.com/wutong-paas/wutong/node/nodem/client"
	"github.com/wutong-paas/wutong/node/nodem/service"
)

// ShellProbe -
type ShellProbe struct {
	Name         string
	Address      string
	ResultsChan  chan *service.HealthStatus
	Ctx          context.Context
	Cancel       context.CancelFunc
	TimeInterval int
	HostNode     *client.HostNode
	MaxErrorsNum int
}

// Check -
func (h *ShellProbe) Check() {
	go h.ShellCheck()
}

// Stop -
func (h *ShellProbe) Stop() {
	h.Cancel()
}

// ShellCheck -
func (h *ShellProbe) ShellCheck() {
	timer := time.NewTimer(time.Second * time.Duration(h.TimeInterval))
	defer timer.Stop()
	for {
		HealthMap := GetShellHealth(h.Address)
		result := &service.HealthStatus{
			Name:   h.Name,
			Status: HealthMap["status"],
			Info:   HealthMap["info"],
		}
		h.ResultsChan <- result
		timer.Reset(time.Second * time.Duration(h.TimeInterval))
		select {
		case <-h.Ctx.Done():
			return
		case <-timer.C:
		}
	}
}

// GetShellHealth get shell health
func GetShellHealth(address string) map[string]string {
	cmd := exec.Command("/bin/bash", "-c", address)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		errStr := stderr.String()
		return map[string]string{"status": service.Stat_death, "info": strings.TrimSpace(errStr)}
	}
	return map[string]string{"status": service.Stat_healthy, "info": "service healthy"}
}
