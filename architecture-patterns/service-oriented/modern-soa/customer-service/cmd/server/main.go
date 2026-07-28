package main

import (
	"log"
	"net/http"

	delivery "customer-service/internal/delivery/http"
	"customer-service/internal/service"
)

func main() {
	svc := service.NewCustomerService()
	handler := delivery.NewCustomerHandler(svc)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /customers/{id}", handler.GetCustomer)

	log.Println("Customer Service starting on :8082")
	if err := http.ListenAndServe(":8082", mux); err != nil {
		log.Fatal(err)
	}
}
