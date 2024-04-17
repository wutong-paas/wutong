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
	"testing"
)

func TestFileSaveMessage(t *testing.T) {

	f := filePlugin{
		homePath: "./test",
	}
	m := &EventLogMessage{EventID: "qwertyuiopasdfghjkl"}
	m.Content = []byte("do you under stand")
	mes := []*EventLogMessage{m}
	for i := 0; i < 100; i++ {
		m := &EventLogMessage{EventID: "qwertyuiopasdfghjkl"}
		m.Content = []byte("do you under stand")
		mes = append(mes, m)
	}
	err := f.SaveMessage(mes)
	if err != nil {
		t.Fatal(err)
	}
}

func TestMvLogFile(t *testing.T) {
	MvLogFile("/Users/qingguo/7b3d5546bd54152d/stdout.log.gz", []string{"/Users/qingguo/7b3d5546bd54152d/stdout.log"})
}

func TestGetMessages(t *testing.T) {
	f := filePlugin{
		homePath: "./test",
	}
	logs, err := f.GetMessages("qwertyuiopasdfghjkl", "", 10)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(logs)
	logs, err = f.GetMessages("qwertyuiopasdfghjkl", "", -10)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(logs)
}

func TestGetServiceAliasID(t *testing.T) {
	should := "fd2b16501f00b10d"
	got := GetServiceAliasID("265f906f94545829b7bb1546d4318d17")
	if should != got {
		t.Fatalf("should get %v, but get %v\n", should, got)
	}
}
