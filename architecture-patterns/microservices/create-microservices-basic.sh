#!/bin/bash
# Базовые микросервисы (Order + Payment) с отдельными БД и REST-общением

set -e

ROOT_DIR="microservices-basic"

mkdir -p "$ROOT_DIR"

# ===== Order Service =====
SERVICE="order-service"
mkdir -p "$ROOT_DIR/$SERVICE/cmd/server"
mkdir -p "$ROOT_DIR/$SERVICE/internal/domain"
mkdir -p "$ROOT_DIR/$SERVICE/internal/repository/inmemory"
mkdir -p "$ROOT_DIR/$SERVICE/internal/service"
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

	"order-service/internal/delivery/http"
	"order-service/internal/repository/inmemory"
	"order-service/internal/service"
)

func main() {
	// Инициализация зависимостей
	orderRepo := inmemory.NewOrderRepository()
	orderSvc := service.NewOrderService(orderRepo)
	orderHandler := http.NewOrderHandler(orderSvc)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /orders", orderHandler.CreateOrder)
	mux.HandleFunc("GET /orders", orderHandler.GetOrder)

	log.Println("Order Service starting on :8081")
	if err := http.ListenAndServe(":8081", mux); err != nil {
		log.Fatal(err)
	}
}
EOF

cat > internal/domain/order.go <<'EOF'
package domain

import "time"

type OrderStatus string

const (
	OrderStatusNew      OrderStatus = "new"
	OrderStatusPaid     OrderStatus = "paid"
	OrderStatusShipped  OrderStatus = "shipped"
	OrderStatusComplete OrderStatus = "complete"
	OrderStatusCanceled OrderStatus = "canceled"
)

type Order struct {
	ID         string
	CustomerID string
	Items      []OrderItem
	Status     OrderStatus
	CreatedAt  time.Time
}

type OrderItem struct {
	ProductID string
	Quantity  int
	Price     float64
}
EOF

cat > internal/repository/inmemory/order_repo.go <<'EOF'
package inmemory

import (
	"errors"
	"sync"

	"order-service/internal/domain"
)

type OrderRepository struct {
	mu     sync.RWMutex
	orders map[string]domain.Order
}

func NewOrderRepository() *OrderRepository {
	return &OrderRepository{
		orders: make(map[string]domain.Order),
	}
}

func (r *OrderRepository) Save(order domain.Order) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.orders[order.ID] = order
	return nil
}

func (r *OrderRepository) FindByID(id string) (domain.Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	order, ok := r.orders[id]
	if !ok {
		return domain.Order{}, errors.New("order not found")
	}
	return order, nil
}
EOF

cat > internal/service/order_service.go <<'EOF'
package service

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"order-service/internal/domain"
	"order-service/internal/repository/inmemory"
)

type OrderService struct {
	repo *inmemory.OrderRepository
}

func NewOrderService(repo *inmemory.OrderRepository) *OrderService {
	return &OrderService{repo: repo}
}

func (s *OrderService) CreateOrder(customerID string, items []domain.OrderItem) (domain.Order, error) {
	if customerID == "" {
		return domain.Order{}, errors.New("customer id is required")
	}
	if len(items) == 0 {
		return domain.Order{}, errors.New("order must have at least one item")
	}

	order := domain.Order{
		ID:         "ord-" + time.Now().Format("20060102150405"),
		CustomerID: customerID,
		Items:      items,
		Status:     domain.OrderStatusNew,
		CreatedAt:  time.Now(),
	}
	if err := s.repo.Save(order); err != nil {
		return domain.Order{}, err
	}

	// Вызов Payment Service для обработки платежа (синхронный REST)
	if err := s.callPaymentService(order); err != nil {
		// В реальности здесь может быть компенсация или повтор
		fmt.Printf("[Order Service] Payment failed: %v\n", err)
		// Можно отменить заказ или оставить как есть
	}

	return order, nil
}

func (s *OrderService) GetOrder(id string) (domain.Order, error) {
	return s.repo.FindByID(id)
}

