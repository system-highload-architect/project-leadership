package repository

import (
	"errors"
	"sync"

	"classic-monolith/internal/domain"
)

var (
	Products   = make(map[string]domain.Product)
	ProductsMu sync.RWMutex
)

type ProductRepository struct{}

func NewProductRepository() *ProductRepository {
	return &ProductRepository{}
}

func (r *ProductRepository) Save(product domain.Product) error {
	ProductsMu.Lock()
	defer ProductsMu.Unlock()
	Products[product.ID] = product
	return nil
}

func (r *ProductRepository) FindByID(id string) (domain.Product, error) {
	ProductsMu.RLock()
	defer ProductsMu.RUnlock()
	product, ok := Products[id]
	if !ok {
		return domain.Product{}, errors.New("product not found")
	}
	return product, nil
}

func (r *ProductRepository) DecreaseStock(productID string, quantity int) error {
	ProductsMu.Lock()
	defer ProductsMu.Unlock()
	product, ok := Products[productID]
	if !ok {
		return errors.New("product not found")
	}
	if product.Stock < quantity {
		return errors.New("not enough stock")
	}
	product.Stock -= quantity
	Products[productID] = product
	return nil
}
