package http

import (
	"encoding/json"
	"net/http"
	"strings"

	"order-service/internal/command"
	"order-service/internal/domain"
	"order-service/internal/query"
)

type Handler struct {
	createOrderCmd  *command.CreateOrderCommand
	updateStatusCmd *command.UpdateStatusCommand
	getOrderQuery   *query.GetOrderQuery
	listOrdersQuery *query.ListOrdersQuery
}

func NewHandler(
	createCmd *command.CreateOrderCommand,
	updateCmd *command.UpdateStatusCommand,
	getQuery *query.GetOrderQuery,
	listQuery *query.ListOrdersQuery,
) *Handler {
	return &Handler{
		createOrderCmd:  createCmd,
		updateStatusCmd: updateCmd,
		getOrderQuery:   getQuery,
		listOrdersQuery: listQuery,
	}
}

func (h *Handler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	var req command.CreateOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	order, err := h.createOrderCmd.Execute(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(order)
}

func (h *Handler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(strings.TrimSuffix(r.URL.Path, "/status"), "/orders/")
	if id == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}
	var req struct {
		Status domain.OrderStatus `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	err := h.updateStatusCmd.Execute(command.UpdateStatusRequest{
		ID:     id,
		Status: req.Status,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}

func (h *Handler) GetOrder(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/orders/")
	if id == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}
	order, err := h.getOrderQuery.Execute(query.GetOrderRequest{ID: id})
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(order)
}

func (h *Handler) ListOrders(w http.ResponseWriter, r *http.Request) {
	orders, err := h.listOrdersQuery.Execute()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(orders)
}
