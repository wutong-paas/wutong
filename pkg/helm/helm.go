package helm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unsafe"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/wutong-paas/wutong/util/commonutil"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/downloader"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/kube"
	"helm.sh/helm/v3/pkg/provenance"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/releaseutil"
	"helm.sh/helm/v3/pkg/repo"
	"helm.sh/helm/v3/pkg/storage/driver"
	"helm.sh/helm/v3/pkg/strvals"
	helmtime "helm.sh/helm/v3/pkg/time"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	yamlDecoder "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

// ReleaseInfo -
type ReleaseInfo struct {
	Revision    int           `json:"revision"`
	Updated     helmtime.Time `json:"updated"`
	Status      string        `json:"status"`
	Chart       string        `json:"chart"`
	AppVersion  string        `json:"app_version"`
	Description string        `json:"description"`
}

// ReleaseHistory -
type ReleaseHistory []ReleaseInfo

// Helm -
type Helm struct {
	cfg       *action.Configuration
	settings  *cli.EnvSettings
	namespace string

	repoFile  string
	repoCache string
}

// NewHelm creates a new helm.
func NewHelm(namespace, repoFile, repoCache string) (*Helm, error) {
	configFlags := genericclioptions.NewConfigFlags(true)
	configFlags.Namespace = commonutil.String(namespace)
	kubeClient := kube.New(configFlags)

	cfg := &action.Configuration{
		KubeClient: kubeClient,
		Log: func(s string, i ...interface{}) {
			logrus.Debugf(s, i)
		},
		RESTClientGetter: configFlags,
	}
	helmDriver := ""
	settings := cli.New()
	settings.Debug = true
	// set namespace
	namespacePtr := (*string)(unsafe.Pointer(settings))
	*namespacePtr = namespace
	settings.RepositoryConfig = repoFile
	settings.RepositoryCache = repoCache
	// initializes the action configuration
	if err := cfg.Init(settings.RESTClientGetter(), settings.Namespace(), helmDriver, func(format string, v ...interface{}) {
		logrus.Debugf(format, v)
	}); err != nil {
		return nil, errors.Wrap(err, "init config")
	}
	return &Helm{
		cfg:       cfg,
		settings:  settings,
		namespace: namespace,
		repoFile:  repoFile,
		repoCache: repoCache,
	}, nil
}

// PreInstall -
func (h *Helm) PreInstall(name, chart, version string) error {
	_, err := h.install(name, chart, version, nil, true, ioutil.Discard)
	return err
}

// Install -
func (h *Helm) Install(name, chart, version string, overrides []string) error {
	_, err := h.install(name, chart, version, overrides, false, ioutil.Discard)
	return err
}

func (h *Helm) locateChart(chart, version string) (string, error) {
	repoAndName := strings.Split(chart, "/")
	if len(repoAndName) != 2 {
		return "", errors.New("invalid chart. expect repo/name, but got " + chart)
	}

	chartCache := path.Join(h.settings.RepositoryCache, chart, version)
	cp := path.Join(chartCache, repoAndName[1]+"-"+version+".tgz")
	if f, err := os.Open(cp); err == nil {
		defer f.Close()

		// check if the chart file is up to date.
		hash, err := provenance.Digest(f)
		if err != nil {
			return "", errors.Wrap(err, "digist chart file")
		}

		// get digiest from repo index.
		digest, err := h.getDigest(chart, version)
		if err != nil {
			return "", err
		}

		if hash == digest {
			return cp, nil
		}
	}

	cpo := &ChartPathOptions{}
	cpo.ChartPathOptions.Version = version
	settings := h.settings
	cp, err := cpo.LocateChart(chart, chartCache, settings)
	if err != nil {
		return "", err
	}

	return cp, err
}

func (h *Helm) getDigest(chart, version string) (string, error) {
	repoAndApp := strings.Split(chart, "/")
	if len(repoAndApp) != 2 {
		return "", errors.New("wrong chart format, expect repo/name, but got " + chart)
	}
	repoName, appName := repoAndApp[0], repoAndApp[1]

	indexFile, err := repo.LoadIndexFile(path.Join(h.repoCache, repoName+"-index.yaml"))
	if err != nil {
		return "", errors.Wrap(err, "load index file")
	}

	entries, ok := indexFile.Entries[appName]
	if !ok {
		return "", errors.New(fmt.Sprintf("chart(%s) not found", chart))
	}

	for _, entry := range entries {
		if entry.Version == version {
			return entry.Digest, nil
		}
	}

	return "", errors.New(fmt.Sprintf("chart(%s) version(%s) not found", chart, version))
}

