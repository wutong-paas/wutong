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
	"context"
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pquerna/ffjson/ffjson"
	"github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
	"github.com/wutong-paas/wutong/chaos"
	"github.com/wutong-paas/wutong/chaos/build"
	"github.com/wutong-paas/wutong/chaos/parser"
	"github.com/wutong-paas/wutong/chaos/parser/code"
	"github.com/wutong-paas/wutong/chaos/sources"
	"github.com/wutong-paas/wutong/db"
	dbmodel "github.com/wutong-paas/wutong/db/model"
	"github.com/wutong-paas/wutong/event"
	"github.com/wutong-paas/wutong/util"
	"k8s.io/client-go/kubernetes"
)

// SourceCodeBuildItem SouceCodeBuildItem
type SourceCodeBuildItem struct {
	Namespace     string       `json:"namespace"`
	TenantEnvName string       `json:"tenant_env_name"`
	WTDataPVCName string       `json:"gr_data_pvc_name"`
	CachePVCName  string       `json:"cache_pvc_name"`
	CacheMode     string       `json:"cache_mode"`
	CachePath     string       `json:"cache_path"`
	ServiceAlias  string       `json:"service_alias"`
	Action        string       `json:"action"`
	DestImage     string       `json:"dest_image"`
	Logger        event.Logger `json:"logger"`
	EventID       string       `json:"event_id"`
	CacheDir      string       `json:"cache_dir"`
	TGZDir        string       `json:"tgz_dir"`
	ImageClient   sources.ImageClient
	KanikoImage   string
	KubeClient    kubernetes.Interface
	WtNamespace   string
	WtRepoName    string
	TenantEnvID   string
	ServiceID     string
	DeployVersion string
	Lang          string
	Runtime       string
	BuildEnvs     map[string]string
	CodeSouceInfo sources.CodeSourceInfo
	RepoInfo      *sources.RepostoryBuildInfo
	commit        Commit
	Configs       map[string]gjson.Result `json:"configs"`
	Ctx           context.Context
	Operator      string `json:"operator"`
}

// Commit code Commit
type Commit struct {
	Hash    string
	Author  string
	Message string
}

// NewSouceCodeBuildItem create
func NewSouceCodeBuildItem(in []byte) *SourceCodeBuildItem {
	eventID := gjson.GetBytes(in, "event_id").String()
	logger := event.GetLogger(eventID)
	csi := sources.CodeSourceInfo{
		ServerType:    strings.Replace(gjson.GetBytes(in, "server_type").String(), " ", "", -1),
		RepositoryURL: gjson.GetBytes(in, "repo_url").String(),
		Branch:        gjson.GetBytes(in, "branch").String(),
		User:          gjson.GetBytes(in, "user").String(),
		Password:      gjson.GetBytes(in, "password").String(),
		TenantEnvID:   gjson.GetBytes(in, "tenant_env_id").String(),
		ServiceID:     gjson.GetBytes(in, "service_id").String(),
	}
	envs := gjson.GetBytes(in, "envs").String()
	be := make(map[string]string)
	if err := ffjson.Unmarshal([]byte(envs), &be); err != nil {
		logrus.Errorf("unmarshal build envs error: %s", err.Error())
	}
	scb := &SourceCodeBuildItem{
		Namespace:     gjson.GetBytes(in, "tenant_env_id").String(),
		TenantEnvName: gjson.GetBytes(in, "tenant_env_name").String(),
		ServiceAlias:  gjson.GetBytes(in, "service_alias").String(),
		TenantEnvID:   gjson.GetBytes(in, "tenant_env_id").String(),
		ServiceID:     gjson.GetBytes(in, "service_id").String(),
		Action:        gjson.GetBytes(in, "action").String(),
		DeployVersion: gjson.GetBytes(in, "deploy_version").String(),
		Logger:        logger,
		EventID:       eventID,
		CodeSouceInfo: csi,
		Lang:          gjson.GetBytes(in, "lang").String(),
		Runtime:       gjson.GetBytes(in, "runtime").String(),
		Configs:       gjson.GetBytes(in, "configs").Map(),
		BuildEnvs:     be,
	}
	scb.CacheDir = fmt.Sprintf("/cache/build/%s/cache/%s", scb.TenantEnvID, scb.ServiceID)
	//scb.SourceDir = scb.CodeSouceInfo.GetCodeSourceDir()
	scb.TGZDir = fmt.Sprintf("/wtdata/build/tenantEnv/%s/slug/%s", scb.TenantEnvID, scb.ServiceID)
	return scb
}

