package util

import (
	"net"
	"testing"
)

func TestGenerateMACAddress(t *testing.T) {
	for i := 0; i < 10; i++ {
		macaddr := GenerateMACAddress()
		t.Log(macaddr)
		if _, err := net.ParseMAC(macaddr); err != nil {
			t.Errorf("GenerateMACAddress() = %v, want a valid MAC address", macaddr)
		}
	}
}
