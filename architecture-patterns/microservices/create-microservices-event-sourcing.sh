#!/bin/bash
# Микросервис с Event Sourcing

set -e

ROOT_DIR="microservices-event-sourcing"
SERVICE="order-service"

mkdir -p "$ROOT_DIR/$SERVICE/cmd/server"
mkdir -p "$ROOT_DIR/$SERVICE/internal/domain"
mkdir -p "$ROOT_DIR/$SERVICE/internal/event"
mkdir -p "$ROOT_DIR/$SERVICE/internal/eventstore"
mkdir -p "$ROOT_DIR/$SERVICE/internal/command"
mkdir -p "$ROOT_DIR/$SERVICE/internal/delivery/http"

cd "$ROOT_DIR/$SERVICE"

cat > go.mod <<EOF
module $SERVICE

go 1.23
EOF

cat > cmd/server/main.go <<'EOF'
package main

import (
	"log"
	"net/http"

	"order-service/internal/command"
	"order-service/internal/delivery/http"
	"order-service/internal/eventstore"
)

func main() {
	// Инициализация Event Store (in-memory)
	es := eventstore.NewEventStore()

	// Команды
	createOrderCmd := command.NewCreateOrderCommand(es)
	payOrderCmd := command.NewPayOrderCommand(es)

	// HTTP-хендлер
	handler := http.NewHandler(createOrderCmd, payOrderCmd)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /orders", handler.CreateOrder)
	mux.HandleFunc("POST /orders/{id}/pay", handler.PayOrder)
	mux.HandleFunc("GET /orders/{id}", handler.GetOrder)

	log.Println("Order Service (Event Sourcing) starting on :8081")
	if err := http.ListenAndServe(":8081", mux); err != nil {
		log.Fatal(err)
	}
}
EOF

cat > internal/domain/order.go <<'EOF'
package domain

import (
	"errors"
	"time"
)

type OrderStatus string

const (
	OrderStatusNew      OrderStatus = "new"
	OrderStatusPaid     OrderStatus = "paid"
	OrderStatusShipped  OrderStatus = "shipped"
	OrderStatusComplete OrderStatus = "complete"
	OrderStatusCanceled OrderStatus = "canceled"
)