// callPaymentService вызывает Payment Service для обработки платежа
func (s *OrderService) callPaymentService(order domain.Order) error {
	// Вычисляем общую сумму
	var total float64
	for _, item := range order.Items {
		total += item.Price * float64(item.Quantity)
	}

	reqBody := map[string]interface{}{
		"order_id": order.ID,
		"amount":   total,
		"customer": order.CustomerID,
	}
	jsonBody, _ := json.Marshal(reqBody)

	// Предполагаем, что Payment Service запущен на порту 8082
	resp, err := http.Post("http://localhost:8082/payments", "application/json", bytes.NewReader(jsonBody))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("payment service returned %d", resp.StatusCode)
	}
	return nil
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
	orderSvc *service.OrderService
}

func NewOrderHandler(orderSvc *service.OrderService) *OrderHandler {
	return &OrderHandler{orderSvc: orderSvc}
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
	order, err := h.orderSvc.CreateOrder(req.CustomerID, req.Items)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(order)
}

func (h *OrderHandler) GetOrder(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}
	order, err := h.orderSvc.GetOrder(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(order)
}
EOF

cd ../..

# ===== Payment Service =====
SERVICE="payment-service"
mkdir -p "$ROOT_DIR/$SERVICE/cmd/server"
mkdir -p "$ROOT_DIR/$SERVICE/internal/domain"
mkdir -p "$ROOT_DIR/$SERVICE/internal/repository/inmemory"
mkdir -p "$ROOT_DIR/$SERVICE/internal/service"
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
EOF

cat > internal/domain/payment.go <<'EOF'
package domain

import "time"

type PaymentStatus string

const (
	PaymentStatusPending PaymentStatus = "pending"
	PaymentStatusSuccess PaymentStatus = "success"
	PaymentStatusFailed  PaymentStatus = "failed"
)

type Payment struct {
	ID        string
	OrderID   string
	Amount    float64
	Customer  string
	Status    PaymentStatus
	CreatedAt time.Time
}
EOF

cat > internal/repository/inmemory/payment_repo.go <<'EOF'
package inmemory

import (
	"errors"
	"sync"

	"payment-service/internal/domain"
)

type PaymentRepository struct {
	mu       sync.RWMutex
	payments map[string]domain.Payment
}

func NewPaymentRepository() *PaymentRepository {
	return &PaymentRepository{
		payments: make(map[string]domain.Payment),
	}
}

func (r *PaymentRepository) Save(payment domain.Payment) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.payments[payment.ID] = payment
	return nil
}

func (r *PaymentRepository) FindByOrderID(orderID string) (domain.Payment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, p := range r.payments {
		if p.OrderID == orderID {
			return p, nil
		}
	}
	return domain.Payment{}, errors.New("payment not found")
}
EOF

cat > internal/service/payment_service.go <<'EOF'
package service

import (
	"errors"
	"time"

	"payment-service/internal/domain"
	"payment-service/internal/repository/inmemory"
)

type PaymentService struct {
	repo *inmemory.PaymentRepository
}

func NewPaymentService(repo *inmemory.PaymentRepository) *PaymentService {
	return &PaymentService{repo: repo}
}

func (s *PaymentService) ProcessPayment(orderID string, amount float64, customer string) (domain.Payment, error) {
	if orderID == "" || amount <= 0 {
		return domain.Payment{}, errors.New("invalid payment data")
	}
	// Имитация успешной обработки
	payment := domain.Payment{
		ID:        "pay-" + time.Now().Format("20060102150405"),
		OrderID:   orderID,
		Amount:    amount,
		Customer:  customer,
		Status:    domain.PaymentStatusSuccess,
		CreatedAt: time.Now(),
	}
	if err := s.repo.Save(payment); err != nil {
		return domain.Payment{}, err
	}
	return payment, nil
}
EOF

cat > internal/delivery/http/payment_handler.go <<'EOF'
package http

import (
	"encoding/json"
	"net/http"

	"payment-service/internal/service"
)

type PaymentHandler struct {
	paymentSvc *service.PaymentService
}

func NewPaymentHandler(paymentSvc *service.PaymentService) *PaymentHandler {
	return &PaymentHandler{paymentSvc: paymentSvc}
}

func (h *PaymentHandler) ProcessPayment(w http.ResponseWriter, r *http.Request) {
	var req struct {
		OrderID  string  `json:"order_id"`
		Amount   float64 `json:"amount"`
		Customer string  `json:"customer"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	payment, err := h.paymentSvc.ProcessPayment(req.OrderID, req.Amount, req.Customer)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(payment)
}
EOF

cd ../..

echo "✅ Microservices basic project created at ./$ROOT_DIR"