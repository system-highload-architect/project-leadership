package main

import (
	"log"
	"net/http"

	delivery "order-service/internal/delivery/http"
	"order-service/internal/saga"
)

func main() {
	// Инициализация саги
	orchestrator := saga.NewOrchestrator()

	// HTTP-хендлер
	handler := delivery.NewHandler(orchestrator)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /orders", handler.CreateOrder)

	log.Println("Order Service (Saga) starting on :8081")
	if err := http.ListenAndServe(":8081", mux); err != nil {
		log.Fatal(err)
	}
}