// Run Run
func (i *SourceCodeBuildItem) Run(timeout time.Duration) error {
	// 1.clone
	// 2.check dockerfile/ source_code
	// 3.build
	// 4.upload image /upload slug
	rbi, err := sources.CreateRepostoryBuildInfo(i.CodeSouceInfo.RepositoryURL, i.CodeSouceInfo.ServerType, i.CodeSouceInfo.Branch, i.TenantEnvID, i.ServiceID)
	if err != nil {
		i.Logger.Error("Git 项目仓库地址格式错误", map[string]string{"step": "parse"})
		return err
	}
	i.RepoInfo = rbi
	if err := i.prepare(); err != nil {
		logrus.Errorf("prepare build code error: %s", err.Error())
		i.Logger.Error("准备源码构建失败，错误信息："+err.Error(), map[string]string{"step": "builder-exector", "status": "failure"})
		return err
	}
	i.CodeSouceInfo.RepositoryURL = rbi.RepostoryURL
	switch i.CodeSouceInfo.ServerType {
	case "svn":
		csi := i.CodeSouceInfo
		svnclient := sources.NewClient(csi, rbi.GetCodeHome(), i.Logger)
		rs, err := svnclient.UpdateOrCheckout(rbi.BuildPath)
		if err != nil {
			logrus.Errorf("checkout svn code error: %s", err.Error())
			i.Logger.Error("Checkout svn code failed, please make sure the code can be downloaded properly", map[string]string{"step": "builder-exector", "status": "failure"})
			return err
		}
		if rs.Logs == nil || len(rs.Logs.CommitEntrys) < 1 {
			logrus.Errorf("get code commit info error")
			i.Logger.Error("读取代码版本信息失败：", map[string]string{"step": "builder-exector", "status": "failure"})
			return err
		}
		i.commit = Commit{
			Hash:    rs.Logs.CommitEntrys[0].Revision,
			Message: rs.Logs.CommitEntrys[0].Msg,
			Author:  rs.Logs.CommitEntrys[0].Author,
		}
	case "oss":
		i.commit = Commit{}
	default:
		//default git
		rs, err := sources.GitCloneOrPull(i.CodeSouceInfo, rbi.GetCodeHome(), i.Logger, 5)
		if err != nil {
			logrus.Errorf("pull git code error: %s", err.Error())
			i.Logger.Error("拉取代码失败，请确保代码可以被正常下载", map[string]string{"step": "builder-exector", "status": "failure"})
			return err
		}
		//get last commit
		commit, err := sources.GetLastCommit(rs)
		if err != nil || commit == nil {
			logrus.Errorf("get code commit info error: %s", err.Error())
			i.Logger.Error("读取代码版本信息失败，错误信息："+err.Error(), map[string]string{"step": "builder-exector", "status": "failure"})
			return err
		}
		i.commit = Commit{
			Hash:    commit.Hash.String(),
			Author:  commit.Author.Name,
			Message: commit.Message,
		}
	}
	// clean cache code
	defer func() {
		if err := os.RemoveAll(rbi.GetCodeHome()); err != nil {
			logrus.Warningf("remove source code: %v", err)
		}
	}()

	hash := i.commit.Hash
	if len(hash) >= 8 {
		hash = i.commit.Hash[0:7]
	}
	info := fmt.Sprintf("CodeVersion:%s Author:%s Commit:%s ", hash, i.commit.Author, i.commit.Message)
	i.Logger.Info(info, map[string]string{"step": "code-version"})
	if _, ok := i.BuildEnvs["REPARSE"]; ok {
		_, lang, err := parser.ReadWtConfigAndLang(rbi)
		if err != nil {
			logrus.Errorf("reparse code lange error %s", err.Error())
			i.Logger.Error("重新解析代码语言错误", map[string]string{"step": "builder-exector", "status": "failure"})
			return err
		}
		i.Lang = string(lang)
	}

	i.Logger.Info("pull or clone code successfully, start code build", map[string]string{"step": "codee-version"})
	res, err := i.codeBuild()
	if err != nil {
		if err.Error() == context.DeadlineExceeded.Error() {
			i.Logger.Error("Build app version from source code timeout, the maximum time is 60 minutes", map[string]string{"step": "builder-exector", "status": "failure"})
		} else {
			i.Logger.Error("Build app version from source code failure,"+err.Error(), map[string]string{"step": "builder-exector", "status": "failure"})
		}
		return err
	}
	if err := i.UpdateBuildVersionInfo(res); err != nil {
		return err
	}
	return nil
}

func (i *SourceCodeBuildItem) codeBuild() (*build.Response, error) {
	codeBuild, err := build.GetBuild(code.Lang(i.Lang))
	if err != nil {
		logrus.Errorf("get code build error: %s lang %s", err.Error(), i.Lang)
		i.Logger.Error(util.Translation("No way of compiling to support this source type was found"), map[string]string{"step": "builder-exector", "status": "failure"})
		return nil, err
	}
	hostAlias, err := i.getHostAlias()
	if err != nil {
		i.Logger.Error(util.Translation("get wt-repo ip failure"), map[string]string{"step": "builder-exector", "status": "failure"})
		return nil, err
	}
	buildReq := &build.Request{
		KanikoImage:   i.KanikoImage,
		WtNamespace:   i.WtNamespace,
		SourceDir:     i.RepoInfo.GetCodeBuildAbsPath(),
		CacheDir:      i.CacheDir,
		TGZDir:        i.TGZDir,
		RepositoryURL: i.RepoInfo.RepostoryURL,
		CodeSouceInfo: i.CodeSouceInfo,
		ServiceAlias:  i.ServiceAlias,
		ServiceID:     i.ServiceID,
		TenantEnvID:   i.TenantEnvID,
		ServerType:    i.CodeSouceInfo.ServerType,
		Runtime:       i.Runtime,
		Branch:        i.CodeSouceInfo.Branch,
		DeployVersion: i.DeployVersion,
		Commit:        build.Commit{User: i.commit.Author, Message: i.commit.Message, Hash: i.commit.Hash},
		Lang:          code.Lang(i.Lang),
		BuildEnvs:     i.BuildEnvs,
		Logger:        i.Logger,
		ImageClient:   i.ImageClient,
		KubeClient:    i.KubeClient,
		HostAlias:     hostAlias,
		Ctx:           i.Ctx,
		WTDataPVCName: i.WTDataPVCName,
		CachePVCName:  i.CachePVCName,
		CacheMode:     i.CacheMode,
		CachePath:     i.CachePath,
	}
	res, err := codeBuild.Build(buildReq)
	return res, err
}

