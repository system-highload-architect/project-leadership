package eventstore

import (
	"errors"
	"sync"

	"order-service/internal/event"
)

type EventStore struct {
	mu     sync.RWMutex
	events map[string][]event.Event // aggregateID → список событий
}

func NewEventStore() *EventStore {
	return &EventStore{
		events: make(map[string][]event.Event),
	}
}

func (es *EventStore) Save(aggregateID string, events ...event.Event) error {
	es.mu.Lock()
	defer es.mu.Unlock()
	es.events[aggregateID] = append(es.events[aggregateID], events...)
	return nil
}

func (es *EventStore) Load(aggregateID string) ([]event.Event, error) {
	es.mu.RLock()
	defer es.mu.RUnlock()
	events, ok := es.events[aggregateID]
	if !ok {
		return nil, errors.New("no events found for aggregate")
	}
	return events, nil
}
