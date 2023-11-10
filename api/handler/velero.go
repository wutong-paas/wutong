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

package handler

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/dustin/go-humanize"
	"github.com/spf13/cast"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"github.com/wutong-paas/wutong/api/client/kube"
	api_model "github.com/wutong-paas/wutong/api/model"
	"github.com/wutong-paas/wutong/db"
	"github.com/wutong-paas/wutong/util"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/util/retry"
)

// CreateBackup create backup for service resources and data
func (s *ServiceAction) CreateBackup(tenantEnvID, serviceID string, req api_model.CreateBackupRequest) error {
	if !kube.IsVeleroInstalled(s.kubeClient, s.apiextClient) {
		return errors.New("集群中未检测到 Velero 服务，使用该功能请联系管理员安装 Velero 服务！")
	}
	tenantEnv, err := db.GetManager().TenantEnvDao().GetTenantEnvByUUID(tenantEnvID)
	if err != nil {
		return errors.New("环境不存在！")
	}
	volumes, _ := db.GetManager().TenantEnvServiceVolumeDao().GetTenantEnvServiceVolumesByServiceID(serviceID)
	if len(volumes) == 0 {
		return errors.New("当前组件没有挂载存储！")
	}

	// 1、如果当前组件没有处于运行中，则需要先启动组件
	pods, err := s.GetPods(serviceID)
	if err != nil {
		return errors.New("获取组件状态失败！")
	}
	if pods == nil || (len(pods.NewPods) == 0 && len(pods.OldPods) == 0) {
		return errors.New("当前组件未运行，请先启动组件！")
	}

	selector := labels.SelectorFromSet(labels.Set{
		"wutong.io/service_id": serviceID,
	})

	// 2、校验是否存在未完成的备份任务
	histories, err := kube.GetVeleroCachedResources(s.kubeClient, s.veleroClient, s.apiextClient).BackupLister.Backups("velero").List(selector)
	if err != nil {
		logrus.Errorf("get velero backup history failed, error: %s", err.Error())
		return errors.New("校验历史备份数据失败！")
	}
	for _, history := range histories {
		if history.Status.Phase != "Completed" {
			return errors.New("当前组件还存在未完成的备份任务！")
		}
	}

	backup := &velerov1.Backup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceID + "-" + time.Now().Format("20060102150405"),
			Namespace: "velero",
			Labels: map[string]string{
				"wutong.io/service_id": serviceID,
				"wutong.io/backup-ttl": req.TTL,
			},
			Annotations: map[string]string{
				"wutong.io/creator": req.Operator,
				"wutong.io/desc":    req.Desc,
			},
		},
		Spec: velerov1.BackupSpec{
			CSISnapshotTimeout: metav1.Duration{Duration: 10 * time.Minute},
			DefaultVolumesToFsBackup: func() *bool {
				b := true
				return &b
			}(),
			IncludedNamespaces: []string{tenantEnv.Namespace},
			LabelSelector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"service_id": serviceID, // 梧桐组件的 Label
				},
			},
			StorageLocation: "default",
			TTL:             parseTTLorDefault(req.TTL),
		},
	}

	_, err = s.veleroClient.VeleroV1().Backups("velero").Create(context.Background(), backup, metav1.CreateOptions{})
	if err != nil {
		logrus.Errorf("create velero backup failed, error: %s", err.Error())
		return errors.New("创建备份任务失败！")
	}
	return nil
}

