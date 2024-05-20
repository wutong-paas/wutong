package handler

import (
	"testing"

	"github.com/wutong-paas/wutong/pkg/prometheus"
)

func TestGetDiskUsage(t *testing.T) {
	prometheusCli, err := prometheus.NewPrometheus(&prometheus.Options{
		Endpoint: "9999.wt5d40c8.2c9v614j.a24839.wtapps.cn",
	})
	if err != nil {
		t.Fatal(err)
	}

	a := ApplicationAction{
		promClient: prometheusCli,
	}

	a.getDiskUsage("4ad713694c934829950f85a26f7c7544")
}
