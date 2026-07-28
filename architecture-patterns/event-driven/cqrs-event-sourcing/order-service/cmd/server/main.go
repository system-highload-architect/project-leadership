package main

import (
	"log"
	"net/http"

	"order-service/internal/command"
	delivery "order-service/internal/delivery/http"
	"order-service/internal/eventstore"
	"order-service/internal/projection"
	"order-service/internal/query"
	readrepo "order-service/internal/repository/read/inmemory"
)

func main() {
	// Event Store (write side)
	es := eventstore.NewEventStore()

	// Read repository (query side)
	readRepo := readrepo.NewReadRepository()

	// Projection (обновляет read-модель из событий)
	proj := projection.NewProjection(readRepo)

	// Подписываемся на события (в реальном проекте через шину, здесь напрямую)
	// Мы будем вызывать проекцию вручную после сохранения события (для простоты)
	// В реальном проекте это делается асинхронно через Kafka.

	// Команды
	createOrderCmd := command.NewCreateOrderCommand(es, proj)
	payOrderCmd := command.NewPayOrderCommand(es, proj)

	// Запросы
	getOrderQuery := query.NewGetOrderQuery(readRepo)
	listOrdersQuery := query.NewListOrdersQuery(readRepo)

	// HTTP-хендлер
	handler := delivery.NewHandler(createOrderCmd, payOrderCmd, getOrderQuery, listOrdersQuery)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /orders", handler.CreateOrder)
	mux.HandleFunc("POST /orders/{id}/pay", handler.PayOrder)
	mux.HandleFunc("GET /orders/{id}", handler.GetOrder)
	mux.HandleFunc("GET /orders", handler.ListOrders)

	log.Println("Order Service (CQRS + Event Sourcing) starting on :8081")
	if err := http.ListenAndServe(":8081", mux); err != nil {
		log.Fatal(err)
	}
}
