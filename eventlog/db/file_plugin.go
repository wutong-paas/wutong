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
	"archive/zip"
	"bufio"
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"strconv"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/wutong-paas/wutong/util"
)

type ByteUnit int64

const (
	B  ByteUnit = 1
	KB          = 1000 * B
	MB          = 1000 * KB
)

type filePlugin struct {
	homePath string // /wtdata/logs
}

func (m *filePlugin) getStdFilePath(serviceID string) (string, error) {
	apath := path.Join(m.homePath, GetServiceAliasID(serviceID))
	_, err := os.Stat(apath)
	if err != nil {
		if os.IsNotExist(err) {
			err := os.MkdirAll(apath, 0755)
			if err != nil {
				return "", err
			}
		} else {
			return "", err
		}
	}
	return apath, nil
}

// dp: 在 1.13.0 版本发现了并发写错误：concurrent map writes，使用 sync.Map 调整代码
// 使用 alpha 版本测试，如果运行正常，则发布 v1.13.1 版本修复。
// var fileCache = make(map[string]int64)
var safeFileCache sync.Map

func (m *filePlugin) SaveMessage(events []*EventLogMessage) error {
	if len(events) == 0 {
		return nil
	}
	logMaxSize := 10 * MB
	if os.Getenv("LOG_MAX_SIZE") != "" {
		if size, err := strconv.Atoi(os.Getenv("LOG_MAX_SIZE")); err == nil {
			logMaxSize = ByteUnit(size) * MB
		}
	}
	key := events[0].EventID
	var logfile *os.File
	filePathDir, err := m.getStdFilePath(key)
	if err != nil {
		return err
	}
	stdoutLogPath := path.Join(filePathDir, "stdout.log")
	stdoutLegacyLogPath := path.Join(filePathDir, "stdout-legacy.log")
	logFile, err := os.Stat(stdoutLogPath)
	if err != nil {
		if os.IsNotExist(err) {
			logfile, err = os.Create(stdoutLogPath)
			if err != nil {
				return err
			}
			defer logfile.Close()
		} else {
			return err
		}
	} else {
		// 如果日志文件不是当天的，将日志文件压缩并重命名
		if logFile.ModTime().Day() != time.Now().Day() {
			logFiles := []string{stdoutLogPath}
			// Assert if stdout-legacy.log is existed , if exists, append to archive
			stdoutLegacyLogFileStat, err := os.Stat(stdoutLegacyLogPath)
			if err == nil && stdoutLegacyLogFileStat.Size() > 0 {
				logFiles = append(logFiles, stdoutLegacyLogPath)
			}
			err = MvLogFile(fmt.Sprintf("%s/%d-%d-%d.log.gz", filePathDir, logFile.ModTime().Year(), logFile.ModTime().Month(), logFile.ModTime().Day()), logFiles)
			if err != nil {
				return err
			}
		}
	}
	// 最后记录日志的时间，如果当前日志的时间小于等于这个时间，不再记录
	// lastLogTimeUnixNano, ok := fileCache[stdoutLogPath]
	lastLogTimeUnixNanoVal, ok := safeFileCache.Load(stdoutLogPath)
	var lastLogTimeUnixNano int64
	if !ok {
		lastLogTimeUnixNano = readLastLogTimeUnixNano(stdoutLogPath)
		// if lastLogTimeUnixNano > 0 {
		// 	fileCache[stdoutLogPath] = lastLogTimeUnixNano
		if lastLogTimeUnixNano > 0 {
			safeFileCache.Store(stdoutLogPath, lastLogTimeUnixNano)
		}
	} else {
		lastLogTimeUnixNano = lastLogTimeUnixNanoVal.(int64)
	}

	if lastLogTimeUnixNano > 0 && events[len(events)-1].TimeUnixNano < lastLogTimeUnixNano {
		return nil
	}

	if logfile == nil {
		logfile, err = os.OpenFile(stdoutLogPath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			return err
		}
		if logfile != nil {
			defer logfile.Close()
		}
	} else {
		defer logfile.Close()
	}
	var content [][]byte
	for _, e := range events {
		content = append(content, e.Content)
	}
	body := bytes.Join(content, []byte("\n"))
	body = append(body, []byte("\n")...)
	if logFile != nil && logFile.Size() > int64(logMaxSize) {
		// 如果日志文件超过一定大小（默认 10M），将日志文件重命名为 stdout-legacy.log
		legacyLogPath := path.Join(filePathDir, "stdout-legacy.log")
		err = os.Rename(stdoutLogPath, legacyLogPath)
		if err != nil {
			logrus.Errorf("[Savemessage]: Rename %v to %v failed %v", stdoutLogPath, legacyLogPath, err)
			return err
		}
		if logfile != nil {
			logfile.Close()
		}
		logfile, err = os.OpenFile(stdoutLogPath, os.O_WRONLY|os.O_CREATE, 0666)
		if err != nil {
			return err
		}
		logrus.Debugf("[SaveMessage]: Old log file size %v, Write content size %v", logFile.Size(), len(body))
		_, err = logfile.Write(body)
		return err
	}
	_, err = logfile.Write(body)
	if err != nil {
		return err
	}
	// fileCache[stdoutLogPath] = events[len(events)-1].TimeUnixNano
	safeFileCache.Store(stdoutLogPath, events[len(events)-1].TimeUnixNano)
	return err
}

