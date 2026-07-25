# Микросервисы с Event Sourcing

**Event Sourcing** — это паттерн, при котором **состояние системы хранится не как текущий снимок данных, а как последовательность событий, которые привели к этому состоянию**. Вместо того чтобы обновлять запись в БД, вы сохраняете каждое изменение как новое событие. Текущее состояние можно восстановить, воспроизведя все события с начала.

Этот подход даёт **полный аудит**, **возможность восстановления на любой момент времени** и **естественную поддержку событийно-ориентированной архитектуры**.

---

## 🧠 Основные принципы

1. **Каждое изменение** (команда) превращается в событие (факт).
2. **Событие сохраняется** в хранилище событий (event store).
3. **Текущее состояние** вычисляется путём воспроизведения всех событий.
4. **События могут использоваться** для построения проекций (read models), отправки уведомлений, аналитики.
5. **События неизменяемы** — они не редактируются и не удаляются.

---

## 🧩 Когда использовать Event Sourcing

- **Аудит** критически важен — нужно знать, кто, когда и что изменил.
- **Возможность восстановления** на любой момент времени — нужна для отладки и откатов.
- **Сложная бизнес-логика** — события позволяют моделировать процессы естественно.
- **Event-Driven Architecture** уже используется в системе.
- **Проекции** — можно построить несколько read models для разных целей.

---

## 🏗️ Архитектура

```
[ Command ] → [ Aggregate ] → [ Event Store ] → [ Kafka ] → [ Projection Service ] → [ Read DB ]
                                                       │
                                                       ▼
                                                  [ Notification Service ]
```

---

## 🔧 Реализация на Go

**Событие:**

```go
type Event interface {
    GetAggregateID() string
    GetTimestamp() time.Time
    GetType() string
}

type OrderCreated struct {
    AggregateID string
    CustomerID  string
    Items       []OrderItem
    Timestamp   time.Time
}
```

**Агрегат:**

```go
type OrderAggregate struct {
    ID     string
    Status string
    Items  []OrderItem
}

func (a *OrderAggregate) Apply(event Event) {
    switch e := event.(type) {
    case *OrderCreated:
        a.ID = e.AggregateID
        a.Status = "NEW"
        a.Items = e.Items
    case *OrderPaid:
        a.Status = "PAID"
    }
}
```

**Event Store:**

```go
type EventStore interface {
    Save(aggregateID string, events ...Event) error
    Load(aggregateID string) ([]Event, error)
}
```

---

## ⚖️ Плюсы и минусы

| Плюсы | Минусы |
|-------|--------|
| ✅ Полный аудит | ❌ Хранилище событий может вырасти до огромных размеров |
| ✅ Возможность восстановления на любой момент | ❌ Сложность работы с данными (eventual consistency) |
| ✅ Естественная интеграция с EDA | ❌ Более сложная модель данных (события вместо таблиц) |
| ✅ Легко строить проекции (CQRS) | ❌ Требуется идемпотентность обработчиков |

---

## 📎 Связанные документы

- [ADR: Event Sourcing Choice](../../../docs/architecture/adr/016-event-sourcing-choice.md)
- [CQRS (следующий шаг)](../cqrs/README.md)
- [Пример реализации Event Sourcing в Go](../../../patterns-examples/event-sourcing/README.md)