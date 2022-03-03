// WUTONG, Application Management Platform
// Copyright (C) 2014-2021 Wutong Co., Ltd.

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

package helmapp

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	wutongv1alpha1 "github.com/wutong-paas/wutong/pkg/apis/wutong/v1alpha1"
	"github.com/wutong-paas/wutong/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("ControlLoop", func() {
	var namespace string
	var helmApp *wutongv1alpha1.HelmApp
	BeforeEach(func() {
		// create namespace
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: util.NewUUID(),
			},
		}
		namespace = ns.Name
		By("create namespace: " + namespace)
		_, err := kubeClient.CoreV1().Namespaces().Create(context.Background(), ns, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())

		helmApp = &wutongv1alpha1.HelmApp{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "phpmyadmin",
				Namespace: namespace,
				Labels: map[string]string{
					"app": "phpmyadmin",
				},
			},
			Spec: wutongv1alpha1.HelmAppSpec{
				EID:          "5bfba91b0ead72f612732535ef802217",
				TemplateName: "phpmyadmin",
				Version:      "8.2.0",
				AppStore: &wutongv1alpha1.HelmAppStore{
					Name: "bitnami",
					URL:  "https://charts.bitnami.com/bitnami",
				},
			},
		}
		By("create helm app: " + helmApp.Name)
		_, err = wutongClient.WutongV1alpha1().HelmApps(helmApp.Namespace).Create(context.Background(), helmApp, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		By("delete namespace: " + namespace)
		err := kubeClient.CoreV1().Namespaces().Delete(context.Background(), namespace, metav1.DeleteOptions{})
		Expect(err).NotTo(HaveOccurred())
	})

	Describe("Reconcile", func() {
		Context("HelmApp created", func() {
			It("should fulfill default values", func() {
				watch, err := wutongClient.WutongV1alpha1().HelmApps(helmApp.Namespace).Watch(context.Background(), metav1.ListOptions{
					LabelSelector: "app=phpmyadmin",
					Watch:         true,
				})
				Expect(err).NotTo(HaveOccurred())

				By("wait until the default values of the helm app were setup")
				for event := range watch.ResultChan() {
					newHelmApp := event.Object.(*wutongv1alpha1.HelmApp)
					// wait status
					for _, conditionType := range defaultConditionTypes {
						_, condition := newHelmApp.Status.GetCondition(conditionType)
						if condition == nil {
							break
						}
					}
					if newHelmApp.Status.Phase == "" {
						continue
					}

					// wait spec
					if newHelmApp.Spec.PreStatus == "" {
						continue
					}

					break
				}
			})

			It("should start detecting", func() {
				newHelmApp, err := wutongClient.WutongV1alpha1().HelmApps(helmApp.Namespace).Get(context.Background(), helmApp.Name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())

				Expect(newHelmApp.Status.Phase).NotTo(Equal(wutongv1alpha1.HelmAppStatusPhaseDetecting))

				By("wait until condition detecting conditions become true")
				watch, err := wutongClient.WutongV1alpha1().HelmApps(helmApp.Namespace).Watch(context.Background(), metav1.ListOptions{
					LabelSelector: "app=phpmyadmin",
					Watch:         true,
				})
				Expect(err).NotTo(HaveOccurred())

				conditionTypes := []wutongv1alpha1.HelmAppConditionType{
					wutongv1alpha1.HelmAppChartReady,
					wutongv1alpha1.HelmAppPreInstalled,
				}

				for event := range watch.ResultChan() {
					newHelmApp = event.Object.(*wutongv1alpha1.HelmApp)
					isFinished := true
					for _, conditionType := range conditionTypes {
						_, condition := newHelmApp.Status.GetCondition(conditionType)
						if condition == nil || condition.Status == corev1.ConditionFalse {
							isFinished = false
							break
						}
					}
					if isFinished {
						break
					}
				}
			})

			It("should start configuring", func() {
				By("wait until phase become configuring")
				err := waitUntilConfiguring(helmApp)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("Install HelmApp", func() {
			It("should ok", func() {
				err := waitUntilConfiguring(helmApp)
				Expect(err).NotTo(HaveOccurred())

				newHelmApp, err := wutongClient.WutongV1alpha1().HelmApps(helmApp.Namespace).Get(context.Background(), helmApp.Name, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())

				By("install helm app: " + helmApp.Name)
				newHelmApp.Spec.PreStatus = wutongv1alpha1.HelmAppPreStatusConfigured
				_, err = wutongClient.WutongV1alpha1().HelmApps(helmApp.Namespace).Update(context.Background(), newHelmApp, metav1.UpdateOptions{})
				Expect(err).NotTo(HaveOccurred())

				err = waitUntilInstalled(helmApp)
				Expect(err).NotTo(HaveOccurred())

				err = waitUntilDeployed(helmApp)
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})
})

func waitUntilConfiguring(helmApp *wutongv1alpha1.HelmApp) error {
	_, err := waitPhaseUntil(helmApp, wutongv1alpha1.HelmAppStatusPhaseConfiguring)
	return err
}

func waitUntilInstalled(helmApp *wutongv1alpha1.HelmApp) error {
	_, err := waitPhaseUntil(helmApp, wutongv1alpha1.HelmAppStatusPhaseInstalled)
	return err
}

func waitPhaseUntil(helmApp *wutongv1alpha1.HelmApp, phase wutongv1alpha1.HelmAppStatusPhase) (*wutongv1alpha1.HelmApp, error) {
	watch, err := wutongClient.WutongV1alpha1().HelmApps(helmApp.Namespace).Watch(context.Background(), metav1.ListOptions{
		LabelSelector: "app=phpmyadmin",
		Watch:         true,
	})
	if err != nil {
		return nil, err
	}

	// TODO: timeout
	for event := range watch.ResultChan() {
		newHelmApp := event.Object.(*wutongv1alpha1.HelmApp)
		if newHelmApp.Status.Phase == phase {
			return newHelmApp, nil
		}
	}

	return nil, nil
}

func waitUntilDeployed(helmApp *wutongv1alpha1.HelmApp) error {
	return waitStatusUntil(helmApp, wutongv1alpha1.HelmAppStatusDeployed)
}

func waitStatusUntil(helmApp *wutongv1alpha1.HelmApp, status wutongv1alpha1.HelmAppStatusStatus) error {
	watch, err := wutongClient.WutongV1alpha1().HelmApps(helmApp.Namespace).Watch(context.Background(), metav1.ListOptions{
		LabelSelector: "app=phpmyadmin",
		Watch:         true,
	})
	if err != nil {
		return err
	}

	// TODO: timeout
	for event := range watch.ResultChan() {
		newHelmApp := event.Object.(*wutongv1alpha1.HelmApp)
		if newHelmApp.Status.Status == status {
			return nil
		}
	}

	return nil
}
