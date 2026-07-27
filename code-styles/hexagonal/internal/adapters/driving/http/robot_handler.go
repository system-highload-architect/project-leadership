package http

import (
	"encoding/json"
	"net/http"
	"strings"

	"hexagonal-architecture/internal/core/domain"
	"hexagonal-architecture/internal/core/usecases"
)

// RobotHandler — Driving Adapter (предоставляемый порт)
// Преобразует HTTP-запросы в вызовы Use Cases
type RobotHandler struct {
	getRobotUC     *usecases.GetRobotUseCase
	updateStatusUC *usecases.UpdateStatusUseCase
}

func NewRobotHandler(
	getUC *usecases.GetRobotUseCase,
	updateUC *usecases.UpdateStatusUseCase,
) *RobotHandler {
	return &RobotHandler{
		getRobotUC:     getUC,
		updateStatusUC: updateUC,
	}
}

// GetRobot — GET /robots/{id}
func (h *RobotHandler) GetRobot(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/robots/")
	if id == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}
	output, err := h.getRobotUC.Execute(r.Context(), usecases.GetRobotInput{ID: id})
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(output)
}

// UpdateStatus — PUT /robots/{id}/status
func (h *RobotHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(strings.TrimSuffix(r.URL.Path, "/status"), "/robots/")
	if id == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}
	var req struct {
		Status domain.RobotStatus `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if err := h.updateStatusUC.Execute(r.Context(), usecases.UpdateStatusInput{
		ID:     id,
		Status: req.Status,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}
