package main

import (
	"context"
	"fmt"
	"sync"
)

// ===== События =====

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
	s := &OrderService{bus: bus}
	// Подписка на события отката
	bus.Subscribe("PaymentFailed", func(event Event) {
		orderID := event.Data.(string)
		s.Compensate(orderID)
	})
	return s
}

func (s *OrderService) CreateOrder(ctx context.Context, orderID string) error {
	fmt.Printf("[Order Service] Order %s created\n", orderID)
	// Публикуем событие OrderCreated
	s.bus.Publish(Event{Type: "OrderCreated", Data: orderID})
	return nil
}

func (s *OrderService) Compensate(orderID string) {
	fmt.Printf("[Order Service] Compensate: %s cancelled\n", orderID)
}

// ===== Сервис 2: Payment Service =====

type PaymentService struct {
	bus        *EventBus
	shouldFail bool // для демонстрации ошибки
}

func NewPaymentService(bus *EventBus, shouldFail bool) *PaymentService {
	s := &PaymentService{bus: bus, shouldFail: shouldFail}
	// Подписка на OrderCreated
	bus.Subscribe("OrderCreated", func(event Event) {
		orderID := event.Data.(string)
		s.ProcessPayment(orderID)
	})
	// Подписка на OrderCancelled (компенсация)
	bus.Subscribe("OrderCancelled", func(event Event) {
		orderID := event.Data.(string)
		s.Compensate(orderID)
	})
	return s
}

func (s *PaymentService) ProcessPayment(orderID string) {
	if s.shouldFail {
		fmt.Printf("[Payment Service] Payment failed for %s\n", orderID)
		s.bus.Publish(Event{Type: "PaymentFailed", Data: orderID})
		return
	}
	fmt.Printf("[Payment Service] Payment for %s processed\n", orderID)
	s.bus.Publish(Event{Type: "PaymentCompleted", Data: orderID})
}

func (s *PaymentService) Compensate(orderID string) {
	fmt.Printf("[Payment Service] Compensate: refund for %s\n", orderID)
}

// ===== Сервис 3: Delivery Service =====

type DeliveryService struct {
	bus *EventBus
}

func NewDeliveryService(bus *EventBus) *DeliveryService {
	s := &DeliveryService{bus: bus}
	// Подписка на PaymentCompleted
	bus.Subscribe("PaymentCompleted", func(event Event) {
		orderID := event.Data.(string)
		s.ScheduleDelivery(orderID)
	})
	// Подписка на PaymentFailed (компенсация)
	bus.Subscribe("PaymentFailed", func(event Event) {
		orderID := event.Data.(string)
		s.Compensate(orderID)
	})
	return s
}

func (s *DeliveryService) ScheduleDelivery(orderID string) {
	fmt.Printf("[Delivery Service] Delivery for %s scheduled\n", orderID)
}

func (s *DeliveryService) Compensate(orderID string) {
	fmt.Printf("[Delivery Service] Compensate: delivery for %s cancelled\n", orderID)
}

// ===== Главная функция =====

func main() {
	bus := NewEventBus()

	fmt.Println("=== Saga Choreography (Успех) ===")
	orderSvc := NewOrderService(bus)
	_ = NewPaymentService(bus, false) // не падает
	_ = NewDeliveryService(bus)

	// Запускаем сагу: создание заказа
	orderSvc.CreateOrder(context.Background(), "order-123")

	// Ждём завершения асинхронных обработчиков
	// В реальном проекте здесь был бы sync.WaitGroup или каналы
	fmt.Scanln() // простая пауза для демонстрации

	// Создаём новую шину для второго сценария
	bus2 := NewEventBus()

	fmt.Println("\n=== Saga Choreography (Ошибка + компенсация) ===")
	orderSvc2 := NewOrderService(bus2)
	_ = NewPaymentService(bus2, true) // падает
	_ = NewDeliveryService(bus2)

	orderSvc2.CreateOrder(context.Background(), "order-456")
	fmt.Scanln()
}
