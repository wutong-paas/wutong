package controller

import (
	"context"
	"fmt"
	"path"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cast"
	"github.com/wutong-paas/wutong/chaos"
	v1 "github.com/wutong-paas/wutong/worker/appm/types/v1"
	appv1 "k8s.io/api/apps/v1"
	"k8s.io/api/autoscaling/v2beta2"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	networkingv1beta1 "k8s.io/api/networking/v1beta1"
	k8sstrings "k8s.io/utils/strings"
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
	End          bool
}

func (s *exportK8sYamlController) Begin() {
	var r WutongExport
	r.ConfigGroups = make(map[string]map[string]string)
	exportApp := fmt.Sprintf("%v-%v", s.AppName, s.AppVersion)
	exportPath := fmt.Sprintf("/wtdata/app/yaml/%v/%v-yaml/%v", exportApp, exportApp, s.AppName)
	for _, service := range s.appService {
		err := s.exportOne(service, &r)
		if err != nil {
			logrus.Errorf("worker export %v failure %v", service.ServiceAlias, err)
		}
	}
	if s.End {
		err := write(path.Join(exportPath, "dependent_image.txt"), []byte(chaos.PROBEMESHIMAGENAME), "\n", false)
		if err != nil {
			logrus.Errorf("write dependent_image.txt failure %v", err)
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
	exportPath := fmt.Sprintf("/wtdata/app/yaml/%v/%v-yaml/%v", exportApp, exportApp, s.AppName)
	logrus.Infof("start export app %s to k8s yaml spec", s.AppName)

	if len(app.GetManifests()) > 0 {
		for _, manifest := range app.GetManifests() {
			manifest.SetNamespace("")
			resourceBytes, err := yaml.Marshal(manifest)
			if err != nil {
				return fmt.Errorf("manifest to yaml failure %v", err)
			}
			err = write(path.Join(exportPath, fmt.Sprintf("%v.yaml", manifest.GetKind())), resourceBytes, "\n---\n", true)
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
			err = write(path.Join(exportPath, "ConfigMap.yaml"), cmBytes, "\n---\n", true)
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
		err = write(path.Join(exportPath, "PersistentVolumeClaim.yaml"), pvcBytes, "\n---\n", true)
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
		err = write(path.Join(exportPath, "StatefulSet.yaml"), statefulsetBytes, "\n---\n", true)
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
		err = write(path.Join(exportPath, "Deployment.yaml"), deploymentBytes, "\n---\n", true)
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
			err = write(path.Join(exportPath, "Service.yaml"), svcBytes, "\n---\n", true)
			if err != nil {
				return fmt.Errorf("write svc yaml failure %v", err)
			}
		}
	}
	v1ingresses, v1beta1ingresses := app.GetIngress(true)
	for _, ing := range v1ingresses {
		ing.Kind = "Ingress"
		ing.Namespace = ""
		ing.APIVersion = APIVersionV1Ingress
		ing.Status = networkingv1.IngressStatus{}

		if len(ing.Spec.Rules) > 0 {
			var port = "http"
			if ing.Spec.DefaultBackend != nil && ing.Spec.DefaultBackend.Service != nil && ing.Spec.DefaultBackend.Service.Port.Number > 0 {
				port = cast.ToString(ing.Spec.DefaultBackend.Service.Port.Number)
			}
			ing.Name = fmt.Sprintf("%s-%s-%s", app.K8sComponentName, port, k8sstrings.ShortenString(ing.Name, 5))
		} else {
			continue
		}

		ingBytes, err := yaml.Marshal(ing)
		if err != nil {
			return fmt.Errorf("networking v1 ingress to yaml failure %v", err)
		}
		err = write(path.Join(exportPath, "Ingress.yaml"), ingBytes, "\n---\n", true)
		if err != nil {
			return fmt.Errorf("write networking v1 ingress yaml failure %v", err)
		}
	}
	for _, ing := range v1beta1ingresses {
		ing.Kind = "Ingress"
		ing.Namespace = ""
		ing.Name = app.K8sComponentName + "-" + k8sstrings.ShortenString(ing.Name, 5)
		ing.APIVersion = APIVersionV1beta1Ingress
		ing.Status = networkingv1beta1.IngressStatus{}

		if len(ing.Spec.Rules) > 0 {
			var port = "http"
			if ing.Spec.Backend != nil {
				port = ing.Spec.Backend.ServicePort.String()
			}
			ing.Name = fmt.Sprintf("%s-%s-%s", app.K8sComponentName, port, k8sstrings.ShortenString(ing.Name, 5))
		} else {
			continue
		}

		ingBytes, err := yaml.Marshal(ing)
		if err != nil {
			return fmt.Errorf("networking v1beta1 ingress to yaml failure %v", err)
		}
		err = write(path.Join(exportPath, "Ingress.yaml"), ingBytes, "\n---\n", true)
		if err != nil {
			return fmt.Errorf("write networking v1beta1 ingress yaml failure %v", err)
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
				err = write(path.Join(exportPath, "Secret.yaml"), secretBytes, "\n---\n", true)
				if err != nil {
					return fmt.Errorf("write secret yaml failure %v", err)
				}
			}
		}
	}
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
				err = write(path.Join(exportPath, "Secret-"+secret.Name+".yaml"), secretBytes, "\n---\n", false)
				if err != nil {
					return fmt.Errorf("configGroup write secret yaml failure %v", err)
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
				err = write(path.Join(exportPath, "HorizontalPodAutoscaler.yaml"), hpaBytes, "\n---\n", true)
				if err != nil {
					return fmt.Errorf("write hpa yaml failure %v", err)
				}
			}
		}
	}
	logrus.Infof("Create all app yaml file success, will waiting app export")
	return nil
}
