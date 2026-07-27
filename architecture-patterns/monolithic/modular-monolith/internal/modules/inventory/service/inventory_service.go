package service

import (
	"errors"

	"modular-monolith/internal/modules/inventory/repository"
	"modular-monolith/internal/shared/domain"
)

type InventoryService struct {
	repo *repository.InventoryRepository
}

func NewInventoryService(repo *repository.InventoryRepository) *InventoryService {
	return &InventoryService{repo: repo}
}

func (s *InventoryService) ReserveProduct(productID string, quantity int) error {
	product, err := s.repo.FindProduct(productID)
	if err != nil {
		return err
	}
	if product.Stock < quantity {
		return errors.New("not enough stock")
	}
	product.Stock -= quantity
	return s.repo.SaveProduct(product)
}

func (s *InventoryService) GetProduct(id string) (domain.Product, error) {
	return s.repo.FindProduct(id)
}

// SaveProduct сохраняет продукт через репозиторий
func (s *InventoryService) SaveProduct(product domain.Product) error {
	return s.repo.SaveProduct(product)
}
