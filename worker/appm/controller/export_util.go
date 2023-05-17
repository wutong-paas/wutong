package controller

import (
	"fmt"
	"os"
)

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
	//APIVersionV1Ingress -
	APIVersionV1Ingress = "networking.k8s.io/v1"
	//APIVersionV1beta1Ingress -
	APIVersionV1beta1Ingress = "networking.k8s.io/v1beta1"
	//APIVersionHorizontalPodAutoscaler -q
	APIVersionHorizontalPodAutoscaler = "autoscaling/v2"
	//APIVersionGateway -
	APIVersionGateway = "gateway.networking.k8s.io/v1beta1"
	//APIVersionHTTPRoute -
	APIVersionHTTPRoute = "gateway.networking.k8s.io/v1beta1"
)

// WutongExport -
type WutongExport struct {
	ImageDomain     string                       `json:"imageDomain"`
	StorageClass    string                       `json:"storageClass"`
	ExternalDomains map[string]map[string]string `json:"externalDomains"`
	ConfigGroups    map[string]map[string]string `json:"secretEnvs"`
}

// CheckFileExist check whether the file exists
func CheckFileExist(fileName string) bool {
	_, err := os.Stat(fileName)
	return !os.IsNotExist(err)
}

func prepareExportDir(exportPath string) error {
	return os.MkdirAll(exportPath, 0755)
}

func write(helmChartFilePath string, meta []byte, endString string, appendFile bool) error {
	var fl *os.File
	var err error
	if CheckFileExist(helmChartFilePath) {
		fl, err = os.OpenFile(helmChartFilePath, os.O_APPEND|os.O_WRONLY, 0755)
		if err != nil {
			return err
		}

		if !appendFile {
			fl.Truncate(0)
		}
	} else {
		fl, err = os.Create(helmChartFilePath)
		if err != nil {
			return err
		}
	}
	defer fl.Close()
	n, err := fl.Write(append(meta, []byte(endString)...))
	if err != nil {
		return err
	}
	if n < len(append(meta, []byte(endString)...)) {
		return fmt.Errorf("write insufficient length")
	}
	return nil
}
