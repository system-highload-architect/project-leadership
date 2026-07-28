package service

import (
	"time"

	"order-service/internal/domain"
	"shared/broker"
)

type OrderService struct {
	broker *broker.Broker
}

func NewOrderService(b *broker.Broker) *OrderService {
	return &OrderService{broker: b}
}

func (s *OrderService) CreateOrder(customerID string, items []domain.OrderItem) (domain.Order, error) {
	order := domain.Order{
		ID:         "ord-" + time.Now().Format("20060102150405"),
		CustomerID: customerID,
		Items:      items,
		Status:     "new",
	}
	s.broker.Publish(broker.Event{
		Type: "OrderCreated",
		Data: order,
	})
	return order, nil
}
