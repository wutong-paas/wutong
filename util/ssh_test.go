package util

import (
	"testing"
)

func TestGenerateRSASSHKeys(t *testing.T) {
	_, _, err := GenOpenSSHKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	t.Log("Generate SSH key pair successfully")
}
