package event

import "time"

type Event interface {
	GetAggregateID() string
	GetTimestamp() time.Time
}

type BaseEvent struct {
	AggregateID string
	Timestamp   time.Time
}

func (e BaseEvent) GetAggregateID() string { return e.AggregateID }
func (e BaseEvent) GetTimestamp() time.Time { return e.Timestamp }

type OrderCreated struct {
	BaseEvent
	CustomerID string
	Items      []struct {
		ProductID string
		Quantity  int
		Price     float64
	}
}

type OrderPaid struct {
	BaseEvent
}