// CreateBackupSchedule create backup schedule for service resources and data
func (s *ServiceAction) CreateBackupSchedule(tenantEnvID, serviceID string, req api_model.CreateBackupScheduleRequest) error {
	if !kube.IsVeleroInstalled(s.kubeClient, s.apiextClient) {
		return errors.New("集群中未检测到 Velero 服务，使用该功能请联系管理员安装 Velero 服务！")
	}
	tenantEnv, err := db.GetManager().TenantEnvDao().GetTenantEnvByUUID(tenantEnvID)
	if err != nil {
		return errors.New("环境不存在！")
	}
	volumes, _ := db.GetManager().TenantEnvServiceVolumeDao().GetTenantEnvServiceVolumesByServiceID(serviceID)
	if len(volumes) == 0 {
		return errors.New("当前组件没有挂载存储！")
	}

	// 1、如果当前组件没有处于运行中，则需要先启动组件
	// pods, err := s.GetPods(serviceID)
	// if err != nil {
	// 	return errors.New("获取组件状态失败！")
	// }
	// if pods == nil || (len(pods.NewPods) == 0 && len(pods.OldPods) == 0) {
	// 	return errors.New("当前组件未运行，请先启动组件！")
	// }

	// 2、校验是否存在未完成的备份定时任务
	_, err = kube.GetVeleroCachedResources(s.kubeClient, s.veleroClient, s.apiextClient).ScheduleLister.Schedules("velero").Get(serviceID)
	if err == nil {
		logrus.Error("schedule already exists")
		return errors.New("当前组件已存在定时备份计划！")
	}
	schedule := &velerov1.Schedule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceID,
			Namespace: "velero",
			Labels: map[string]string{
				"wutong.io/service_id": serviceID,
				"wutong.io/backup-ttl": req.TTL,
			},
			Annotations: map[string]string{
				"wutong.io/creator": req.Operator,
				"wutong.io/desc":    req.Desc,
			},
		},
		Spec: velerov1.ScheduleSpec{
			Schedule: req.Cron,
			Template: velerov1.BackupSpec{
				CSISnapshotTimeout: metav1.Duration{Duration: 10 * time.Minute},
				DefaultVolumesToFsBackup: func() *bool {
					b := true
					return &b
				}(),
				IncludedNamespaces: []string{tenantEnv.Namespace},
				LabelSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"service_id": serviceID, // 梧桐组件的 Label
					},
				},
				StorageLocation: "default",
				TTL:             parseTTLorDefault(req.TTL),
			},
		},
	}

	_, err = s.veleroClient.VeleroV1().Schedules("velero").Create(context.Background(), schedule, metav1.CreateOptions{})
	if err != nil {
		logrus.Errorf("create velero backup schedule failed, error: %s", err.Error())
		return errors.New("创建定时备份计划失败！")
	}
	return nil
}

// UpdateBackupSchedule update backup schedule for service resources and data
func (s *ServiceAction) UpdateBackupSchedule(tenantEnvID, serviceID string, req api_model.UpdateBackupScheduleRequest) error {
	if !kube.IsVeleroInstalled(s.kubeClient, s.apiextClient) {
		return errors.New("集群中未检测到 Velero 服务，使用该功能请联系管理员安装 Velero 服务！")
	}

	volumes, _ := db.GetManager().TenantEnvServiceVolumeDao().GetTenantEnvServiceVolumesByServiceID(serviceID)
	if len(volumes) == 0 {
		return errors.New("当前组件没有挂载存储！")
	}

	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		latest, err := s.veleroClient.VeleroV1().Schedules("velero").Get(context.Background(), serviceID, metav1.GetOptions{})
		if err != nil {
			return err
		}
		latest.Labels["wutong.io/backup-ttl"] = req.TTL
		latest.Annotations["wutong.io/last_modifier"] = req.Operator
		latest.Annotations["wutong.io/desc"] = req.Desc
		latest.Spec.Schedule = req.Cron
		latest.Spec.Template.TTL = parseTTLorDefault(req.TTL)
		_, err = s.veleroClient.VeleroV1().Schedules("velero").Update(context.Background(), latest, metav1.UpdateOptions{})
		return err
	})
	if err != nil {
		return errors.New("更新定时备份计划失败！")
	}
	return nil
}

// DeleteBackupSchedule
func (s *ServiceAction) DeleteBackupSchedule(serviceID string) error {
	if !kube.IsVeleroInstalled(s.kubeClient, s.apiextClient) {
		return errors.New("集群中未检测到 Velero 服务，使用该功能请联系管理员安装 Velero 服务！")
	}

	err := s.veleroClient.VeleroV1().Schedules("velero").Delete(context.Background(), serviceID, metav1.DeleteOptions{})
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			return nil
		}
		return errors.New("删除备份计划失败！")
	}
	return nil
}

