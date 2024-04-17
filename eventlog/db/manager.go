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
	"fmt"
)

type Manager interface {
	// SaveMessage 保存消息
	SaveMessage([]*EventLogMessage) error
	// Close 关闭
	Close() error
	// GetMessages 获取消息
	GetMessages(id, level string, length int) (interface{}, error)
}

// NewManager 创建存储管理器
func NewManager(plugin, storePath string) (Manager, error) {
	switch plugin {
	case "file":
		return &filePlugin{
			homePath: storePath,
		}, nil
	case "eventfile":
		return &EventFilePlugin{
			HomePath: storePath,
		}, nil
	default:
		return nil, fmt.Errorf("do not support plugin")
	}
}
