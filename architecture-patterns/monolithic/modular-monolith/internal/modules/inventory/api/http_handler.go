package api

import (
	"encoding/json"
	"net/http"

	"modular-monolith/internal/modules/inventory/service"
	"modular-monolith/internal/shared/domain"
)

type InventoryHandler struct {
	invSvc *service.InventoryService
}

func NewInventoryHandler(invSvc *service.InventoryService) *InventoryHandler {
	return &InventoryHandler{invSvc: invSvc}
}

func (h *InventoryHandler) AddProduct(w http.ResponseWriter, r *http.Request) {
	var product domain.Product
	if err := json.NewDecoder(r.Body).Decode(&product); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	// Используем метод сервиса, а не обращаемся к repo напрямую
	if err := h.invSvc.SaveProduct(product); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(product)
}

func (h *InventoryHandler) GetProduct(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}
	product, err := h.invSvc.GetProduct(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(product)
}
