package main

import (
	"log"
	"net/http"

	"payment-service/internal/delivery/http"
	"payment-service/internal/repository/inmemory"
	"payment-service/internal/service"
)

func main() {
	paymentRepo := inmemory.NewPaymentRepository()
	paymentSvc := service.NewPaymentService(paymentRepo)
	paymentHandler := http.NewPaymentHandler(paymentSvc)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /payments", paymentHandler.ProcessPayment)

	log.Println("Payment Service starting on :8082")
	if err := http.ListenAndServe(":8082", mux); err != nil {
		log.Fatal(err)
	}
}
