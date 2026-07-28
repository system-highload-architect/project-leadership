#!/bin/bash
# Классический монолит (структурированный, но с сильной связанностью)

set -e

PROJECT_NAME="classic-monolith"

mkdir -p "$PROJECT_NAME/cmd/server"
mkdir -p "$PROJECT_NAME/internal/domain"
mkdir -p "$PROJECT_NAME/internal/repository"
mkdir -p "$PROJECT_NAME/internal/service"
mkdir -p "$PROJECT_NAME/internal/delivery/http"
mkdir -p "$PROJECT_NAME/internal/bootstrap"
mkdir -p "$PROJECT_NAME/pkg/logger"

cd "$PROJECT_NAME"

cat > go.mod <<'EOF'
module classic-monolith

go 1.23
EOF

# ===== DOMAIN =====
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

cat > internal/domain/product.go <<'EOF'
package domain

type Product struct {
	ID    string
	Name  string
	Stock int
}
EOF

# ===== REPOSITORY =====
cat > internal/repository/order_repo.go <<'EOF'
package repository

import (
	"errors"
	"sync"

	"classic-monolith/internal/domain"
)

// Глобальное состояние — классический подход
var (
	Orders   = make(map[string]domain.Order)
	OrdersMu sync.RWMutex
)

type OrderRepository struct{}

func NewOrderRepository() *OrderRepository {
	return &OrderRepository{}
}

func (r *OrderRepository) Save(order domain.Order) error {
	OrdersMu.Lock()
	defer OrdersMu.Unlock()
	Orders[order.ID] = order
	return nil
}

func (r *OrderRepository) FindByID(id string) (domain.Order, error) {
	OrdersMu.RLock()
	defer OrdersMu.RUnlock()
	order, ok := Orders[id]
	if !ok {
		return domain.Order{}, errors.New("order not found")
	}
	return order, nil
}
EOF

cat > internal/repository/product_repo.go <<'EOF'
package repository

import (
	"errors"
	"sync"

	"classic-monolith/internal/domain"
)

var (
	Products   = make(map[string]domain.Product)
	ProductsMu sync.RWMutex
)

type ProductRepository struct{}

func NewProductRepository() *ProductRepository {
	return &ProductRepository{}
}

func (r *ProductRepository) Save(product domain.Product) error {
	ProductsMu.Lock()
	defer ProductsMu.Unlock()
	Products[product.ID] = product
	return nil
}

func (r *ProductRepository) FindByID(id string) (domain.Product, error) {
	ProductsMu.RLock()
	defer ProductsMu.RUnlock()
	product, ok := Products[id]
	if !ok {
		return domain.Product{}, errors.New("product not found")
	}
	return product, nil
}

func (r *ProductRepository) DecreaseStock(productID string, quantity int) error {
	ProductsMu.Lock()
	defer ProductsMu.Unlock()
	product, ok := Products[productID]
	if !ok {
		return errors.New("product not found")
	}
	if product.Stock < quantity {
		return errors.New("not enough stock")
	}
	product.Stock -= quantity
	Products[productID] = product
	return nil
}
EOF

# ===== SERVICE =====
cat > internal/service/order_service.go <<'EOF'
package service

import (
	"errors"
	"time"

	"classic-monolith/internal/domain"
	"classic-monolith/internal/repository"
)

// OrderService — бизнес-логика (всё в одном месте)
type OrderService struct {
	orderRepo   *repository.OrderRepository
	productRepo *repository.ProductRepository
}

func NewOrderService(
	orderRepo *repository.OrderRepository,
	productRepo *repository.ProductRepository,
) *OrderService {
	return &OrderService{
		orderRepo:   orderRepo,
		productRepo: productRepo,
	}
}

func (s *OrderService) CreateOrder(customerID string, items []domain.OrderItem) (domain.Order, error) {
	if customerID == "" {
		return domain.Order{}, errors.New("customer id is required")
	}
	if len(items) == 0 {
		return domain.Order{}, errors.New("order must have at least one item")
	}

	// Проверяем остатки (прямо здесь, в сервисе)
	for _, item := range items {
		if err := s.productRepo.DecreaseStock(item.ProductID, item.Quantity); err != nil {
			return domain.Order{}, err
		}
	}

	order := domain.Order{
		ID:         "ord-" + time.Now().Format("20060102150405"),
		CustomerID: customerID,
		Items:      items,
		Status:     domain.OrderStatusNew,
		CreatedAt:  time.Now(),
	}
	if err := s.orderRepo.Save(order); err != nil {
		return domain.Order{}, err
	}
	return order, nil
}

func (s *OrderService) GetOrder(id string) (domain.Order, error) {
	return s.orderRepo.FindByID(id)
}
EOF

# ===== DELIVERY (HTTP) =====
cat > internal/delivery/http/order_handler.go <<'EOF'
package http

import (
	"encoding/json"
	"net/http"

	"classic-monolith/internal/domain"
	"classic-monolith/internal/service"
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

cat > internal/delivery/http/product_handler.go <<'EOF'
package http

import (
	"encoding/json"
	"net/http"

	"classic-monolith/internal/domain"
	"classic-monolith/internal/repository"
)

type ProductHandler struct {
	productRepo *repository.ProductRepository
}

func NewProductHandler(productRepo *repository.ProductRepository) *ProductHandler {
	return &ProductHandler{productRepo: productRepo}
}

func (h *ProductHandler) AddProduct(w http.ResponseWriter, r *http.Request) {
	var p domain.Product
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if err := h.productRepo.Save(p); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(p)
}

func (h *ProductHandler) GetProduct(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}
	product, err := h.productRepo.FindByID(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(product)
}
EOF

# ===== BOOTSTRAP (DI) =====
cat > internal/bootstrap/container.go <<'EOF'
package bootstrap

import (
	"classic-monolith/internal/delivery/http"
	"classic-monolith/internal/repository"
	"classic-monolith/internal/service"
)

type Container struct {
	OrderHandler   *http.OrderHandler
	ProductHandler *http.ProductHandler
}

func NewContainer() *Container {
	// Инициализация репозиториев
	orderRepo := repository.NewOrderRepository()
	productRepo := repository.NewProductRepository()

	// Инициализация сервисов
	orderSvc := service.NewOrderService(orderRepo, productRepo)

	// Инициализация хендлеров
	orderHandler := http.NewOrderHandler(orderSvc)
	productHandler := http.NewProductHandler(productRepo)

	return &Container{
		OrderHandler:   orderHandler,
		ProductHandler: productHandler,
	}
}
EOF

# ===== MAIN =====
cat > cmd/server/main.go <<'EOF'
package main

import (
	"log"
	"net/http"

	"classic-monolith/internal/bootstrap"
)

func main() {
	container := bootstrap.NewContainer()

	mux := http.NewServeMux()
	mux.HandleFunc("POST /orders", container.OrderHandler.CreateOrder)
	mux.HandleFunc("GET /orders", container.OrderHandler.GetOrder)
	mux.HandleFunc("POST /products", container.ProductHandler.AddProduct)
	mux.HandleFunc("GET /products", container.ProductHandler.GetProduct)

	log.Println("Classic monolith starting on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}
EOF

# ===== LOGGER =====
cat > pkg/logger/logger.go <<'EOF'
package logger

import "log"

var (
	Info  = log.New(log.Writer(), "[INFO] ", log.LstdFlags)
	Error = log.New(log.Writer(), "[ERROR] ", log.LstdFlags)
)
EOF

echo "✅ Classic monolith project created at ./$PROJECT_NAME"
cd ..