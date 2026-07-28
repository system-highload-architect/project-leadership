package main

import (
	"log"
	"net/http"

	"order-service/internal/command"
	"order-service/internal/delivery"
	"order-service/internal/query"
	readrepo "order-service/internal/repository/read/inmemory"
	writerepo "order-service/internal/repository/write/inmemory"
)

func main() {
	// Репозитории
	writeRepo := writerepo.NewWriteRepository()
	readRepo := readrepo.NewReadRepository()

	// Команды
	createOrderCmd := command.NewCreateOrderCommand(writeRepo, readRepo)
	updateStatusCmd := command.NewUpdateStatusCommand(writeRepo, readRepo)

	// Запросы
	getOrderQuery := query.NewGetOrderQuery(readRepo)
	listOrdersQuery := query.NewListOrdersQuery(readRepo)

	// HTTP-хендлер
	handler := delivery.NewHandler(createOrderCmd, updateStatusCmd, getOrderQuery, listOrdersQuery)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /orders", handler.CreateOrder)
	mux.HandleFunc("PUT /orders/{id}/status", handler.UpdateStatus)
	mux.HandleFunc("GET /orders/{id}", handler.GetOrder)
	mux.HandleFunc("GET /orders", handler.ListOrders)

	log.Println("Order Service (CQRS) starting on :8081")
	if err := http.ListenAndServe(":8081", mux); err != nil {
		log.Fatal(err)
	}
}
