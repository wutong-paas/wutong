package controller

import "os"

var (
	//APIVersionSecret -
	APIVersionSecret = "v1"
	//APIVersionConfigMap -
	APIVersionConfigMap = "v1"
	//APIVersionPersistentVolumeClaim -
	APIVersionPersistentVolumeClaim = "v1"
	//APIVersionStatefulSet -
	APIVersionStatefulSet = "apps/v1"
	//APIVersionDeployment -
	APIVersionDeployment = "apps/v1"
	//APIVersionJob -
	APIVersionJob = "batch/v1"
	//APIVersionCronJob -
	APIVersionCronJob = "batch/v1"
	//APIVersionBetaCronJob -
	APIVersionBetaCronJob = "batch/v1beta1"
	//APIVersionService -
	APIVersionService = "v1"
	//APIVersionHorizontalPodAutoscaler -q
	APIVersionHorizontalPodAutoscaler = "autoscaling/v2"
	//APIVersionGateway -
	APIVersionGateway = "gateway.networking.k8s.io/v1beta1"
	//APIVersionHTTPRoute -
	APIVersionHTTPRoute = "gateway.networking.k8s.io/v1beta1"
)

// WutongExport -
type WutongExport struct {
	ImageDomain  string                       `json:"imageDomain"`
	StorageClass string                       `json:"storageClass"`
	ConfigGroups map[string]map[string]string `json:"-"`
}

// CheckFileExist check whether the file exists
func CheckFileExist(fileName string) bool {
	_, err := os.Stat(fileName)
	return !os.IsNotExist(err)
}

func prepareExportDir(exportPath string) error {
	return os.MkdirAll(exportPath, 0755)
}
