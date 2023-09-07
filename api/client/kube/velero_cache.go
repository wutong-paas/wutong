package kube

import (
	"context"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/vmware-tanzu/velero/pkg/generated/clientset/versioned"
	informers "github.com/vmware-tanzu/velero/pkg/generated/informers/externalversions"
	velerov1 "github.com/vmware-tanzu/velero/pkg/generated/listers/velero/v1"
	apiextclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"k8s.io/client-go/tools/cache"
)

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

func GetVeleroCachedResources(clientset versioned.Interface, apiextClientset apiextclient.Interface) *VeleroCachedResources {
	if veleroCachedResources == nil {
		if IsVeleroInstalled(apiextClientset) {
			veleroCachedResources = initializeVeleroCachedResources(clientset)
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

func IsVeleroInstalled(client apiextclient.Interface) bool {
	_, err := client.ApiextensionsV1().CustomResourceDefinitions().Get(context.Background(), "backups.velero.io", metav1.GetOptions{})
	return err == nil
}
