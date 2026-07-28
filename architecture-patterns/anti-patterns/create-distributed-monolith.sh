#!/bin/bash
# Distributed Monolith — анти-паттерн с общей БД и жёсткой связью

set -e

PROJECT_NAME="distributed-monolith"

mkdir -p "$PROJECT_NAME/order-service/cmd/server"
mkdir -p "$PROJECT_NAME/order-service/internal"
mkdir -p "$PROJECT_NAME/payment-service/cmd/server"
mkdir -p "$PROJECT_NAME/payment-service/internal"
mkdir -p "$PROJECT_NAME/shared/db"

cd "$PROJECT_NAME"

# ===== Shared DB =====
cat > shared/db/db.go <<'EOF'
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
	mu       sync.RWMutex
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
EOF

cat > shared/go.mod <<'EOF'
module shared

go 1.23
EOF

# ===== Order Service =====
cd order-service
cat > go.mod <<'EOF'
module order-service

go 1.23

replace shared => ../shared
EOF

cat > cmd/server/main.go <<'EOF'
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"shared/db"
)

func main() {
	db := db.GetDB()

	http.HandleFunc("POST /orders", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			CustomerID string `json:"customer_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		orderID := "ord-" + time.Now().Format("20060102150405")
		db.mu.Lock()
		db.Orders[orderID] = db.Order{ID: orderID, Status: "new"}
		db.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"id": orderID})
	})

	http.HandleFunc("GET /orders/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		db.mu.RLock()
		order, ok := db.Orders[id]
		db.mu.RUnlock()
		if !ok {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(order)
	})

	log.Println("Order Service (distributed monolith) on :8081")
	log.Fatal(http.ListenAndServe(":8081", nil))
}
EOF
cd ..

# ===== Payment Service =====
cd payment-service
cat > go.mod <<'EOF'
module payment-service

go 1.23

replace shared => ../shared
EOF

cat > cmd/server/main.go <<'EOF`
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"shared/db"
)

func main() {
	db := db.GetDB()

	http.HandleFunc("POST /payments", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			OrderID string  `json:"order_id"`
			Amount  float64 `json:"amount"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		// Проверяем, существует ли заказ (прямое чтение из общей БД!)
		db.mu.RLock()
		_, ok := db.Orders[req.OrderID]
		db.mu.RUnlock()
		if !ok {
			http.Error(w, "order not found", http.StatusBadRequest)
			return
		}
		payment := db.Payment{
			ID:      "pay-" + time.Now().Format("20060102150405"),
			OrderID: req.OrderID,
			Amount:  req.Amount,
		}
		db.mu.Lock()
		db.Payments[payment.ID] = payment
		// Обновляем статус заказа прямо из платёжного сервиса!
		order := db.Orders[req.OrderID]
		order.Status = "paid"
		db.Orders[req.OrderID] = order
		db.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(payment)
	})

	log.Println("Payment Service (distributed monolith) on :8082")
	log.Fatal(http.ListenAndServe(":8082", nil))
}
EOF
cd ..

echo "✅ Distributed Monolith demo created at ./$PROJECT_NAME"