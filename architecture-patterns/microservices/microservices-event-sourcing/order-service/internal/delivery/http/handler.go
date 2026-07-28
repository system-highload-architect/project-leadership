package http

import (
	"encoding/json"
	"net/http"
	"strings"

	"order-service/internal/command"
	"order-service/internal/domain"
	"order-service/internal/eventstore"
)

type Handler struct {
	createOrderCmd *command.CreateOrderCommand
	payOrderCmd    *command.PayOrderCommand
	es             *eventstore.EventStore
}

func NewHandler(
	createCmd *command.CreateOrderCommand,
	payCmd *command.PayOrderCommand,
) *Handler {
	return &Handler{
		createOrderCmd: createCmd,
		payOrderCmd:    payCmd,
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

func (h *Handler) PayOrder(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(strings.TrimSuffix(r.URL.Path, "/pay"), "/orders/")
	if id == "" {
		http.Error(w, "order id is required", http.StatusBadRequest)
		return
	}
	order, err := h.payOrderCmd.Execute(command.PayOrderRequest{OrderID: id})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(order)
}

func (h *Handler) GetOrder(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/orders/")
	if id == "" {
		http.Error(w, "order id is required", http.StatusBadRequest)
		return
	}
	events, err := h.es.Load(id)
	if err != nil {
		http.Error(w, "order not found", http.StatusNotFound)
		return
	}
	order := &domain.Order{}
	for _, e := range events {
		order.ApplyEvent(e)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(order)
}
