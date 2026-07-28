#!/bin/bash
# Modern SOA (Service-Oriented Architecture) — два сервиса через API Gateway

set -e

ROOT_DIR="modern-soa"

mkdir -p "$ROOT_DIR/api-gateway/cmd/server"
mkdir -p "$ROOT_DIR/api-gateway/internal/proxy"
mkdir -p "$ROOT_DIR/order-service/cmd/server"
mkdir -p "$ROOT_DIR/order-service/internal/domain"
mkdir -p "$ROOT_DIR/order-service/internal/service"
mkdir -p "$ROOT_DIR/order-service/internal/delivery/http"
mkdir -p "$ROOT_DIR/customer-service/cmd/server"
mkdir -p "$ROOT_DIR/customer-service/internal/domain"
mkdir -p "$ROOT_DIR/customer-service/internal/service"
mkdir -p "$ROOT_DIR/customer-service/internal/delivery/http"

cd "$ROOT_DIR"

# ===== API Gateway =====
cd api-gateway
cat > go.mod <<'EOF'
module api-gateway

go 1.23
EOF

cat > cmd/server/main.go <<'EOF'
package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
)

func main() {
	orderURL, _ := url.Parse("http://localhost:8081")
	customerURL, _ := url.Parse("http://localhost:8082")

	orderProxy := httputil.NewSingleHostReverseProxy(orderURL)
	customerProxy := httputil.NewSingleHostReverseProxy(customerURL)

	mux := http.NewServeMux()
	mux.HandleFunc("/orders/", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[Gateway] Routing to Order Service: %s %s", r.Method, r.URL.Path)
		orderProxy.ServeHTTP(w, r)
	})
	mux.HandleFunc("/customers/", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[Gateway] Routing to Customer Service: %s %s", r.Method, r.URL.Path)
		customerProxy.ServeHTTP(w, r)
	})

	log.Println("API Gateway starting on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}
EOF
cd ..

# ===== Order Service =====
cd order-service
cat > go.mod <<'EOF'
module order-service

go 1.23
EOF

cat > cmd/server/main.go <<'EOF'
package main

import (
	"log"
	"net/http"

	"order-service/internal/delivery/http"
	"order-service/internal/service"
)

func main() {
	svc := service.NewOrderService()
	handler := http.NewOrderHandler(svc)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /orders", handler.CreateOrder)
	mux.HandleFunc("GET /orders/{id}", handler.GetOrder)

	log.Println("Order Service starting on :8081")
	if err := http.ListenAndServe(":8081", mux); err != nil {
		log.Fatal(err)
	}
}
EOF

cat > internal/domain/order.go <<'EOF'
package domain

type Order struct {
	ID         string `json:"id"`
	CustomerID string `json:"customer_id"`
	Status     string `json:"status"`
}
EOF

cat > internal/service/order_service.go <<'EOF'
package service

import (
	"errors"
	"time"
)

type OrderService struct{}

func NewOrderService() *OrderService {
	return &OrderService{}
}

func (s *OrderService) CreateOrder(customerID string) (map[string]interface{}, error) {
	if customerID == "" {
		return nil, errors.New("customer id is required")
	}
	return map[string]interface{}{
		"id":          "ord-" + time.Now().Format("20060102150405"),
		"customer_id": customerID,
		"status":      "new",
	}, nil
}

func (s *OrderService) GetOrder(id string) (map[string]interface{}, error) {
	if id == "" {
		return nil, errors.New("order id is required")
	}
	return map[string]interface{}{
		"id":          id,
		"customer_id": "cust-123",
		"status":      "paid",
	}, nil
}
EOF

cat > internal/delivery/http/order_handler.go <<'EOF'
package http

import (
	"encoding/json"
	"net/http"
	"strings"

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
		CustomerID string `json:"customer_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	order, err := h.svc.CreateOrder(req.CustomerID)
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
cd ..

# ===== Customer Service =====
cd customer-service
cat > go.mod <<'EOF'
module customer-service

go 1.23
EOF

cat > cmd/server/main.go <<'EOF'
package main

import (
	"log"
	"net/http"

	"customer-service/internal/delivery/http"
	"customer-service/internal/service"
)

func main() {
	svc := service.NewCustomerService()
	handler := http.NewCustomerHandler(svc)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /customers/{id}", handler.GetCustomer)

	log.Println("Customer Service starting on :8082")
	if err := http.ListenAndServe(":8082", mux); err != nil {
		log.Fatal(err)
	}
}
EOF

cat > internal/domain/customer.go <<'EOF'
package domain

type Customer struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	City string `json:"city"`
}
EOF

cat > internal/service/customer_service.go <<'EOF'
package service

import (
	"errors"
)

type CustomerService struct{}

func NewCustomerService() *CustomerService {
	return &CustomerService{}
}

func (s *CustomerService) GetCustomer(id string) (map[string]interface{}, error) {
	if id == "" {
		return nil, errors.New("customer id is required")
	}
	return map[string]interface{}{
		"id":   id,
		"name": "John Doe",
		"city": "Moscow",
	}, nil
}
EOF

cat > internal/delivery/http/customer_handler.go <<'EOF'
package http

import (
	"encoding/json"
	"net/http"
	"strings"

	"customer-service/internal/service"
)

type CustomerHandler struct {
	svc *service.CustomerService
}

func NewCustomerHandler(svc *service.CustomerService) *CustomerHandler {
	return &CustomerHandler{svc: svc}
}

func (h *CustomerHandler) GetCustomer(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/customers/")
	if id == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}
	customer, err := h.svc.GetCustomer(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(customer)
}
EOF
cd ..

echo "✅ Modern SOA project created at ./$ROOT_DIR"