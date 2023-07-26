package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strings"

	k8sstrings "k8s.io/utils/strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cast"
	"github.com/wutong-paas/wutong/chaos"
	v1 "github.com/wutong-paas/wutong/worker/appm/types/v1"
	appv1 "k8s.io/api/apps/v1"
	"k8s.io/api/autoscaling/v2beta2"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	networkingv1beta1 "k8s.io/api/networking/v1beta1"
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

func newWutongExport() *WutongExport {
	return &WutongExport{
		ConfigGroups:    make(map[string]map[string]string),
		ExternalDomains: make(map[string]map[string]string),
		StorageClass:    "wutongvolumerwx",
	}
}

func (s *exportHelmChartController) Begin() {
	exportApp := fmt.Sprintf("%v-%v", s.AppName, s.AppVersion)
	exportPath := fmt.Sprintf("/wtdata/app/helm_chart/%v/%v-helm/%v", exportApp, exportApp, s.AppName)
	r := s.readHelmValuesFromFileOrInit(path.Join(exportPath, "values.yaml"))

	for _, service := range s.appService {
		err := s.exportOne(service, r)
		if err != nil {
			logrus.Errorf("worker export %v failure %v", service.ServiceAlias, err)
		}
	}

	// Write values.yaml
	valuesYamlByte, err := yaml.Marshal(r)
	if err != nil {
		logrus.Errorf("yaml marshal valueYaml failure %v", err)
	}
	err = write(path.Join(exportPath, "values.yaml"), valuesYamlByte, "\n", false)
	if err != nil {
		logrus.Errorf("write values.yaml other failure %v", err)
	}

	if s.End {
		err = write(path.Join(exportPath, "dependent_image.txt"), []byte(chaos.PROBEMESHIMAGENAME), "\n", false)
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
	exportPath := fmt.Sprintf("/wtdata/app/helm_chart/%v/%v-helm/%v", exportApp, exportApp, s.AppName)
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
			err = write(path.Join(exportTemplatePath, fmt.Sprintf("%v.yaml", manifest.GetKind())), resourceBytes, "\n---\n", true)
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
			err = write(path.Join(exportTemplatePath, "ConfigMap.yaml"), cmBytes, "\n---\n", true)
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
		err = write(path.Join(exportTemplatePath, "PersistentVolumeClaim.yaml"), pvcBytes, "\n---\n", true)
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
			statefulset.Spec.VolumeClaimTemplates[i].Namespace = ""
		}
		statefulset.Status = appv1.StatefulSetStatus{}
		statefulsetBytes, err := yaml.Marshal(statefulset)
		if err != nil {
			return fmt.Errorf("statefulset to yaml failure %v", err)
		}
		err = write(path.Join(exportTemplatePath, "StatefulSet.yaml"), statefulsetBytes, "\n---\n", true)
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
		err = write(path.Join(exportTemplatePath, "Deployment.yaml"), deploymentBytes, "\n---\n", true)
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
			err = write(path.Join(exportTemplatePath, "Service.yaml"), svcBytes, "\n---\n", true)
			if err != nil {
				return fmt.Errorf("write svc yaml failure %v", err)
			}
		}
	}
	v1ingresses, v1beta1ingresses := app.GetIngress(true)
	domains := make(map[string]string)
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
			domains[ing.Name] = ing.Spec.Rules[0].Host
			ing.Spec.Rules[0].Host = fmt.Sprintf("{{ $host := index .Values \"externalDomains\" \"%s\" \"%s\" }}{{ default \"%v\" $host }}", app.K8sComponentName, ing.Name, ing.Spec.Rules[0].Host)
		} else {
			continue
		}

		ingBytes, err := yaml.Marshal(ing)
		if err != nil {
			return fmt.Errorf("networking v1 ingress to yaml failure %v", err)
		}
		err = write(path.Join(exportTemplatePath, "Ingress.yaml"), ingBytes, "\n---\n", true)
		if err != nil {
			return fmt.Errorf("write networking v1 ingress yaml failure %v", err)
		}
	}

	for _, ing := range v1beta1ingresses {
		ing.Kind = "Ingress"
		ing.Namespace = ""
		ing.APIVersion = APIVersionV1beta1Ingress
		ing.Status = networkingv1beta1.IngressStatus{}

		if len(ing.Spec.Rules) > 0 {
			var port = "http"
			if ing.Spec.Backend != nil {
				port = ing.Spec.Backend.ServicePort.String()
			}
			ing.Name = fmt.Sprintf("%s-%s-%s", app.K8sComponentName, port, k8sstrings.ShortenString(ing.Name, 5))
			domains[ing.Name] = ing.Spec.Rules[0].Host
			ing.Spec.Rules[0].Host = fmt.Sprintf("{{ $host := index .Values \"externalDomains\" \"%s\" \"%s\" }}{{ default \"%v\" $host }}", app.K8sComponentName, ing.Name, ing.Spec.Rules[0].Host)
		} else {
			continue
		}

		ingBytes, err := yaml.Marshal(ing)
		if err != nil {
			return fmt.Errorf("networking v1beta1 ingress to yaml failure %v", err)
		}
		err = write(path.Join(exportTemplatePath, "Ingress.yaml"), ingBytes, "\n---\n", true)
		if err != nil {
			return fmt.Errorf("write networking v1beta1 ingress yaml failure %v", err)
		}
	}
	if len(domains) > 0 {
		r.ExternalDomains[app.K8sComponentName] = domains
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
				err = write(path.Join(exportTemplatePath, "Secret.yaml"), secretBytes, "\n---\n", true)
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
				templateConfigGroup := make(map[string]string)
				for key, value := range data {
					templateConfigGroup[key] = string(value)
				}
				r.ConfigGroups[secret.Name] = templateConfigGroup
				dataTemplate := fmt.Sprintf("  {{ $secret := index .Values \"secretEnvs\" \"%s\"}}\n  {{- range $key, $val := $secret }}\n  {{ $key }}: {{ $val | quote}}\n  {{- end }}", secret.Name)
				ySecret.StringData = dataTemplate
				secretBytes, err := yaml.Marshal(ySecret)
				if err != nil {
					return fmt.Errorf("configGroup secret to yaml failure %v", err)
				}
				secretStr := strings.Replace(string(secretBytes), "|2-", "", 1)
				err = write(path.Join(exportTemplatePath, "Secret-"+secret.Name+".yaml"), []byte(secretStr), "\n---\n", false)
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
				err = write(path.Join(exportTemplatePath, "HorizontalPodAutoscaler.yaml"), hpaBytes, "\n---\n", true)
				if err != nil {
					return fmt.Errorf("write hpa yaml failure %v", err)
				}
			}
		}
	}
	logrus.Infof("Create all app yaml file success, will waiting app export")
	return nil
}

func (s *exportHelmChartController) readHelmValuesFromFileOrInit(helmChartFilePath string) *WutongExport {
	if CheckFileExist(helmChartFilePath) {
		bytes, err := os.ReadFile(helmChartFilePath)
		if err != nil {
			logrus.Errorf("read helm values file error: %v", err)
			return newWutongExport()
		}
		var wutongExport WutongExport
		err = yaml.Unmarshal(bytes, &wutongExport)
		if err != nil {
			logrus.Errorf("unmarshal helm values error: %v", err)
			return newWutongExport()
		}
		return &wutongExport
	}

	return newWutongExport()
}
