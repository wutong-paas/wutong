package helm

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wutong-paas/wutong/util"
)

func TestRepoAdd(t *testing.T) {
	repo := NewRepo(
		"/tmp/helm/repoName/repositories.yaml",
		"/tmp/helm/cache")
	err := repo.Add(util.NewUUID(), "https://openchart.wutong-paas.com/wutong-paas/wutong", "", "")
	assert.Nil(t, err)
}
