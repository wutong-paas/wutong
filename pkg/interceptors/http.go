package interceptors

import (
	"encoding/json"
	"net/http"

	"github.com/wutong-paas/wutong/pkg/component/cr"
	"github.com/wutong-paas/wutong/pkg/component/etcd"
	"github.com/wutong-paas/wutong/pkg/component/grpc"
	"github.com/wutong-paas/wutong/pkg/component/mq"
	"github.com/wutong-paas/wutong/pkg/component/prom"
)

// Recoverer -
func Recoverer(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rvr := recover(); rvr != nil && rvr != http.ErrAbortHandler {
				handleServiceUnavailable(w, r)
			}
		}()

		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}

// isNilPointerException Check if the panic is a nil pointer exception
// func isNilPointerException(p interface{}) bool {
// 	if p == nil {
// 		return false
// 	}

// 	errMsg := fmt.Sprintf("%v", p)
// 	return strings.Contains(errMsg, "runtime error: invalid memory address or nil pointer dereference") || strings.Contains(errMsg, "runtime error: slice bounds out of range")
// }

// handleServiceUnavailable -
func handleServiceUnavailable(w http.ResponseWriter, _ *http.Request) {
	// Additional information about why etcd service is not available
	errorMessage := "部分服务不可用"

	if etcd.Default().EtcdClient == nil {
		errorMessage = "etcd 服务不可用"
	} else if grpc.Default().StatusClient == nil {
		errorMessage = "worker 服务不可用"
	} else if cr.Default().RegistryCli == nil {
		errorMessage = "私有镜像仓库 服务不可用"
	} else if mq.Default().MqClient == nil {
		errorMessage = "mq 服务不可用"
	} else if prom.Default().PrometheusCli == nil {
		errorMessage = "monitor 服务不可用"
	}

	// Create a response JSON
	response := map[string]interface{}{
		"error": errorMessage,
	}

	// Convert the response to JSON
	responseJSON, err := json.Marshal(response)
	if err != nil {
		// Handle JSON marshaling error
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Set appropriate headers
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusServiceUnavailable)

	// Write the JSON response to the client
	_, _ = w.Write(responseJSON)
}
