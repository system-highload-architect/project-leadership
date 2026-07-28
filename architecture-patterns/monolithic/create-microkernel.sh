#!/bin/bash
# Microkernel architecture (плагинная)

set -e

PROJECT_NAME="microkernel"

mkdir -p "$PROJECT_NAME/cmd/server"
mkdir -p "$PROJECT_NAME/internal/core"
mkdir -p "$PROJECT_NAME/internal/plugins/greeting"
mkdir -p "$PROJECT_NAME/internal/plugins/calculator"
mkdir -p "$PROJECT_NAME/pkg/logger"

cd "$PROJECT_NAME"

cat > go.mod <<'EOF'
module microkernel

go 1.23
EOF

cat > cmd/server/main.go <<'EOF'
package main

import (
	"log"
	"net/http"

	"microkernel/internal/core"
	"microkernel/internal/plugins/calculator"
	"microkernel/internal/plugins/greeting"
)

func main() {
	// Создаём ядро
	kernel := core.NewKernel()

	// Регистрируем плагины
	kernel.RegisterPlugin(greeting.NewPlugin())
	kernel.RegisterPlugin(calculator.NewPlugin())

	// Запускаем HTTP-сервер ядра
	mux := http.NewServeMux()
	mux.HandleFunc("GET /plugin/{name}", kernel.HandlePlugin)

	log.Println("Microkernel server starting on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}
EOF

cat > internal/core/kernel.go <<'EOF'
package core

import (
	"encoding/json"
	"net/http"
	"strings"
)

// Plugin — интерфейс, который должны реализовать все плагины
type Plugin interface {
	Name() string
	Execute(input map[string]interface{}) (map[string]interface{}, error)
}

// Kernel — ядро системы
type Kernel struct {
	plugins map[string]Plugin
}

func NewKernel() *Kernel {
	return &Kernel{
		plugins: make(map[string]Plugin),
	}
}

func (k *Kernel) RegisterPlugin(p Plugin) {
	k.plugins[p.Name()] = p
}

func (k *Kernel) HandlePlugin(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/plugin/")
	if name == "" {
		http.Error(w, "plugin name is required", http.StatusBadRequest)
		return
	}
	plugin, ok := k.plugins[name]
	if !ok {
		http.Error(w, "plugin not found", http.StatusNotFound)
		return
	}
	var input map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		input = make(map[string]interface{})
	}
	result, err := plugin.Execute(input)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
EOF

cat > internal/plugins/greeting/plugin.go <<'EOF'
package greeting

type Plugin struct{}

func NewPlugin() *Plugin {
	return &Plugin{}
}

func (p *Plugin) Name() string {
	return "greeting"
}

func (p *Plugin) Execute(input map[string]interface{}) (map[string]interface{}, error) {
	name, ok := input["name"].(string)
	if !ok || name == "" {
		name = "World"
	}
	return map[string]interface{}{
		"message": "Hello, " + name + "!",
	}, nil
}
EOF

cat > internal/plugins/calculator/plugin.go <<'EOF'
package calculator

import (
	"fmt"
)

type Plugin struct{}

func NewPlugin() *Plugin {
	return &Plugin{}
}

func (p *Plugin) Name() string {
	return "calculator"
}

func (p *Plugin) Execute(input map[string]interface{}) (map[string]interface{}, error) {
	a, okA := input["a"].(float64)
	b, okB := input["b"].(float64)
	if !okA || !okB {
		return nil, fmt.Errorf("missing 'a' or 'b' (numbers)")
	}
	op, _ := input["op"].(string)
	var result float64
	switch op {
	case "add":
		result = a + b
	case "sub":
		result = a - b
	case "mul":
		result = a * b
	case "div":
		if b == 0 {
			return nil, fmt.Errorf("division by zero")
		}
		result = a / b
	default:
		return nil, fmt.Errorf("unknown op: %s", op)
	}
	return map[string]interface{}{
		"result": result,
	}, nil
}
EOF

cat > pkg/logger/logger.go <<'EOF'
package logger

import "log"

var (
	Info  = log.New(log.Writer(), "[INFO] ", log.LstdFlags)
	Error = log.New(log.Writer(), "[ERROR] ", log.LstdFlags)
)
EOF

echo "✅ Microkernel project created at ./$PROJECT_NAME"
cd ..