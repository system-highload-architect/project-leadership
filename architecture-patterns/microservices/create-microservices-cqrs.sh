#!/bin/bash
# Микросервис с CQRS (разделение команд и запросов) — исправленная версия

set -e

ROOT_DIR="microservices-cqrs"
SERVICE="order-service"

mkdir -p "$ROOT_DIR/$SERVICE/cmd/server"
mkdir -p "$ROOT_DIR/$SERVICE/internal/domain"
mkdir -p "$ROOT_DIR/$SERVICE/internal/command"
mkdir -p "$ROOT_DIR/$SERVICE/internal/query"
mkdir -p "$ROOT_DIR/$SERVICE/internal/repository/write/inmemory"
mkdir -p "$ROOT_DIR/$SERVICE/internal/repository/read/inmemory"
mkdir -p "$ROOT_DIR/$SERVICE/internal/delivery"

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
	"order-service/internal/delivery"
	"order-service/internal/query"
	readrepo "order-service/internal/repository/read/inmemory"
	writerepo "order-service/internal/repository/write/inmemory"
)

func main() {
	// Репозитории
	writeRepo := writerepo.NewWriteRepository()
	readRepo := readrepo.NewReadRepository()

	// Команды
	createOrderCmd := command.NewCreateOrderCommand(writeRepo, readRepo)
	updateStatusCmd := command.NewUpdateStatusCommand(writeRepo, readRepo)

	// Запросы
	getOrderQuery := query.NewGetOrderQuery(readRepo)
	listOrdersQuery := query.NewListOrdersQuery(readRepo)

	// HTTP-хендлер
	handler := delivery.NewHandler(createOrderCmd, updateStatusCmd, getOrderQuery, listOrdersQuery)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /orders", handler.CreateOrder)
	mux.HandleFunc("PUT /orders/{id}/status", handler.UpdateStatus)
	mux.HandleFunc("GET /orders/{id}", handler.GetOrder)
	mux.HandleFunc("GET /orders", handler.ListOrders)

	log.Println("Order Service (CQRS) starting on :8081")
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

// OrderReadModel — денормализованная модель для чтения
type OrderReadModel struct {
	ID         string
	CustomerID string
	Items      []OrderItemReadModel
	Status     string
	Total      float64
	CreatedAt  time.Time
}

type OrderItemReadModel struct {
	ProductID string
	Quantity  int
	Price     float64
	Total     float64
}
EOF

cat > internal/repository/write/inmemory/write_repo.go <<'EOF'
package inmemory

import (
	"errors"
	"sync"

	"order-service/internal/domain"
)

type WriteRepository struct {
	mu     sync.RWMutex
	orders map[string]domain.Order
}

func NewWriteRepository() *WriteRepository {
	return &WriteRepository{
		orders: make(map[string]domain.Order),
	}
}

func (r *WriteRepository) Save(order domain.Order) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.orders[order.ID] = order
	return nil
}

func (r *WriteRepository) FindByID(id string) (domain.Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	order, ok := r.orders[id]
	if !ok {
		return domain.Order{}, errors.New("order not found")
	}
	return order, nil
}
EOF

cat > internal/repository/read/inmemory/read_repo.go <<'EOF'
package inmemory

import (
	"errors"
	"sync"

	"order-service/internal/domain"
)

type ReadRepository struct {
	mu     sync.RWMutex
	orders map[string]domain.OrderReadModel
}

func NewReadRepository() *ReadRepository {
	return &ReadRepository{
		orders: make(map[string]domain.OrderReadModel),
	}
}

func (r *ReadRepository) Save(order domain.OrderReadModel) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.orders[order.ID] = order
	return nil
}

func (r *ReadRepository) FindByID(id string) (domain.OrderReadModel, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	order, ok := r.orders[id]
	if !ok {
		return domain.OrderReadModel{}, errors.New("order not found")
	}
	return order, nil
}

func (r *ReadRepository) ListAll() ([]domain.OrderReadModel, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]domain.OrderReadModel, 0, len(r.orders))
	for _, order := range r.orders {
		result = append(result, order)
	}
	return result, nil
}
EOF

cat > internal/command/create_order.go <<'EOF'
package command

import (
	"errors"
	"time"

	"order-service/internal/domain"
	"order-service/internal/repository/read/inmemory"
	"order-service/internal/repository/write/inmemory"
)

type CreateOrderCommand struct {
	writeRepo *inmemory.WriteRepository
	readRepo  *inmemory.ReadRepository
}

