package db

import (
	"sync"

	"modular-monolith/internal/shared/domain"
)

type InMemoryDB struct {
	Mu       sync.RWMutex
	Orders   map[string]domain.Order
	Products map[string]domain.Product
}

func NewInMemoryDB() *InMemoryDB {
	return &InMemoryDB{
		Orders:   make(map[string]domain.Order),
		Products: make(map[string]domain.Product),
	}
}
