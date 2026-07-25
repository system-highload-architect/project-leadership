# Event Sourcing в событийно-ориентированной архитектуре

**Event Sourcing** — это паттерн, при котором все изменения состояния системы сохраняются как последовательность событий, а не как текущее состояние. Вместо того чтобы обновлять запись в БД, вы сохраняете каждое изменение как новое событие. Текущее состояние может быть восстановлено путём воспроизведения всех событий с начала.

В контексте событийно-ориентированной архитектуры (EDA), Event Sourcing является естественным расширением: события становятся источником истины и основой для интеграции между компонентами.

---

## 🧠 Основные принципы

1. **События — источник истины.**  
   Все изменения фиксируются как неизменяемые события. Это даёт полный аудит и историю.

2. **Восстановление состояния.**  
   Текущее состояние агрегата вычисляется путём воспроизведения всех его событий.

3. **Интеграция с EDA.**  
   События публикуются в шину (Kafka) и могут быть использованы другими сервисами для построения проекций, уведомлений, аналитики.

4. **Неизменяемость.**  
   События нельзя изменить или удалить — только добавить новые.

---

## 🧩 Когда использовать Event Sourcing

- **Аудит** критически важен (финансовые операции, логистика, медицинские записи).
- **Возможность восстановления** на любой момент времени необходима для отладки или откатов.
- **Сложная бизнес-логика** — события позволяют моделировать процессы естественно.
- **Система уже использует EDA** — Event Sourcing легко вписывается в неё.
- **Несколько проекций** — можно построить несколько read-моделей для разных целей.

---

## 🏗️ Архитектура

```
[ Command ] → [ Aggregate ] → [ Event Store ] → [ Event Bus (Kafka) ]
    │                              │                    │
    │                              ▼                    ▼
    │                         [ Snapshot ]      [ Projection Service ]
    │                                                   │
    └───────────────────────────────────────────────────▼
                                              [ Read DB (CQRS) ]
```

---

## 🔧 Реализация на Go (пример)

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

func (a *OrderAggregate) Replay(events []Event) {
    for _, event := range events {
        a.Apply(event)
    }
}
```

**Event Store (PostgreSQL):**

```sql
CREATE TABLE events (
    id SERIAL PRIMARY KEY,
    aggregate_id UUID NOT NULL,
    event_type VARCHAR(100) NOT NULL,
    event_data JSONB NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);
```

---

## ⚖️ Плюсы и минусы

| Плюсы | Минусы |
|-------|--------|
| ✅ Полный аудит | ❌ Хранилище событий может вырасти до огромных размеров |
| ✅ Возможность восстановления на любой момент | ❌ Сложность работы с данными (eventual consistency) |
| ✅ Естественная интеграция с EDA | ❌ Более сложная модель данных (события вместо таблиц) |
| ✅ Лёгкость построения проекций | ❌ Требуется идемпотентность обработчиков |

---

## 📎 Связанные документы

- [ADR: Event Sourcing Choice](../../../docs/architecture/adr/016-event-sourcing-choice.md)
- [CQRS + Event Sourcing](../cqrs-event-sourcing/README.md)
- [Пример реализации Event Sourcing](../../../patterns-examples/event-sourcing/README.md)