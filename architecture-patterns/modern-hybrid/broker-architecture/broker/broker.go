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