func (h *Helm) install(name, chart, version string, overrides []string, dryRun bool, out io.Writer) (*release.Release, error) {
	client := action.NewInstall(h.cfg)
	client.ReleaseName = name
	client.Namespace = h.namespace
	client.Version = version
	client.DryRun = dryRun

	cp, err := h.locateChart(chart, version)
	if err != nil {
		return nil, err
	}

	logrus.Debugf("CHART PATH: %s\n", cp)

	p := getter.All(h.settings)
	// User specified a value via --set
	vals, err := h.parseOverrides(overrides)
	if err != nil {
		return nil, err
	}

	// Check chart dependencies to make sure all are present in /charts
	chartRequested, err := loader.Load(cp)
	if err != nil {
		return nil, err
	}

	if err := checkIfInstallable(chartRequested); err != nil {
		return nil, err
	}

	if chartRequested.Metadata.Deprecated {
		logrus.Warningf("This chart is deprecated")
	}

	if req := chartRequested.Metadata.Dependencies; req != nil {
		// If CheckDependencies returns an error, we have unfulfilled dependencies.
		// As of Helm 2.4.0, this is treated as a stopping condition:
		// https://github.com/helm/helm/issues/2209
		if err := action.CheckDependencies(chartRequested, req); err != nil {
			if client.DependencyUpdate {
				man := &downloader.Manager{
					Out:              out,
					ChartPath:        cp,
					Keyring:          client.ChartPathOptions.Keyring,
					SkipUpdate:       false,
					Getters:          p,
					RepositoryConfig: h.settings.RepositoryConfig,
					RepositoryCache:  h.settings.RepositoryCache,
					Debug:            h.settings.Debug,
				}
				if err := man.Update(); err != nil {
					return nil, err
				}
				// Reload the chart with the updated Chart.lock file.
				if chartRequested, err = loader.Load(cp); err != nil {
					return nil, errors.Wrap(err, "failed reloading chart after repoName update")
				}
			} else {
				return nil, err
			}
		}
	}

	return client.Run(chartRequested, vals)
}

func (h *Helm) parseOverrides(overrides []string) (map[string]interface{}, error) {
	vals := make(map[string]interface{})
	for _, value := range overrides {
		if err := strvals.ParseInto(value, vals); err != nil {
			return nil, errors.Wrap(err, "failed parsing --set data")
		}
	}
	return vals, nil
}

// Upgrade -
func (h *Helm) Upgrade(name string, chart, version string, overrides []string) error {
	client := action.NewUpgrade(h.cfg)
	client.Namespace = h.namespace
	client.Version = version

	chartPath, err := h.locateChart(chart, version)
	if err != nil {
		return err
	}

	// User specified a value via --set
	vals, err := h.parseOverrides(overrides)
	if err != nil {
		return err
	}

	// Check chart dependencies to make sure all are present in /charts
	ch, err := loader.Load(chartPath)
	if err != nil {
		return err
	}
	if req := ch.Metadata.Dependencies; req != nil {
		if err := action.CheckDependencies(ch, req); err != nil {
			return err
		}
	}

	if ch.Metadata.Deprecated {
		logrus.Warningf("This chart is deprecated")
	}

	upgrade := action.NewUpgrade(h.cfg)
	upgrade.Namespace = h.namespace
	_, err = upgrade.Run(name, ch, vals)
	return err
}

// Status -
func (h *Helm) Status(name string) (*release.Release, error) {
	// helm status RELEASE_NAME [flags]
	client := action.NewStatus(h.cfg)
	rel, err := client.Run(name)
	return rel, errors.Wrap(err, "helm status")
}

// Uninstall -
func (h *Helm) Uninstall(name string) error {
	logrus.Infof("uninstall helm app(%s/%s)", h.namespace, name)
	uninstall := action.NewUninstall(h.cfg)
	_, err := uninstall.Run(name)
	return err
}

