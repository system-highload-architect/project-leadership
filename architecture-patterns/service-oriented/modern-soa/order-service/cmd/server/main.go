package main

import (
	"log"
	"net/http"

	delivery "order-service/internal/delivery/http"
	"order-service/internal/service"
)

func main() {
	svc := service.NewOrderService()
	handler := delivery.NewOrderHandler(svc)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /orders", handler.CreateOrder)
	mux.HandleFunc("GET /orders/{id}", handler.GetOrder)

	log.Println("Order Service starting on :8081")
	if err := http.ListenAndServe(":8081", mux); err != nil {
		log.Fatal(err)
	}
}
