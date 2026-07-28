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
