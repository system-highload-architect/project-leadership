package repository

import (
	"errors"

	"modular-monolith/internal/infrastructure/db"
	"modular-monolith/internal/shared/domain"
)

type OrderRepository struct {
	db *db.InMemoryDB
}

func NewOrderRepository(db *db.InMemoryDB) *OrderRepository {
	return &OrderRepository{db: db}
}

func (r *OrderRepository) Save(order domain.Order) error {
	r.db.Mu.Lock()
	defer r.db.Mu.Unlock()
	r.db.Orders[order.ID] = order
	return nil
}

func (r *OrderRepository) FindByID(id string) (domain.Order, error) {
	r.db.Mu.RLock()
	defer r.db.Mu.RUnlock()
	order, ok := r.db.Orders[id]
	if !ok {
		return domain.Order{}, errors.New("order not found")
	}
	return order, nil
}
