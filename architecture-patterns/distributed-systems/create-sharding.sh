#!/bin/bash
# Sharding — распределение данных по ключу

set -e

PROJECT_NAME="sharding"

mkdir -p "$PROJECT_NAME"

cd "$PROJECT_NAME"

cat > go.mod <<'EOF'
module sharding

go 1.23
EOF

cat > main.go <<'EOF'
package main

import (
	"fmt"
	"hash/fnv"
	"sync"
)

type Shard struct {
	ID   int
	Data map[string]string
	mu   sync.RWMutex
}

func NewShard(id int) *Shard {
	return &Shard{
		ID:   id,
		Data: make(map[string]string),
	}
}

func (s *Shard) Write(key, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Data[key] = value
	fmt.Printf("[Shard %d] wrote %s=%s\n", s.ID, key, value)
}

func (s *Shard) Read(key string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Data[key]
}

type ShardManager struct {
	shards []*Shard
}

func NewShardManager(numShards int) *ShardManager {
	shards := make([]*Shard, numShards)
	for i := 0; i < numShards; i++ {
		shards[i] = NewShard(i)
	}
	return &ShardManager{shards: shards}
}

func (sm *ShardManager) getShard(key string) *Shard {
	h := fnv.New32a()
	h.Write([]byte(key))
	idx := int(h.Sum32()) % len(sm.shards)
	return sm.shards[idx]
}

func (sm *ShardManager) Write(key, value string) {
	shard := sm.getShard(key)
	shard.Write(key, value)
}

func (sm *ShardManager) Read(key string) string {
	shard := sm.getShard(key)
	return shard.Read(key)
}

func main() {
	sm := NewShardManager(3)

	keys := []string{"order1", "order2", "order3", "product1", "product2"}
	for _, k := range keys {
		sm.Write(k, "value-"+k)
	}

	for _, k := range keys {
		fmt.Printf("Read %s = %s\n", k, sm.Read(k))
	}
}
EOF

echo "✅ Sharding demo created at ./$PROJECT_NAME"