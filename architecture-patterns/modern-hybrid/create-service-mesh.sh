#!/bin/bash
# Service Mesh — демонстрация маршрутизации между версиями сервиса

set -e

ROOT_DIR="service-mesh"

mkdir -p "$ROOT_DIR/service-v1/cmd/server"
mkdir -p "$ROOT_DIR/service-v1/internal"
mkdir -p "$ROOT_DIR/service-v2/cmd/server"
mkdir -p "$ROOT_DIR/service-v2/internal"
mkdir -p "$ROOT_DIR/sidecar/cmd/server"
mkdir -p "$ROOT_DIR/sidecar/internal/proxy"
mkdir -p "$ROOT_DIR/sidecar/internal/middleware"

cd "$ROOT_DIR"

# ===== Service v1 =====
cd service-v1
cat > go.mod <<'EOF'
module service-v1

go 1.23
EOF

cat > cmd/server/main.go <<'EOF'
package main

import (
	"encoding/json"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{
			"version": "v1",
			"message": "Hello from service v1",
		})
	})
	log.Println("Service v1 starting on :8081")
	log.Fatal(http.ListenAndServe(":8081", nil))
}
EOF
cd ..

# ===== Service v2 =====
cd service-v2
cat > go.mod <<'EOF'
module service-v2

go 1.23
EOF

cat > cmd/server/main.go <<'EOF'
package main

import (
	"encoding/json"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{
			"version": "v2",
			"message": "Hello from service v2 (new version!)",
		})
	})
	log.Println("Service v2 starting on :8082")
	log.Fatal(http.ListenAndServe(":8082", nil))
}
EOF
cd ..

# ===== Sidecar (прокси с маршрутизацией) =====
cd sidecar
cat > go.mod <<'EOF'
module sidecar

go 1.23
EOF

cat > cmd/server/main.go <<'EOF'
package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"math/rand"
)

func main() {
	// Целевые сервисы
	v1, _ := url.Parse("http://localhost:8081")
	v2, _ := url.Parse("http://localhost:8082")

	proxyV1 := httputil.NewSingleHostReverseProxy(v1)
	proxyV2 := httputil.NewSingleHostReverseProxy(v2)

	// Маршрутизация с балансировкой (50/50)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if rand.Intn(2) == 0 {
			proxyV1.ServeHTTP(w, r)
		} else {
			proxyV2.ServeHTTP(w, r)
		}
	})

	log.Println("Sidecar (Service Mesh) starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
EOF
cd ..

echo "✅ Service Mesh project created at ./$ROOT_DIR"