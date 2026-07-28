package main

import (
	"fmt"
	"time"

	"event-sourcing/internal/domain"
	"event-sourcing/internal/event"
	"event-sourcing/internal/eventstore"
)

func main() {
	es := eventstore.NewEventStore()
	orderID := "ord-123"

	// 1. Создаём заказ
	createEvt := &event.OrderCreated{
		BaseEvent: event.BaseEvent{
			AggregateID: orderID,
			Timestamp:   time.Now(),
		},
		CustomerID: "cust-1",
		Items: []event.OrderItem{
			{ProductID: "prod-1", Quantity: 2, Price: 100},
			{ProductID: "prod-2", Quantity: 1, Price: 50},
		},
	}
	_ = es.Save(orderID, createEvt)

	// 2. Оплачиваем заказ
	payEvt := &event.OrderPaid{
		BaseEvent: event.BaseEvent{
			AggregateID: orderID,
			Timestamp:   time.Now(),
		},
	}
	_ = es.Save(orderID, payEvt)

	// 3. Восстанавливаем агрегат
	events, _ := es.Load(orderID)
	order := &domain.Order{}
	for _, e := range events {
		order.ApplyEvent(e)
	}

	// 4. Выводим состояние и историю
	fmt.Printf("Order: ID=%s, Status=%s, Customer=%s\n", order.ID, order.Status, order.CustomerID)
	fmt.Println("Items:")
	for _, item := range order.Items {
		fmt.Printf("  - %s x%d = %.2f\n", item.ProductID, item.Quantity, item.Price*float64(item.Quantity))
	}
	fmt.Println("\nEvent history:")
	for i, e := range events {
		fmt.Printf("  %d. %s at %s\n", i+1, e.GetType(), e.GetTimestamp().Format("15:04:05"))
	}
}
