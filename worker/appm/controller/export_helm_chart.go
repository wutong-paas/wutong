package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/sirupsen/logrus"
	v1 "github.com/wutong-paas/wutong/worker/appm/types/v1"
	appv1 "k8s.io/api/apps/v1"
	"k8s.io/api/autoscaling/v2beta2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

type exportHelmChartController struct {
	stopChan     chan struct{}
	controllerID string
	appService   []v1.AppService
	manager      *Manager
	ctx          context.Context
	AppName      string
	AppVersion   string
	End          bool
}

func (s *exportHelmChartController) Begin() {
	var r WutongExport
	r.ConfigGroups = make(map[string]map[string]string)
	exportApp := fmt.Sprintf("%v-%v", s.AppName, s.AppVersion)
	exportPath := fmt.Sprintf("/wtdata/app/helm-chart/%v/%v-helm/%v", exportApp, exportApp, s.AppName)
	for _, service := range s.appService {
		err := s.exportOne(service, &r)
		if err != nil {
			logrus.Errorf("worker export %v failure %v", service.ServiceAlias, err)
		}
	}

	if s.End {
		if len(r.ConfigGroups) != 0 {
			configGroupByte, err := yaml.Marshal(r.ConfigGroups)
			if err != nil {
				logrus.Errorf("yaml marshal valueYaml failure %v", err)
			} else {
				err = s.write(path.Join(exportPath, "values.yaml"), configGroupByte, "")
				if err != nil {
					logrus.Errorf("write values.yaml configgroup failure %v", err)
				}
			}
		}
		r.StorageClass = "wutongvolumerwx"
		volumeYamlByte, err := yaml.Marshal(r)
		if err != nil {
			logrus.Errorf("yaml marshal valueYaml failure %v", err)
		}
		err = s.write(path.Join(exportPath, "values.yaml"), volumeYamlByte, "\n")
		if err != nil {
			logrus.Errorf("write values.yaml other failure %v", err)
		}
		err = s.write(path.Join(exportPath, "dependent_image.txt"), []byte(v1.GetOnlineProbeMeshImageName()), "\n")
		if err != nil {
			logrus.Errorf("write dependent_image.txt failure %v", err)
		}
	}
	s.manager.callback(s.controllerID, nil)
}

func (s *exportHelmChartController) Stop() error {
	close(s.stopChan)
	return nil
}