// DownloadBackup download backup data
func (s *ServiceAction) DownloadBackup(serviceID, backupID string) ([]byte, error) {
	if !kube.IsVeleroInstalled(s.kubeClient, s.apiextClient) {
		return nil, errors.New("集群中未检测到 Velero 服务，使用该功能请联系管理员安装 Velero 服务！")
	}

	backup, err := kube.GetVeleroCachedResources(s.kubeClient, s.veleroClient, s.apiextClient).BackupLister.Backups("velero").Get(backupID)
	if err != nil {
		logrus.Errorf("download backup data failed, error: %s", err.Error())
		return nil, errors.New("获取备份失败！")
	}

	if backup.Labels["wutong.io/service_id"] != serviceID {
		return nil, errors.New("当前备份不属于该组件！")
	}

	if backup.Status.Phase != "Completed" {
		return nil, errors.New("当前备份还未完成！")
	}

	veleroStatus := kube.GetVeleroStatus(s.kubeClient, s.veleroClient, s.apiextClient)
	if veleroStatus == nil {
		return nil, errors.New("获取 Velero 存储信息失败！")
	}

	object := fmt.Sprintf("/backups/%s/%s.tar.gz", backupID, backupID)

	// Minio client
	// useSSL := u.Scheme == "https"
	// minioClient, err := minio.NewWithRegion(u.Host, accessKey, secretKey, useSSL, region)
	// if err != nil {
	// 	logrus.Errorf("download backup data failed, bucket: %s, object: %s, error: %s", veleroStatus.S3Bucket, object, err.Error())
	// 	return nil, errors.New("下载备份数据失败！")
	// }

	// obj, err := minioClient.GetObject(bucket, object, minio.GetObjectOptions{})
	// if err != nil {
	// 	return nil, errors.New("下载备份数据失败！")
	// }
	// defer obj.Close()

	// bytes, err := io.ReadAll(obj)
	// if err != nil {
	// 	return nil, errors.New("下载备份数据失败！")
	// }

	// S3 standard client
	disableSSL := veleroStatus.S3UrlScheme != "https"
	sess, err := session.NewSession(&aws.Config{
		Endpoint:         &veleroStatus.S3Host,
		Region:           &veleroStatus.S3Region,
		S3ForcePathStyle: util.Ptr(true),
		Credentials:      credentials.NewStaticCredentials(veleroStatus.S3AccessKeyID, veleroStatus.S3SecretAccessKey, ""),
		DisableSSL:       util.Ptr(disableSSL),
	})
	if err != nil {
		logrus.Errorf("download backup data failed, bucket: %s, object: %s, error: %s", veleroStatus.S3Bucket, object, err.Error())
		return nil, errors.New("下载备份数据失败！")
	}

	out, err := s3.New(sess).GetObject(&s3.GetObjectInput{
		Bucket: util.Ptr(veleroStatus.S3Bucket),
		Key:    util.Ptr(object),
	})
	if err != nil {
		logrus.Errorf("download backup data failed, bucket: %s, object: %s, error: %s", veleroStatus.S3Bucket, object, err.Error())
		return nil, errors.New("下载备份数据失败！")
	}
	defer out.Body.Close()

	tarBuffer := new(bytes.Buffer)
	tarWriter := tar.NewWriter(tarBuffer)

	manifests, err := io.ReadAll(out.Body)
	if err != nil {
		return nil, errors.New("下载备份数据失败！")
	}
	addFileToTar(tarWriter, "manifests.tar", manifests)

	pvbl, _ := kube.GetVeleroCachedResources(s.kubeClient, s.veleroClient, s.apiextClient).PodVolumeBackupLister.List(labels.SelectorFromSet(labels.Set{
		"velero.io/backup-name": backupID,
	}))

	for _, pvb := range pvbl {
		switch pvb.Spec.UploaderType {
		case velerov1.BackupRepositoryTypeKopia:
			// TODO: 不是很好的解决方案，后续需要优化
			// 1. 如果不存在，添加 kopia 操作用户，以命名空间名作为用户名
			kopiaUser := "kopia-" + pvb.Spec.Pod.Namespace
			exec.Command("adduser", kopiaUser, "-D").Run()

			// 2. 当前用户执行 kopia 用户命令
			rawConnectCmd := fmt.Sprintf("kopia repository connect s3 --endpoint %s --region %s --bucket %s --prefix kopia/%s/", veleroStatus.S3Host, veleroStatus.S3Region, veleroStatus.S3Bucket, pvb.Spec.Pod.Namespace)
			if disableSSL {
				rawConnectCmd = rawConnectCmd + " --disable-tls"
			}
			connetctCmd := exec.Command("su", kopiaUser, "-c", rawConnectCmd)
			err := connetctCmd.Run()
			if err != nil {
				logrus.Warningf("kopia repository connect error: %s", err.Error())
				continue
			}

			tmpFile := fmt.Sprintf("/home/%s/kopia-%s.tar", kopiaUser, pvb.Status.SnapshotID)
			rawRestoreCmd := fmt.Sprintf("kopia snapshot restore %s %s", pvb.Status.SnapshotID, tmpFile)
			restoreCmd := exec.Command("su", kopiaUser, "-c", rawRestoreCmd)
			err = restoreCmd.Run()
			if err != nil {
				logrus.Warningf("kopia snapshot restore error: %s", err.Error())
				continue
			}
			volumeData, err := os.ReadFile(tmpFile)
			if err != nil {
				logrus.Warningf("read kopia snapshot file error: %s", err.Error())
				continue
			}
			if len(volumeData) > 0 {
				addFileToTar(tarWriter, fmt.Sprintf("volumes/%s.tar", pvb.Name), volumeData)
			}
			os.Remove(tmpFile)
		case velerov1.BackupRepositoryTypeRestic:
			dumpCmd := exec.Command("restic", "-r", pvb.Spec.RepoIdentifier, "--verbose", "dump", pvb.Status.SnapshotID, "/")
			volumeData, err := dumpCmd.Output()
			if err != nil {
				logrus.Warningf("restic dump error: %s", err.Error())
				continue
			}
			if len(volumeData) > 0 {
				addFileToTar(tarWriter, fmt.Sprintf("volumes/%s.tar", pvb.Name), volumeData)
			}
		}
	}

	// 关闭 tar.Writer，完成归档文件的创建
	tarWriter.Close()

	gzipBuffer := new(bytes.Buffer)
	gzipWriter := gzip.NewWriter(gzipBuffer)

	_, err = io.Copy(gzipWriter, tarBuffer)
	if err != nil {
		return nil, errors.New("下载备份数据失败！")
	}

	// 关闭 gzip.Writer，完成压缩文件的创建
	gzipWriter.Close()

	return gzipBuffer.Bytes(), nil
}

