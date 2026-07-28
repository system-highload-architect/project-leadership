#!/bin/bash
# Микросервисы с API Gateway

set -e

ROOT_DIR="microservices-api-gateway"

mkdir -p "$ROOT_DIR"

# ===== API Gateway =====
SERVICE="api-gateway"
mkdir -p "$ROOT_DIR/$SERVICE/cmd/server"
mkdir -p "$ROOT_DIR/$SERVICE/internal/middleware"
mkdir -p "$ROOT_DIR/$SERVICE/internal/proxy"

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

	"api-gateway/internal/middleware"
	"api-gateway/internal/proxy"
)

func main() {
	// Создаём прокси для сервисов
	orderProxy := proxy.NewProxy("http://localhost:8081")
	paymentProxy := proxy.NewProxy("http://localhost:8082")

	mux := http.NewServeMux()

	// Маршруты с middleware
	mux.HandleFunc("/orders/", middleware.Chain(
		orderProxy.ServeHTTP,
		middleware.Logging,
		middleware.Auth,
	))

	mux.HandleFunc("/payments/", middleware.Chain(
		paymentProxy.ServeHTTP,
		middleware.Logging,
		middleware.Auth,
	))

	log.Println("API Gateway starting on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}
EOF

cat > internal/proxy/proxy.go <<'EOF'
package proxy

import (
	"net/http"
	"net/http/httputil"
	"net/url"
)

func NewProxy(target string) *httputil.ReverseProxy {
	parsedURL, _ := url.Parse(target)
	return httputil.NewSingleHostReverseProxy(parsedURL)
}
EOF

cat > internal/middleware/logging.go <<'EOF'
package middleware

import (
	"log"
	"net/http"
	"time"
)

func Logging(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next(w, r)
		log.Printf("%s %s %v", r.Method, r.URL.Path, time.Since(start))
	}
}
EOF

cat > internal/middleware/auth.go <<'EOF'
package middleware

import (
	"net/http"
	"strings"
)

// Auth — простая проверка JWT (для демонстрации)
func Auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "missing Authorization header", http.StatusUnauthorized)
			return
		}
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, "invalid Authorization header format", http.StatusUnauthorized)
			return
		}
		// В реальном проекте здесь проверка JWT
		// Для демонстрации считаем любой токен валидным, кроме "invalid"
		if parts[1] == "invalid" {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}
EOF

cat > internal/middleware/chain.go <<'EOF'
package middleware

import "net/http"

// Chain применяет цепочку middleware к хендлеру
func Chain(handler http.HandlerFunc, middlewares ...func(http.HandlerFunc) http.HandlerFunc) http.HandlerFunc {
	for _, mw := range middlewares {
		handler = mw(handler)
	}
	return handler
}
EOF

cd ../..

# ===== Order Service (упрощённая версия) =====
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
	"errors"
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
	return order, nil
}

func (s *OrderService) GetOrder(id string) (domain.Order, error) {
	return s.repo.FindByID(id)
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

# ===== Payment Service (упрощённая версия) =====
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

echo "✅ Microservices with API Gateway created at ./$ROOT_DIR"