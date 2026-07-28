package main

import (
	"log"

	"payment-service/internal/service"
	"shared/broker"
)

func main() {
	b := broker.GetBroker()
	svc := service.NewPaymentService(b)

	ch := b.Subscribe("OrderCreated")
	go func() {
		for event := range ch {
			svc.HandleOrderCreated(event.Data)
		}
	}()

	log.Println("Payment Service started (listening for events)")
	select {}
}
