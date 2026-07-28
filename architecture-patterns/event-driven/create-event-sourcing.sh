#!/bin/bash
# Event Sourcing (простой пример без HTTP)

set -e

PROJECT_NAME="event-sourcing"

mkdir -p "$PROJECT_NAME/cmd/server"
mkdir -p "$PROJECT_NAME/internal/domain"
mkdir -p "$PROJECT_NAME/internal/event"
mkdir -p "$PROJECT_NAME/internal/eventstore"

cd "$PROJECT_NAME"

cat > go.mod <<'EOF'
module event-sourcing

go 1.23
EOF

cat > cmd/server/main.go <<'EOF'
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
EOF

cat > internal/domain/order.go <<'EOF'
package domain

import (
	"time"

	"event-sourcing/internal/event"
)

type OrderStatus string

const (
	OrderStatusNew  OrderStatus = "new"
	OrderStatusPaid OrderStatus = "paid"
)

type Order struct {
	ID         string
	CustomerID string
	Items      []event.OrderItem
	Status     OrderStatus
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func (o *Order) ApplyEvent(e event.Event) {
	switch ev := e.(type) {
	case *event.OrderCreated:
		o.ID = ev.AggregateID
		o.CustomerID = ev.CustomerID
		o.Items = ev.Items
		o.Status = OrderStatusNew
		o.CreatedAt = ev.Timestamp
		o.UpdatedAt = ev.Timestamp
	case *event.OrderPaid:
		o.Status = OrderStatusPaid
		o.UpdatedAt = ev.Timestamp
	}
}
EOF

cat > internal/event/events.go <<'EOF'
package event

import "time"

type Event interface {
	GetAggregateID() string
	GetTimestamp() time.Time
	GetType() string
}

type BaseEvent struct {
	AggregateID string
	Timestamp   time.Time
}

func (e BaseEvent) GetAggregateID() string { return e.AggregateID }
func (e BaseEvent) GetTimestamp() time.Time { return e.Timestamp }

type OrderCreated struct {
	BaseEvent
	CustomerID string
	Items      []OrderItem
}

func (e *OrderCreated) GetType() string { return "OrderCreated" }

type OrderItem struct {
	ProductID string
	Quantity  int
	Price     float64
}

type OrderPaid struct {
	BaseEvent
}

func (e *OrderPaid) GetType() string { return "OrderPaid" }
EOF

cat > internal/eventstore/eventstore.go <<'EOF'
package eventstore

import (
	"errors"
	"sync"

	"event-sourcing/internal/event"
)

type EventStore struct {
	mu     sync.RWMutex
	events map[string][]event.Event
}

func NewEventStore() *EventStore {
	return &EventStore{
		events: make(map[string][]event.Event),
	}
}

func (es *EventStore) Save(aggregateID string, events ...event.Event) error {
	es.mu.Lock()
	defer es.mu.Unlock()
	es.events[aggregateID] = append(es.events[aggregateID], events...)
	return nil
}

func (es *EventStore) Load(aggregateID string) ([]event.Event, error) {
	es.mu.RLock()
	defer es.mu.RUnlock()
	events, ok := es.events[aggregateID]
	if !ok {
		return nil, errors.New("no events found")
	}
	return events, nil
}
EOF

echo "✅ Event Sourcing project created at ./$PROJECT_NAME"