func (s *exportHelmChartController) exportOne(app v1.AppService, r *WutongExport) error {
	exportApp := fmt.Sprintf("%v-%v", s.AppName, s.AppVersion)
	exportPath := fmt.Sprintf("/wtdata/app/helm-chart/%v/%v-helm/%v", exportApp, exportApp, s.AppName)
	logrus.Infof("start export app %s to helm chart spec", s.AppName)

	exportTemplatePath := path.Join(exportPath, "templates")
	if err := prepareExportDir(exportTemplatePath); err != nil {
		return fmt.Errorf("create exportTemplatePath( %v )failure %v", exportTemplatePath, err)
	}
	logrus.Infof("success prepare helm chart template dir")

	if len(app.GetManifests()) > 0 {
		for _, manifest := range app.GetManifests() {
			manifest.SetNamespace("")
			resourceBytes, err := yaml.Marshal(manifest)
			if err != nil {
				return fmt.Errorf("manifest to yaml failure %v", err)
			}
			err = s.write(path.Join(exportTemplatePath, fmt.Sprintf("%v.yaml", manifest.GetKind())), resourceBytes, "\n---\n")
			if err != nil {
				return fmt.Errorf("write manifest yaml failure %v", err)
			}
		}
	}

	if configs := app.GetConfigMaps(); configs != nil {
		for _, config := range configs {
			config.Kind = "ConfigMap"
			config.APIVersion = APIVersionConfigMap
			config.Namespace = ""
			cmBytes, err := yaml.Marshal(config)
			if err != nil {
				return fmt.Errorf("configmap to yaml failure %v", err)
			}
			err = s.write(path.Join(exportTemplatePath, "ConfigMap.yaml"), cmBytes, "\n---\n")
			if err != nil {
				return fmt.Errorf("write configmap yaml failure %v", err)
			}
		}
	}

	for _, claim := range app.GetClaimsManually() {
		if *claim.Spec.StorageClassName != "" {
			sc := "{{ .Values.storageClass }}"
			claim.Spec.StorageClassName = &sc
		}
		claim.Kind = "PersistentVolumeClaim"
		claim.APIVersion = APIVersionPersistentVolumeClaim
		claim.Namespace = ""
		claim.Status = corev1.PersistentVolumeClaimStatus{}
		pvcBytes, err := yaml.Marshal(claim)
		if err != nil {
			return fmt.Errorf("pvc to yaml failure %v", err)
		}
		err = s.write(path.Join(exportTemplatePath, "PersistentVolumeClaim.yaml"), pvcBytes, "\n---\n")
		if err != nil {
			return fmt.Errorf("write pvc yaml failure %v", err)
		}
	}

	if statefulset := app.GetStatefulSet(); statefulset != nil {
		statefulset.Name = app.K8sComponentName
		statefulset.Spec.Template.Name = app.K8sComponentName + "-pod-spec"
		statefulset.Kind = "StatefulSet"
		statefulset.APIVersion = APIVersionStatefulSet
		statefulset.Namespace = ""
		image := statefulset.Spec.Template.Spec.Containers[0].Image
		imageCut := strings.Split(image, "/")
		Image := fmt.Sprintf("{{ default \"%v\" .Values.imageDomain }}/%v", strings.Join(imageCut[:len(imageCut)-1], "/"), imageCut[len(imageCut)-1])
		statefulset.Spec.Template.Spec.Containers[0].Image = Image
		for i := range statefulset.Spec.VolumeClaimTemplates {
			if *statefulset.Spec.VolumeClaimTemplates[i].Spec.StorageClassName != "" {
				sc := "{{ .Values.storageClass }}"
				statefulset.Spec.VolumeClaimTemplates[i].Spec.StorageClassName = &sc
			}
		}
		statefulset.Status = appv1.StatefulSetStatus{}
		statefulsetBytes, err := yaml.Marshal(statefulset)
		if err != nil {
			return fmt.Errorf("statefulset to yaml failure %v", err)
		}
		err = s.write(path.Join(exportTemplatePath, "StatefulSet.yaml"), statefulsetBytes, "\n---\n")
		if err != nil {
			return fmt.Errorf("write statefulset yaml failure %v", err)
		}
	}
	if deployment := app.GetDeployment(); deployment != nil {
		deployment.Name = app.K8sComponentName
		deployment.Spec.Template.Name = app.K8sComponentName + "-pod-spec"
		deployment.Kind = "Deployment"
		deployment.Namespace = ""
		deployment.APIVersion = APIVersionDeployment
		image := deployment.Spec.Template.Spec.Containers[0].Image
		imageCut := strings.Split(image, "/")
		Image := fmt.Sprintf("{{ default \"%v\" .Values.imageDomain }}/%v", strings.Join(imageCut[:len(imageCut)-1], "/"), imageCut[len(imageCut)-1])
		deployment.Spec.Template.Spec.Containers[0].Image = Image
		deployment.Status = appv1.DeploymentStatus{}
		deploymentBytes, err := yaml.Marshal(deployment)
		if err != nil {
			return fmt.Errorf("deployment to yaml failure %v", err)
		}
		err = s.write(path.Join(exportTemplatePath, "Deployment.yaml"), deploymentBytes, "\n---\n")
		if err != nil {
			return fmt.Errorf("write deployment yaml failure %v", err)
		}
	}

	if services := app.GetServices(true); services != nil {
		for _, svc := range services {
			svc.Kind = "Service"
			svc.Namespace = ""
			svc.APIVersion = APIVersionService
			if svc.Labels["service_type"] == "outer" {
				svc.Spec.Type = corev1.ServiceTypeNodePort
			}
			svc.Status = corev1.ServiceStatus{}
			svcBytes, err := yaml.Marshal(svc)
			if err != nil {
				return fmt.Errorf("svc to yaml failure %v", err)
			}
			err = s.write(path.Join(exportTemplatePath, "Service.yaml"), svcBytes, "\n---\n")
			if err != nil {
				return fmt.Errorf("write svc yaml failure %v", err)
			}
		}
	}
	if secrets := app.GetSecrets(true); secrets != nil {
		for _, secret := range secrets {
			if len(secret.ResourceVersion) == 0 {
				secret.Kind = "Secret"
				secret.APIVersion = APIVersionSecret
				secret.Namespace = ""
				secret.Type = ""
				secretBytes, err := yaml.Marshal(secret)
				if err != nil {
					return fmt.Errorf("secret to yaml failure %v", err)
				}
				err = s.write(path.Join(exportTemplatePath, "Secret.yaml"), secretBytes, "\n---\n")
				if err != nil {
					return fmt.Errorf("write secret yaml failure %v", err)
				}
			}
		}
	}
	type SecretType string
	type YamlSecret struct {
		metav1.TypeMeta   `json:",inline"`
		metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
		Immutable         *bool      `json:"immutable,omitempty" protobuf:"varint,5,opt,name=immutable"`
		Data              string     `json:"data,omitempty" protobuf:"bytes,2,rep,name=data"`
		StringData        string     `json:"stringData,omitempty" protobuf:"bytes,4,rep,name=stringData"`
		Type              SecretType `json:"type,omitempty" protobuf:"bytes,3,opt,name=type,casttype=SecretType"`
	}
	if s.End {
		if secrets := app.GetEnvVarSecrets(true); secrets != nil {
			for _, secret := range secrets {
				if len(secret.ResourceVersion) == 0 {
					secret.APIVersion = APIVersionSecret
					secret.Namespace = ""
					secret.Type = ""
					secret.Kind = "Secret"
					data := secret.Data
					secret.Data = nil
					var ySecret YamlSecret
					jsonSecret, err := json.Marshal(secret)
					if err != nil {
						return fmt.Errorf("json.Marshal configGroup secret failure %v", err)
					}
					err = json.Unmarshal(jsonSecret, &ySecret)
					if err != nil {
						return fmt.Errorf("json.Unmarshal configGroup secret failure %v", err)
					}
					templateConfigGroupName := strings.Split(ySecret.Name, "-")[0]
					templateConfigGroup := make(map[string]string)
					for key, value := range data {
						templateConfigGroup[key] = string(value)
					}
					r.ConfigGroups[templateConfigGroupName] = templateConfigGroup
					dataTemplate := fmt.Sprintf("  {{- range $key, $val := .Values.%v }}\n  {{ $key }}: {{ $val | quote}}\n  {{- end }}", templateConfigGroupName)
					ySecret.StringData = dataTemplate
					secretBytes, err := yaml.Marshal(ySecret)
					if err != nil {
						return fmt.Errorf("configGroup secret to yaml failure %v", err)
					}
					secretStr := strings.Replace(string(secretBytes), "|2-", "", 1)
					err = s.write(path.Join(exportTemplatePath, "Secret.yaml"), []byte(secretStr), "\n---\n")
					if err != nil {
						return fmt.Errorf("configGroup write secret yaml failure %v", err)
					}
				}
			}
		}
	}

	if hpas := app.GetHPAs(); len(hpas) != 0 {
		for _, hpa := range hpas {
			hpa.Kind = "HorizontalPodAutoscaler"
			hpa.Namespace = ""
			hpa.APIVersion = APIVersionHorizontalPodAutoscaler
			hpa.Status = v2beta2.HorizontalPodAutoscalerStatus{}
			if len(hpa.ResourceVersion) == 0 {
				hpaBytes, err := yaml.Marshal(hpa)
				if err != nil {
					return fmt.Errorf("hpa to yaml failure %v", err)
				}
				err = s.write(path.Join(exportTemplatePath, "HorizontalPodAutoscaler.yaml"), hpaBytes, "\n---\n")
				if err != nil {
					return fmt.Errorf("write hpa yaml failure %v", err)
				}
			}
		}
	}
	logrus.Infof("Create all app yaml file success, will waiting app export")
	return nil
}

func (s *exportHelmChartController) write(helmChartFilePath string, meta []byte, endString string) error {
	var fl *os.File
	var err error
	if CheckFileExist(helmChartFilePath) {
		fl, err = os.OpenFile(helmChartFilePath, os.O_APPEND|os.O_WRONLY, 0755)
		if err != nil {
			return err
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
