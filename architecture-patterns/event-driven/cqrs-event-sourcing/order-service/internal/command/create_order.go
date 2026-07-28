package command

import (
	"errors"
	"time"

	"order-service/internal/domain"
	"order-service/internal/event"
	"order-service/internal/eventstore"
	"order-service/internal/projection"
)

type CreateOrderCommand struct {
	es   *eventstore.EventStore
	proj *projection.Projection
}

func NewCreateOrderCommand(es *eventstore.EventStore, proj *projection.Projection) *CreateOrderCommand {
	return &CreateOrderCommand{es: es, proj: proj}
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
	}
	for _, item := range req.Items {
		evt.Items = append(evt.Items, struct {
			ProductID string
			Quantity  int
			Price     float64
		}{
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
			Price:     item.Price,
		})
	}

	// Сохраняем событие
	if err := c.es.Save(orderID, evt); err != nil {
		return domain.Order{}, err
	}

	// Обновляем проекцию (синхронно для простоты)
	c.proj.Apply(evt)

	// Возвращаем агрегат
	events, _ := c.es.Load(orderID)
	order := domain.NewOrderFromEvents(events)
	return *order, nil
}
