#!/bin/bash
# Broker Architecture — центральный брокер (in-memory)

set -e

ROOT_DIR="broker-architecture"

mkdir -p "$ROOT_DIR/broker"
mkdir -p "$ROOT_DIR/producer/cmd/server"
mkdir -p "$ROOT_DIR/consumer/cmd/server"

cd "$ROOT_DIR"

# ===== Broker (пакет без internal) =====
cat > broker/broker.go <<'EOF'
package broker

import (
	"sync"
)

type Message struct {
	Topic   string
	Payload interface{}
}

type Broker struct {
	subscribers map[string][]chan Message
	mu          sync.RWMutex
}

var (
	instance *Broker
	once     sync.Once
)

func GetBroker() *Broker {
	once.Do(func() {
		instance = &Broker{
			subscribers: make(map[string][]chan Message),
		}
	})
	return instance
}

func (b *Broker) Subscribe(topic string) <-chan Message {
	b.mu.Lock()
	defer b.mu.Unlock()
	ch := make(chan Message, 10)
	b.subscribers[topic] = append(b.subscribers[topic], ch)
	return ch
}

func (b *Broker) Publish(topic string, payload interface{}) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	msg := Message{Topic: topic, Payload: payload}
	for _, ch := range b.subscribers[topic] {
		ch <- msg
	}
}
EOF

# ===== Producer =====
cd producer
cat > go.mod <<'EOF'
module producer

go 1.23

replace broker => ../broker
EOF

cat > cmd/server/main.go <<'EOF'
package main

import (
	"fmt"
	"time"

	"broker"
)

func main() {
	b := broker.GetBroker()
	for i := 1; i <= 5; i++ {
		msg := fmt.Sprintf("Message #%d", i)
		b.Publish("orders", msg)
		fmt.Printf("[Producer] Sent: %s\n", msg)
		time.Sleep(500 * time.Millisecond)
	}
}
EOF
cd ..

# ===== Consumer =====
cd consumer
cat > go.mod <<'EOF'
module consumer

go 1.23

replace broker => ../broker
EOF

cat > cmd/server/main.go <<'EOF'
package main

import (
	"fmt"

	"broker"
)

func main() {
	b := broker.GetBroker()
	ch := b.Subscribe("orders")

	fmt.Println("[Consumer] Listening for messages...")
	for msg := range ch {
		fmt.Printf("[Consumer] Received: %v\n", msg.Payload)
	}
}
EOF
cd ..

echo "✅ Broker Architecture project created at ./$ROOT_DIR"