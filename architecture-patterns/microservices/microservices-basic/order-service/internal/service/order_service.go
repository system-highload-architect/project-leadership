package service

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
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

	// Вызов Payment Service для обработки платежа (синхронный REST)
	if err := s.callPaymentService(order); err != nil {
		// В реальности здесь может быть компенсация или повтор
		fmt.Printf("[Order Service] Payment failed: %v\n", err)
		// Можно отменить заказ или оставить как есть
	}

	return order, nil
}

func (s *OrderService) GetOrder(id string) (domain.Order, error) {
	return s.repo.FindByID(id)
}

// callPaymentService вызывает Payment Service для обработки платежа
func (s *OrderService) callPaymentService(order domain.Order) error {
	// Вычисляем общую сумму
	var total float64
	for _, item := range order.Items {
		total += item.Price * float64(item.Quantity)
	}

	reqBody := map[string]interface{}{
		"order_id": order.ID,
		"amount":   total,
		"customer": order.CustomerID,
	}
	jsonBody, _ := json.Marshal(reqBody)

	// Предполагаем, что Payment Service запущен на порту 8082
	resp, err := http.Post("http://localhost:8082/payments", "application/json", bytes.NewReader(jsonBody))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("payment service returned %d", resp.StatusCode)
	}
	return nil
}
