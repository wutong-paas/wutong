package handler

import (
	"testing"
)

func TestValidatePassword(t *testing.T) {
	testdata := []struct {
		password string
		want     bool
	}{
		{
			password: "123456",
			want:     false,
		},
		{
			password: "12345678",
			want:     false,
		},
		{
			password: "abcdefgh",
			want:     false,
		},
		{
			password: "ABCDEFGH",
			want:     false,
		},
		{
			password: "Abcabc123",
			want:     true,
		}, {
			password: "Abab12@>",
			want:     true,
		},
	}

	for _, v := range testdata {
		got, err := validatePassword(v.password)
		if got != v.want {
			t.Errorf("ValidatePassword(%v) = %v, want %v", v.password, got, v.want)
		}
		if err != nil {
			t.Logf("ValidatePassword(%v) = %v, want %v, err: %v", v.password, got, v.want, err)
		}
	}
}
