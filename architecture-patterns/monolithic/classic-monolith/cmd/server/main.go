package main

import (
	"log"
	"net/http"

	"classic-monolith/internal/bootstrap"
)

func main() {
	container := bootstrap.NewContainer()

	mux := http.NewServeMux()
	mux.HandleFunc("POST /orders", container.OrderHandler.CreateOrder)
	mux.HandleFunc("GET /orders", container.OrderHandler.GetOrder)
	mux.HandleFunc("POST /products", container.ProductHandler.AddProduct)
	mux.HandleFunc("GET /products", container.ProductHandler.GetProduct)

	log.Println("Classic monolith starting on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}
