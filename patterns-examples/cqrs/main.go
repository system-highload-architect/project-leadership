package main

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

// ===== Domain =====

// Order — доменная сущность
type Order struct {
	ID         string
	CustomerID string
	Status     string
	Items      []OrderItem
}

type OrderItem struct {
	ProductID string
	Quantity  int
	Price     float64
	Total     float64
}

// ===== Команды =====

// CreateOrderCommand — команда создания заказа
type CreateOrderCommand struct {
	ID         string
	CustomerID string
	Items      []OrderItem
}

// ===== Запросы =====

// GetOrderQuery — запрос на получение заказа
type GetOrderQuery struct {
	ID string
}

// ListOrdersQuery — запрос на список заказов
type ListOrdersQuery struct {
	Limit int
}

// ===== Command Handler =====

// CommandHandler — интерфейс обработчика команд
type CommandHandler interface {
	Handle(ctx context.Context, cmd interface{}) error
}

// OrderCommandHandler — обработчик команд для заказов
type OrderCommandHandler struct {
	repo WriteRepository
}

func NewOrderCommandHandler(repo WriteRepository) *OrderCommandHandler {
	return &OrderCommandHandler{repo: repo}
}

func (h *OrderCommandHandler) Handle(ctx context.Context, cmd interface{}) error {
	switch c := cmd.(type) {
	case CreateOrderCommand:
		return h.handleCreate(ctx, c)
	default:
		return errors.New("unknown command")
	}
}

func (h *OrderCommandHandler) handleCreate(ctx context.Context, cmd CreateOrderCommand) error {
	// Вычисляем Total для каждого Item
	for i := range cmd.Items {
		cmd.Items[i].Total = float64(cmd.Items[i].Quantity) * cmd.Items[i].Price
	}
	order := Order{
		ID:         cmd.ID,
		CustomerID: cmd.CustomerID,
		Status:     "NEW",
		Items:      cmd.Items,
	}
	return h.repo.Save(ctx, order)
}

// ===== Query Handler =====

// QueryHandler — интерфейс обработчика запросов
type QueryHandler interface {
	Handle(ctx context.Context, query interface{}) (interface{}, error)
}

// OrderQueryHandler — обработчик запросов для заказов
type OrderQueryHandler struct {
	repo ReadRepository
}

func NewOrderQueryHandler(repo ReadRepository) *OrderQueryHandler {
	return &OrderQueryHandler{repo: repo}
}

func (h *OrderQueryHandler) Handle(ctx context.Context, query interface{}) (interface{}, error) {
	switch q := query.(type) {
	case GetOrderQuery:
		return h.handleGet(ctx, q)
	case ListOrdersQuery:
		return h.handleList(ctx, q)
	default:
		return nil, errors.New("unknown query")
	}
}

func (h *OrderQueryHandler) handleGet(ctx context.Context, q GetOrderQuery) (interface{}, error) {
	return h.repo.Get(ctx, q.ID)
}

func (h *OrderQueryHandler) handleList(ctx context.Context, q ListOrdersQuery) (interface{}, error) {
	return h.repo.List(ctx, q.Limit)
}

// ===== Repositories =====

// WriteRepository — хранилище для записи
type WriteRepository interface {
	Save(ctx context.Context, order Order) error
	Get(ctx context.Context, id string) (Order, error)
}

// ReadRepository — хранилище для чтения
type ReadRepository interface {
	Get(ctx context.Context, id string) (Order, error)
	List(ctx context.Context, limit int) ([]Order, error)
	Save(ctx context.Context, order Order) error // для синхронизации
}

// InMemoryWriteRepository — in-memory реализация для записи
type InMemoryWriteRepository struct {
	mu     sync.RWMutex
	orders map[string]Order
}

func NewInMemoryWriteRepository() *InMemoryWriteRepository {
	return &InMemoryWriteRepository{
		orders: make(map[string]Order),
	}
}

func (r *InMemoryWriteRepository) Save(ctx context.Context, order Order) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.orders[order.ID] = order
	return nil
}

func (r *InMemoryWriteRepository) Get(ctx context.Context, id string) (Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	order, ok := r.orders[id]
	if !ok {
		return Order{}, errors.New("order not found")
	}
	return order, nil
}

// InMemoryReadRepository — in-memory реализация для чтения
type InMemoryReadRepository struct {
	mu     sync.RWMutex
	orders map[string]Order
}

func NewInMemoryReadRepository() *InMemoryReadRepository {
	return &InMemoryReadRepository{
		orders: make(map[string]Order),
	}
}

func (r *InMemoryReadRepository) Save(ctx context.Context, order Order) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.orders[order.ID] = order
	return nil
}

func (r *InMemoryReadRepository) Get(ctx context.Context, id string) (Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	order, ok := r.orders[id]
	if !ok {
		return Order{}, errors.New("order not found")
	}
	return order, nil
}

func (r *InMemoryReadRepository) List(ctx context.Context, limit int) ([]Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]Order, 0, limit)
	for _, order := range r.orders {
		if len(result) >= limit {
			break
		}
		result = append(result, order)
	}
	return result, nil
}

// ===== Синхронизация (в реальном проекте — через события) =====

func syncReadModel(writeRepo WriteRepository, readRepo ReadRepository, orderID string) error {
	order, err := writeRepo.Get(context.Background(), orderID)
	if err != nil {
		return err
	}
	return readRepo.Save(context.Background(), order)
}

// ===== Главная функция =====

func main() {
	ctx := context.Background()

	// Инициализация репозиториев
	writeRepo := NewInMemoryWriteRepository()
	readRepo := NewInMemoryReadRepository()

	// Инициализация обработчиков
	cmdHandler := NewOrderCommandHandler(writeRepo)
	queryHandler := NewOrderQueryHandler(readRepo)

	// 1. Создаём заказ (команда)
	cmd := CreateOrderCommand{
		ID:         "order-1",
		CustomerID: "customer-1",
		Items: []OrderItem{
			{ProductID: "product-1", Quantity: 2, Price: 100},
			{ProductID: "product-2", Quantity: 1, Price: 50},
		},
	}
	err := cmdHandler.Handle(ctx, cmd)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Println("✅ Order created:", cmd.ID)

	// 2. Синхронизация read model (в реальном проекте — через события)
	if err := syncReadModel(writeRepo, readRepo, cmd.ID); err != nil {
		fmt.Println("Sync error:", err)
		return
	}

	// 3. Запрос: получить заказ
	result, err := queryHandler.Handle(ctx, GetOrderQuery{ID: cmd.ID})
	if err != nil {
		fmt.Println("Query error:", err)
		return
	}
	order := result.(Order)
	fmt.Printf("📖 Order: %+v\n", order)

	// 4. Запрос: список заказов
	result, err = queryHandler.Handle(ctx, ListOrdersQuery{Limit: 10})
	if err != nil {
		fmt.Println("Query error:", err)
		return
	}
	orders := result.([]Order)
	fmt.Printf("📖 All orders: %+v\n", orders)
}
