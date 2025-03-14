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

package event

import (
	"testing"
	"time"
)

func TestLogger(t *testing.T) {
	loggerManager, err := NewLoggerManager()
	if err != nil {
		t.Fatal(err)
	}
	defer loggerManager.Close()
	time.Sleep(time.Second * 3)
	for i := 0; i < 500; i++ {
		GetLogger("qwdawdasdasasfafa").Info("hello word", nil)
		GetLogger("asdasdasdasdads").Debug("hello word", nil)
		GetLogger("1234124124124").Error("hello word", nil)
		time.Sleep(time.Millisecond * 1)
	}
	select {}
}
