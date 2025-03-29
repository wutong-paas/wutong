package kube

import "testing"

func TestVersion(t *testing.T) {
	v := Version()
	t.Log(v)

	te := VersionGTE(1, 23)
	t.Log(te)

	te = VersionGTE(1, 22)
	t.Log(te)
}
