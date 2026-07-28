package main

import (
	"log"
	"net/http"

	"sidecar/internal/middleware"
	"sidecar/internal/proxy"
)

func main() {
	// Создаём прокси к основному сервису (localhost:8081)
	proxyHandler := proxy.NewProxy("http://localhost:8081")

	// Оборачиваем в middleware (логирование, метрики)
	handler := middleware.Logging(
		middleware.Metrics(proxyHandler),
	)

	log.Println("Sidecar starting on :8080 (proxying to http://localhost:8081)")
	if err := http.ListenAndServe(":8080", handler); err != nil {
		log.Fatal(err)
	}
}
