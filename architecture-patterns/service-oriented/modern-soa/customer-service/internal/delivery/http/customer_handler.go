package http

import (
	"encoding/json"
	"net/http"
	"strings"

	"customer-service/internal/service"
)

type CustomerHandler struct {
	svc *service.CustomerService
}

func NewCustomerHandler(svc *service.CustomerService) *CustomerHandler {
	return &CustomerHandler{svc: svc}
}

func (h *CustomerHandler) GetCustomer(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/customers/")
	if id == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}
	customer, err := h.svc.GetCustomer(id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(customer)
}
