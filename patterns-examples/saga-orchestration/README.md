# Saga Orchestration (Оркестрация саги)

**Saga Orchestration** — это паттерн управления распределёнными транзакциями, при котором **один центральный координатор (оркестратор)** управляет последовательностью шагов и вызывает сервисы, выполняющие локальные транзакции. В случае ошибки оркестратор инициирует компенсирующие действия (откат).

Этот подход даёт **полный контроль** над ходом транзакции, упрощает мониторинг и отладку, но создаёт **единую точку отказа** (оркестратор). Он идеально подходит для сценариев, где важна **строгая последовательность** и **согласованность**.

---

## 🧠 Как это работает

1. **Оркестратор** получает команду на выполнение транзакции (например, "создать заказ").
2. **Оркестратор вызывает первый сервис** (например, "Создать заказ").
3. **После успешного ответа** вызывает следующий сервис (например, "Провести оплату").
4. **После успешного ответа** вызывает следующий сервис (например, "Запланировать доставку").
5. **Если любой шаг завершился ошибкой**, оркестратор вызывает компенсирующие операции в обратном порядке (откат).

Оркестратор хранит состояние саги и может быть восстановлен при сбое (если сохранять состояние в БД).

---

## 🧩 Пример реализации на Go

```go
package saga_orchestration

import (
	"context"
	"errors"
	"fmt"
)

// ===== Шаги саги =====

// Step — интерфейс шага саги
type Step interface {
	Execute(ctx context.Context, data map[string]interface{}) error
	Compensate(ctx context.Context, data map[string]interface{}) error
}

// ===== Конкретные шаги =====

// CreateOrderStep — создание заказа
type CreateOrderStep struct{}

func (s *CreateOrderStep) Execute(ctx context.Context, data map[string]interface{}) error {
	orderID, ok := data["orderID"].(string)
	if !ok {
		return errors.New("orderID missing")
	}
	fmt.Printf("[CreateOrderStep] Создан заказ %s\n", orderID)
	// В реальности здесь был бы запрос к БД или вызов сервиса
	return nil
}

func (s *CreateOrderStep) Compensate(ctx context.Context, data map[string]interface{}) error {
	orderID, _ := data["orderID"].(string)
	fmt.Printf("[CreateOrderStep] Компенсация: заказ %s отменён\n", orderID)
	return nil
}

// ProcessPaymentStep — обработка оплаты
type ProcessPaymentStep struct {
	shouldFail bool // для демонстрации ошибки
}

func (s *ProcessPaymentStep) Execute(ctx context.Context, data map[string]interface{}) error {
	orderID, _ := data["orderID"].(string)
	if s.shouldFail {
		fmt.Printf("[ProcessPaymentStep] Ошибка оплаты для заказа %s\n", orderID)
		return errors.New("payment failed")
	}
	fmt.Printf("[ProcessPaymentStep] Оплата для заказа %s успешно проведена\n", orderID)
	return nil
}

func (s *ProcessPaymentStep) Compensate(ctx context.Context, data map[string]interface{}) error {
	orderID, _ := data["orderID"].(string)
	fmt.Printf("[ProcessPaymentStep] Компенсация: возврат средств для заказа %s\n", orderID)
	return nil
}

// ScheduleDeliveryStep — планирование доставки
type ScheduleDeliveryStep struct{}

func (s *ScheduleDeliveryStep) Execute(ctx context.Context, data map[string]interface{}) error {
	orderID, _ := data["orderID"].(string)
	fmt.Printf("[ScheduleDeliveryStep] Доставка для заказа %s запланирована\n", orderID)
	return nil
}

func (s *ScheduleDeliveryStep) Compensate(ctx context.Context, data map[string]interface{}) error {
	orderID, _ := data["orderID"].(string)
	fmt.Printf("[ScheduleDeliveryStep] Компенсация: доставка для заказа %s отменена\n", orderID)
	return nil
}

// ===== Оркестратор =====

// Orchestrator — управляет последовательностью шагов
type Orchestrator struct {
	steps []Step
}

// NewOrchestrator создаёт оркестратор с заданной последовательностью шагов
func NewOrchestrator(steps []Step) *Orchestrator {
	return &Orchestrator{steps: steps}
}

// Execute запускает сагу
func (o *Orchestrator) Execute(ctx context.Context, data map[string]interface{}) error {
	history := []int{} // хранит индексы успешно выполненных шагов

	for i, step := range o.steps {
		if err := step.Execute(ctx, data); err != nil {
			// Ошибка → откат (компенсация в обратном порядке)
			fmt.Printf("\n[Orchestrator] Ошибка на шаге %d: %v. Запуск компенсации...\n", i, err)
			o.compensate(ctx, data, history)
			return fmt.Errorf("saga failed at step %d: %w", i, err)
		}
		history = append(history, i)
	}
	fmt.Println("[Orchestrator] Сага успешно завершена")
	return nil
}

// compensate выполняет компенсацию в обратном порядке
func (o *Orchestrator) compensate(ctx context.Context, data map[string]interface{}, history []int) {
	for i := len(history) - 1; i >= 0; i-- {
		idx := history[i]
		if err := o.steps[idx].Compensate(ctx, data); err != nil {
			fmt.Printf("[Orchestrator] Ошибка при компенсации шага %d: %v\n", idx, err)
		}
	}
}
```

