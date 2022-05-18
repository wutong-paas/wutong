// WUTONG, Application Management Platform
// Copyright (C) 2014-2017 Wutong Co., Ltd.

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

package v1

import (
	"encoding/json"
	"fmt"
	"testing"

	corev1 "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "k8s.io/api/apps/v1"
)

func TestGetStatefulsetModifiedConfiguration(t *testing.T) {
	var replicas int32 = 1
	var replicasnew int32 = 2
	bytes, err := getStatefulsetModifiedConfiguration(&v1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name: "teststatefulset",
			Labels: map[string]string{
				"version": "1",
			},
		},
		Spec: v1.StatefulSetSpec{
			Replicas: &replicas,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					NodeName: "v1",
					NodeSelector: map[string]string{
						"test": "1111",
					},
					InitContainers: []corev1.Container{
						{
							Name:  "busybox",
							Image: "busybox",
						},
					},
					Containers: []corev1.Container{
						{
							Image: "nginx",
							Name:  "nginx1",
							Env: []corev1.EnvVar{
								{
									Name:  "version",
									Value: "V1",
								},
								{
									Name:  "delete",
									Value: "true",
								},
							},
						},
						{
							Image: "nginx",
							Name:  "nginx2",
						},
					},
				},
			},
		},
	}, &v1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name: "teststatefulset",
			Labels: map[string]string{
				"version": "2",
			},
		},
		Spec: v1.StatefulSetSpec{
			Replicas: &replicasnew,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					NodeName: "v2",
					NodeSelector: map[string]string{
						"test": "1111",
					},
					Containers: []corev1.Container{
						{
							Image: "nginx",
							Name:  "nginx1",
							Env: []corev1.EnvVar{
								{
									Name:  "version",
									Value: "V2",
								},
							},
						},
						{
							Image: "nginx",
							Name:  "nginx3",
						},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("here")
	fmt.Println(string(bytes))
	// t.Log("here")
	// t.Log(string(bytes))
}

type A struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Spec  Spec   `json:"spec"`
	Spec2 Spec   `json:"spec2"`
}
type Spec struct {
	Replicas int    `json:"replicas"`
	InitC    []Cont `json:"initC,omitempty"`
	C        []Cont `json:"c"`
}

type Cont struct {
	Name  string `json:"name"`
	Image string `json:"image"`
}

type AliasA A

func TestGetchange(t *testing.T) {
	bytes, err := getchange(&A{
		ID:   "1",
		Name: "test",
		Spec: Spec{
			Replicas: 1,
			InitC: []Cont{
				{
					Name:  "nginx0",
					Image: "nginx0",
				},
			},
			C: []Cont{
				{
					Name:  "nginx",
					Image: "nginx1",
				}, {
					Name:  "nginx2",
					Image: "nginx2",
				},
			},
		},
		Spec2: Spec{
			Replicas: 1,
		},
	}, &A{
		ID:   "1",
		Name: "test2",
		Spec: Spec{
			C: []Cont{
				{
					// Name:  "nginx",
					Image: "nginx10",
				},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(string(bytes))
}

func TestJson(t *testing.T) {
	bytes, _ := json.Marshal(&A{
		ID:   "1",
		Name: "test",
		Spec: Spec{
			Replicas: 1,
			InitC:    []Cont{},
			C: []Cont{
				{
					Name:  "nginx",
					Image: "nginx1",
				},
			},
		},
	})
	fmt.Println(string(bytes))
}
