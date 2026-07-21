# Saga Choreography (Хореография саги)

**Saga Choreography** — это паттерн управления распределёнными транзакциями, при котором **нет центрального координатора**. Вместо этого каждый участник (сервис) публикует события, а другие сервисы подписываются на них и реагируют, выполняя свои локальные транзакции или компенсирующие действия.

Этот подход — идеальный выбор для систем, где важна **слабая связанность**, **масштабируемость** и **отказоустойчивость**, и где можно допустить eventual consistency.

---

## 🧠 Как это работает

1. **Процесс начинается** — первый сервис выполняет свою локальную транзакцию и публикует событие о её успехе.
2. **Другие сервисы** подписаны на это событие и запускают свои собственные транзакции.
3. **Каждый сервис** публикует своё событие (успех/неудача) после завершения.
4. **Компенсация** (откат) инициируется, если какой-то сервис завершился с ошибкой — он публикует событие отката, и подписчики реагируют компенсирующими действиями.

Никакой центральный координатор не управляет процессом — каждый сервис знает только о событиях, на которые он подписан, и действует независимо.

---

## 🧩 Пример реализации на Go

```go
package saga

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

// Event — базовое событие
type Event struct {
	Type string
	Data interface{}
}

// EventBus — шина событий (in-memory для демонстрации)
type EventBus struct {
	subscribers map[string][]func(Event)
	mu          sync.RWMutex
}

func NewEventBus() *EventBus {
	return &EventBus{
		subscribers: make(map[string][]func(Event)),
	}
}

func (eb *EventBus) Subscribe(eventType string, handler func(Event)) {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	eb.subscribers[eventType] = append(eb.subscribers[eventType], handler)
}

func (eb *EventBus) Publish(event Event) {
	eb.mu.RLock()
	defer eb.mu.RUnlock()
	for _, handler := range eb.subscribers[event.Type] {
		go handler(event) // асинхронно
	}
}

// ===== Сервис 1: Order Service =====
type OrderService struct {
	bus *EventBus
}

func NewOrderService(bus *EventBus) *OrderService {
	return &OrderService{bus: bus}
}

func (s *OrderService) CreateOrder(ctx context.Context, orderID string) error {
	// Локальная транзакция: создание заказа в БД
	fmt.Printf("[OrderService] Создан заказ %s\n", orderID)

	// Публикуем событие "OrderCreated"
	s.bus.Publish(Event{Type: "OrderCreated", Data: orderID})
	return nil
}

func (s *OrderService) Compensate(orderID string) {
	// Компенсация: отмена заказа
	fmt.Printf("[OrderService] Компенсация: заказ %s отменён\n", orderID)
}

// ===== Сервис 2: Payment Service =====
type PaymentService struct {
	bus *EventBus
}

func NewPaymentService(bus *EventBus) *PaymentService {
	return &PaymentService{bus: bus}
}

func (s *PaymentService) Subscribe() {
	s.bus.Subscribe("OrderCreated", func(event Event) {
		orderID := event.Data.(string)
		if err := s.ProcessPayment(orderID); err != nil {
			// Ошибка → публикуем событие "PaymentFailed"
			s.bus.Publish(Event{Type: "PaymentFailed", Data: orderID})
			return
		}
		s.bus.Publish(Event{Type: "PaymentCompleted", Data: orderID})
	})
	s.bus.Subscribe("OrderCanceled", func(event Event) {
		orderID := event.Data.(string)
		s.Compensate(orderID)
	})
}

func (s *PaymentService) ProcessPayment(orderID string) error {
	// Локальная транзакция: списание средств
	fmt.Printf("[PaymentService] Оплата для заказа %s успешно проведена\n", orderID)
	return nil
}

func (s *PaymentService) Compensate(orderID string) {
	fmt.Printf("[PaymentService] Компенсация: возврат средств за заказ %s\n", orderID)
}

// ===== Сервис 3: Delivery Service =====
type DeliveryService struct {
	bus *EventBus
}

func NewDeliveryService(bus *EventBus) *DeliveryService {
	return &DeliveryService{bus: bus}
}

func (s *DeliveryService) Subscribe() {
	s.bus.Subscribe("PaymentCompleted", func(event Event) {
		orderID := event.Data.(string)
		if err := s.ScheduleDelivery(orderID); err != nil {
			s.bus.Publish(Event{Type: "DeliveryFailed", Data: orderID})
			return
		}
		s.bus.Publish(Event{Type: "DeliveryScheduled", Data: orderID})
	})
	s.bus.Subscribe("PaymentFailed", func(event Event) {
		orderID := event.Data.(string)
		s.Compensate(orderID)
	})
}

func (s *DeliveryService) ScheduleDelivery(orderID string) error {
	// Локальная транзакция: планирование доставки
	fmt.Printf("[DeliveryService] Доставка для заказа %s запланирована\n", orderID)
	return nil
}

func (s *DeliveryService) Compensate(orderID string) {
	fmt.Printf("[DeliveryService] Компенсация: доставка отменена для заказа %s\n", orderID)
}
```