// DeleteBackup delete backup for service resources and data
func (s *ServiceAction) DeleteBackup(serviceID, backupID string) error {
	if !kube.IsVeleroInstalled(s.kubeClient, s.apiextClient) {
		return errors.New("集群中未检测到 Velero 服务，使用该功能请联系管理员安装 Velero 服务！")
	}

	backup, err := kube.GetVeleroCachedResources(s.kubeClient, s.veleroClient, s.apiextClient).BackupLister.Backups("velero").Get(backupID)
	if err != nil {
		return errors.New("获取待删除备份记录失败！")
	}

	if backup.Labels["wutong.io/service_id"] != serviceID {
		return errors.New("当前备份不属于该组件！")
	}

	dbrl, _ := kube.GetVeleroCachedResources(s.kubeClient, s.veleroClient, s.apiextClient).DeleteBackupRequestLister.List(labels.SelectorFromSet(labels.Set{
		"velero.io/backup-name": backupID,
	}))
	if len(dbrl) > 0 {
		return errors.New("当前备份记录已存在删除请求，请稍后再试！")
	}

	_, err = s.veleroClient.VeleroV1().DeleteBackupRequests("velero").Create(context.Background(), &velerov1.DeleteBackupRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceID + "-" + time.Now().Format("20060102150405"),
			Namespace: "velero",
			Labels: map[string]string{
				"velero.io/backup-name": backupID,
			},
		},
		Spec: velerov1.DeleteBackupRequestSpec{
			BackupName: backupID,
		},
	}, metav1.CreateOptions{})
	if err != nil {
		return errors.New("创建删除备份请求失败！")
	}
	return nil
}

