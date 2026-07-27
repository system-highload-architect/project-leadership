#!/bin/bash
# Создание структуры модульного монолита с кодом (исправленная версия)

set -e

PROJECT_NAME="modular-monolith"

mkdir -p "$PROJECT_NAME/cmd/server"
mkdir -p "$PROJECT_NAME/internal/modules/order/api"
mkdir -p "$PROJECT_NAME/internal/modules/order/service"
mkdir -p "$PROJECT_NAME/internal/modules/order/repository"
mkdir -p "$PROJECT_NAME/internal/modules/inventory/api"
mkdir -p "$PROJECT_NAME/internal/modules/inventory/service"
mkdir -p "$PROJECT_NAME/internal/modules/inventory/repository"
mkdir -p "$PROJECT_NAME/internal/shared/domain"
mkdir -p "$PROJECT_NAME/internal/infrastructure/db"
mkdir -p "$PROJECT_NAME/pkg/logger"

cd "$PROJECT_NAME"

cat > go.mod <<'EOF'
module modular-monolith

go 1.23
EOF

cat > cmd/server/main.go <<'EOF'
package main

import (
	"log"
	"net/http"

	"modular-monolith/internal/infrastructure/db"
	invApi "modular-monolith/internal/modules/inventory/api"
	invRepo "modular-monolith/internal/modules/inventory/repository"
	invSvc "modular-monolith/internal/modules/inventory/service"
	orderApi "modular-monolith/internal/modules/order/api"
	orderRepo "modular-monolith/internal/modules/order/repository"
	orderSvc "modular-monolith/internal/modules/order/service"
)

