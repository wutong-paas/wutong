package kube

import (
	"context"
	"log"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/vmware-tanzu/velero/pkg/generated/clientset/versioned"
	informers "github.com/vmware-tanzu/velero/pkg/generated/informers/externalversions"
	velerov1 "github.com/vmware-tanzu/velero/pkg/generated/listers/velero/v1"
	"github.com/wutong-paas/wutong/util"
	"gopkg.in/ini.v1"
	apiextclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

var isVeleroInstalled *bool
var veleroStatus *VeleroStatus
var veleroCachedResources *VeleroCachedResources

type VeleroCachedResources struct {
	BackupLister                velerov1.BackupLister
	BackupRepositoryLister      velerov1.BackupRepositoryLister
	BackupStorageLocationLister velerov1.BackupStorageLocationLister
	RestoreLister               velerov1.RestoreLister
	PodVolumeBackupLister       velerov1.PodVolumeBackupLister
	PodVolumeRestoreLister      velerov1.PodVolumeRestoreLister
	DeleteBackupRequestLister   velerov1.DeleteBackupRequestLister
	ScheduleLister              velerov1.ScheduleLister
	DownloadRequestLister       velerov1.DownloadRequestLister
}

type VeleroStatus struct {
	S3Url             string
	S3UrlScheme       string
	S3Host            string
	S3Region          string
	S3Bucket          string
	S3AccessKeyID     string
	S3SecretAccessKey string
	ResticPassword    string
}

func GetVeleroStatus(kubeClient kubernetes.Interface, veleroClient versioned.Interface, apiextClient apiextclient.Interface) *VeleroStatus {
	if veleroStatus == nil {
		veleroStatus = initializeVeleroStatus(kubeClient, veleroClient, apiextClient)
	}
	return veleroStatus
}

func GetVeleroCachedResources(kubeClient kubernetes.Interface, veleroClient versioned.Interface, apiextClientset apiextclient.Interface) *VeleroCachedResources {
	if veleroCachedResources == nil {
		if IsVeleroInstalled(kubeClient, apiextClientset) {
			veleroCachedResources = initializeVeleroCachedResources(veleroClient)
		}
	}
	return veleroCachedResources
}

func initializeVeleroCachedResources(clientset versioned.Interface) *VeleroCachedResources {
	clientset.Discovery().ServerGroupsAndResources()
	sharedInformers := informers.NewSharedInformerFactory(clientset, time.Hour*8)

	// informer
	backupInfromer := sharedInformers.Velero().V1().Backups()
	backupRepositoryInformer := sharedInformers.Velero().V1().BackupRepositories()
	backupStorageInformer := sharedInformers.Velero().V1().BackupStorageLocations()
	restoreInformer := sharedInformers.Velero().V1().Restores()
	podVolumeBackupInformer := sharedInformers.Velero().V1().PodVolumeBackups()
	podVolumeRestoreInformer := sharedInformers.Velero().V1().PodVolumeRestores()
	deleteBackupRequestInformer := sharedInformers.Velero().V1().DeleteBackupRequests()
	scheduleInformer := sharedInformers.Velero().V1().Schedules()
	downloadRequestInformer := sharedInformers.Velero().V1().DownloadRequests()

	// shared informers
	backupSharedInformer := backupInfromer.Informer()
	backupRepositorySharedInformer := backupRepositoryInformer.Informer()
	backupStorageSharedInformer := backupStorageInformer.Informer()
	restoreSharedInformer := restoreInformer.Informer()
	podVolumeBackupSharedInformer := podVolumeBackupInformer.Informer()
	podVolumeRestoreSharedInformer := podVolumeRestoreInformer.Informer()
	deleteBackupRequestSharedInformer := deleteBackupRequestInformer.Informer()
	scheduleSharedInformer := scheduleInformer.Informer()
	downloadRequestSharedInformer := downloadRequestInformer.Informer()

	informers := map[string]cache.SharedInformer{
		"backupSharedInformer":              backupSharedInformer,
		"backupRepositorySharedInformer":    backupRepositorySharedInformer,
		"backupStorageSharedInformer":       backupStorageSharedInformer,
		"restoreSharedInformer":             restoreSharedInformer,
		"podVolumeBackupSharedInformer":     podVolumeBackupSharedInformer,
		"podVolumeRestoreSharedInformer":    podVolumeRestoreSharedInformer,
		"deleteBackupRequestSharedInformer": deleteBackupRequestSharedInformer,
		"scheduleSharedInformer":            scheduleSharedInformer,
		"downloadRequestSharedInformer":     downloadRequestSharedInformer,
	}
	var wg sync.WaitGroup
	wg.Add(len(informers))
	for k, v := range informers {
		go func(name string, informer cache.SharedInformer) {
			if !cache.WaitForCacheSync(wait.NeverStop, informer.HasSynced) {
				logrus.Warningln("wait for cached synced failed:", name)
			}
			wg.Done()
		}(k, v)
	}

	sharedInformers.Start(wait.NeverStop)
	sharedInformers.WaitForCacheSync(wait.NeverStop)
	return &VeleroCachedResources{
		BackupLister:                backupInfromer.Lister(),
		BackupRepositoryLister:      backupRepositoryInformer.Lister(),
		BackupStorageLocationLister: backupStorageInformer.Lister(),
		RestoreLister:               restoreInformer.Lister(),
		PodVolumeBackupLister:       podVolumeBackupInformer.Lister(),
		PodVolumeRestoreLister:      podVolumeRestoreInformer.Lister(),
		DeleteBackupRequestLister:   deleteBackupRequestInformer.Lister(),
		ScheduleLister:              scheduleInformer.Lister(),
		DownloadRequestLister:       downloadRequestInformer.Lister(),
	}
}

func initializeVeleroStatus(kubeClient kubernetes.Interface, veleroClient versioned.Interface, apiextClient apiextclient.Interface) *VeleroStatus {
	bsl, err := GetVeleroCachedResources(kubeClient, veleroClient, apiextClient).BackupStorageLocationLister.BackupStorageLocations("velero").Get("default")
	if err != nil {
		return nil
	}

	cloudCredentialsSecret, err := GetCachedResources(kubeClient).SecretLister.Secrets("velero").Get("cloud-credentials")
	if err != nil {
		return nil
	}
	cloudCredentials := cloudCredentialsSecret.Data["cloud"]
	if len(cloudCredentials) == 0 {
		return nil
	}
	cloudCredentialsData, err := ini.Load(cloudCredentials)
	if err != nil {
		return nil
	}

	accessKeyID := cloudCredentialsData.Section("default").Key("aws_access_key_id").String()
	secretAccessKey := cloudCredentialsData.Section("default").Key("aws_secret_access_key").String()
	if accessKeyID == "" || secretAccessKey == "" {
		return nil
	}
	os.Setenv("AWS_ACCESS_KEY_ID", accessKeyID)
	os.Setenv("AWS_SECRET_ACCESS_KEY", secretAccessKey)

	veleroRepoCredentialsSecret, err := GetCachedResources(kubeClient).SecretLister.Secrets("velero").Get("velero-repo-credentials")
	if err != nil {
		return nil
	}
	veleroRepositoryPassword := string(veleroRepoCredentialsSecret.Data["repository-password"])
	if len(veleroRepositoryPassword) == 0 {
		return nil
	}
	os.Setenv("RESTIC_PASSWORD", veleroRepositoryPassword)
	os.Setenv("KOPIA_PASSWORD", veleroRepositoryPassword)

	s3Url := bsl.Spec.Config["s3Url"]
	u, err := url.Parse(s3Url)
	if err != nil {
		return nil
	}

	return &VeleroStatus{
		S3Url:             bsl.Spec.Config["s3Url"],
		S3UrlScheme:       u.Scheme,
		S3Host:            u.Host,
		S3Region:          bsl.Spec.Config["region"],
		S3Bucket:          bsl.Spec.ObjectStorage.Bucket,
		S3AccessKeyID:     accessKeyID,
		S3SecretAccessKey: secretAccessKey,
		ResticPassword:    veleroRepositoryPassword,
	}
}

func IsVeleroInstalled(kubeClient kubernetes.Interface, apiextClient apiextclient.Interface) bool {
	if isVeleroInstalled == nil {
		_, err := apiextClient.ApiextensionsV1().CustomResourceDefinitions().Get(context.Background(), "backups.velero.io", metav1.GetOptions{})
		if err != nil {
			log.Println("not found velero crd: backups.velero.io")
			isVeleroInstalled = util.Ptr(false)
		}

		_, err = GetCachedResources(kubeClient).DeploymentLister.Deployments("velero").Get("velero")
		if err != nil {
			log.Println("not found velero deployment: velero/velero")
			isVeleroInstalled = util.Ptr(false)
		} else {
			isVeleroInstalled = util.Ptr(true)
		}
	}

	return *isVeleroInstalled
}
