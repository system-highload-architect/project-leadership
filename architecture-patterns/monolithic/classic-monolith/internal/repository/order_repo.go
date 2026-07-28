package repository

import (
	"errors"
	"sync"

	"classic-monolith/internal/domain"
)

// Глобальное состояние — классический подход
var (
	Orders   = make(map[string]domain.Order)
	OrdersMu sync.RWMutex
)

type OrderRepository struct{}

func NewOrderRepository() *OrderRepository {
	return &OrderRepository{}
}

func (r *OrderRepository) Save(order domain.Order) error {
	OrdersMu.Lock()
	defer OrdersMu.Unlock()
	Orders[order.ID] = order
	return nil
}

func (r *OrderRepository) FindByID(id string) (domain.Order, error) {
	OrdersMu.RLock()
	defer OrdersMu.RUnlock()
	order, ok := Orders[id]
	if !ok {
		return domain.Order{}, errors.New("order not found")
	}
	return order, nil
}