// Order — агрегат
type Order struct {
	ID         string
	CustomerID string
	Items      []OrderItem
	Status     OrderStatus
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type OrderItem struct {
	ProductID string
	Quantity  int
	Price     float64
}

// NewOrder создаёт новый заказ (без событий)
func NewOrder(id, customerID string, items []OrderItem) *Order {
	return &Order{
		ID:         id,
		CustomerID: customerID,
		Items:      items,
		Status:     OrderStatusNew,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
}

// ApplyEvent применяет событие к агрегату
func (o *Order) ApplyEvent(event interface{}) {
	switch e := event.(type) {
	case *OrderCreated:
		o.ID = e.AggregateID
		o.CustomerID = e.CustomerID
		o.Items = e.Items
		o.Status = OrderStatusNew
		o.CreatedAt = e.Timestamp
		o.UpdatedAt = e.Timestamp
	case *OrderPaid:
		o.Status = OrderStatusPaid
		o.UpdatedAt = e.Timestamp
	}
}
EOF

cat > internal/event/events.go <<'EOF'
package event

import "time"

type Event interface {
	GetAggregateID() string
	GetTimestamp() time.Time
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

type OrderItem struct {
	ProductID string
	Quantity  int
	Price     float64
}

type OrderPaid struct {
	BaseEvent
}
EOF

cat > internal/eventstore/eventstore.go <<'EOF'
package eventstore

import (
	"errors"
	"sync"

	"order-service/internal/event"
)

type EventStore struct {
	mu     sync.RWMutex
	events map[string][]event.Event // aggregateID → список событий
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
		return nil, errors.New("no events found for aggregate")
	}
	return events, nil
}
EOF

cat > internal/command/create_order.go <<'EOF'
package command

import (
	"errors"
	"time"

	"order-service/internal/domain"
	"order-service/internal/event"
	"order-service/internal/eventstore"
)

type CreateOrderCommand struct {
	es *eventstore.EventStore
}

func NewCreateOrderCommand(es *eventstore.EventStore) *CreateOrderCommand {
	return &CreateOrderCommand{es: es}
}

type CreateOrderRequest struct {
	CustomerID string             `json:"customer_id"`
	Items      []domain.OrderItem `json:"items"`
}

func (c *CreateOrderCommand) Execute(req CreateOrderRequest) (domain.Order, error) {
	if req.CustomerID == "" {
		return domain.Order{}, errors.New("customer id is required")
	}
	if len(req.Items) == 0 {
		return domain.Order{}, errors.New("order must have at least one item")
	}

	orderID := "ord-" + time.Now().Format("20060102150405")

	// Создаём событие
	evt := &event.OrderCreated{
		BaseEvent: event.BaseEvent{
			AggregateID: orderID,
			Timestamp:   time.Now(),
		},
		CustomerID: req.CustomerID,
		Items:      make([]event.OrderItem, len(req.Items)),
	}
	for i, item := range req.Items {
		evt.Items[i] = event.OrderItem{
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
			Price:     item.Price,
		}
	}

	// Сохраняем событие
	if err := c.es.Save(orderID, evt); err != nil {
		return domain.Order{}, err
	}

	// Восстанавливаем агрегат (для возврата)
	order := domain.NewOrder(orderID, req.CustomerID, req.Items)
	return *order, nil
}
EOF

cat > internal/command/pay_order.go <<'EOF'
package command

import (
	"errors"
	"time"

	"order-service/internal/domain"
	"order-service/internal/event"
	"order-service/internal/eventstore"
)

type PayOrderCommand struct {
	es *eventstore.EventStore
}

func NewPayOrderCommand(es *eventstore.EventStore) *PayOrderCommand {
	return &PayOrderCommand{es: es}
}

type PayOrderRequest struct {
	OrderID string `json:"order_id"`
}

func (c *PayOrderCommand) Execute(req PayOrderRequest) (domain.Order, error) {
	if req.OrderID == "" {
		return domain.Order{}, errors.New("order id is required")
	}

	// Загружаем события
	events, err := c.es.Load(req.OrderID)
	if err != nil {
		return domain.Order{}, err
	}

	// Восстанавливаем агрегат
	order := &domain.Order{}
	for _, e := range events {
		order.ApplyEvent(e)
	}

	// Проверяем, можно ли оплатить
	if order.Status != domain.OrderStatusNew {
		return domain.Order{}, errors.New("order cannot be paid in current status")
	}

	// Создаём событие оплаты
	evt := &event.OrderPaid{
		BaseEvent: event.BaseEvent{
			AggregateID: req.OrderID,
			Timestamp:   time.Now(),
		},
	}

	// Сохраняем событие
	if err := c.es.Save(req.OrderID, evt); err != nil {
		return domain.Order{}, err
	}

	// Обновляем агрегат и возвращаем
	order.ApplyEvent(evt)
	return *order, nil
}
EOF

cat > internal/delivery/http/handler.go <<'EOF'
package http

import (
	"encoding/json"
	"net/http"
	"strings"

	"order-service/internal/command"
	"order-service/internal/domain"
	"order-service/internal/eventstore"
)

type Handler struct {
	createOrderCmd *command.CreateOrderCommand
	payOrderCmd    *command.PayOrderCommand
	es             *eventstore.EventStore
}

func NewHandler(
	createCmd *command.CreateOrderCommand,
	payCmd *command.PayOrderCommand,
) *Handler {
	return &Handler{
		createOrderCmd: createCmd,
		payOrderCmd:    payCmd,
	}
}

func (h *Handler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	var req command.CreateOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	order, err := h.createOrderCmd.Execute(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(order)
}

func (h *Handler) PayOrder(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(strings.TrimSuffix(r.URL.Path, "/pay"), "/orders/")
	if id == "" {
		http.Error(w, "order id is required", http.StatusBadRequest)
		return
	}
	order, err := h.payOrderCmd.Execute(command.PayOrderRequest{OrderID: id})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(order)
}

func (h *Handler) GetOrder(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/orders/")
	if id == "" {
		http.Error(w, "order id is required", http.StatusBadRequest)
		return
	}
	events, err := h.es.Load(id)
	if err != nil {
		http.Error(w, "order not found", http.StatusNotFound)
		return
	}
	order := &domain.Order{}
	for _, e := range events {
		order.ApplyEvent(e)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(order)
}
EOF

cd ../..

echo "✅ Microservice with Event Sourcing created at ./$ROOT_DIR/$SERVICE"s