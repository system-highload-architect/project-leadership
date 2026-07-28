package main

import (
	"log"
	"net/http"

	"api-gateway/internal/middleware"
	"api-gateway/internal/proxy"
)

func main() {
	// Создаём прокси для сервисов
	orderProxy := proxy.NewProxy("http://localhost:8081")
	paymentProxy := proxy.NewProxy("http://localhost:8082")

	mux := http.NewServeMux()

	// Маршруты с middleware
	mux.HandleFunc("/orders/", middleware.Chain(
		orderProxy.ServeHTTP,
		middleware.Logging,
		middleware.Auth,
	))

	mux.HandleFunc("/payments/", middleware.Chain(
		paymentProxy.ServeHTTP,
		middleware.Logging,
		middleware.Auth,
	))

	log.Println("API Gateway starting on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}
