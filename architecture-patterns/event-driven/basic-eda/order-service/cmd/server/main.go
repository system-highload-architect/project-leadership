package main

import (
	"log"
	"net/http"

	delivery "order-service/internal/delivery/http"
	"order-service/internal/service"
	"shared/broker"
)

func main() {
	b := broker.GetBroker()
	svc := service.NewOrderService(b)
	handler := delivery.NewOrderHandler(svc)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /orders", handler.CreateOrder)

	log.Println("Order Service starting on :8081")
	if err := http.ListenAndServe(":8081", mux); err != nil {
		log.Fatal(err)
	}
}
