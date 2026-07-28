package inmemory

import (
	"errors"
	"sync"

	"order-service/internal/domain"
)

type ReadRepository struct {
	mu     sync.RWMutex
	orders map[string]domain.OrderReadModel
}

func NewReadRepository() *ReadRepository {
	return &ReadRepository{
		orders: make(map[string]domain.OrderReadModel),
	}
}

func (r *ReadRepository) Save(order domain.OrderReadModel) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.orders[order.ID] = order
	return nil
}

func (r *ReadRepository) FindByID(id string) (domain.OrderReadModel, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	order, ok := r.orders[id]
	if !ok {
		return domain.OrderReadModel{}, errors.New("order not found")
	}
	return order, nil
}

func (r *ReadRepository) ListAll() ([]domain.OrderReadModel, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]domain.OrderReadModel, 0, len(r.orders))
	for _, order := range r.orders {
		result = append(result, order)
	}
	return result, nil
}
