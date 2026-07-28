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