// CreateRestore create restore for service resources and data from backup
func (s *ServiceAction) CreateRestore(tenantEnvID, serviceID string, req api_model.CreateRestoreRequest) error {
	if !kube.IsVeleroInstalled(s.kubeClient, s.apiextClient) {
		return errors.New("集群中未检测到 Velero 服务，使用该功能请联系管理员安装 Velero 服务！")
	}

	// 1、如果当前组件处于运行中，则先关闭组件
	pods, err := s.GetPods(serviceID)
	if err != nil {
		return errors.New("获取组件状态失败！")
	}
	if pods != nil {
		if len(pods.NewPods) > 0 || len(pods.OldPods) > 0 {
			return errors.New("当前组件处于运行中，请先关闭组件！")
		}
	}
	// 2、校验当前是否有未完成的还原任务
	restores, err := kube.GetVeleroCachedResources(s.kubeClient, s.veleroClient, s.apiextClient).RestoreLister.Restores("velero").List(labels.SelectorFromSet(labels.Set{
		"wutong.io/service_id": serviceID,
	}))
	if err != nil {
		return errors.New("获取历史恢复数据失败！")
	}

	for _, restore := range restores {
		if restore.Status.Phase == velerov1.RestorePhaseNew ||
			restore.Status.Phase == velerov1.RestorePhaseInProgress ||
			restore.Status.Phase == velerov1.RestorePhaseWaitingForPluginOperations ||
			restore.Status.Phase == velerov1.RestorePhaseWaitingForPluginOperationsPartiallyFailed {
			return errors.New("当前组件还存在未完成的还原任务！")
		}
	}

	// 3、校验备份数据是否存在
	backup, err := kube.GetVeleroCachedResources(s.kubeClient, s.veleroClient, s.apiextClient).BackupLister.Backups("velero").Get(req.BackupID)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			return errors.New("备份数据不存在！")
		}
		return errors.New("获取备份数据失败！")
	}

	if backup.Labels["wutong.io/service_id"] != serviceID {
		return errors.New("当前备份不属于该组件！")
	}

	restore := &velerov1.Restore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceID + "-" + time.Now().Format("20060102150405"),
			Namespace: "velero",
			Labels:    backup.Labels,
			Annotations: map[string]string{
				"wutong.io/creator": req.Operator,
			},
		},
		Spec: velerov1.RestoreSpec{
			BackupName: req.BackupID,
			ExcludedResources: []string{
				"nodes",
				"events",
				"endpoints",
				"events.events.k8s.io",
				"backups.velero.io",
				"restores.velero.io",
				"resticrepositories.velero.io",
				"csinodes.storage.k8s.io",
				"volumeattachments.storage.k8s.io",
				"backuprepositories.velero.io",
			},
		},
	}

	_, err = s.veleroClient.VeleroV1().Restores("velero").Create(context.Background(), restore, metav1.CreateOptions{})
	if err != nil {
		return errors.New("创建还原任务失败！")
	}
	return nil
}

// DeleteRestore delete restore for service resources and data from backup
func (s *ServiceAction) DeleteRestore(serviceID, restoreID string) error {
	if !kube.IsVeleroInstalled(s.kubeClient, s.apiextClient) {
		return errors.New("集群中未检测到 Velero 服务，使用该功能请联系管理员安装 Velero 服务！")
	}

	r, err := kube.GetVeleroCachedResources(s.kubeClient, s.veleroClient, s.apiextClient).RestoreLister.Restores("velero").Get(restoreID)
	if err != nil {
		return errors.New("获取待删除还原记录失败！")
	}

	if r.Labels["wutong.io/service_id"] != serviceID {
		return errors.New("当前还原记录不属于该组件！")
	}

	err = s.veleroClient.VeleroV1().Restores("velero").Delete(context.Background(), restoreID, metav1.DeleteOptions{})
	if err != nil {
		logrus.Error(err)
		return errors.New("删除恢复历史失败！")
	}
	return nil
}

