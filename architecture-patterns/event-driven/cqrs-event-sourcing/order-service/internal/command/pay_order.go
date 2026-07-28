package command

import (
	"errors"
	"time"

	"order-service/internal/domain"
	"order-service/internal/event"
	"order-service/internal/eventstore"
	"order-service/internal/projection"
)

type PayOrderCommand struct {
	es   *eventstore.EventStore
	proj *projection.Projection
}

func NewPayOrderCommand(es *eventstore.EventStore, proj *projection.Projection) *PayOrderCommand {
	return &PayOrderCommand{es: es, proj: proj}
}

type PayOrderRequest struct {
	OrderID string
}

func (c *PayOrderCommand) Execute(req PayOrderRequest) (domain.Order, error) {
	if req.OrderID == "" {
		return domain.Order{}, errors.New("order id is required")
	}

	// Загружаем события
	events, err := c.es.Load(req.OrderID)
	if err != nil {
		return domain.Order{}, err
	}

	// Восстанавливаем агрегат
	order := domain.NewOrderFromEvents(events)
	if order.Status != domain.OrderStatusNew {
		return domain.Order{}, errors.New("order cannot be paid in current status")
	}

	// Создаём событие оплаты
	evt := &event.OrderPaid{
		BaseEvent: event.BaseEvent{
			AggregateID: req.OrderID,
			Timestamp:   time.Now(),
		},
	}

	// Сохраняем событие
	if err := c.es.Save(req.OrderID, evt); err != nil {
		return domain.Order{}, err
	}

	// Обновляем проекцию
	c.proj.Apply(evt)

	// Возвращаем обновлённый агрегат
	events, _ = c.es.Load(req.OrderID)
	order = domain.NewOrderFromEvents(events)
	return *order, nil
}
