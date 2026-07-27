package http

import (
	"encoding/json"
	"net/http"
	"strings"

	"layered-architecture/internal/domain"
	"layered-architecture/internal/service"
)

// RobotHandler — слой представления (HTTP)
type RobotHandler struct {
	robotSvc *service.RobotService
}

func NewRobotHandler(svc *service.RobotService) *RobotHandler {
	return &RobotHandler{robotSvc: svc}
}

// GetRobot — GET /robots/{id}
func (h *RobotHandler) GetRobot(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/robots/")
	if id == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}
	robot, err := h.robotSvc.GetRobot(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(robot)
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
	if err := h.robotSvc.UpdateStatus(r.Context(), id, req.Status); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}