// GetBackupSchedule get velero backup schedule
func (s *ServiceAction) GetBackupSchedule(tenantEnvID, serviceID string) (*api_model.BackupSchedule, bool) {
	if !kube.IsVeleroInstalled(s.kubeClient, s.apiextClient) {
		return nil, false
	}

	schedule, err := kube.GetVeleroCachedResources(s.kubeClient, s.veleroClient, s.apiextClient).ScheduleLister.Schedules("velero").Get(serviceID)
	if err != nil {
		return nil, false
	}

	result := &api_model.BackupSchedule{
		ScheduleID:   serviceID,
		ServiceID:    serviceID,
		Cron:         schedule.Spec.Schedule,
		TTL:          ttlStr(schedule.Labels["wutong.io/backup-ttl"], schedule.Spec.Template.TTL),
		Desc:         schedule.Annotations["wutong.io/desc"],
		Creator:      schedule.Annotations["wutong.io/creator"],
		LastModifier: schedule.Annotations["wutong.io/last_modifier"],
	}
	return result, true
}

// BackupRecords get velero backup histories
func (s *ServiceAction) BackupRecords(tenantEnvID, serviceID string) ([]*api_model.BackupRecord, error) {
	if !kube.IsVeleroInstalled(s.kubeClient, s.apiextClient) {
		return nil, nil
	}
	var result []*api_model.BackupRecord
	backups, err := kube.GetVeleroCachedResources(s.kubeClient, s.veleroClient, s.apiextClient).BackupLister.Backups("velero").List(labels.SelectorFromSet(labels.Set{
		"wutong.io/service_id": serviceID,
	}))
	if err != nil {
		return nil, errors.New("获取历史备份数据失败！")
	}
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].CreationTimestamp.Time.After(backups[j].CreationTimestamp.Time)
	})
	for _, backup := range backups {
		var restorable = true
		pvbs, _ := kube.GetVeleroCachedResources(s.kubeClient, s.veleroClient, s.apiextClient).PodVolumeBackupLister.PodVolumeBackups("velero").List(labels.SelectorFromSet(labels.Set{
			"velero.io/backup-name": backup.Name,
		}))
		if len(pvbs) == 0 || backup.Status.Phase != velerov1.BackupPhaseCompleted {
			restorable = false
		}
		var totalBytes, completedBytes int64
		for _, pvb := range pvbs {
			if pvb != nil {
				totalBytes += pvb.Status.Progress.TotalBytes
				completedBytes += pvb.Status.Progress.BytesDone
			}
		}
		var totalItems, completedItems int
		if backup.Status.Progress != nil {
			totalItems = backup.Status.Progress.TotalItems
			completedItems = backup.Status.Progress.ItemsBackedUp
		}
		result = append(result, &api_model.BackupRecord{
			BackupID:       backup.Name,
			ServiceID:      serviceID,
			TTL:            ttlStr(backup.Labels["wutong.io/backup-ttl"], backup.Spec.TTL),
			Desc:           backup.Annotations["wutong.io/desc"],
			Mode:           backupMode(backup),
			CreatedAt:      convertMetaV1Time(backup.Status.StartTimestamp),
			CompletedAt:    convertMetaV1Time(backup.Status.CompletionTimestamp),
			ExpiredAt:      convertMetaV1Time(backup.Status.Expiration),
			Size:           formatBytesSize(totalBytes),
			ProgressRate:   formatProcessRate(totalBytes, completedBytes),
			CompletedItems: completedItems,
			TotalItems:     totalItems,
			Status:         string(backup.Status.Phase),
			FailureReason:  backup.Status.FailureReason,
			Operator:       backup.Annotations["wutong.io/creator"],
			Restorable:     restorable,
		})
	}
	return result, nil
}

