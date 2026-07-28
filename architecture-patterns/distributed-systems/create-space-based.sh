#!/bin/bash
# Space-Based — общая распределённая память

set -e

PROJECT_NAME="space-based"

mkdir -p "$PROJECT_NAME"

cd "$PROJECT_NAME"

cat > go.mod <<'EOF'
module space-based

go 1.23
EOF

cat > main.go <<'EOF'
package main

import (
	"fmt"
	"sync"
)

type Space struct {
	Data map[string]string
	mu   sync.RWMutex
}

var (
	space *Space
	once  sync.Once
)

func GetSpace() *Space {
	once.Do(func() {
		space = &Space{Data: make(map[string]string)}
	})
	return space
}

func (s *Space) Put(key, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Data[key] = value
	fmt.Printf("[Space] put %s=%s\n", key, value)
}

func (s *Space) Get(key string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Data[key]
}

func main() {
	space := GetSpace()

	// Несколько узлов обращаются к общему пространству
	var wg sync.WaitGroup
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			key := fmt.Sprintf("order%d", id)
			space.Put(key, fmt.Sprintf("value-%d", id))
			fmt.Printf("[Node %d] read %s = %s\n", id, key, space.Get(key))
		}(i)
	}
	wg.Wait()
}
EOF

echo "✅ Space-Based demo created at ./$PROJECT_NAME"