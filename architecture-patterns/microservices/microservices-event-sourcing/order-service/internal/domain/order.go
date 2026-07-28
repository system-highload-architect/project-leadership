package domain

import (
	"order-service/internal/event"
	"time"
)

type OrderStatus string

const (
	OrderStatusNew      OrderStatus = "new"
	OrderStatusPaid     OrderStatus = "paid"
	OrderStatusShipped  OrderStatus = "shipped"
	OrderStatusComplete OrderStatus = "complete"
	OrderStatusCanceled OrderStatus = "canceled"
)

// Order — агрегат
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

// NewOrder создаёт новый заказ (без событий)
func NewOrder(id, customerID string, items []OrderItem) *Order {
	return &Order{
		ID:         id,
		CustomerID: customerID,
		Items:      items,
		Status:     OrderStatusNew,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
}

// ApplyEvent применяет событие к агрегату
func (o *Order) ApplyEvent(ev interface{}) {
	switch e := ev.(type) {
	case *event.OrderCreated:
		o.ID = e.AggregateID
		o.CustomerID = e.CustomerID
		if len(e.Items) > 0 {
			o.Items = make([]OrderItem, len(e.Items))
			for i := 0; i < len(e.Items); i++ {
				o.Items[i] = OrderItem{
					ProductID: e.Items[i].ProductID,
					Price:     e.Items[i].Price,
					Quantity:  e.Items[i].Quantity,
				}
			}
		}

		o.Status = OrderStatusNew
		o.CreatedAt = e.Timestamp
		o.UpdatedAt = e.Timestamp
	case *event.OrderPaid:
		o.Status = OrderStatusPaid
		o.UpdatedAt = e.Timestamp
	}
}
