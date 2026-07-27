package api

import (
	"encoding/json"
	"net/http"

	"modular-monolith/internal/modules/order/service"
	invService "modular-monolith/internal/modules/inventory/service"
	"modular-monolith/internal/shared/domain"
)

type OrderHandler struct {
	orderSvc *service.OrderService
	invSvc   *invService.InventoryService
}

func NewOrderHandler(orderSvc *service.OrderService, invSvc *invService.InventoryService) *OrderHandler {
	return &OrderHandler{
		orderSvc: orderSvc,
		invSvc:   invSvc,
	}
}

func (h *OrderHandler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CustomerID string             `json:"customer_id"`
		Items      []domain.OrderItem `json:"items"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	for _, item := range req.Items {
		if err := h.invSvc.ReserveProduct(item.ProductID, item.Quantity); err != nil {
			http.Error(w, "inventory error: "+err.Error(), http.StatusBadRequest)
			return
		}
	}
	order, err := h.orderSvc.CreateOrder(req.CustomerID, req.Items)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(order)
}

func (h *OrderHandler) GetOrder(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}
	order, err := h.orderSvc.GetOrder(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(order)
}