// Rollback -
func (h *Helm) Rollback(name string, revision int) error {
	logrus.Infof("name: %s; revision: %d; rollback helm app", name, revision)
	client := action.NewRollback(h.cfg)
	client.Version = revision

	if err := client.Run(name); err != nil {
		return errors.Wrap(err, "helm rollback")
	}
	return nil
}

// History -
func (h *Helm) History(name string) (ReleaseHistory, error) {
	logrus.Debugf("name: %s; list helm app history", name)
	client := action.NewHistory(h.cfg)
	client.Max = 256

	hist, err := client.Run(name)
	if err != nil {
		if errors.Is(err, driver.ErrReleaseNotFound) {
			return nil, nil
		}
		return nil, errors.Wrap(err, "list helm app history")
	}

	releaseutil.Reverse(hist, releaseutil.SortByRevision)

	var rels []*release.Release
	for i := 0; i < min(len(hist), client.Max); i++ {
		rels = append(rels, hist[i])
	}

	if len(rels) == 0 {
		logrus.Debugf("name: %s; helm app history not found", name)
		return nil, nil
	}

	releaseHistory := getReleaseHistory(rels)

	return releaseHistory, nil
}

// Load loads the chart from the repository.
func (h *Helm) Load(chart, version string) (string, error) {
	return h.locateChart(chart, version)
}

// ChartPathOptions -
type ChartPathOptions struct {
	action.ChartPathOptions
}

// LocateChart looks for a chart directory in known places, and returns either the full path or an error.
func (c *ChartPathOptions) LocateChart(name, dest string, settings *cli.EnvSettings) (string, error) {
	name = strings.TrimSpace(name)
	version := strings.TrimSpace(c.ChartPathOptions.Version)

	if _, err := os.Stat(name); err == nil {
		abs, err := filepath.Abs(name)
		if err != nil {
			return abs, err
		}
		if c.ChartPathOptions.Verify {
			if _, err := downloader.VerifyChart(abs, c.ChartPathOptions.Keyring); err != nil {
				return "", err
			}
		}
		return abs, nil
	}
	if filepath.IsAbs(name) || strings.HasPrefix(name, ".") {
		return name, errors.Errorf("path %q not found", name)
	}

	dl := downloader.ChartDownloader{
		Out:     os.Stdout,
		Keyring: c.ChartPathOptions.Keyring,
		Getters: getter.All(settings),
		Options: []getter.Option{
			getter.WithBasicAuth(c.ChartPathOptions.Username, c.ChartPathOptions.Password),
			getter.WithTLSClientConfig(c.ChartPathOptions.CertFile, c.ChartPathOptions.KeyFile, c.ChartPathOptions.CaFile),
			getter.WithInsecureSkipVerifyTLS(c.ChartPathOptions.InsecureSkipTLSverify),
		},
		RepositoryConfig: settings.RepositoryConfig,
		RepositoryCache:  settings.RepositoryCache,
	}
	if c.ChartPathOptions.Verify {
		dl.Verify = downloader.VerifyAlways
	}
	if c.ChartPathOptions.RepoURL != "" {
		chartURL, err := repo.FindChartInAuthAndTLSRepoURL(c.ChartPathOptions.RepoURL, c.ChartPathOptions.Username, c.ChartPathOptions.Password, name, version,
			c.ChartPathOptions.CertFile, c.ChartPathOptions.KeyFile, c.ChartPathOptions.CaFile, c.ChartPathOptions.InsecureSkipTLSverify, getter.All(settings))
		if err != nil {
			return "", err
		}
		name = chartURL
	}

	if err := os.MkdirAll(dest, 0755); err != nil {
		return "", err
	}

	filename, _, err := dl.DownloadTo(name, version, dest)
	if err == nil {
		lname, err := filepath.Abs(filename)
		if err != nil {
			return filename, err
		}
		return lname, nil
	} else if settings.Debug {
		return filename, err
	}

	atVersion := ""
	if version != "" {
		atVersion = fmt.Sprintf(" at version %q", version)
	}
	return filename, errors.Errorf("failed to download %q%s (hint: running `helm repo update` may help)", name, atVersion)
}

// checkIfInstallable validates if a chart can be installed
//
// Application chart type is only installable
func checkIfInstallable(ch *chart.Chart) error {
	switch ch.Metadata.Type {
	case "", "application":
		return nil
	}
	return errors.Errorf("%s charts are not installable", ch.Metadata.Type)
}