// RestoreRecords get velero restore histories
func (s *ServiceAction) RestoreRecords(tenantEnvID, serviceID string) ([]*api_model.RestoreRecord, error) {
	if !kube.IsVeleroInstalled(s.kubeClient, s.apiextClient) {
		return nil, nil
	}
	var result []*api_model.RestoreRecord
	restores, err := kube.GetVeleroCachedResources(s.kubeClient, s.veleroClient, s.apiextClient).RestoreLister.Restores("velero").List(labels.SelectorFromSet(labels.Set{
		"wutong.io/service_id": serviceID,
	}))
	if err != nil {
		return nil, errors.New("获取历史恢复数据失败！")
	}

	sort.Slice(restores, func(i, j int) bool {
		return restores[i].CreationTimestamp.Time.After(restores[j].CreationTimestamp.Time)
	})
	for _, restore := range restores {
		pvbs, _ := kube.GetVeleroCachedResources(s.kubeClient, s.veleroClient, s.apiextClient).PodVolumeRestoreLister.PodVolumeRestores("velero").List(labels.SelectorFromSet(labels.Set{
			"velero.io/restore-name": restore.Name,
		}))
		var totalBytes, completedBytes int64
		for _, pvb := range pvbs {
			if pvb != nil {
				totalBytes += pvb.Status.Progress.TotalBytes
				completedBytes += pvb.Status.Progress.BytesDone
			}
		}
		var totalItems, completedItems int
		if restore.Status.Progress != nil {
			totalItems = restore.Status.Progress.TotalItems
			completedItems = restore.Status.Progress.ItemsRestored
		}
		result = append(result, &api_model.RestoreRecord{
			RestoreID:      restore.Name,
			BackupID:       restore.Spec.BackupName,
			ServiceID:      serviceID,
			CreatedAt:      convertMetaV1Time(restore.Status.StartTimestamp),
			CompletedAt:    convertMetaV1Time(restore.Status.CompletionTimestamp),
			Size:           formatBytesSize(totalBytes),
			ProgressRate:   formatProcessRate(totalBytes, completedBytes),
			CompletedItems: completedItems,
			TotalItems:     totalItems,
			Status:         string(restore.Status.Phase),
			FailureReason:  restore.Status.FailureReason,
			Operator:       restore.Annotations["wutong.io/creator"],
		})
	}
	return result, nil
}

func backupMode(backup *velerov1.Backup) string {
	if _, ok := backup.Labels["velero.io/schedule-name"]; ok {
		return "Scheduled"
	}
	return "Manual"
}

func ttlStr(labelVal string, ttl metav1.Duration) string {
	if labelVal != "" {
		return labelVal
	}
	durStr := ttl.Duration.String()
	s := strings.Split(durStr, "h")
	return s[0] + "h"
}

func formatBytesSize(size int64) string {
	if size == 0 {
		return "-"
	}
	return humanize.Bytes(uint64(size))
}

func formatProcessRate(total, completed int64) string {
	if total == 0 {
		return "-"
	}
	return fmt.Sprintf("%s%%", fmt.Sprintf("%.2f", float64(completed)/float64(total)*100))
}

// parseDayOrHourTTL parse day or hour ttl to hour ttl
func parseTTLorDefault(ttl string) metav1.Duration {
	if strings.HasSuffix(ttl, "d") {
		dayNoStr := strings.TrimSuffix(ttl, "d")
		dayNo := cast.ToInt(dayNoStr)
		ttl = fmt.Sprintf("%dh", dayNo*24)
	}
	dur, err := time.ParseDuration(ttl)
	if err != nil {
		dur = time.Hour * 24 * 30
	}
	if dur == 0 {
		dur = time.Hour * 24 * 30
	}
	return metav1.Duration{Duration: dur}
}

func convertMetaV1Time(t *metav1.Time) string {
	if t == nil {
		return "-"
	}
	return t.Format("2006-01-02 15:04:05")
}

func addFileToTar(tarWriter *tar.Writer, fileName string, content []byte) {
	header := &tar.Header{
		Name: fileName,
		Mode: 0644,
		Size: int64(len(content)),
	}
	err := tarWriter.WriteHeader(header)
	if err != nil {
		panic(err)
	}
	_, err = tarWriter.Write(content)
	if err != nil {
		panic(err)
	}
}
