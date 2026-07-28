package db

import (
	"sync"
)

type Order struct {
	ID     string
	Status string
}

type Payment struct {
	ID      string
	OrderID string
	Amount  float64
}

type DB struct {
	Mu       sync.RWMutex
	Orders   map[string]Order
	Payments map[string]Payment
}

var (
	instance *DB
	once     sync.Once
)

func GetDB() *DB {
	once.Do(func() {
		instance = &DB{
			Orders:   make(map[string]Order),
			Payments: make(map[string]Payment),
		}
	})
	return instance
}
