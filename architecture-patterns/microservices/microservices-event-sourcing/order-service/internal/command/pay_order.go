package command

import (
	"errors"
	"time"

	"order-service/internal/domain"
	"order-service/internal/event"
	"order-service/internal/eventstore"
)

type PayOrderCommand struct {
	es *eventstore.EventStore
}

func NewPayOrderCommand(es *eventstore.EventStore) *PayOrderCommand {
	return &PayOrderCommand{es: es}
}

type PayOrderRequest struct {
	OrderID string `json:"order_id"`
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
	order := &domain.Order{}
	for _, e := range events {
		order.ApplyEvent(e)
	}

	// Проверяем, можно ли оплатить
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

	// Обновляем агрегат и возвращаем
	order.ApplyEvent(evt)
	return *order, nil
}
