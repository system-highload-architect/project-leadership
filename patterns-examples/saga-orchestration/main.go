package main

import (
	"context"
	"errors"
	"fmt"
)

// ===== Шаги саги =====

// Step — интерфейс шага саги
type Step interface {
	Execute(ctx context.Context, data map[string]interface{}) error
	Compensate(ctx context.Context, data map[string]interface{}) error
}

// ===== Конкретные шаги =====

// CreateOrderStep — создание заказа
type CreateOrderStep struct{}

func (s *CreateOrderStep) Execute(ctx context.Context, data map[string]interface{}) error {
	orderID, ok := data["orderID"].(string)
	if !ok {
		return errors.New("orderID missing")
	}
	fmt.Printf("[Step 1] Create Order: %s created\n", orderID)
	return nil
}

func (s *CreateOrderStep) Compensate(ctx context.Context, data map[string]interface{}) error {
	orderID, _ := data["orderID"].(string)
	fmt.Printf("[Step 1] Compensate: %s cancelled\n", orderID)
	return nil
}

// ProcessPaymentStep — обработка оплаты
type ProcessPaymentStep struct {
	shouldFail bool // для демонстрации ошибки
}

func (s *ProcessPaymentStep) Execute(ctx context.Context, data map[string]interface{}) error {
	orderID, _ := data["orderID"].(string)
	if s.shouldFail {
		fmt.Printf("[Step 2] Process Payment: payment failed for %s\n", orderID)
		return errors.New("payment failed")
	}
	fmt.Printf("[Step 2] Process Payment: payment for %s processed\n", orderID)
	return nil
}

func (s *ProcessPaymentStep) Compensate(ctx context.Context, data map[string]interface{}) error {
	orderID, _ := data["orderID"].(string)
	fmt.Printf("[Step 2] Compensate: refund for %s\n", orderID)
	return nil
}

// ScheduleDeliveryStep — планирование доставки
type ScheduleDeliveryStep struct{}

func (s *ScheduleDeliveryStep) Execute(ctx context.Context, data map[string]interface{}) error {
	orderID, _ := data["orderID"].(string)
	fmt.Printf("[Step 3] Schedule Delivery: delivery for %s scheduled\n", orderID)
	return nil
}

func (s *ScheduleDeliveryStep) Compensate(ctx context.Context, data map[string]interface{}) error {
	orderID, _ := data["orderID"].(string)
	fmt.Printf("[Step 3] Compensate: delivery for %s cancelled\n", orderID)
	return nil
}

// ===== Оркестратор =====

// SagaOrchestrator — управляет последовательностью шагов
type SagaOrchestrator struct {
	steps []Step
}

// NewSagaOrchestrator создаёт оркестратор с заданной последовательностью шагов
func NewSagaOrchestrator(steps []Step) *SagaOrchestrator {
	return &SagaOrchestrator{steps: steps}
}

// Execute запускает сагу
func (o *SagaOrchestrator) Execute(ctx context.Context, data map[string]interface{}) error {
	history := []int{} // хранит индексы успешно выполненных шагов

	for i, step := range o.steps {
		if err := step.Execute(ctx, data); err != nil {
			// Ошибка → откат (компенсация в обратном порядке)
			fmt.Printf("Saga failed at step %d: %v. Starting compensation...\n", i, err)
			o.compensate(ctx, data, history)
			return fmt.Errorf("saga failed at step %d: %w", i, err)
		}
		history = append(history, i)
	}
	fmt.Println("Saga completed successfully")
	return nil
}

// compensate выполняет компенсацию в обратном порядке
func (o *SagaOrchestrator) compensate(ctx context.Context, data map[string]interface{}, history []int) {
	for i := len(history) - 1; i >= 0; i-- {
		idx := history[i]
		if err := o.steps[idx].Compensate(ctx, data); err != nil {
			fmt.Printf("Compensation error at step %d: %v\n", idx, err)
		}
	}
}

// ===== Главная функция =====

func main() {
	fmt.Println("=== Saga Orchestration (Успех) ===")
	// Сценарий успеха
	steps := []Step{
		&CreateOrderStep{},
		&ProcessPaymentStep{shouldFail: false},
		&ScheduleDeliveryStep{},
	}
	orchestrator := NewSagaOrchestrator(steps)
	data := map[string]interface{}{"orderID": "order-123"}
	if err := orchestrator.Execute(context.Background(), data); err != nil {
		fmt.Println("Error:", err)
	}

	fmt.Println("\n=== Saga Orchestration (Ошибка + компенсация) ===")
	// Сценарий с ошибкой
	stepsFail := []Step{
		&CreateOrderStep{},
		&ProcessPaymentStep{shouldFail: true},
		&ScheduleDeliveryStep{},
	}
	orchestratorFail := NewSagaOrchestrator(stepsFail)
	if err := orchestratorFail.Execute(context.Background(), data); err != nil {
		fmt.Println("Error:", err)
	}
}
