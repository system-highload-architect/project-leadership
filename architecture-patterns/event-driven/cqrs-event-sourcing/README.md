# CQRS + Event Sourcing

**CQRS (Command Query Responsibility Segregation)** и **Event Sourcing** — это два паттерна, которые часто используются вместе. CQRS разделяет команды (запись) и запросы (чтение), а Event Sourcing хранит все изменения как последовательность событий.

Вместе они создают мощную архитектуру для сложных, аудитируемых и масштабируемых систем.

---

## 🧠 Основные принципы

1. **Разделение команд и запросов (CQRS).**  
   Команды изменяют состояние, запросы только читают данные. Это позволяет оптимизировать каждую сторону независимо.

2. **Хранение событий (Event Sourcing).**  
   Все изменения фиксируются как события, которые являются источником истины. Текущее состояние восстанавливается через воспроизведение событий.

3. **Проекции (Read Models).**  
   Запросы обращаются к отдельным проекциям (денормализованным моделям), которые строятся из событий асинхронно.

4. **Eventual consistency.**  
   Данные в read-моделях могут быть немного устаревшими по сравнению с write-моделью, но это допустимо для большинства сценариев.

---

## 🧩 Когда использовать CQRS + Event Sourcing

- **Сложная бизнес-логика** с множеством состояний и переходов.
- **Аудит** критически важен (финансовые операции, медицина, логистика).
- **Нагрузка на чтение и запись сильно различается.**
- **Несколько проекций** для разных целей (аналитика, поиск, отчёты).
- **Необходимость восстановления состояния** на любой момент времени.

---

## 🏗️ Архитектура

```
[ Команда ] → [ Command Handler ] → [ Aggregate ] → [ Event Store ] → [ Event Bus ]
                                                                          │
    [ Запрос ] → [ Query Handler ] ← [ Read DB (Projection) ] ← [ Projection Service ]
```

---

## 🔧 Реализация на Go (пример)

**Команда:**

```go
type CreateOrderCommand struct {
    CustomerID string
    Items      []OrderItem
}
```

**Command Handler (с сохранением события):**

```go
func (h *OrderCommandHandler) HandleCreateOrder(ctx context.Context, cmd CreateOrderCommand) error {
    order := domain.NewOrder(cmd.CustomerID, cmd.Items)
    // сохраняем событие OrderCreated в Event Store
    event := order.GetEvent()
    err := h.eventStore.Save(order.ID, event)
    if err != nil {
        return err
    }
    // публикуем событие в Kafka для проекций
    h.eventBus.Publish(event)
    return nil
}
```

**Проекция (Read Model):**

```go
func (s *OrderProjection) OnOrderCreated(event OrderCreated) {
    // обновляем read-модель (денормализованная таблица)
    readModel := &OrderRead{
        ID:         event.AggregateID,
        CustomerID: event.CustomerID,
        Status:     "NEW",
        Items:      event.Items,
    }
    s.readRepo.Save(readModel)
}
```

---

## ⚖️ Плюсы и минусы

| Плюсы | Минусы |
|-------|--------|
| ✅ Полный аудит | ❌ Высокая сложность |
| ✅ Независимое масштабирование чтения и записи | ❌ Eventual consistency (согласованность в конечном счёте) |
| ✅ Гибкость в построении проекций | ❌ Большой объём данных (события) |
| ✅ Возможность восстановления на любой момент | ❌ Требуется идемпотентность обработчиков |

---

## 📎 Связанные документы

- [ADR: CQRS Choice](../../../docs/architecture/adr/017-cqrs-choice.md)
- [ADR: Event Sourcing Choice](../../../docs/architecture/adr/016-event-sourcing-choice.md)
- [Пример реализации CQRS + Event Sourcing](../../../patterns-examples/cqrs/README.md)
- [Event Sourcing (отдельно)](./event-sourcing/README.md)