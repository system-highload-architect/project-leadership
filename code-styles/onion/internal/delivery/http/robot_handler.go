package http

import (
	"encoding/json"
	"net/http"
	"strings"

	"onion-architecture/internal/application/robot"
)

type RobotHandler struct {
	getRobotUC   *robot.GetRobotUseCase
	assignTaskUC *robot.AssignTaskUseCase
}

func NewRobotHandler(
	getUC *robot.GetRobotUseCase,
	assignUC *robot.AssignTaskUseCase,
) *RobotHandler {
	return &RobotHandler{
		getRobotUC:   getUC,
		assignTaskUC: assignUC,
	}
}

// GetRobot — GET /robots/{id}
func (h *RobotHandler) GetRobot(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/robots/")
	if id == "" {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}
	robot, err := h.getRobotUC.Execute(r.Context(), robot.GetRobotInput{ID: id})
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(robot)
}

// AssignTask — POST /robots/{id}/tasks
func (h *RobotHandler) AssignTask(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/robots/")
	id = strings.TrimSuffix(id, "/tasks")
	if id == "" {
		http.Error(w, "robot id is required", http.StatusBadRequest)
		return
	}
	var req struct {
		TaskID      string `json:"task_id"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if err := h.assignTaskUC.Execute(r.Context(), robot.AssignTaskInput{
		RobotID:     id,
		TaskID:      req.TaskID,
		Description: req.Description,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "task assigned"})
}
