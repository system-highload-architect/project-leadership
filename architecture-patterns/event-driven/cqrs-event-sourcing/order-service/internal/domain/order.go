package domain

import (
	"time"

	"order-service/internal/event"
)

type OrderStatus string

const (
	OrderStatusNew      OrderStatus = "new"
	OrderStatusPaid     OrderStatus = "paid"
	OrderStatusShipped  OrderStatus = "shipped"
	OrderStatusComplete OrderStatus = "complete"
	OrderStatusCanceled OrderStatus = "canceled"
)

type Order struct {
	ID         string
	CustomerID string
	Items      []OrderItem
	Status     OrderStatus
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type OrderItem struct {
	ProductID string
	Quantity  int
	Price     float64
}

// ApplyEvent применяет событие к агрегату
func (o *Order) ApplyEvent(e event.Event) {
	switch ev := e.(type) {
	case *event.OrderCreated:
		o.ID = ev.AggregateID
		o.CustomerID = ev.CustomerID
		o.Items = make([]OrderItem, len(ev.Items))
		for i, item := range ev.Items {
			o.Items[i] = OrderItem{
				ProductID: item.ProductID,
				Quantity:  item.Quantity,
				Price:     item.Price,
			}
		}
		o.Status = OrderStatusNew
		o.CreatedAt = ev.Timestamp
		o.UpdatedAt = ev.Timestamp
	case *event.OrderPaid:
		o.Status = OrderStatusPaid
		o.UpdatedAt = ev.Timestamp
	}
}

// Replay восстанавливает агрегат из списка событий
func (o *Order) Replay(events []event.Event) {
	for _, e := range events {
		o.ApplyEvent(e)
	}
}

// NewOrderFromEvents создаёт агрегат из событий
func NewOrderFromEvents(events []event.Event) *Order {
	o := &Order{}
	o.Replay(events)
	return o
}
