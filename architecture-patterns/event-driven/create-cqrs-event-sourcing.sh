#!/bin/bash
# CQRS + Event Sourcing (объединённый пример)

set -e

ROOT_DIR="cqrs-event-sourcing"
SERVICE="order-service"

mkdir -p "$ROOT_DIR/$SERVICE/cmd/server"
mkdir -p "$ROOT_DIR/$SERVICE/internal/domain"
mkdir -p "$ROOT_DIR/$SERVICE/internal/event"
mkdir -p "$ROOT_DIR/$SERVICE/internal/eventstore"
mkdir -p "$ROOT_DIR/$SERVICE/internal/command"
mkdir -p "$ROOT_DIR/$SERVICE/internal/query"
mkdir -p "$ROOT_DIR/$SERVICE/internal/projection"
mkdir -p "$ROOT_DIR/$SERVICE/internal/repository/read/inmemory"
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
	"order-service/internal/projection"
	"order-service/internal/query"
	readrepo "order-service/internal/repository/read/inmemory"
)

func main() {
	// Event Store (write side)
	es := eventstore.NewEventStore()

	// Read repository (query side)
	readRepo := readrepo.NewReadRepository()

	// Projection (обновляет read-модель из событий)
	proj := projection.NewProjection(readRepo)

	// Подписываемся на события (в реальном проекте через шину, здесь напрямую)
	// Мы будем вызывать проекцию вручную после сохранения события (для простоты)
	// В реальном проекте это делается асинхронно через Kafka.

	// Команды
	createOrderCmd := command.NewCreateOrderCommand(es, proj)
	payOrderCmd := command.NewPayOrderCommand(es, proj)

	// Запросы
	getOrderQuery := query.NewGetOrderQuery(readRepo)
	listOrdersQuery := query.NewListOrdersQuery(readRepo)

	// HTTP-хендлер
	handler := http.NewHandler(createOrderCmd, payOrderCmd, getOrderQuery, listOrdersQuery)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /orders", handler.CreateOrder)
	mux.HandleFunc("POST /orders/{id}/pay", handler.PayOrder)
	mux.HandleFunc("GET /orders/{id}", handler.GetOrder)
	mux.HandleFunc("GET /orders", handler.ListOrders)

	log.Println("Order Service (CQRS + Event Sourcing) starting on :8081")
	if err := http.ListenAndServe(":8081", mux); err != nil {
		log.Fatal(err)
	}
}
EOF

cat > internal/domain/order.go <<'EOF'
package domain

import (
	"time"

	"order-service/internal/event"
)

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

// ApplyEvent применяет событие к агрегату
func (o *Order) ApplyEvent(e event.Event) {
	switch ev := e.(type) {
	case *event.OrderCreated:
		o.ID = ev.AggregateID
		o.CustomerID = ev.CustomerID
		o.Items = make([]OrderItem, len(ev.Items))
		for i, item := range ev.Items {
			o.Items[i] = OrderItem{
				ProductID: item.ProductID,
				Quantity:  item.Quantity,
				Price:     item.Price,
			}
		}
		o.Status = OrderStatusNew
		o.CreatedAt = ev.Timestamp
		o.UpdatedAt = ev.Timestamp
	case *event.OrderPaid:
		o.Status = OrderStatusPaid
		o.UpdatedAt = ev.Timestamp
	}
}

// Replay восстанавливает агрегат из списка событий
func (o *Order) Replay(events []event.Event) {
	for _, e := range events {
		o.ApplyEvent(e)
	}
}

