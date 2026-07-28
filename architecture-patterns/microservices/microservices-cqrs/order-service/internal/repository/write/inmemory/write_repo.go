package inmemory

import (
	"errors"
	"sync"

	"order-service/internal/domain"
)

type WriteRepository struct {
	mu     sync.RWMutex
	orders map[string]domain.Order
}

func NewWriteRepository() *WriteRepository {
	return &WriteRepository{
		orders: make(map[string]domain.Order),
	}
}

func (r *WriteRepository) Save(order domain.Order) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.orders[order.ID] = order
	return nil
}

func (r *WriteRepository) FindByID(id string) (domain.Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	order, ok := r.orders[id]
	if !ok {
		return domain.Order{}, errors.New("order not found")
	}
	return order, nil
}
