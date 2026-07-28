#!/bin/bash
# Peer-to-Peer — прямое общение между узлами

set -e

PROJECT_NAME="peer-to-peer"

mkdir -p "$PROJECT_NAME"

cd "$PROJECT_NAME"

cat > go.mod <<'EOF'
module peer-to-peer

go 1.23
EOF

cat > main.go <<'EOF'
package main

import (
	"fmt"
	"sync"
)

type Peer struct {
	ID   int
	Data map[string]string
	mu   sync.RWMutex
}

func NewPeer(id int) *Peer {
	return &Peer{
		ID:   id,
		Data: make(map[string]string),
	}
}

func (p *Peer) Send(other *Peer, key, value string) {
	p.mu.Lock()
	p.Data[key] = value
	p.mu.Unlock()
	fmt.Printf("[Peer %d] sends %s=%s to Peer %d\n", p.ID, key, value, other.ID)
	other.Receive(key, value)
}

func (p *Peer) Receive(key, value string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Data[key] = value
	fmt.Printf("[Peer %d] received %s=%s\n", p.ID, key, value)
}

func (p *Peer) Print() {
	p.mu.RLock()
	defer p.mu.RUnlock()
	fmt.Printf("[Peer %d] data: %v\n", p.ID, p.Data)
}

func main() {
	p1 := NewPeer(1)
	p2 := NewPeer(2)
	p3 := NewPeer(3)

	// Обмен сообщениями
	p1.Send(p2, "order1", "active")
	p2.Send(p3, "order2", "paid")
	p3.Send(p1, "order3", "shipped")

	p1.Print()
	p2.Print()
	p3.Print()
}
EOF

echo "✅ Peer-to-Peer demo created at ./$PROJECT_NAME"