func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

func getReleaseHistory(rls []*release.Release) (history ReleaseHistory) {
	for i := len(rls) - 1; i >= 0; i-- {
		r := rls[i]
		c := formatChartName(r.Chart)
		s := r.Info.Status.String()
		v := r.Version
		d := r.Info.Description
		a := formatAppVersion(r.Chart)

		rInfo := ReleaseInfo{
			Revision:    v,
			Status:      s,
			Chart:       c,
			AppVersion:  a,
			Description: d,
		}
		if !r.Info.LastDeployed.IsZero() {
			rInfo.Updated = r.Info.LastDeployed
		}
		history = append(history, rInfo)
	}

	return history
}

func formatChartName(c *chart.Chart) string {
	if c == nil || c.Metadata == nil {
		// This is an edge case that has happened in prod, though we don't
		// know how: https://github.com/helm/helm/issues/1347
		return "MISSING"
	}
	return fmt.Sprintf("%s-%s", c.Name(), c.Metadata.Version)
}

func formatAppVersion(c *chart.Chart) string {
	if c == nil || c.Metadata == nil {
		// This is an edge case that has happened in prod, though we don't
		// know how: https://github.com/helm/helm/issues/1347
		return "MISSING"
	}
	return c.AppVersion()
}

type Release struct {
	Name        string               `json:"name"`
	Namespace   string               `json:"namespace"`
	Revision    int                  `json:"revision"`
	Updated     time.Time            `json:"updated"`
	Status      string               `json:"status"`
	Chart       string               `json:"chart"`
	AppVersion  string               `json:"appVersion"`
	Description string               `json:"description"`
	Histories   []ReleaseHistoryInfo `json:"histories"`
}

type ReleaseHistoryInfo struct {
	Revision   int       `json:"revision"`
	Updated    time.Time `json:"updated"`
	Status     string    `json:"status"`
	AppVersion string    `json:"appVersion"`
}

type DeployInfo struct {
	Info        Resource                  `json:"info"`
	ApiResource unstructured.Unstructured `json:"apiResource"`
}

type Resource struct {
	APIVersion   string `json:"apiVersion"`
	gk           schema.GroupKind
	Kind         string            `json:"kind"`
	Name         string            `json:"name"`
	Namespace    string            `json:"namespace"`
	CreationTime time.Time         `json:"creationTime"`
	Labels       map[string]string `json:"labels"`
	Annotations  map[string]string `json:"annotations"`
}

func Resources(name, namespace string) ([]Resource, error) {
	actionConfig, err := getActionConfig(namespace)
	if err != nil {
		return nil, err
	}
	getAction := action.NewGet(actionConfig)
	release, err := getAction.Run(name)
	if err != nil {
		return nil, err
	}
	return resourcesFromManifest(namespace, release.Manifest)
}

func ToObjects(in io.Reader) ([]runtime.Object, error) {
	var result []runtime.Object
	reader := yamlDecoder.NewYAMLReader(bufio.NewReaderSize(in, 4096))
	for {
		raw, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		obj, err := toObjects(raw)
		if err != nil {
			return nil, err
		}

		result = append(result, obj...)
	}

	return result, nil
}

func toObjects(bytes []byte) ([]runtime.Object, error) {
	bytes, err := yamlDecoder.ToJSON(bytes)
	if err != nil {
		return nil, err
	}

	check := map[string]interface{}{}
	if err := json.Unmarshal(bytes, &check); err != nil || len(check) == 0 {
		return nil, err
	}

	obj, _, err := unstructured.UnstructuredJSONScheme.Decode(bytes, nil, nil)
	if err != nil {
		return nil, err
	}

	if l, ok := obj.(*unstructured.UnstructuredList); ok {
		var result []runtime.Object
		for _, obj := range l.Items {
			copy := obj
			result = append(result, &copy)
		}
		return result, nil
	}

	return []runtime.Object{obj}, nil
}