// NewOrderFromEvents создаёт агрегат из событий
func NewOrderFromEvents(events []event.Event) *Order {
	o := &Order{}
	o.Replay(events)
	return o
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
	Items      []struct {
		ProductID string
		Quantity  int
		Price     float64
	}
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

cat > internal/command/create_order.go <<'EOF'
package command

import (
	"errors"
	"time"

	"order-service/internal/domain"
	"order-service/internal/event"
	"order-service/internal/eventstore"
	"order-service/internal/projection"
)

type CreateOrderCommand struct {
	es   *eventstore.EventStore
	proj *projection.Projection
}

func NewCreateOrderCommand(es *eventstore.EventStore, proj *projection.Projection) *CreateOrderCommand {
	return &CreateOrderCommand{es: es, proj: proj}
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
	}
	for _, item := range req.Items {
		evt.Items = append(evt.Items, struct {
			ProductID string
			Quantity  int
			Price     float64
		}{
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
			Price:     item.Price,
		})
	}

	// Сохраняем событие
	if err := c.es.Save(orderID, evt); err != nil {
		return domain.Order{}, err
	}

	// Обновляем проекцию (синхронно для простоты)
	c.proj.Apply(evt)

	// Возвращаем агрегат
	events, _ := c.es.Load(orderID)
	order := domain.NewOrderFromEvents(events)
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
	"order-service/internal/projection"
)

type PayOrderCommand struct {
	es   *eventstore.EventStore
	proj *projection.Projection
}

func NewPayOrderCommand(es *eventstore.EventStore, proj *projection.Projection) *PayOrderCommand {
	return &PayOrderCommand{es: es, proj: proj}
}

type PayOrderRequest struct {
	OrderID string
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
	order := domain.NewOrderFromEvents(events)
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

	// Обновляем проекцию
	c.proj.Apply(evt)

	// Возвращаем обновлённый агрегат
	events, _ = c.es.Load(req.OrderID)
	order = domain.NewOrderFromEvents(events)
	return *order, nil
}
EOF

cat > internal/projection/projection.go <<'EOF'
package projection

import (
	"order-service/internal/domain"
	"order-service/internal/event"
	"order-service/internal/repository/read/inmemory"
)

type Projection struct {
	readRepo *inmemory.ReadRepository
}

func NewProjection(readRepo *inmemory.ReadRepository) *Projection {
	return &Projection{readRepo: readRepo}
}

// Apply обновляет read-модель на основе события
func (p *Projection) Apply(e event.Event) {
	switch ev := e.(type) {
	case *event.OrderCreated:
		// Создаём read-модель
		readModel := domain.OrderReadModel{
			ID:         ev.AggregateID,
			CustomerID: ev.CustomerID,
			Status:     string(domain.OrderStatusNew),
			CreatedAt:  ev.Timestamp,
		}
		var total float64
		for _, item := range ev.Items {
			readModel.Items = append(readModel.Items, domain.OrderItemReadModel{
				ProductID: item.ProductID,
				Quantity:  item.Quantity,
				Price:     item.Price,
				Total:     float64(item.Quantity) * item.Price,
			})
			total += float64(item.Quantity) * item.Price
		}
		readModel.Total = total
		_ = p.readRepo.Save(readModel)

	case *event.OrderPaid:
		// Обновляем статус в read-модели
		existing, err := p.readRepo.FindByID(ev.AggregateID)
		if err != nil {
			return
		}
		existing.Status = string(domain.OrderStatusPaid)
		_ = p.readRepo.Save(existing)
	}
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

cat > internal/domain/read_model.go <<'EOF'
package domain

import "time"

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

cat > internal/delivery/http/handler.go <<'EOF'
package http

import (
	"encoding/json"
	"net/http"
	"strings"

	"order-service/internal/command"
	"order-service/internal/query"
)

type Handler struct {
	createOrderCmd *command.CreateOrderCommand
	payOrderCmd    *command.PayOrderCommand
	getOrderQuery  *query.GetOrderQuery
	listOrdersQuery *query.ListOrdersQuery
}

func NewHandler(
	createCmd *command.CreateOrderCommand,
	payCmd *command.PayOrderCommand,
	getQuery *query.GetOrderQuery,
	listQuery *query.ListOrdersQuery,
) *Handler {
	return &Handler{
		createOrderCmd: createCmd,
		payOrderCmd:    payCmd,
		getOrderQuery:  getQuery,
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

echo "✅ CQRS + Event Sourcing project created at ./$ROOT_DIR"