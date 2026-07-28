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
