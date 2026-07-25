# Микросервисы с Saga

**Saga** — это паттерн управления распределёнными транзакциями в микросервисной архитектуре. Вместо использования двухфазного коммита (2PC), Saga разбивает транзакцию на последовательность локальных транзакций, каждая из которых имеет компенсирующее действие (откат). Если один из шагов завершается ошибкой, Saga инициирует компенсацию в обратном порядке.

Существует две модели реализации:

- **Saga Choreography** — децентрализованная координация через события.
- **Saga Orchestration** — централизованная координация через оркестратор.

---

## 🧠 Основные принципы

1. **Распределённая транзакция** разбивается на несколько локальных транзакций.
2. **Каждая локальная транзакция** имеет компенсирующее действие (откат).
3. **Координация** может быть:
   - **Хореография** — каждый сервис публикует события, другие сервисы подписываются и реагируют.
   - **Оркестрация** — центральный координатор управляет последовательностью шагов и компенсацией.
4. **Eventual consistency** — согласованность достигается в конечном счёте, а не мгновенно.

---

## 🧩 Когда использовать Saga

- Нужно обеспечить согласованность данных между несколькими сервисами.
- Невозможно использовать двухфазный коммит (блокировки, масштабируемость).
- Допустима eventual consistency (не требуется строгая атомарность).
- Система уже использует события (EDA) или есть центральный координатор.

---

## 🏗️ Архитектура

### Хореография (Choreography)

```
[ Service A ] → событие → [ Service B ] → событие → [ Service C ]
       │                                            │
       └──────────────── событие отката ─────────────┘
```

### Оркестрация (Orchestration)

```
[ Coordinator ]
    │
    ├── 1. Service A → успех
    ├── 2. Service B → успех
    ├── 3. Service C → ошибка
    │
    └── Компенсация: Service B → Service A (в обратном порядке)
```

---

## 🔧 Реализация на Go

### Saga Choreography (на событиях)

```go
// Order Service
func (s *OrderService) CreateOrder(order Order) error {
    // сохраняем заказ
    s.repo.Save(order)
    // публикуем событие OrderCreated
    s.events.Publish(OrderCreated{OrderID: order.ID})
    return nil
}

// Payment Service (подписчик)
func (s *PaymentService) OnOrderCreated(event OrderCreated) {
    err := s.processPayment(event.OrderID)
    if err != nil {
        // публикуем событие отката
        s.events.Publish(PaymentFailed{OrderID: event.OrderID})
        return
    }
    s.events.Publish(PaymentCompleted{OrderID: event.OrderID})
}
```

### Saga Orchestration (с координатором)

```go
type SagaCoordinator struct {
    steps []Step
}

func (s *SagaCoordinator) Execute(ctx context.Context, data map[string]interface{}) error {
    history := []int{}
    for i, step := range s.steps {
        if err := step.Execute(ctx, data); err != nil {
            // откат
            for j := len(history) - 1; j >= 0; j-- {
                s.steps[history[j]].Compensate(ctx, data)
            }
            return err
        }
        history = append(history, i)
    }
    return nil
}
```

---

## ⚖️ Плюсы и минусы

| Плюсы | Минусы |
|-------|--------|
| ✅ Слабая связанность (хореография) | ❌ Сложность отладки (хореография) |
| ✅ Нет единой точки отказа (хореография) | ❌ Единая точка отказа (оркестрация) |
| ✅ Лёгкость добавления новых участников (хореография) | ❌ Сложность мониторинга (хореография) |
| ✅ Полный контроль над процессом (оркестрация) | ❌ Увеличение сложности (оба варианта) |
| ✅ Простота мониторинга (оркестрация) | ❌ Требуется идемпотентность |

---

## 📎 Связанные документы

- [ADR: Saga Choice](../../../docs/architecture/adr/015-saga-choice.md)
- [Пример Saga Orchestration](../../../patterns-examples/saga-orchestration/README.md)
- [Пример Saga Choreography](../../../patterns-examples/saga-choreography/README.md)