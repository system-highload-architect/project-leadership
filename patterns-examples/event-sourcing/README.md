# Event Sourcing (Хранение событий)

**Event Sourcing** — это паттерн, при котором **состояние системы хранится не как текущий снимок данных, а как последовательность событий, которые привели к этому состоянию**. Вместо того чтобы обновлять запись в БД, вы сохраняете каждое изменение как новое событие. Текущее состояние можно восстановить, воспроизведя все события с начала.

Этот подход даёт **полный аудит**, **возможность восстановления на любой момент времени** и **естественную поддержку событийно-ориентированной архитектуры**.

---

## 🧠 Как это работает

1. **Каждое изменение** (команда) превращается в событие (факт).
2. **Событие сохраняется** в хранилище событий (event store).
3. **Текущее состояние** вычисляется путём воспроизведения всех событий.
4. **События могут использоваться** для построения проекций (read models), отправки уведомлений, аналитики.

События неизменяемы — они не редактируются и не удаляются. Это делает систему **аудит-дружественной** и позволяет воспроизводить историю.

---

## 🧩 Пример реализации на Go

```go
package eventsourcing

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"
)

// ===== События =====

// Event — интерфейс любого события
type Event interface {
	GetAggregateID() string
	GetTimestamp() time.Time
	GetType() string
}

// BaseEvent — базовая структура для всех событий
type BaseEvent struct {
	AggregateID string    `json:"aggregate_id"`
	Timestamp   time.Time `json:"timestamp"`
	Type        string    `json:"type"`
}

func (e BaseEvent) GetAggregateID() string { return e.AggregateID }
func (e BaseEvent) GetTimestamp() time.Time { return e.Timestamp }
func (e BaseEvent) GetType() string        { return e.Type }

// RobotCreated — событие создания робота
type RobotCreated struct {
	BaseEvent
	Name string `json:"name"`
	Type string `json:"type"`
}

// RobotStatusChanged — событие изменения статуса робота
type RobotStatusChanged struct {
	BaseEvent
	OldStatus string `json:"old_status"`
	NewStatus string `json:"new_status"`
}

// RobotDeleted — событие удаления робота
type RobotDeleted struct {
	BaseEvent
}

// ===== Event Store =====

// EventStore — хранилище событий (in-memory для демонстрации)
type EventStore struct {
	mu      sync.RWMutex
	events  map[string][]Event // aggregateID → список событий
}

func NewEventStore() *EventStore {
	return &EventStore{
		events: make(map[string][]Event),
	}
}

func (es *EventStore) Save(aggregateID string, events ...Event) error {
	es.mu.Lock()
	defer es.mu.Unlock()
	es.events[aggregateID] = append(es.events[aggregateID], events...)
	return nil
}

func (es *EventStore) Load(aggregateID string) ([]Event, error) {
	es.mu.RLock()
	defer es.mu.RUnlock()
	events, ok := es.events[aggregateID]
	if !ok {
		return nil, nil
	}
	return events, nil
}

// ===== Aggregate =====

// RobotAggregate — агрегат робота
type RobotAggregate struct {
	ID     string
	Name   string
	Type   string
	Status string
	Deleted bool
}

// NewRobotAggregate создаёт агрегат и применяет события
func NewRobotAggregate(events []Event) *RobotAggregate {
	r := &RobotAggregate{}
	for _, e := range events {
		r.Apply(e)
	}
	return r
}

// Apply применяет событие к агрегату (восстановление состояния)
func (r *RobotAggregate) Apply(e Event) {
	switch ev := e.(type) {
	case *RobotCreated:
		r.ID = ev.AggregateID
		r.Name = ev.Name
		r.Type = ev.Type
		r.Status = "idle"
	case *RobotStatusChanged:
		r.Status = ev.NewStatus
	case *RobotDeleted:
		r.Deleted = true
	}
}

// ===== Command Handlers =====

// CreateRobotCommand — команда создания робота
type CreateRobotCommand struct {
	ID   string
	Name string
	Type string
}

func HandleCreateRobot(ctx context.Context, es *EventStore, cmd CreateRobotCommand) error {
	// Проверяем, не существует ли уже
	events, _ := es.Load(cmd.ID)
	if len(events) > 0 {
		return errors.New("robot already exists")
	}
	// Создаём событие
	event := &RobotCreated{
		BaseEvent: BaseEvent{
			AggregateID: cmd.ID,
			Timestamp:   time.Now(),
			Type:        "RobotCreated",
		},
		Name: cmd.Name,
		Type: cmd.Type,
	}
	return es.Save(cmd.ID, event)
}

// ChangeRobotStatusCommand — команда изменения статуса
type ChangeRobotStatusCommand struct {
	ID     string
	Status string
}

func HandleChangeRobotStatus(ctx context.Context, es *EventStore, cmd ChangeRobotStatusCommand) error {
	events, _ := es.Load(cmd.ID)
	if len(events) == 0 {
		return errors.New("robot not found")
	}
	// Восстанавливаем текущее состояние
	robot := NewRobotAggregate(events)
	if robot.Deleted {
		return errors.New("robot already deleted")
	}
	if robot.Status == cmd.Status {
		return nil // статус не изменился
	}
	// Создаём событие
	event := &RobotStatusChanged{
		BaseEvent: BaseEvent{
			AggregateID: cmd.ID,
			Timestamp:   time.Now(),
			Type:        "RobotStatusChanged",
		},
		OldStatus: robot.Status,
		NewStatus: cmd.Status,
	}
	return es.Save(cmd.ID, event)
}

// DeleteRobotCommand — команда удаления робота
type DeleteRobotCommand struct {
	ID string
}

func HandleDeleteRobot(ctx context.Context, es *EventStore, cmd DeleteRobotCommand) error {
	events, _ := es.Load(cmd.ID)
	if len(events) == 0 {
		return errors.New("robot not found")
	}
	robot := NewRobotAggregate(events)
	if robot.Deleted {
		return nil // уже удалён
	}
	event := &RobotDeleted{
		BaseEvent: BaseEvent{
			AggregateID: cmd.ID,
			Timestamp:   time.Now(),
			Type:        "RobotDeleted",
		},
	}
	return es.Save(cmd.ID, event)
}
```

