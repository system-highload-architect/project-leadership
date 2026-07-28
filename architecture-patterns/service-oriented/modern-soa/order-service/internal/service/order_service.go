package service

import (
	"errors"
	"time"
)

type OrderService struct{}

func NewOrderService() *OrderService {
	return &OrderService{}
}

func (s *OrderService) CreateOrder(customerID string) (map[string]interface{}, error) {
	if customerID == "" {
		return nil, errors.New("customer id is required")
	}
	return map[string]interface{}{
		"id":          "ord-" + time.Now().Format("20060102150405"),
		"customer_id": customerID,
		"status":      "new",
	}, nil
}

func (s *OrderService) GetOrder(id string) (map[string]interface{}, error) {
	if id == "" {
		return nil, errors.New("order id is required")
	}
	return map[string]interface{}{
		"id":          id,
		"customer_id": "cust-123",
		"status":      "paid",
	}, nil
}
