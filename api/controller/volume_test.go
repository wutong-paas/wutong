package controller

import "testing"

func Test(t *testing.T) {
	testdata := []struct {
		id   string
		arg  string
		want bool
	}{
		{
			// correct yaml
			id: "correct-yaml",
			arg: `file1.log: |
  {
    "log": "log1"
  }
file2.log: |
  {
    "log": "log2"
  }`, want: true,
		},
		{
			// noformat plain
			id:  "noformat-plain",
			arg: `08d266ee-35c5-4b57-b7e4-ba720f29bdfd`, want: false,
		},
		{
			// json
			id: "json",
			arg: `{
  "log": "log1"
}`, want: false,
		},
		{
			// yaml
			id: "yaml2",
			arg: `name: test
friends:
- name: tom
  age: 20
- name: jerry
  age: 18`, want: true,
		},
	}

	for _, tt := range testdata {
		if got, _ := correctConfigMapContent(tt.arg); got != tt.want {
			t.Errorf("Test faile, id: %s, got %v, want %v", tt.id, got, tt.want)
		}
	}
}
