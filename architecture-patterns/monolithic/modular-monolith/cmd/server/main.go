package main

import (
	"log"
	"net/http"

	"modular-monolith/internal/infrastructure/db"
	invApi "modular-monolith/internal/modules/inventory/api"
	invRepo "modular-monolith/internal/modules/inventory/repository"
	invSvc "modular-monolith/internal/modules/inventory/service"
	orderApi "modular-monolith/internal/modules/order/api"
	orderRepo "modular-monolith/internal/modules/order/repository"
	orderSvc "modular-monolith/internal/modules/order/service"
)

func main() {
	database := db.NewInMemoryDB()

	// Inventory module
	invRepo := invRepo.NewInventoryRepository(database)
	invService := invSvc.NewInventoryService(invRepo)
	invHandler := invApi.NewInventoryHandler(invService)

	// Order module (depends on Inventory)
	orderRepo := orderRepo.NewOrderRepository(database)
	orderService := orderSvc.NewOrderService(orderRepo)
	orderHandler := orderApi.NewOrderHandler(orderService, invService)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /products", invHandler.AddProduct)
	mux.HandleFunc("GET /products", invHandler.GetProduct)
	mux.HandleFunc("POST /orders", orderHandler.CreateOrder)
	mux.HandleFunc("GET /orders", orderHandler.GetOrder)

	log.Println("Modular monolith server starting on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}
