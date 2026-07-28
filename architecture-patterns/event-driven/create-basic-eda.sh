#!/bin/bash
# Basic EDA (рабочая версия с shared/go.mod)

set -e

ROOT_DIR="basic-eda"

mkdir -p "$ROOT_DIR/order-service/cmd/server"
mkdir -p "$ROOT_DIR/order-service/internal/domain"
mkdir -p "$ROOT_DIR/order-service/internal/service"
mkdir -p "$ROOT_DIR/order-service/internal/delivery/http"
mkdir -p "$ROOT_DIR/payment-service/cmd/server"
mkdir -p "$ROOT_DIR/payment-service/internal/domain"
mkdir -p "$ROOT_DIR/payment-service/internal/service"
mkdir -p "$ROOT_DIR/shared/broker"

cd "$ROOT_DIR"

# ===== Shared module =====
cat > shared/go.mod <<'EOF'
module shared

go 1.23
EOF

cat > shared/broker/broker.go <<'EOF'
package broker

import (
	"sync"
)

type Event struct {
	Type string
	Data interface{}
}

type Broker struct {
	subscribers map[string][]chan Event
	mu          sync.RWMutex
}

var (
	instance *Broker
	once     sync.Once
)

func GetBroker() *Broker {
	once.Do(func() {
		instance = &Broker{
			subscribers: make(map[string][]chan Event),
		}
	})
	return instance
}

func (b *Broker) Subscribe(eventType string) <-chan Event {
	b.mu.Lock()
	defer b.mu.Unlock()
	ch := make(chan Event, 10)
	b.subscribers[eventType] = append(b.subscribers[eventType], ch)
	return ch
}

func (b *Broker) Publish(event Event) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for _, ch := range b.subscribers[event.Type] {
		ch <- event
	}
}
EOF

# ===== Order Service =====
cd order-service
cat > go.mod <<'EOF'
module order-service

go 1.23

replace shared => ../shared
EOF

cat > cmd/server/main.go <<'EOF'
package main

import (
	"log"
	"net/http"

	"order-service/internal/delivery/http"
	"order-service/internal/service"
	"shared/broker"
)

func main() {
	b := broker.GetBroker()
	svc := service.NewOrderService(b)
	handler := http.NewOrderHandler(svc)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /orders", handler.CreateOrder)

	log.Println("Order Service starting on :8081")
	if err := http.ListenAndServe(":8081", mux); err != nil {
		log.Fatal(err)
	}
}
EOF

cat > internal/domain/order.go <<'EOF'
package domain

type Order struct {
	ID         string      `json:"id"`
	CustomerID string      `json:"customer_id"`
	Items      []OrderItem `json:"items"`
	Status     string      `json:"status"`
}

type OrderItem struct {
	ProductID string  `json:"product_id"`
	Quantity  int     `json:"quantity"`
	Price     float64 `json:"price"`
}
EOF

cat > internal/service/order_service.go <<'EOF'
package service

import (
	"time"

	"order-service/internal/domain"
	"shared/broker"
)

type OrderService struct {
	broker *broker.Broker
}

func NewOrderService(b *broker.Broker) *OrderService {
	return &OrderService{broker: b}
}

func (s *OrderService) CreateOrder(customerID string, items []domain.OrderItem) (domain.Order, error) {
	order := domain.Order{
		ID:         "ord-" + time.Now().Format("20060102150405"),
		CustomerID: customerID,
		Items:      items,
		Status:     "new",
	}
	s.broker.Publish(broker.Event{
		Type: "OrderCreated",
		Data: order,
	})
	return order, nil
}
EOF

cat > internal/delivery/http/order_handler.go <<'EOF'
package http

import (
	"encoding/json"
	"net/http"

	"order-service/internal/domain"
	"order-service/internal/service"
)

type OrderHandler struct {
	svc *service.OrderService
}

func NewOrderHandler(svc *service.OrderService) *OrderHandler {
	return &OrderHandler{svc: svc}
}

func (h *OrderHandler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CustomerID string             `json:"customer_id"`
		Items      []domain.OrderItem `json:"items"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	order, err := h.svc.CreateOrder(req.CustomerID, req.Items)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(order)
}
EOF

cd ..

# ===== Payment Service =====
cd payment-service
cat > go.mod <<'EOF'
module payment-service

go 1.23

replace shared => ../shared
EOF

cat > cmd/server/main.go <<'EOF'
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
EOF

cat > internal/domain/payment.go <<'EOF'
package domain

type Payment struct {
	ID      string  `json:"id"`
	OrderID string  `json:"order_id"`
	Amount  float64 `json:"amount"`
	Status  string  `json:"status"`
}
EOF

cat > internal/service/payment_service.go <<'EOF'
package service

import (
	"encoding/json"
	"fmt"
	"time"

	"payment-service/internal/domain"
	"shared/broker"
)

type PaymentService struct {
	broker *broker.Broker
}

func NewPaymentService(b *broker.Broker) *PaymentService {
	return &PaymentService{broker: b}
}

func (s *PaymentService) HandleOrderCreated(data interface{}) {
	bytes, err := json.Marshal(data)
	if err != nil {
		fmt.Println("Failed to marshal order data")
		return
	}
	var order struct {
		ID         string `json:"id"`
		CustomerID string `json:"customer_id"`
		Items      []struct {
			ProductID string  `json:"product_id"`
			Quantity  int     `json:"quantity"`
			Price     float64 `json:"price"`
		} `json:"items"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal(bytes, &order); err != nil {
		fmt.Println("Failed to unmarshal order data")
		return
	}
	var total float64
	for _, item := range order.Items {
		total += item.Price * float64(item.Quantity)
	}
	payment := domain.Payment{
		ID:      "pay-" + time.Now().Format("20060102150405"),
		OrderID: order.ID,
		Amount:  total,
		Status:  "success",
	}
	fmt.Printf("[Payment Service] Payment processed: %+v\n", payment)
	s.broker.Publish(broker.Event{
		Type: "PaymentProcessed",
		Data: payment,
	})
}
EOF

cd ..

echo "✅ Basic EDA (fixed) created at ./$ROOT_DIR"