---

## 🧪 Пример использования

```go
func main() {
	es := NewEventStore()
	ctx := context.Background()

	// 1. Создаём робота
	err := HandleCreateRobot(ctx, es, CreateRobotCommand{
		ID:   "robot-1",
		Name: "R2D2",
		Type: "delivery",
	})
	if err != nil {
		fmt.Println(err)
	}

	// 2. Меняем статус
	err = HandleChangeRobotStatus(ctx, es, ChangeRobotStatusCommand{
		ID:     "robot-1",
		Status: "busy",
	})
	if err != nil {
		fmt.Println(err)
	}

	// 3. Загружаем события и восстанавливаем состояние
	events, _ := es.Load("robot-1")
	robot := NewRobotAggregate(events)
	fmt.Printf("Robot: %+v, Status: %s\n", robot, robot.Status)

	// 4. Просматриваем историю
	for _, e := range events {
		fmt.Printf("Event: %s at %s\n", e.GetType(), e.GetTimestamp())
	}
}
```

**Ожидаемый вывод:**
```
Robot: &{ID:robot-1 Name:R2D2 Type:delivery Status:busy Deleted:false}, Status: busy
Event: RobotCreated at 2026-07-21 12:00:00
Event: RobotStatusChanged at 2026-07-21 12:00:01
```

---

## 🧠 Когда я выбираю Event Sourcing

Я выбираю Event Sourcing, когда:

- **Аудит** критически важен — нужно знать, кто, когда и что изменил.
- **Возможность восстановления** на любой момент времени — нужна для отладки и откатов.
- **Сложная бизнес-логика** — события позволяют моделировать процессы естественно.
- **Event-Driven Architecture** уже используется в системе.
- **Проекции** — можно построить несколько read models для разных целей.

---

## ⚖️ Плюсы и минусы

| Плюсы | Минусы |
|-------|--------|
| ✅ Полный аудит | ❌ Хранилище событий может вырасти до огромных размеров |
| ✅ Возможность восстановления на любой момент | ❌ Сложность работы с данными (eventual consistency) |
| ✅ Естественная интеграция с EDA | ❌ Более сложная модель данных (события вместо таблиц) |
| ✅ Легко строить проекции (CQRS) | ❌ Требуется идемпотентность обработчиков |

---

## 🚀 Как использовать в реальном проекте

1. **Определи события** для каждой команды.
2. **Реализуй Event Store** (PostgreSQL, EventStoreDB, Kafka с retention).
3. **Построй агрегаты** как восстанавливаемые состояния из событий.
4. **Используй CQRS** для разделения записи (события) и чтения (проекции).
5. **Обеспечь идемпотентность** — обработка одного события не должна менять состояние дважды.

---

## 📎 Связанные документы

- [CQRS](../cqrs/README.md) — часто используется с Event Sourcing
- [Event-Driven Architecture](../../architecture-patterns/event-driven/README.md)
- [ADR: Выбор Event Sourcing для аудита](../../docs/architecture/adr/016-event-sourcing-choice.md)

---

*Event Sourcing — это не «медленно», а **история изменений, которую можно переигрывать**.*