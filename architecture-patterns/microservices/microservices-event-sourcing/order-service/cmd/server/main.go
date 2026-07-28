package main

import (
	"log"
	"net/http"

	"order-service/internal/command"
	delivery "order-service/internal/delivery/http"
	"order-service/internal/eventstore"
)

func main() {
	// Инициализация Event Store (in-memory)
	es := eventstore.NewEventStore()

	// Команды
	createOrderCmd := command.NewCreateOrderCommand(es)
	payOrderCmd := command.NewPayOrderCommand(es)

	// HTTP-хендлер
	handler := delivery.NewHandler(createOrderCmd, payOrderCmd)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /orders", handler.CreateOrder)
	mux.HandleFunc("POST /orders/{id}/pay", handler.PayOrder)
	mux.HandleFunc("GET /orders/{id}", handler.GetOrder)

	log.Println("Order Service (Event Sourcing) starting on :8081")
	if err := http.ListenAndServe(":8081", mux); err != nil {
		log.Fatal(err)
	}
}
