#!/bin/bash
# Микросервис с Saga Orchestration

set -e

ROOT_DIR="microservices-saga"
SERVICE="order-service"

mkdir -p "$ROOT_DIR/$SERVICE/cmd/server"
mkdir -p "$ROOT_DIR/$SERVICE/internal/domain"
mkdir -p "$ROOT_DIR/$SERVICE/internal/saga"
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
	"order-service/internal/saga"
)

func main() {
	// Инициализация саги
	orchestrator := saga.NewOrchestrator()

	// HTTP-хендлер
	handler := http.NewHandler(orchestrator)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /orders", handler.CreateOrder)

	log.Println("Order Service (Saga) starting on :8081")
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

cat > internal/saga/steps.go <<'EOF'
package saga

import (
	"context"
	"errors"
	"fmt"
	"time"

	"order-service/internal/domain"
)

// Step — интерфейс шага саги
type Step interface {
	Execute(ctx context.Context, data map[string]interface{}) error
	Compensate(ctx context.Context, data map[string]interface{}) error
}

// CreateOrderStep — создание заказа
type CreateOrderStep struct{}

func (s *CreateOrderStep) Execute(ctx context.Context, data map[string]interface{}) error {
	orderID, ok := data["orderID"].(string)
	if !ok {
		return errors.New("orderID missing")
	}
	fmt.Printf("[Saga Step 1] Create Order: %s created\n", orderID)
	// В реальном проекте здесь сохранение в БД
	return nil
}

func (s *CreateOrderStep) Compensate(ctx context.Context, data map[string]interface{}) error {
	orderID, _ := data["orderID"].(string)
	fmt.Printf("[Saga Compensation] Order %s cancelled\n", orderID)
	return nil
}

// ProcessPaymentStep — обработка оплаты
type ProcessPaymentStep struct {
	shouldFail bool
}

func NewProcessPaymentStep(shouldFail bool) *ProcessPaymentStep {
	return &ProcessPaymentStep{shouldFail: shouldFail}
}

func (s *ProcessPaymentStep) Execute(ctx context.Context, data map[string]interface{}) error {
	orderID, _ := data["orderID"].(string)
	if s.shouldFail {
		fmt.Printf("[Saga Step 2] Payment failed for %s\n", orderID)
		return errors.New("payment failed")
	}
	fmt.Printf("[Saga Step 2] Payment for %s processed\n", orderID)
	return nil
}

func (s *ProcessPaymentStep) Compensate(ctx context.Context, data map[string]interface{}) error {
	orderID, _ := data["orderID"].(string)
	fmt.Printf("[Saga Compensation] Refund for %s\n", orderID)
	return nil
}

// ScheduleDeliveryStep — планирование доставки
type ScheduleDeliveryStep struct{}

func (s *ScheduleDeliveryStep) Execute(ctx context.Context, data map[string]interface{}) error {
	orderID, _ := data["orderID"].(string)
	fmt.Printf("[Saga Step 3] Delivery for %s scheduled\n", orderID)
	return nil
}

func (s *ScheduleDeliveryStep) Compensate(ctx context.Context, data map[string]interface{}) error {
	orderID, _ := data["orderID"].(string)
	fmt.Printf("[Saga Compensation] Delivery for %s cancelled\n", orderID)
	return nil
}
EOF

cat > internal/saga/orchestrator.go <<'EOF'
package saga

import (
	"context"
	"fmt"
)

// Orchestrator — управляет последовательностью шагов
type Orchestrator struct {
	steps []Step
}

func NewOrchestrator() *Orchestrator {
	// Создаём шаги (можно менять порядок и добавлять новые)
	steps := []Step{
		&CreateOrderStep{},
		NewProcessPaymentStep(false), // true — для демонстрации ошибки
		&ScheduleDeliveryStep{},
	}
	return &Orchestrator{steps: steps}
}

// Execute запускает сагу
func (o *Orchestrator) Execute(ctx context.Context, data map[string]interface{}) error {
	history := []int{} // хранит индексы успешно выполненных шагов

	for i, step := range o.steps {
		if err := step.Execute(ctx, data); err != nil {
			// Ошибка → откат
			fmt.Printf("[Saga] Failed at step %d: %v. Starting compensation...\n", i, err)
			o.compensate(ctx, data, history)
			return fmt.Errorf("saga failed at step %d: %w", i, err)
		}
		history = append(history, i)
	}
	fmt.Println("[Saga] Completed successfully")
	return nil
}

func (o *Orchestrator) compensate(ctx context.Context, data map[string]interface{}, history []int) {
	for i := len(history) - 1; i >= 0; i-- {
		idx := history[i]
		if err := o.steps[idx].Compensate(ctx, data); err != nil {
			fmt.Printf("[Saga] Compensation error at step %d: %v\n", idx, err)
		}
	}
}
EOF

cat > internal/delivery/http/handler.go <<'EOF'
package http

import (
	"encoding/json"
	"net/http"
	"time"

	"order-service/internal/saga"
)

type Handler struct {
	orchestrator *saga.Orchestrator
}

func NewHandler(orchestrator *saga.Orchestrator) *Handler {
	return &Handler{orchestrator: orchestrator}
}

func (h *Handler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CustomerID string `json:"customer_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	// Генерируем ID заказа
	orderID := "ord-" + time.Now().Format("20060102150405")

	// Данные для саги
	data := map[string]interface{}{
		"orderID":    orderID,
		"customerID": req.CustomerID,
	}

	// Запускаем сагу
	err := h.orchestrator.Execute(r.Context(), data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"order_id": orderID,
		"status":   "created",
	})
}
EOF

cd ../..

echo "✅ Saga microservice created at ./$ROOT_DIR/$SERVICE"