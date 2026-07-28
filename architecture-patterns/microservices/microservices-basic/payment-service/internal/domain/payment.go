package domain

import "time"

type PaymentStatus string

const (
	PaymentStatusPending PaymentStatus = "pending"
	PaymentStatusSuccess PaymentStatus = "success"
	PaymentStatusFailed  PaymentStatus = "failed"
)

type Payment struct {
	ID        string
	OrderID   string
	Amount    float64
	Customer  string
	Status    PaymentStatus
	CreatedAt time.Time
}
