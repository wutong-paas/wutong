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

package db

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cast"
	eventutil "github.com/wutong-paas/wutong/eventlog/util"
	"github.com/wutong-paas/wutong/util"
)

// EventFilePlugin EventFilePlugin
type EventFilePlugin struct {
	HomePath string // /wtdata/logs
}

// SaveMessage save event log to file
func (m *EventFilePlugin) SaveMessage(events []*EventLogMessage) error {
	if len(events) == 0 {
		return nil
	}
	filePath := eventutil.EventLogFilePath(m.HomePath) // /wtdata/logs/eventlog
	if err := util.CheckAndCreateDir(filePath); err != nil {
		return err
	}
	filename := eventutil.EventLogFileName(filePath, events[0].EventID) // 根据 eventID 生成文件名
	writeFile, err := os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0755)
	if err != nil {
		return err
	}
	defer writeFile.Close()
	var lastTime int64
	for _, e := range events {
		if e == nil {
			continue
		}
		logtime := GetTimeUnix(e.Time)
		if logtime != 0 {
			lastTime = logtime
		}
		writeFile.Write([]byte(fmt.Sprintf("%d %d %s\n", GetLevelFlag(e.Level), lastTime, e.Message)))
	}
	return nil
}

// MessageData message data 获取指定操作的操作日志
type MessageData struct {
	Message  string `json:"message"`
	Time     string `json:"time"`
	Unixtime int64  `json:"utime"`
}

// MessageDataList MessageDataList
type MessageDataList []MessageData

func (a MessageDataList) Len() int           { return len(a) }
func (a MessageDataList) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a MessageDataList) Less(i, j int) bool { return a[i].Unixtime <= a[j].Unixtime }

// GetMessages GetMessages
func (m *EventFilePlugin) GetMessages(eventID, level string, length int) (interface{}, error) {
	var message MessageDataList
	apath := path.Join(m.HomePath, "eventlog", eventID+".log")
	if ok, err := util.FileExists(apath); !ok {
		if err != nil {
			logrus.Errorf("check file exist error %s", err.Error())
		}
		return message, nil
	}
	eventFile, err := os.Open(apath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer eventFile.Close()
	reader := bufio.NewReader(eventFile)
	for {
		line, _, err := reader.ReadLine()
		if err != nil {
			if err != io.EOF {
				logrus.Error("read event log file error:", err.Error())
			}
			break
		}
		if len(line) > 2 {
			flag := line[0]
			if CheckLevel(string(flag), level) {
				info := strings.SplitN(string(line), " ", 3)
				if len(info) == 3 {
					timestr := info[1]
					unixnano := cast.ToInt64(timestr)
					md := MessageData{
						Message:  info[2],
						Unixtime: unixnano,
						Time:     time.Unix(0, unixnano).Format(time.RFC3339Nano),
					}
					message = append(message, md)
					if len(message) > length && length != 0 {
						break
					}
				}
			}
		}
	}
	sort.Sort(message)
	return message, nil
}

// CheckLevel check log level
func CheckLevel(flag, level string) bool {
	switch flag {
	case "0":
		return true
	case "1":
		if level != "error" {
			return true
		}
	case "2":
		if level == "debug" {
			return true
		}
	}
	return false
}

// GetTimeUnix get specified time unix
func GetTimeUnix(timeStr string) int64 {
	utime, _ := time.Parse(time.RFC3339Nano, timeStr)
	return utime.UnixNano()
}

// GetLevelFlag get log level flag
func GetLevelFlag(level string) int {
	switch level {
	case "error":
		return 0
	case "info":
		return 1
	case "debug":
		return 2
	default:
		return 0
	}
}

// Close Close
func (m *EventFilePlugin) Close() error {
	return nil
}