func NewCreateOrderCommand(writeRepo *inmemory.WriteRepository, readRepo *inmemory.ReadRepository) *CreateOrderCommand {
	return &CreateOrderCommand{
		writeRepo: writeRepo,
		readRepo:  readRepo,
	}
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

	order := domain.Order{
		ID:         "ord-" + time.Now().Format("20060102150405"),
		CustomerID: req.CustomerID,
		Items:      req.Items,
		Status:     domain.OrderStatusNew,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	if err := c.writeRepo.Save(order); err != nil {
		return domain.Order{}, err
	}

	// Обновление read-модели (синхронно)
	c.updateReadModel(order)

	return order, nil
}

func (c *CreateOrderCommand) updateReadModel(order domain.Order) {
	readModel := domain.OrderReadModel{
		ID:         order.ID,
		CustomerID: order.CustomerID,
		Status:     string(order.Status),
		CreatedAt:  order.CreatedAt,
	}
	var total float64
	for _, item := range order.Items {
		readModel.Items = append(readModel.Items, domain.OrderItemReadModel{
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
			Price:     item.Price,
			Total:     item.Price * float64(item.Quantity),
		})
		total += item.Price * float64(item.Quantity)
	}
	readModel.Total = total
	_ = c.readRepo.Save(readModel)
}
EOF

cat > internal/command/update_status.go <<'EOF'
package command

import (
	"errors"
	"time"

	"order-service/internal/domain"
	readrepo "order-service/internal/repository/read/inmemory"
	writerepo "order-service/internal/repository/write/inmemory"
)

type UpdateStatusCommand struct {
	writeRepo *writerepo.WriteRepository
	readRepo  *readrepo.ReadRepository
}

func NewUpdateStatusCommand(writeRepo *writerepo.WriteRepository, readRepo *readrepo.ReadRepository) *UpdateStatusCommand {
	return &UpdateStatusCommand{
		writeRepo: writeRepo,
		readRepo:  readRepo,
	}
}

type UpdateStatusRequest struct {
	ID     string
	Status domain.OrderStatus
}

func (c *UpdateStatusCommand) Execute(req UpdateStatusRequest) error {
	if req.ID == "" {
		return errors.New("order id is required")
	}
	order, err := c.writeRepo.FindByID(req.ID)
	if err != nil {
		return err
	}
	// Простая валидация: можно менять статус только если он не равен текущему
	if order.Status == req.Status {
		return nil
	}
	// В реальном проекте здесь может быть сложная логика валидации переходов
	order.Status = req.Status
	order.UpdatedAt = time.Now()
	if err := c.writeRepo.Save(order); err != nil {
		return err
	}
	// Обновление read-модели
	c.updateReadModel(order)
	return nil
}

func (c *UpdateStatusCommand) updateReadModel(order domain.Order) {
	// Получаем существующую read-модель или создаём новую
	existing, _ := c.readRepo.FindByID(order.ID)
	readModel := domain.OrderReadModel{
		ID:         order.ID,
		CustomerID: order.CustomerID,
		Status:     string(order.Status),
		CreatedAt:  order.CreatedAt,
	}
	// Если есть существующие предметы, используем их
	if len(existing.Items) > 0 {
		readModel.Items = existing.Items
		readModel.Total = existing.Total
	}
	_ = c.readRepo.Save(readModel)
}
EOF

cat > internal/query/get_order.go <<'EOF'
package query

import (
	"errors"

	"order-service/internal/domain"
	"order-service/internal/repository/read/inmemory"
)

type GetOrderQuery struct {
	readRepo *inmemory.ReadRepository
}

func NewGetOrderQuery(readRepo *inmemory.ReadRepository) *GetOrderQuery {
	return &GetOrderQuery{readRepo: readRepo}
}

type GetOrderRequest struct {
	ID string
}

func (q *GetOrderQuery) Execute(req GetOrderRequest) (domain.OrderReadModel, error) {
	if req.ID == "" {
		return domain.OrderReadModel{}, errors.New("order id is required")
	}
	return q.readRepo.FindByID(req.ID)
}
EOF

cat > internal/query/list_orders.go <<'EOF'
package query

import (
	"order-service/internal/domain"
	"order-service/internal/repository/read/inmemory"
)

type ListOrdersQuery struct {
	readRepo *inmemory.ReadRepository
}

func NewListOrdersQuery(readRepo *inmemory.ReadRepository) *ListOrdersQuery {
	return &ListOrdersQuery{readRepo: readRepo}
}

func (q *ListOrdersQuery) Execute() ([]domain.OrderReadModel, error) {
	return q.readRepo.ListAll()
}
EOF

cat > internal/delivery/handler.go <<'EOF'
package delivery

import (
	"encoding/json"
	"net/http"
	"strings"

	"order-service/internal/command"
	"order-service/internal/domain"
	"order-service/internal/query"
)

type Handler struct {
	createOrderCmd  *command.CreateOrderCommand
	updateStatusCmd *command.UpdateStatusCommand
	getOrderQuery   *query.GetOrderQuery
	listOrdersQuery *query.ListOrdersQuery
}

func NewHandler(
	createCmd *command.CreateOrderCommand,
	updateCmd *command.UpdateStatusCommand,
	getQuery *query.GetOrderQuery,
	listQuery *query.ListOrdersQuery,
) *Handler {
	return &Handler{
		createOrderCmd:  createCmd,
		updateStatusCmd: updateCmd,
		getOrderQuery:   getQuery,
		listOrdersQuery: listQuery,
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

func (h *Handler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(strings.TrimSuffix(r.URL.Path, "/status"), "/orders/")
	if id == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}
	var req struct {
		Status domain.OrderStatus `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	err := h.updateStatusCmd.Execute(command.UpdateStatusRequest{
		ID:     id,
		Status: req.Status,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}

func (h *Handler) GetOrder(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/orders/")
	if id == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}
	order, err := h.getOrderQuery.Execute(query.GetOrderRequest{ID: id})
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(order)
}

func (h *Handler) ListOrders(w http.ResponseWriter, r *http.Request) {
	orders, err := h.listOrdersQuery.Execute()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(orders)
}
EOF

cd ../..

echo "✅ Microservice with CQRS created at ./$ROOT_DIR/$SERVICE"