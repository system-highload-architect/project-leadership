package middleware

import (
	"fmt"
	"net/http"
	"sync/atomic"
)

var requestCount uint64

func Metrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddUint64(&requestCount, 1)
		fmt.Printf("[Metrics] Requests: %d\n", requestCount)
		next.ServeHTTP(w, r)
	})
}
