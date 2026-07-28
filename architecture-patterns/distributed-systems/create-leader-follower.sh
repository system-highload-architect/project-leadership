#!/bin/bash
# Leader-Follower — выбор лидера и репликация

set -e

PROJECT_NAME="leader-follower"

mkdir -p "$PROJECT_NAME"

cd "$PROJECT_NAME"

cat > go.mod <<'EOF'
module leader-follower

go 1.23
EOF

cat > main.go <<'EOF'
package main

import (
	"fmt"
	"sync"
	"time"
)

type Node struct {
	ID       int
	IsLeader bool
	Data     map[string]string
	mu       sync.RWMutex
}

func NewNode(id int) *Node {
	return &Node{
		ID:       id,
		IsLeader: false,
		Data:     make(map[string]string),
	}
}

func (n *Node) SetLeader() {
	n.IsLeader = true
	fmt.Printf("[Node %d] I am the LEADER\n", n.ID)
}

func (n *Node) Write(key, value string) {
	if !n.IsLeader {
		fmt.Printf("[Node %d] ERROR: not leader, cannot write\n", n.ID)
		return
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	n.Data[key] = value
	fmt.Printf("[Node %d] Wrote %s=%s\n", n.ID, key, value)
}

func (n *Node) Read(key string) string {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.Data[key]
}

func (n *Node) ReplicateFrom(leader *Node) {
	if leader == nil || !leader.IsLeader {
		return
	}
	leader.mu.RLock()
	defer leader.mu.RUnlock()
	n.mu.Lock()
	defer n.mu.Unlock()
	for k, v := range leader.Data {
		n.Data[k] = v
	}
	fmt.Printf("[Node %d] Replicated data from leader\n", n.ID)
}

func main() {
	nodes := []*Node{NewNode(1), NewNode(2), NewNode(3)}

	// Выбираем лидера (первый узел)
	leader := nodes[0]
	leader.SetLeader()

	// Лидер пишет данные
	leader.Write("order1", "active")
	leader.Write("order2", "paid")

	// Другие узлы реплицируются
	for _, n := range nodes[1:] {
		n.ReplicateFrom(leader)
	}

	// Проверяем чтение у последователей
	for _, n := range nodes {
		fmt.Printf("[Node %d] order1 = %s\n", n.ID, n.Read("order1"))
	}

	// Попытка записи у последователя
	nodes[1].Write("order3", "shipped")
}
EOF

echo "✅ Leader-Follower demo created at ./$PROJECT_NAME"