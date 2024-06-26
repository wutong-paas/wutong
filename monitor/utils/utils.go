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

package utils

import (
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"

	"github.com/sirupsen/logrus"
	"github.com/wutong-paas/wutong/discover/config"
)

// TrimAndSort TrimAndSort
func TrimAndSort(endpoints []*config.Endpoint) []string {
	arr := make([]string, 0, len(endpoints))
	for _, end := range endpoints {
		if strings.HasPrefix(end.URL, "https://") {
			url := strings.TrimPrefix(end.URL, "https://")
			arr = append(arr, url)
			continue
		}
		url := strings.TrimPrefix(end.URL, "http://")
		arr = append(arr, url)
	}
	sort.Strings(arr)
	return arr
}

// ArrCompare ArrCompare
func ArrCompare(arr1, arr2 []string) bool {
	if len(arr1) != len(arr2) {
		return false
	}

	for i, item := range arr1 {
		if item != arr2[i] {
			return false
		}
	}

	return true
}

// ListenStop ListenStop
func ListenStop() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigs
	signal.Ignore(syscall.SIGINT, syscall.SIGTERM)

	logrus.Warn("monitor manager received signal: ", sig.String())
	close(sigs)
}
