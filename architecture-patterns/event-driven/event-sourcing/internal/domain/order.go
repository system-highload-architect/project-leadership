package domain

import (
	"time"

	"event-sourcing/internal/event"
)

type OrderStatus string

const (
	OrderStatusNew  OrderStatus = "new"
	OrderStatusPaid OrderStatus = "paid"
)

type Order struct {
	ID         string
	CustomerID string
	Items      []event.OrderItem
	Status     OrderStatus
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func (o *Order) ApplyEvent(e event.Event) {
	switch ev := e.(type) {
	case *event.OrderCreated:
		o.ID = ev.AggregateID
		o.CustomerID = ev.CustomerID
		o.Items = ev.Items
		o.Status = OrderStatusNew
		o.CreatedAt = ev.Timestamp
		o.UpdatedAt = ev.Timestamp
	case *event.OrderPaid:
		o.Status = OrderStatusPaid
		o.UpdatedAt = ev.Timestamp
	}
}
