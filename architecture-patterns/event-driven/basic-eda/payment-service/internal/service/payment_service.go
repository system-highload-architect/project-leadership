package service

import (
	"encoding/json"
	"fmt"
	"time"

	"payment-service/internal/domain"
	"shared/broker"
)

type PaymentService struct {
	broker *broker.Broker
}

func NewPaymentService(b *broker.Broker) *PaymentService {
	return &PaymentService{broker: b}
}

func (s *PaymentService) HandleOrderCreated(data interface{}) {
	bytes, err := json.Marshal(data)
	if err != nil {
		fmt.Println("Failed to marshal order data")
		return
	}
	var order struct {
		ID         string `json:"id"`
		CustomerID string `json:"customer_id"`
		Items      []struct {
			ProductID string  `json:"product_id"`
			Quantity  int     `json:"quantity"`
			Price     float64 `json:"price"`
		} `json:"items"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal(bytes, &order); err != nil {
		fmt.Println("Failed to unmarshal order data")
		return
	}
	var total float64
	for _, item := range order.Items {
		total += item.Price * float64(item.Quantity)
	}
	payment := domain.Payment{
		ID:      "pay-" + time.Now().Format("20060102150405"),
		OrderID: order.ID,
		Amount:  total,
		Status:  "success",
	}
	fmt.Printf("[Payment Service] Payment processed: %+v\n", payment)
	s.broker.Publish(broker.Event{
		Type: "PaymentProcessed",
		Data: payment,
	})
}
