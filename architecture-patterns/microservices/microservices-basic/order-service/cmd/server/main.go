package main

import (
	"log"
	"net/http"

	"order-service/internal/delivery/http"
	"order-service/internal/repository/inmemory"
	"order-service/internal/service"
)

func main() {
	// Инициализация зависимостей
	orderRepo := inmemory.NewOrderRepository()
	orderSvc := service.NewOrderService(orderRepo)
	orderHandler := http.NewOrderHandler(orderSvc)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /orders", orderHandler.CreateOrder)
	mux.HandleFunc("GET /orders", orderHandler.GetOrder)

	log.Println("Order Service starting on :8081")
	if err := http.ListenAndServe(":8081", mux); err != nil {
		log.Fatal(err)
	}
}