func resourcesFromManifest(namespace string, manifest string) (result []Resource, err error) {
	objs, err := ToObjects(bytes.NewReader([]byte(manifest)))
	if err != nil {
		return nil, err
	}

	for _, obj := range objs {
		o, err := meta.Accessor(obj)

		if err != nil {
			return nil, err
		}
		ns := o.GetNamespace()
		if len(ns) == 0 {
			ns = namespace
		}
		r := Resource{
			Name:         o.GetName(),
			Namespace:    ns,
			Labels:       o.GetLabels(),
			CreationTime: o.GetCreationTimestamp().Time,
		}

		gvk := obj.GetObjectKind().GroupVersionKind()
		r.APIVersion, r.Kind = gvk.ToAPIVersionAndKind()
		r.gk = gvk.GroupKind()
		result = append(result, r)
	}

	return result, nil
}

func clientGetter(namespace string) *genericclioptions.ConfigFlags {
	kc := KubeConfig()
	cf := genericclioptions.NewConfigFlags(false)
	cf.APIServer = &kc.Host
	cf.BearerToken = &kc.BearerToken
	cf.CAFile = &kc.CAFile
	cf.Namespace = &namespace
	return cf
}

func getActionConfig(namespace string) (*action.Configuration, error) {
	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(clientGetter(namespace), namespace, os.Getenv("HELM_DRIVER"), log.Printf); err != nil {
		return nil, err
	}
	return actionConfig, nil
}

func AllResources(name, namespace string) ([]DeployInfo, error) {
	res := make([]DeployInfo, 0)
	resources, err := Resources(name, namespace)
	if err != nil {
		return nil, err
	}

	for i := range resources {
		gvr, ok := KubeGVRFromGK(resources[i].gk)
		if !ok {
			continue
		}

		list, listErr := KubeDynamicClient().Resource(gvr).Namespace(metav1.NamespaceAll).List(context.TODO(), metav1.ListOptions{})
		if listErr != nil {
			return nil, listErr
		}
		for _, item := range list.Items {
			if item.GetName() == resources[i].Name && item.GetAnnotations()["meta.helm.sh/release-name"] == name && item.GetAnnotations()["meta.helm.sh/release-namespace"] == namespace {
				res = append(res, DeployInfo{
					Info: Resource{
						APIVersion:   item.GetAPIVersion(),
						Kind:         item.GetKind(),
						Namespace:    item.GetNamespace(),
						Name:         item.GetName(),
						Labels:       item.GetLabels(),
						Annotations:  item.GetAnnotations(),
						CreationTime: item.GetCreationTimestamp().Time,
					},
					ApiResource: item,
				})
			}

		}
	}
	return res, nil
}

func AllReleases(namespace string) ([]Release, error) {
	res := make([]Release, 0)
	actionConfig, err := getActionConfig(namespace)
	if err != nil {
		return nil, err
	}

	listAction := action.NewList(actionConfig)
	releases, err := listAction.Run()
	if err != nil {
		return nil, err
	}
	historyAction := action.NewHistory(actionConfig)
	for _, release := range releases {
		r := resourceFrom(release)

		histories, _ := historyAction.Run(release.Name)
		if len(histories) > 0 {
			for _, h := range histories {
				rh := resourceHistoryFrom(h)
				if rh.Revision == r.Revision {
					continue
				}
				r.Histories = append(r.Histories, rh)
			}
			sort.Slice(r.Histories, func(i, j int) bool {
				return r.Histories[i].Revision > (r.Histories[j].Revision)
			})
		}

		res = append(res, r)
	}
	return res, nil
}

func resourceFrom(release *release.Release) Release {
	r := Release{
		Revision:   release.Version,
		Name:       release.Name,
		Namespace:  release.Namespace,
		Updated:    release.Info.LastDeployed.Time,
		Status:     release.Info.Status.String(),
		Chart:      formatChartName(release.Chart),
		AppVersion: formatAppVersion(release.Chart),
	}

	if !release.Info.LastDeployed.IsZero() {
		r.Updated = release.Info.LastDeployed.Time
	}
	return r
}

func resourceHistoryFrom(release *release.Release) ReleaseHistoryInfo {
	rh := ReleaseHistoryInfo{
		Revision:   release.Version,
		Updated:    release.Info.LastDeployed.Time,
		Status:     release.Info.Status.String(),
		AppVersion: formatAppVersion(release.Chart),
	}

	if !release.Info.LastDeployed.IsZero() {
		rh.Updated = release.Info.LastDeployed.Time
	}
	return rh
}
