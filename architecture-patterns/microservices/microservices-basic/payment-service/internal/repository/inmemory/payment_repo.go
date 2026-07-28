package inmemory

import (
	"errors"
	"sync"

	"payment-service/internal/domain"
)

type PaymentRepository struct {
	mu       sync.RWMutex
	payments map[string]domain.Payment
}

func NewPaymentRepository() *PaymentRepository {
	return &PaymentRepository{
		payments: make(map[string]domain.Payment),
	}
}

func (r *PaymentRepository) Save(payment domain.Payment) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.payments[payment.ID] = payment
	return nil
}

func (r *PaymentRepository) FindByOrderID(orderID string) (domain.Payment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, p := range r.payments {
		if p.OrderID == orderID {
			return p, nil
		}
	}
	return domain.Payment{}, errors.New("payment not found")
}
