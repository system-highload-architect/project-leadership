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
	}
	if err := s.repo.Save(order); err != nil {
		return domain.Order{}, err
	}
	return order, nil
}

func (s *OrderService) GetOrder(id string) (domain.Order, error) {
	return s.repo.FindByID(id)
}
