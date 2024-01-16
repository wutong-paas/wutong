package model

import (
	"encoding/json"
	"testing"

	corev1 "k8s.io/api/core/v1"
)

func TestTryAppend(t *testing.T) {
	var taints TaintForSelectList

	taints = taints.TryAppend(corev1.Taint{
		Key:   "key0",
		Value: "",
	})

	taints = taints.TryAppend(corev1.Taint{
		Key:   "key1",
		Value: "val1",
	})

	taints = taints.TryAppend(corev1.Taint{
		Key:   "key1",
		Value: "val2",
	})

	for _, taint := range taints {
		t.Log("Key: ", taint.Key)
		for _, v := range taint.Values {
			t.Log("\tValue: ", v)
		}
	}
}

func TestJsonToleration(t *testing.T) {
	toleration := corev1.Toleration{
		Key:      "a",
		Value:    "b",
		Operator: corev1.TolerationOpEqual,
		Effect:   corev1.TaintEffectNoSchedule,
	}

	b, _ := json.Marshal(toleration)
	t.Log(string(b))
}
