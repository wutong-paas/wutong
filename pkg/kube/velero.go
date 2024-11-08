package kube

import (
	"context"
	"log"
	"net/url"
	"os"

	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"github.com/wutong-paas/wutong/util"
	"gopkg.in/ini.v1"
	apiextclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

var isVeleroInstalled *bool
var veleroStatus *VeleroStatus

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

func GetVeleroStatus(kubeClient kubernetes.Interface) *VeleroStatus {
	if veleroStatus == nil {
		veleroStatus = initializeVeleroStatus(kubeClient)
	}
	return veleroStatus
}

func initializeVeleroStatus(kubeClient kubernetes.Interface) *VeleroStatus {
	var bsl velerov1.BackupStorageLocation
	if err := RuntimeClient().Get(context.Background(), types.NamespacedName{Name: "default", Namespace: "velero"}, &bsl); err != nil {
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
			return *isVeleroInstalled
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
