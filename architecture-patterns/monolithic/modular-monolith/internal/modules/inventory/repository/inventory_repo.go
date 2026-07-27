package repository

import (
	"errors"

	"modular-monolith/internal/infrastructure/db"
	"modular-monolith/internal/shared/domain"
)

type InventoryRepository struct {
	db *db.InMemoryDB
}

func NewInventoryRepository(db *db.InMemoryDB) *InventoryRepository {
	return &InventoryRepository{db: db}
}

func (r *InventoryRepository) FindProduct(id string) (domain.Product, error) {
	r.db.Mu.RLock()
	defer r.db.Mu.RUnlock()
	product, ok := r.db.Products[id]
	if !ok {
		return domain.Product{}, errors.New("product not found")
	}
	return product, nil
}

func (r *InventoryRepository) SaveProduct(product domain.Product) error {
	r.db.Mu.Lock()
	defer r.db.Mu.Unlock()
	r.db.Products[product.ID] = product
	return nil
}
