package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"shared/db"
	database "shared/db"
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
		db.Mu.Lock()
		db.Orders[orderID] = database.Order{ID: orderID, Status: "new"}
		db.Mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"id": orderID})
	})

	http.HandleFunc("GET /orders/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		db.Mu.RLock()
		order, ok := db.Orders[id]
		db.Mu.RUnlock()
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
