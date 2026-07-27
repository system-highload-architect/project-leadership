package main

import (
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
	AggregateID string
	Timestamp   time.Time
	Type        string
}

func (e BaseEvent) GetAggregateID() string  { return e.AggregateID }
func (e BaseEvent) GetTimestamp() time.Time { return e.Timestamp }
func (e BaseEvent) GetType() string         { return e.Type }

// OrderCreated — событие создания заказа
type OrderCreated struct {
	BaseEvent
	CustomerID string
	Items      []OrderItem
}

// OrderPaid — событие оплаты заказа
type OrderPaid struct {
	BaseEvent
	PaymentID string
}

// OrderCancelled — событие отмены заказа
type OrderCancelled struct {
	BaseEvent
	Reason string
}

// ===== Доменные сущности =====

// OrderItem — позиция заказа
type OrderItem struct {
	ProductID string
	Quantity  int
	Price     float64
}

// Order — агрегат (восстанавливается из событий)
type Order struct {
	ID         string
	Status     string
	CustomerID string
	Items      []OrderItem
}

// Apply применяет событие к агрегату
func (o *Order) Apply(event Event) {
	switch e := event.(type) {
	case *OrderCreated:
		o.ID = e.AggregateID
		o.Status = "NEW"
		o.CustomerID = e.CustomerID
		o.Items = e.Items
	case *OrderPaid:
		o.Status = "PAID"
	case *OrderCancelled:
		o.Status = "CANCELLED"
	}
}

// Replay восстанавливает агрегат из списка событий
func (o *Order) Replay(events []Event) {
	for _, event := range events {
		o.Apply(event)
	}
}

// ===== Event Store =====

// EventStore — хранилище событий (in-memory для демонстрации)
type EventStore struct {
	mu     sync.RWMutex
	events map[string][]Event // aggregateID → список событий
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

// ===== Command Handlers =====

// CreateOrderCommand — команда создания заказа
type CreateOrderCommand struct {
	ID         string
	CustomerID string
	Items      []OrderItem
}

// HandleCreateOrder обрабатывает команду создания заказа
func HandleCreateOrder(es *EventStore, cmd CreateOrderCommand) error {
	// Проверяем, не существует ли уже
	events, _ := es.Load(cmd.ID)
	if len(events) > 0 {
		return errors.New("order already exists")
	}
	// Создаём событие
	event := &OrderCreated{
		BaseEvent: BaseEvent{
			AggregateID: cmd.ID,
			Timestamp:   time.Now(),
			Type:        "OrderCreated",
		},
		CustomerID: cmd.CustomerID,
		Items:      cmd.Items,
	}
	return es.Save(cmd.ID, event)
}

// PayOrderCommand — команда оплаты заказа
type PayOrderCommand struct {
	OrderID   string
	PaymentID string
}

// HandlePayOrder обрабатывает команду оплаты заказа
func HandlePayOrder(es *EventStore, cmd PayOrderCommand) error {
	events, _ := es.Load(cmd.OrderID)
	if len(events) == 0 {
		return errors.New("order not found")
	}
	// Восстанавливаем текущее состояние
	order := &Order{}
	order.Replay(events)
	if order.Status != "NEW" {
		return errors.New("order cannot be paid in current status: " + order.Status)
	}
	// Создаём событие
	event := &OrderPaid{
		BaseEvent: BaseEvent{
			AggregateID: cmd.OrderID,
			Timestamp:   time.Now(),
			Type:        "OrderPaid",
		},
		PaymentID: cmd.PaymentID,
	}
	return es.Save(cmd.OrderID, event)
}

// CancelOrderCommand — команда отмены заказа
type CancelOrderCommand struct {
	OrderID string
	Reason  string
}

// HandleCancelOrder обрабатывает команду отмены заказа
func HandleCancelOrder(es *EventStore, cmd CancelOrderCommand) error {
	events, _ := es.Load(cmd.OrderID)
	if len(events) == 0 {
		return errors.New("order not found")
	}
	// Восстанавливаем текущее состояние
	order := &Order{}
	order.Replay(events)
	if order.Status == "CANCELLED" {
		return nil // уже отменён
	}
	// Создаём событие
	event := &OrderCancelled{
		BaseEvent: BaseEvent{
			AggregateID: cmd.OrderID,
			Timestamp:   time.Now(),
			Type:        "OrderCancelled",
		},
		Reason: cmd.Reason,
	}
	return es.Save(cmd.OrderID, event)
}

// ===== Вспомогательные функции =====

// LoadOrder загружает и восстанавливает заказ из Event Store
func LoadOrder(es *EventStore, orderID string) (*Order, error) {
	events, err := es.Load(orderID)
	if err != nil {
		return nil, err
	}
	if len(events) == 0 {
		return nil, errors.New("order not found")
	}
	order := &Order{}
	order.Replay(events)
	return order, nil
}

// PrintEvents печатает историю событий заказа
func PrintEvents(es *EventStore, orderID string) {
	events, _ := es.Load(orderID)
	fmt.Printf("📜 Event history for %s:\n", orderID)
	for i, event := range events {
		fmt.Printf("  [%d] %s at %s\n", i, event.GetType(), event.GetTimestamp().Format("15:04:05"))
	}
}

// ===== Главная функция =====

func main() {
	es := NewEventStore()
	orderID := "order-123"

	// 1. Создаём заказ
	err := HandleCreateOrder(es, CreateOrderCommand{
		ID:         orderID,
		CustomerID: "customer-1",
		Items: []OrderItem{
			{ProductID: "product-1", Quantity: 2, Price: 100},
		},
	})
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Println("✅ Order created:", orderID)

	// 2. Оплачиваем заказ
	err = HandlePayOrder(es, PayOrderCommand{
		OrderID:   orderID,
		PaymentID: "payment-1",
	})
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Println("✅ Order status updated: PAID")

	// 3. Загружаем и выводим текущее состояние
	order, err := LoadOrder(es, orderID)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Printf("📖 Order state: %+v\n", order)

	// 4. Выводим историю событий
	PrintEvents(es, orderID)

	// 5. Демонстрация отмены (компенсация)
	fmt.Println("\n--- Отмена заказа (компенсация) ---")
	err = HandleCancelOrder(es, CancelOrderCommand{
		OrderID: orderID,
		Reason:  "User request",
	})
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	order, _ = LoadOrder(es, orderID)
	fmt.Printf("📖 Order state after cancellation: %+v\n", order)
	PrintEvents(es, orderID)
}
