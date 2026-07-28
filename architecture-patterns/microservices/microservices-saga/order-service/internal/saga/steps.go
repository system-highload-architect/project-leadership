package saga

import (
	"context"
	"errors"
	"fmt"
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
