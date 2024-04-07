package util

import "testing"

func TestExternalIP(t *testing.T) {
	ip, err := ExternalIP()
	if err != nil {
		t.Errorf("TestExternalIP failed, err: %v", err)
	}
	t.Log(ip.String())
}
