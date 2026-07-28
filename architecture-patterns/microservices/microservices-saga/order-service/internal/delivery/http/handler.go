package http

import (
	"encoding/json"
	"net/http"
	"time"

	"order-service/internal/saga"
)

type Handler struct {
	orchestrator *saga.Orchestrator
}

func NewHandler(orchestrator *saga.Orchestrator) *Handler {
	return &Handler{orchestrator: orchestrator}
}

func (h *Handler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CustomerID string `json:"customer_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	// Генерируем ID заказа
	orderID := "ord-" + time.Now().Format("20060102150405")

	// Данные для саги
	data := map[string]interface{}{
		"orderID":    orderID,
		"customerID": req.CustomerID,
	}

	// Запускаем сагу
	err := h.orchestrator.Execute(r.Context(), data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{
		"order_id": orderID,
		"status":   "created",
	})
}