func (i *SourceCodeBuildItem) getHostAlias() (hostAliasList []build.HostAlias, err error) {
	endpoints, err := i.KubeClient.CoreV1().Endpoints(i.WtNamespace).Get(context.Background(), i.WtRepoName, metav1.GetOptions{})
	if err != nil {
		logrus.Errorf("do not found ep by name: %s in namespace: %s", i.WtRepoName, i.Namespace)
		return nil, err
	}
	hostNames := []string{"maven.goodrain.me", "lang.goodrain.me"}
	for _, subset := range endpoints.Subsets {
		for _, addr := range subset.Addresses {
			hostAliasList = append(hostAliasList, build.HostAlias{IP: addr.IP, Hostnames: hostNames})
		}
	}
	return
}

// IsDockerfile CheckDockerfile
func (i *SourceCodeBuildItem) IsDockerfile() bool {
	filepath := path.Join(i.RepoInfo.GetCodeBuildAbsPath(), "Dockerfile")
	_, err := os.Stat(filepath)

	return err == nil
}

func (i *SourceCodeBuildItem) prepare() error {
	if err := util.CheckAndCreateDir(i.CacheDir); err != nil {
		return err
	}
	if err := util.CheckAndCreateDir(i.TGZDir); err != nil {
		return err
	}
	if err := util.CheckAndCreateDir(i.RepoInfo.GetCodeHome()); err != nil {
		return err
	}
	if _, ok := i.BuildEnvs["NO_CACHE"]; ok {
		if !util.DirIsEmpty(i.RepoInfo.GetCodeHome()) {
			os.RemoveAll(i.RepoInfo.GetCodeHome())
		}
		if err := os.RemoveAll(i.CacheDir); err != nil {
			logrus.Error("remove cache dir error", err.Error())
		}
		if err := os.MkdirAll(i.CacheDir, 0755); err != nil {
			logrus.Error("make cache dir error", err.Error())
		}
	}
	os.Chown(i.CacheDir, 200, 200)
	os.Chown(i.TGZDir, 200, 200)
	return nil
}

// UpdateVersionInfo Update build application service version info
func (i *SourceCodeBuildItem) UpdateVersionInfo(vi *dbmodel.VersionInfo) error {
	version, err := db.GetManager().VersionInfoDao().GetVersionByDeployVersion(i.DeployVersion, i.ServiceID)
	if err != nil {
		return err
	}
	if vi.DeliveredType != "" {
		version.DeliveredType = vi.DeliveredType
	}
	if vi.DeliveredPath != "" {
		version.DeliveredPath = vi.DeliveredPath
		if vi.DeliveredType == "image" {
			version.ImageName = vi.DeliveredPath
		}
	}
	if vi.FinalStatus != "" {
		version.FinalStatus = vi.FinalStatus
	}
	version.CommitMsg = vi.CommitMsg
	version.Author = vi.Author
	version.CodeVersion = vi.CodeVersion
	version.CodeBranch = vi.CodeBranch
	version.FinishTime = time.Now()
	if err := db.GetManager().VersionInfoDao().UpdateModel(version); err != nil {
		return err
	}
	return nil
}

// UpdateBuildVersionInfo update service build version info to db
func (i *SourceCodeBuildItem) UpdateBuildVersionInfo(res *build.Response) error {
	vi := &dbmodel.VersionInfo{
		DeliveredType: string(res.MediumType),
		DeliveredPath: res.MediumPath,
		EventID:       i.EventID,
		ImageName:     chaos.RUNNERIMAGENAME,
		FinalStatus:   "success",
		CodeBranch:    i.CodeSouceInfo.Branch,
		CodeVersion:   i.commit.Hash,
		CommitMsg:     i.commit.Message,
		Author:        i.commit.Author,
		FinishTime:    time.Now(),
	}
	if err := i.UpdateVersionInfo(vi); err != nil {
		logrus.Errorf("update version info error: %s", err.Error())
		i.Logger.Error("Update application service version information failed", map[string]string{"step": "build-code", "status": "failure"})
		return err
	}
	return nil
}

// UpdateCheckResult UpdateCheckResult
func (i *SourceCodeBuildItem) UpdateCheckResult(result *dbmodel.CodeCheckResult) error {
	return nil
}