---

## 🧪 Пример использования

```go
package main

import (
	"context"
	"fmt"
)

func main() {
	// Создаём шаги
	steps := []Step{
		&CreateOrderStep{},
		&ProcessPaymentStep{shouldFail: false}, // true — для демонстрации ошибки
		&ScheduleDeliveryStep{},
	}

	// Создаём оркестратор
	orchestrator := NewOrchestrator(steps)

	// Данные саги
	data := map[string]interface{}{
		"orderID": "order-123",
	}

	// Запуск саги
	err := orchestrator.Execute(context.Background(), data)
	if err != nil {
		fmt.Printf("Saga завершилась с ошибкой: %v\n", err)
	}
}
```

**Ожидаемый вывод (успех):**
```
[CreateOrderStep] Создан заказ order-123
[ProcessPaymentStep] Оплата для заказа order-123 успешно проведена
[ScheduleDeliveryStep] Доставка для заказа order-123 запланирована
[Orchestrator] Сага успешно завершена
```

**Вывод при ошибке (shouldFail=true):**
```
[CreateOrderStep] Создан заказ order-123
[ProcessPaymentStep] Ошибка оплаты для заказа order-123
[Orchestrator] Ошибка на шаге 1: payment failed. Запуск компенсации...
[ProcessPaymentStep] Компенсация: возврат средств для заказа order-123
[CreateOrderStep] Компенсация: заказ order-123 отменён
Saga завершилась с ошибкой: payment failed
```

---

## 🧠 Когда я выбираю Saga Orchestration

Я выбираю оркестрацию, когда:

- **Строгая последовательность** обязательна (шаги должны выполняться в чётком порядке).
- **Нужен полный контроль** над процессом и состоянием транзакции.
- **Мониторинг и отладка** критичны — легче отслеживать одну точку управления.
- **Можно допустить единую точку отказа** (оркестратор должен быть восстановлен при сбое).

---

## ⚖️ Choreography vs Orchestration

| Критерий | Choreography | Orchestration |
|----------|--------------|---------------|
| **Управление** | Децентрализованное | Централизованное |
| **Связанность** | Слабая | Более жёсткая (зависимость от оркестратора) |
| **Мониторинг** | Сложный (распределённые логи) | Простой (все шаги в одном месте) |
| **Масштабируемость** | Высокая | Ограничена оркестратором |
| **Сложность** | Высокая (события, eventual consistency) | Средняя (чёткая последовательность) |
| **Пример** | EDA, Kafka | Workflow-движки (Camunda, Temporal) |

---

## 🚀 Как использовать в реальном проекте

1. **Определи последовательность шагов** и компенсирующие действия для каждого.
2. **Реализуй интерфейс Step** для каждого шага.
3. **Обеспечь идемпотентность** Execute и Compensate (на случай повторных вызовов).
4. **Сохраняй состояние саги** в БД (для восстановления после сбоя оркестратора).
5. **Внедри мониторинг** — логируй переходы состояний для отладки.

---

## 📎 Связанные документы

- [Saga Choreography](../saga-choreography/README.md) — альтернатива без центрального координатора
- [Event-Driven Architecture](../../architecture-patterns/event-driven/README.md)
- [ADR: Выбор Saga Orchestration vs Choreography](../../docs/architecture/adr/015-saga-choice.md)

---

*Saga Orchestration — это не «бюрократия», а **инструмент управления сложными процессами**.*