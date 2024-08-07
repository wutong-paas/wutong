package handler

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	dbmodel "github.com/wutong-paas/wutong/db/model"
	"github.com/wutong-paas/wutong/util"
)

var (
	ErrTenantEnvLackOfMemory = errors.New("tenant_env_lack_of_memory")
	ErrClusterLackOfMemory   = errors.New("cluster_lack_of_memory")
)

// CheckTenantEnvResource check tenantEnv's resource is support action or not
func CheckTenantEnvResource(ctx context.Context, tenantEnv *dbmodel.TenantEnvs, needMemory int) error {
	ts, err := GetServiceManager().GetTenantEnvRes(tenantEnv.UUID)
	if err != nil {
		return err
	}
	logrus.Debugf("tenant env limitMemory: %v, usedMemory: %v", tenantEnv.LimitMemory, ts.UsedMEM)
	if tenantEnv.LimitMemory != 0 {
		avaiMemory := tenantEnv.LimitMemory - ts.UsedMEM
		if needMemory > avaiMemory {
			logrus.Errorf("tenant env available memory is %d, To apply for %d, not enough", avaiMemory, needMemory)
			// return ErrTenantEnvLackOfMemory
			return fmt.Errorf("超出当前环境内存限额（%dM）", tenantEnv.LimitMemory)
		}
	}

	allcm, err := ClusterAllocMemory(ctx)
	if err != nil {
		return err
	}

	if int64(needMemory) > allcm {
		logrus.Errorf("cluster available memory is %d, To apply for %d, not enough", allcm, needMemory)
		return ErrClusterLackOfMemory
	}

	return nil
}

// ClusterAllocMemory returns the allocatable memory of the cluster.
func ClusterAllocMemory(ctx context.Context) (int64, error) {
	if logrus.IsLevelEnabled(logrus.DebugLevel) {
		defer util.Elapsed("ClusterAllocMemory")()
	}

	clusterInfo, err := GetTenantEnvManager().GetAllocatableResources(ctx)
	if err != nil {
		return 0, err
	}
	return clusterInfo.AllMemory - clusterInfo.RequestMemory, nil
}