// readLastLogTimeUnixNano 读取归档日志文件最后一行日志的 Timestamp UnixNano
func readLastLogTimeUnixNano(stdoutLogPath string) int64 {
	lastln, err := fileLastln(stdoutLogPath)
	if err != nil {
		return 0
	}

	if len(lastln) < 23 || !bytes.HasPrefix([]byte(lastln), []byte("v2:")) {
		return 0
	}
	// v2:[19位 UnixNano 时间戳] [12位 containerID]:[YYYY/MM/DD HH:MM:SS] [日志内容]
	logTimeUnixNano, _ := strconv.ParseInt(string(lastln[3:22]), 10, 64)
	return logTimeUnixNano
}

// fileLastln 读取文件最后一行
func fileLastln(path string) ([]byte, error) {
	var lastln []byte
	file, err := os.OpenFile(path, os.O_RDONLY, 0600)
	if err != nil {
		return lastln, err
	}
	defer file.Close()

	info, _ := file.Stat()
	if info.Size() > 0 {
		index := int64(-1)
		r := bufio.NewReader(file)
		for {
			index--
			file.Seek(index, io.SeekEnd)
			readByte, err := r.ReadByte()
			if readByte == '\n' {
				file.Seek(0, io.SeekEnd)
				break
			}
			if err != nil {
				if err == io.EOF {
					break
				}
				logrus.Errorf("failed to read file %s: %v", path, err)
			}
		}
		lastln, _, _ = r.ReadLine()
	}
	return lastln, nil
}

func (m *filePlugin) GetMessages(serviceID, level string, length int) (interface{}, error) {
	if length <= 0 {
		return nil, nil
	}
	filePathDir, err := m.getStdFilePath(serviceID)
	if err != nil {
		return nil, err
	}
	filePath := path.Join(filePathDir, "stdout.log")
	if ok, err := util.FileExists(filePath); !ok {
		if err != nil {
			logrus.Errorf("check file exist error %s", err.Error())
		}
		return nil, nil
	}
	f, err := exec.Command("tail", "-n", fmt.Sprintf("%d", length), filePath).Output()
	if err != nil {
		return nil, err
	}
	reader := bufio.NewReader(bytes.NewBuffer(f))
	var lines []string
	for {
		line, _, err := reader.ReadLine()
		if err != nil {
			break
		}
		if len(line) == 0 {
			continue
		}
		// v2 log archivement
		// 去除前缀以及 Timestamp 部分
		if bytes.HasPrefix(line, []byte("v2:")) && len(line) > 23 {
			line = line[23:]
		}
		lines = append(lines, string(line))
	}
	return lines, nil
}

func (m *filePlugin) Close() error {
	return nil
}

// GetServiceAliasID python:
// new_word = str(ord(string[10])) + string + str(ord(string[3])) + 'log' + str(ord(string[2]) / 7)
// new_id = hashlib.sha224(new_word).hexdigest()[0:16]
func GetServiceAliasID(ServiceID string) string {
	if len(ServiceID) > 11 {
		newWord := strconv.Itoa(int(ServiceID[10])) + ServiceID + strconv.Itoa(int(ServiceID[3])) + "log" + strconv.Itoa(int(ServiceID[2])/7)
		ha := sha256.New224()
		ha.Write([]byte(newWord))
		return fmt.Sprintf("%x", ha.Sum(nil))[0:16]
	}
	return ServiceID
}

// MvLogFile 更改文件名称，压缩
func MvLogFile(newName string, filePaths []string) error {
	// 将压缩文档内容写入文件
	f, err := os.OpenFile(newName, os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return err
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	defer zw.Close()

	for _, filePath := range filePaths {
		info, err := os.Stat(filePath)
		if err != nil {
			return err
		}
		reader, err := os.OpenFile(filePath, os.O_RDONLY, 0666)
		if err != nil {
			return err
		}
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		writer, err := zw.CreateHeader(header)
		if err != nil {
			return err
		}
		_, err = io.Copy(writer, reader)
		if err != nil {
			return err
		}
		err = os.Remove(filePath)
		if err != nil {
			return err
		}
	}

	return nil
}
