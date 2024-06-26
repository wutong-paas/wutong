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

package exector

import (
	"fmt"
	"os"

	"github.com/pquerna/ffjson/ffjson"
	"github.com/sirupsen/logrus"
	"github.com/wutong-paas/wutong/cmd/chaos/option"
	eventutil "github.com/wutong-paas/wutong/eventlog/util"
)

// GarbageCollectionItem -
type GarbageCollectionItem struct {
	TenantEnvID string        `json:"tenant_env_id"`
	ServiceID   string        `json:"service_id"`
	EventIDs    []string      `json:"event_ids"`
	Cfg         option.Config `json:"-"`
}

// NewGarbageCollectionItem creates a new GarbageCollectionItem
func NewGarbageCollectionItem(cfg option.Config, in []byte) (*GarbageCollectionItem, error) {
	logrus.Debugf("garbage collection; request body: %v", string(in))
	var gci GarbageCollectionItem
	if err := ffjson.Unmarshal(in, &gci); err != nil {
		return nil, err
	}
	gci.Cfg = cfg
	// validate

	return &gci, nil
}

// delLogFile deletes persistent data related to the service based on serviceID.
func (g *GarbageCollectionItem) delLogFile() {
	logrus.Infof("service id: %s;delete log file.", g.ServiceID)
	// log generated during service running
	dockerLogPath := eventutil.DockerLogFilePath(g.Cfg.LogPath, g.ServiceID)
	if err := os.RemoveAll(dockerLogPath); err != nil {
		logrus.Warningf("remove docker log files: %v", err)
	}
	// log generated by the service event
	eventLogPath := eventutil.EventLogFilePath(g.Cfg.LogPath)
	for _, eventID := range g.EventIDs {
		eventLogFileName := eventutil.EventLogFileName(eventLogPath, eventID)
		logrus.Debugf("remove event log file: %s", eventLogFileName)
		if err := os.RemoveAll(eventLogFileName); err != nil {
			logrus.Warningf("file: %s; remove event log file: %v", eventLogFileName, err)
		}
	}
}

func (g *GarbageCollectionItem) delVolumeData() {
	logrus.Infof("service id: %s; delete volume data.", g.ServiceID)
	dir := fmt.Sprintf("/wtdata/tenantEnv/%s/service/%s", g.TenantEnvID, g.ServiceID)
	if err := os.RemoveAll(dir); err != nil {
		logrus.Warningf("dir: %s; remove volume data: %v", dir, err)
	}
}
