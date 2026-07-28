package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Request struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

type Response struct {
	Message string `json:"message"`
	Time    string `json:"time"`
}

func Handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "only POST allowed", http.StatusMethodNotAllowed)
		return
	}
	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if req.Name == "" {
		req.Name = "World"
	}
	resp := Response{
		Message: fmt.Sprintf("Hello, %s! You are %d years old.", req.Name, req.Age),
		Time:    time.Now().Format(time.RFC3339),
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