---

## 🧪 Пример использования

```go
func main() {
	bus := NewEventBus()

	orderSvc := NewOrderService(bus)
	paymentSvc := NewPaymentService(bus)
	deliverySvc := NewDeliveryService(bus)

	// Подписка
	paymentSvc.Subscribe()
	deliverySvc.Subscribe()

	// Запуск саги: создание заказа
	orderSvc.CreateOrder(context.Background(), "order-123")

	// (В реальном приложении — долгий процесс, здесь просто демонстрация)
}
```

**Ожидаемый вывод:**
```
[OrderService] Создан заказ order-123
[PaymentService] Оплата для заказа order-123 успешно проведена
[DeliveryService] Доставка для заказа order-123 запланирована
```

Если PaymentService вернёт ошибку, произойдёт компенсация:
```
[OrderService] Создан заказ order-123
[PaymentService] (ошибка)
[OrderService] Компенсация: заказ order-123 отменён
[PaymentService] Компенсация: возврат средств за заказ order-123
```

---

## 🧠 Когда я выбираю Saga Choreography

Я выбираю хореографию, когда:

- **Слабая связанность** критична — добавление нового сервиса не требует изменений в существующих.
- **Событийная архитектура** уже используется в системе.
- **Масштабируемость** важнее детального контроля над процессом.
- **Распределённая природа** системы позволяет eventual consistency.

---

## ⚖️ Плюсы и минусы

| Плюсы | Минусы |
|-------|--------|
| ✅ Нет единой точки отказа | ❌ Сложность отладки (распределённый лог) |
| ✅ Лёгкость добавления новых участников | ❌ Сложность мониторинга |
| ✅ Естественная интеграция с EDA | ❌ Риск бесконечных циклов (если события зациклены) |
| ✅ Масштабируемость | ❌ Сложность обеспечения идемпотентности |

---

## 🚀 Как использовать в реальном проекте

1. **Определи события** — какие события публикуются и на какие подписываются сервисы.
2. **Реализуй шину событий** (Kafka, RabbitMQ, NATS) — для продакшена лучше использовать готовый брокер.
3. **Обеспечь идемпотентность** — каждое событие может быть доставлено несколько раз.
4. **Внедри мониторинг** — отслеживай цепочки событий, чтобы видеть прогресс саги.

---

## 📎 Связанные документы

- [Saga Orchestration](../saga-orchestration/README.md) — альтернатива с центральным координатором
- [Event-Driven Architecture](../../architecture-patterns/event-driven/README.md)
- [ADR: Выбор Saga Choreography vs Orchestration](../../docs/architecture/adr/015-saga-choice.md)

---

*Saga Choreography — это не «хаос», а **децентрализованная координация**.*