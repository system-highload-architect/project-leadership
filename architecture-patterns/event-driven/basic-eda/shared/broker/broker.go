package broker

import (
	"sync"
)

type Event struct {
	Type string
	Data interface{}
}

type Broker struct {
	subscribers map[string][]chan Event
	mu          sync.RWMutex
}

var (
	instance *Broker
	once     sync.Once
)

func GetBroker() *Broker {
	once.Do(func() {
		instance = &Broker{
			subscribers: make(map[string][]chan Event),
		}
	})
	return instance
}

func (b *Broker) Subscribe(eventType string) <-chan Event {
	b.mu.Lock()
	defer b.mu.Unlock()
	ch := make(chan Event, 10)
	b.subscribers[eventType] = append(b.subscribers[eventType], ch)
	return ch
}

func (b *Broker) Publish(event Event) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for _, ch := range b.subscribers[event.Type] {
		ch <- event
	}
}
