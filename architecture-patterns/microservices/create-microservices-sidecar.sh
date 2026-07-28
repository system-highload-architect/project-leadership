#!/bin/bash
# Микросервисы с Sidecar (прокси + логирование + метрики)

set -e

ROOT_DIR="microservices-sidecar"

mkdir -p "$ROOT_DIR"

# ===== Order Service (основной) =====
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
	repo := inmemory.NewOrderRepository()
	svc := service.NewOrderService(repo)
	handler := http.NewOrderHandler(svc)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /orders", handler.CreateOrder)
	mux.HandleFunc("GET /orders/{id}", handler.GetOrder)

	log.Println("Order Service (main) starting on :8081")
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
	UpdatedAt  time.Time
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
		UpdatedAt:  time.Now(),
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
	"strings"

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

func (h *OrderHandler) GetOrder(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/orders/")
	if id == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}
	order, err := h.svc.GetOrder(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(order)
}
EOF

cd ../..

# ===== Sidecar =====
SERVICE="sidecar"
mkdir -p "$ROOT_DIR/$SERVICE/cmd/server"
mkdir -p "$ROOT_DIR/$SERVICE/internal/proxy"
mkdir -p "$ROOT_DIR/$SERVICE/internal/middleware"

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

	"sidecar/internal/middleware"
	"sidecar/internal/proxy"
)

func main() {
	// Создаём прокси к основному сервису (localhost:8081)
	proxy := proxy.NewProxy("http://localhost:8081")

	// Оборачиваем в middleware (логирование, метрики)
	handler := middleware.Logging(
		middleware.Metrics(proxy.ServeHTTP),
	)

	log.Println("Sidecar starting on :8080 (proxying to http://localhost:8081)")
	if err := http.ListenAndServe(":8080", handler); err != nil {
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

func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("[Sidecar] %s %s %v", r.Method, r.URL.Path, time.Since(start))
	})
}
EOF

cat > internal/middleware/metrics.go <<'EOF'
package middleware

import (
	"fmt"
	"net/http"
	"sync/atomic"
)

var requestCount uint64

func Metrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddUint64(&requestCount, 1)
		fmt.Printf("[Metrics] Requests: %d\n", requestCount)
		next.ServeHTTP(w, r)
	})
}
EOF

cd ../..

echo "✅ Microservices with Sidecar created at ./$ROOT_DIR"