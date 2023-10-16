// WUTONG, Application Management Platform
// Copyright (C) 2014-2017 Wutong Co., Ltd.

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

package util

import "os"

var translationMetadata = map[string]string{
	"write console level metadata success":           "写控制台级应用元数据成功",
	"write region level metadata success":            "写数据中心级应用元数据成功",
	"Asynchronous tasks are sent successfully":       "异步任务发送成功",
	"create ftp client error":                        "创建FTP客户端失败",
	"push slug file to local dir error":              "上传应用代码包到本地目录失败",
	"push slug file to sftp server error":            "上传应用代码包到服务端失败",
	"down slug file from sftp server error":          "从服务端下载文件失败",
	"save image to local dir error":                  "保存镜像到本地目录失败",
	"save image to hub error":                        "保存镜像到仓库失败",
	"Please try again or contact customer service":   "后端服务开小差，请重试或联系客服",
	"unzip metadata file error":                      "解压数据失败",
	"start service error":                            "组件启动失败，请检查集组件信息或查看日志",
	"start service timeout":                          "组件启动超时，建议观察组件实例运行状态",
	"stop service error":                             "组件停止失败，建议观察组件实例运行状态",
	"stop service timeout":                           "组件停止超时，建议观察组件实例运行状态",
	"(restart)stop service error":                    "组件停止失败，建议观察组件实例运行状态，待其停止后手动启动",
	"(restart)Application model init create failure": "初始化应用元数据模型失败，请检查集群运行状态或查看日志",
	"horizontal scaling service error":               "组件水平伸缩失败，请检查集群运行状态或查看日志",
	"horizontal scaling service timeout":             "组件水平伸缩超时，建议观察组件实例运行状态",
	"upgrade service error":                          "组件升级失败，请检查组件信息或查看日志",
	"upgrade service timeout":                        "组件升级超时，建议观察更新组件实例运行状态或运行日志",
	"Check for log location code errors":             "建议查看日志定位代码错误",
	"Check for log location imgae source errors":     "建议查看日志定位镜像源错误",
	"create share image task error":                  "分享任务失败，请检查组件信息或查看日志",
	"get wt-repo ip failure":                         "获取依赖仓库IP地址失败，请检查wt-repo组件信息",
}

// Translation Translation English to Chinese
func Translation(english string) string {
	if chinese, ok := translationMetadata[english]; ok {
		if os.Getenv("WUTONG_LANG") == "en" {
			return english
		}
		return chinese
	}
	return english
}
