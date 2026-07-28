package domain

import "time"

// OrderReadModel — денормализованная модель для чтения
type OrderReadModel struct {
	ID         string
	CustomerID string
	Items      []OrderItemReadModel
	Status     string
	Total      float64
	CreatedAt  time.Time
}

type OrderItemReadModel struct {
	ProductID string
	Quantity  int
	Price     float64
	Total     float64
}
