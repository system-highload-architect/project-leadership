#!/bin/bash
# Actor Model (простой пример на Go)

set -e

PROJECT_NAME="actor-model"

mkdir -p "$PROJECT_NAME/cmd/server"
mkdir -p "$PROJECT_NAME/internal/actor"

cd "$PROJECT_NAME"

cat > go.mod <<'EOF'
module actor-model

go 1.23
EOF

cat > cmd/server/main.go <<'EOF'
package main

import (
	"fmt"
	"time"

	"actor-model/internal/actor"
)

func main() {
	// Создаём систему акторов
	system := actor.NewSystem()

	// Создаём актора-принтера
	printer := system.Spawn("printer", func(msg interface{}) {
		fmt.Printf("[Printer] %v\n", msg)
	})

	// Создаём актора-счётчика
	counter := system.Spawn("counter", func(msg interface{}) {
		count, ok := msg.(int)
		if !ok {
			return
		}
		fmt.Printf("[Counter] Received: %d\n", count)
		if count < 5 {
			// Отправляем следующее число
			time.Sleep(500 * time.Millisecond)
			counter.Send(count + 1)
		} else {
			printer.Send("Counter finished!")
		}
	})

	// Стартуем счётчик с 1
	counter.Send(1)

	// Ждём завершения
	time.Sleep(5 * time.Second)
}
EOF

cat > internal/actor/system.go <<'EOF'
package actor

import "sync"

// Actor — структура актора
type Actor struct {
	ID       string
	Mailbox  chan interface{}
	Behavior func(interface{})
}

// System — система акторов
type System struct {
	actors map[string]*Actor
	mu     sync.RWMutex
}

func NewSystem() *System {
	return &System{
		actors: make(map[string]*Actor),
	}
}

func (s *System) Spawn(id string, behavior func(interface{})) *Actor {
	s.mu.Lock()
	defer s.mu.Unlock()
	actor := &Actor{
		ID:       id,
		Mailbox:  make(chan interface{}, 10),
		Behavior: behavior,
	}
	s.actors[id] = actor
	go actor.run()
	return actor
}

func (a *Actor) run() {
	for msg := range a.Mailbox {
		a.Behavior(msg)
	}
}

func (a *Actor) Send(msg interface{}) {
	a.Mailbox <- msg
}
EOF

echo "✅ Actor Model project created at ./$PROJECT_NAME"