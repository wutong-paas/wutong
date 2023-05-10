package controller

import (
	"context"
	"fmt"
	"os"
	"path"

	"github.com/sirupsen/logrus"
	"github.com/wutong-paas/wutong/db"
	v1 "github.com/wutong-paas/wutong/worker/appm/types/v1"
	appv1 "k8s.io/api/apps/v1"
	"k8s.io/api/autoscaling/v2beta2"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"
)

type exportK8sYamlController struct {
	stopChan     chan struct{}
	controllerID string
	appService   []v1.AppService
	manager      *Manager
	ctx          context.Context
	AppName      string
	AppVersion   string
	EventIDs     []string
	End          bool
}

func (s *exportK8sYamlController) Begin() {
	var r WutongExport
	r.ConfigGroups = make(map[string]map[string]string)
	exportApp := fmt.Sprintf("%v-%v", s.AppName, s.AppVersion)
	exportPath := fmt.Sprintf("/wtdata/app/k8s-yaml/%v/%v-yaml/%v", exportApp, exportApp, s.AppName)
	for _, service := range s.appService {
		err := s.exportOne(service, &r)
		if err != nil {
			logrus.Errorf("worker export %v failure %v", service.ServiceAlias, err)
		}
	}
	if s.End {
		err := s.write(path.Join(exportPath, "dependent_image.txt"), []byte(v1.GetOnlineProbeMeshImageName()), "\n")
		if err != nil {
			logrus.Errorf("write dependent_image.txt failure %v", err)
		}
		err = db.GetManager().ServiceEventDao().DeleteEvents(s.EventIDs)
		if err != nil {
			logrus.Errorf("delete event failure %v", err)
		}
	}
	s.manager.callback(s.controllerID, nil)
}

func (s *exportK8sYamlController) Stop() error {
	close(s.stopChan)
	return nil
}

func (s *exportK8sYamlController) exportOne(app v1.AppService, r *WutongExport) error {
	exportApp := fmt.Sprintf("%v-%v", s.AppName, s.AppVersion)
	exportPath := fmt.Sprintf("/wtdata/app/k8s-yaml/%v/%v-yaml/%v", exportApp, exportApp, s.AppName)
	logrus.Infof("start export app %s to k8s yaml spec", s.AppName)

	if len(app.GetManifests()) > 0 {
		for _, manifest := range app.GetManifests() {
			manifest.SetNamespace("")
			resourceBytes, err := yaml.Marshal(manifest)
			if err != nil {
				return fmt.Errorf("manifest to yaml failure %v", err)
			}
			err = s.write(path.Join(exportPath, fmt.Sprintf("%v.yaml", manifest.GetKind())), resourceBytes, "\n---\n")
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
			err = s.write(path.Join(exportPath, "ConfigMap.yaml"), cmBytes, "\n---\n")
			if err != nil {
				return fmt.Errorf("write configmap yaml failure %v", err)
			}
		}
	}

	for _, claim := range app.GetClaimsManually() {
		*claim.Spec.StorageClassName = ""
		claim.Kind = "PersistentVolumeClaim"
		claim.APIVersion = APIVersionPersistentVolumeClaim
		claim.Namespace = ""
		claim.Status = corev1.PersistentVolumeClaimStatus{}
		pvcBytes, err := yaml.Marshal(claim)
		if err != nil {
			return fmt.Errorf("pvc to yaml failure %v", err)
		}
		err = s.write(path.Join(exportPath, "PersistentVolumeClaim.yaml"), pvcBytes, "\n---\n")
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
		for i := range statefulset.Spec.VolumeClaimTemplates {
			*statefulset.Spec.VolumeClaimTemplates[i].Spec.StorageClassName = ""
		}
		statefulset.Status = appv1.StatefulSetStatus{}
		statefulsetBytes, err := yaml.Marshal(statefulset)
		if err != nil {
			return fmt.Errorf("statefulset to yaml failure %v", err)
		}
		err = s.write(path.Join(exportPath, "StatefulSet.yaml"), statefulsetBytes, "\n---\n")
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
		deployment.Status = appv1.DeploymentStatus{}
		deploymentBytes, err := yaml.Marshal(deployment)
		if err != nil {
			return fmt.Errorf("deployment to yaml failure %v", err)
		}
		err = s.write(path.Join(exportPath, "Deployment.yaml"), deploymentBytes, "\n---\n")
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
			err = s.write(path.Join(exportPath, "Service.yaml"), svcBytes, "\n---\n")
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
				err = s.write(path.Join(exportPath, "Secret.yaml"), secretBytes, "\n---\n")
				if err != nil {
					return fmt.Errorf("write secret yaml failure %v", err)
				}
			}
		}
	}
	if s.End {
		if secrets := app.GetEnvVarSecrets(true); secrets != nil {
			for _, secret := range secrets {
				if len(secret.ResourceVersion) == 0 {
					secret.APIVersion = APIVersionSecret
					secret.Namespace = ""
					secret.Type = ""
					secret.Kind = "Secret"

					secretBytes, err := yaml.Marshal(secret)
					if err != nil {
						return fmt.Errorf("configGroup secret to yaml failure %v", err)
					}
					err = s.write(path.Join(exportPath, "Secret.yaml"), secretBytes, "\n---\n")
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
				err = s.write(path.Join(exportPath, "HorizontalPodAutoscaler.yaml"), hpaBytes, "\n---\n")
				if err != nil {
					return fmt.Errorf("write hpa yaml failure %v", err)
				}
			}
		}
	}
	logrus.Infof("Create all app yaml file success, will waiting app export")
	return nil
}

func (s *exportK8sYamlController) write(k8sYamlFilePath string, meta []byte, endString string) error {
	var fl *os.File
	var err error
	if CheckFileExist(k8sYamlFilePath) {
		fl, err = os.OpenFile(k8sYamlFilePath, os.O_APPEND|os.O_WRONLY, 0755)
		if err != nil {
			return err
		}
	} else {
		fl, err = os.Create(k8sYamlFilePath)
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
