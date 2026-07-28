package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"math/rand"
)

func main() {
	// Целевые сервисы
	v1, _ := url.Parse("http://localhost:8081")
	v2, _ := url.Parse("http://localhost:8082")

	proxyV1 := httputil.NewSingleHostReverseProxy(v1)
	proxyV2 := httputil.NewSingleHostReverseProxy(v2)

	// Маршрутизация с балансировкой (50/50)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if rand.Intn(2) == 0 {
			proxyV1.ServeHTTP(w, r)
		} else {
			proxyV2.ServeHTTP(w, r)
		}
	})

	log.Println("Sidecar (Service Mesh) starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
