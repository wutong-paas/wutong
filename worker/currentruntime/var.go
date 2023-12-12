package currentruntime

import "os"

var (
	otelServerHost string
)

func GetOTELServerHost() string {
	if otelServerHost == "" {
		otelServerHost = os.Getenv("WT_OTEL_SERVER_HOST")
	}
	return otelServerHost
}
