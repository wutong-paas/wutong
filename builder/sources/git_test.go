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

package sources

import (
	"io"
	"testing"
	"time"

	"github.com/wutong-paas/wutong/event"
)

func TestGitClone(t *testing.T) {
	start := time.Now()
	csi := CodeSourceInfo{
		RepositoryURL: "git@gitee.com:zhoujunhaowutong/webhook_test.git",
		Branch:        "master",
	}
	res, err := GitClone(csi, "/tmp/wutongdoc3", event.GetTestLogger(), 1)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Take %d ms", time.Now().Unix()-start.Unix())
	commit, err := GetLastCommit(res)
	t.Logf("%+v %+v", commit, err)
}
func TestGitCloneByTag(t *testing.T) {
	start := time.Now()
	csi := CodeSourceInfo{
		RepositoryURL: "https://github.com/wutong-paas/wutong-ui.git",
		Branch:        "master",
	}
	res, err := GitClone(csi, "/tmp/wutongdoc4", event.GetTestLogger(), 1)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Take %d ms", time.Now().Unix()-start.Unix())
	commit, err := GetLastCommit(res)
	t.Logf("%+v %+v", commit, err)
}

func TestGitPull(t *testing.T) {
	csi := CodeSourceInfo{
		RepositoryURL: "git@gitee.com:zhoujunhaowutong/webhook_test.git",
		Branch:        "master2",
	}
	res, err := GitPull(csi, "/tmp/master2", event.GetTestLogger(), 1)
	if err != nil {
		t.Fatal(err)
	}
	commit, err := GetLastCommit(res)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%v", commit)
}

func TestGitPullOrClone(t *testing.T) {
	csi := CodeSourceInfo{
		RepositoryURL: "git@gitee.com:zhoujunhaowutong/webhook_test.git",
	}
	res, err := GitCloneOrPull(csi, "/tmp/wutongweb2", event.GetTestLogger(), 1)
	if err != nil {
		t.Fatal(err)
	}
	//get last commit
	commit, err := GetLastCommit(res)
	if err != nil && err != io.EOF {
		t.Fatal(err)
	}
	t.Logf("%+v", commit)
}

func TestGetCodeCacheDir(t *testing.T) {
	csi := CodeSourceInfo{
		RepositoryURL: "git@121.196.222.148:summersoft/yycx_push.git",
		Branch:        "test",
	}
	t.Log(csi.GetCodeSourceDir())
}

func TestGetShowURL(t *testing.T) {
	t.Log(getShowURL("https://zsl1526:79890ffc74014b34b49040d42b95d5af@github.com:9090/zsl1549/python-demo.git"))
}
