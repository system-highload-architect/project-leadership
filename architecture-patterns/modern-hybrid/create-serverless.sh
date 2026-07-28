#!/bin/bash
# Serverless (FaaS) — функция как сервис на Go

set -e

PROJECT_NAME="serverless"

mkdir -p "$PROJECT_NAME/cmd/function"
mkdir -p "$PROJECT_NAME/internal/handler"

cd "$PROJECT_NAME"

cat > go.mod <<'EOF'
module serverless

go 1.23
EOF

cat > cmd/function/main.go <<'EOF'
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"serverless/internal/handler"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/", handler.Handle)

	log.Printf("Function listening on :%s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
EOF

cat > internal/handler/handler.go <<'EOF'
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
EOF

echo "✅ Serverless project created at ./$PROJECT_NAME"