package command

import (
	"errors"
	"time"

	"order-service/internal/domain"
	"order-service/internal/event"
	"order-service/internal/eventstore"
)

type CreateOrderCommand struct {
	es *eventstore.EventStore
}

func NewCreateOrderCommand(es *eventstore.EventStore) *CreateOrderCommand {
	return &CreateOrderCommand{es: es}
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
		Items:      make([]event.OrderItem, len(req.Items)),
	}
	for i, item := range req.Items {
		evt.Items[i] = event.OrderItem{
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
			Price:     item.Price,
		}
	}

	// Сохраняем событие
	if err := c.es.Save(orderID, evt); err != nil {
		return domain.Order{}, err
	}

	// Восстанавливаем агрегат (для возврата)
	order := domain.NewOrder(orderID, req.CustomerID, req.Items)
	return *order, nil
}