func main() {
	database := db.NewInMemoryDB()

	// Inventory module
	invRepo := invRepo.NewInventoryRepository(database)
	invService := invSvc.NewInventoryService(invRepo)
	invHandler := invApi.NewInventoryHandler(invService)

	// Order module (depends on Inventory)
	orderRepo := orderRepo.NewOrderRepository(database)
	orderService := orderSvc.NewOrderService(orderRepo)
	orderHandler := orderApi.NewOrderHandler(orderService, invService)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /products", invHandler.AddProduct)
	mux.HandleFunc("GET /products", invHandler.GetProduct)
	mux.HandleFunc("POST /orders", orderHandler.CreateOrder)
	mux.HandleFunc("GET /orders", orderHandler.GetOrder)

	log.Println("Modular monolith server starting on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}
EOF

cat > internal/infrastructure/db/inmemory_db.go <<'EOF'
package db

import (
	"sync"

	"modular-monolith/internal/shared/domain"
)

type InMemoryDB struct {
	Mu       sync.RWMutex
	Orders   map[string]domain.Order
	Products map[string]domain.Product
}

func NewInMemoryDB() *InMemoryDB {
	return &InMemoryDB{
		Orders:   make(map[string]domain.Order),
		Products: make(map[string]domain.Product),
	}
}
EOF

cat > internal/shared/domain/order.go <<'EOF'
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

cat > internal/shared/domain/inventory.go <<'EOF'
package domain

type Product struct {
	ID    string
	Name  string
	Stock int
}
EOF

cat > internal/modules/order/repository/order_repo.go <<'EOF'
package repository

import (
	"errors"

	"modular-monolith/internal/infrastructure/db"
	"modular-monolith/internal/shared/domain"
)

type OrderRepository struct {
	db *db.InMemoryDB
}

func NewOrderRepository(db *db.InMemoryDB) *OrderRepository {
	return &OrderRepository{db: db}
}

func (r *OrderRepository) Save(order domain.Order) error {
	r.db.Mu.Lock()
	defer r.db.Mu.Unlock()
	r.db.Orders[order.ID] = order
	return nil
}

func (r *OrderRepository) FindByID(id string) (domain.Order, error) {
	r.db.Mu.RLock()
	defer r.db.Mu.RUnlock()
	order, ok := r.db.Orders[id]
	if !ok {
		return domain.Order{}, errors.New("order not found")
	}
	return order, nil
}
EOF

cat > internal/modules/order/service/order_service.go <<'EOF'
package service

import (
	"errors"
	"time"

	"modular-monolith/internal/modules/order/repository"
	"modular-monolith/internal/shared/domain"
)

type OrderService struct {
	repo *repository.OrderRepository
}

func NewOrderService(repo *repository.OrderRepository) *OrderService {
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

cat > internal/modules/order/api/http_handler.go <<'EOF'
package api

import (
	"encoding/json"
	"net/http"

	"modular-monolith/internal/modules/order/service"
	invService "modular-monolith/internal/modules/inventory/service"
	"modular-monolith/internal/shared/domain"
)

type OrderHandler struct {
	orderSvc *service.OrderService
	invSvc   *invService.InventoryService
}

func NewOrderHandler(orderSvc *service.OrderService, invSvc *invService.InventoryService) *OrderHandler {
	return &OrderHandler{
		orderSvc: orderSvc,
		invSvc:   invSvc,
	}
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
	for _, item := range req.Items {
		if err := h.invSvc.ReserveProduct(item.ProductID, item.Quantity); err != nil {
			http.Error(w, "inventory error: "+err.Error(), http.StatusBadRequest)
			return
		}
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

cat > internal/modules/inventory/repository/inventory_repo.go <<'EOF'
package repository

import (
	"errors"

	"modular-monolith/internal/infrastructure/db"
	"modular-monolith/internal/shared/domain"
)

type InventoryRepository struct {
	db *db.InMemoryDB
}

func NewInventoryRepository(db *db.InMemoryDB) *InventoryRepository {
	return &InventoryRepository{db: db}
}

func (r *InventoryRepository) FindProduct(id string) (domain.Product, error) {
	r.db.Mu.RLock()
	defer r.db.Mu.RUnlock()
	product, ok := r.db.Products[id]
	if !ok {
		return domain.Product{}, errors.New("product not found")
	}
	return product, nil
}

func (r *InventoryRepository) SaveProduct(product domain.Product) error {
	r.db.Mu.Lock()
	defer r.db.Mu.Unlock()
	r.db.Products[product.ID] = product
	return nil
}
EOF

cat > internal/modules/inventory/service/inventory_service.go <<'EOF'
package service

import (
	"errors"

	"modular-monolith/internal/modules/inventory/repository"
	"modular-monolith/internal/shared/domain"
)

type InventoryService struct {
	repo *repository.InventoryRepository
}

func NewInventoryService(repo *repository.InventoryRepository) *InventoryService {
	return &InventoryService{repo: repo}
}

func (s *InventoryService) ReserveProduct(productID string, quantity int) error {
	product, err := s.repo.FindProduct(productID)
	if err != nil {
		return err
	}
	if product.Stock < quantity {
		return errors.New("not enough stock")
	}
	product.Stock -= quantity
	return s.repo.SaveProduct(product)
}

func (s *InventoryService) GetProduct(id string) (domain.Product, error) {
	return s.repo.FindProduct(id)
}
EOF

cat > internal/modules/inventory/api/http_handler.go <<'EOF'
package api

import (
	"encoding/json"
	"net/http"

	"modular-monolith/internal/modules/inventory/service"
	"modular-monolith/internal/shared/domain"
)

type InventoryHandler struct {
	invSvc *service.InventoryService
}

func NewInventoryHandler(invSvc *service.InventoryService) *InventoryHandler {
	return &InventoryHandler{invSvc: invSvc}
}

func (h *InventoryHandler) AddProduct(w http.ResponseWriter, r *http.Request) {
	var product domain.Product
	if err := json.NewDecoder(r.Body).Decode(&product); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if err := h.invSvc.repo.SaveProduct(product); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(product)
}

func (h *InventoryHandler) GetProduct(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}
	product, err := h.invSvc.GetProduct(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(product)
}
EOF

cat > pkg/logger/logger.go <<'EOF'
package logger

import "log"

var (
	Info  = log.New(log.Writer(), "[INFO] ", log.LstdFlags)
	Error = log.New(log.Writer(), "[ERROR] ", log.LstdFlags)
)
EOF

echo "✅ Project $PROJECT_NAME created successfully!"
cd ..