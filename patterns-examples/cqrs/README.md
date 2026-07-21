# CQRS (Command Query Responsibility Segregation)

**CQRS** — это паттерн, разделяющий модели для **чтения** (Query) и **записи** (Command). Вместо использования одной модели данных для всех операций, CQRS предлагает использовать отдельные модели для обновления состояния и для получения данных.

Это позволяет **оптимизировать каждую сторону независимо** (например, использовать разные БД для чтения и записи), упрощает масштабирование и улучшает производительность.

---

## 🧠 Как это работает

1. **Команда (Command)** — изменяет состояние системы (CREATE, UPDATE, DELETE). Не возвращает данные.
2. **Запрос (Query)** — получает данные, не изменяет состояние.
3. **Модель записи** оптимизирована для вставки/обновления (нормализованная, транзакционная).
4. **Модель чтения** оптимизирована для выборки (денормализованная, кэшированная).

Между моделями может быть **синхронизация** (часто асинхронная через события). Это позволяет читать данные из специализированных хранилищ (Elasticsearch, Redis, ClickHouse) без влияния на производительность записи.

---

## 🧩 Пример реализации на Go

```go
package cqrs

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

// ===== Domain =====

// Robot — доменная сущность
type Robot struct {
	ID     string
	Name   string
	Status string
}

// ===== Команды =====

// CreateRobotCommand — команда создания робота
type CreateRobotCommand struct {
	ID   string
	Name string
}

// UpdateRobotStatusCommand — команда изменения статуса
type UpdateRobotStatusCommand struct {
	ID     string
	Status string
}

// ===== Query =====

// GetRobotQuery — запрос на получение робота
type GetRobotQuery struct {
	ID string
}

// ListRobotsQuery — запрос на список роботов
type ListRobotsQuery struct {
	Limit int
}

// ===== Command Handlers =====

// CommandHandler — интерфейс обработчика команд
type CommandHandler interface {
	Handle(ctx context.Context, cmd interface{}) error
}

// RobotCommandHandler — обработчик команд для роботов
type RobotCommandHandler struct {
	repo WriteRepository
}

func NewRobotCommandHandler(repo WriteRepository) *RobotCommandHandler {
	return &RobotCommandHandler{repo: repo}
}

func (h *RobotCommandHandler) Handle(ctx context.Context, cmd interface{}) error {
	switch c := cmd.(type) {
	case CreateRobotCommand:
		return h.handleCreate(ctx, c)
	case UpdateRobotStatusCommand:
		return h.handleUpdateStatus(ctx, c)
	default:
		return errors.New("unknown command")
	}
}

func (h *RobotCommandHandler) handleCreate(ctx context.Context, cmd CreateRobotCommand) error {
	robot := Robot{
		ID:     cmd.ID,
		Name:   cmd.Name,
		Status: "idle",
	}
	// Сохраняем в хранилище записи
	return h.repo.Save(ctx, robot)
}

func (h *RobotCommandHandler) handleUpdateStatus(ctx context.Context, cmd UpdateRobotStatusCommand) error {
	// Загружаем, обновляем, сохраняем
	robot, err := h.repo.Get(ctx, cmd.ID)
	if err != nil {
		return err
	}
	if robot.Status == cmd.Status {
		return nil // нечего менять
	}
	robot.Status = cmd.Status
	return h.repo.Save(ctx, robot)
}

// ===== Query Handlers =====

// QueryHandler — интерфейс обработчика запросов
type QueryHandler interface {
	Handle(ctx context.Context, query interface{}) (interface{}, error)
}

// RobotQueryHandler — обработчик запросов для роботов
type RobotQueryHandler struct {
	repo ReadRepository
}

func NewRobotQueryHandler(repo ReadRepository) *RobotQueryHandler {
	return &RobotQueryHandler{repo: repo}
}

func (h *RobotQueryHandler) Handle(ctx context.Context, query interface{}) (interface{}, error) {
	switch q := query.(type) {
	case GetRobotQuery:
		return h.handleGet(ctx, q)
	case ListRobotsQuery:
		return h.handleList(ctx, q)
	default:
		return nil, errors.New("unknown query")
	}
}

func (h *RobotQueryHandler) handleGet(ctx context.Context, q GetRobotQuery) (interface{}, error) {
	return h.repo.Get(ctx, q.ID)
}

func (h *RobotQueryHandler) handleList(ctx context.Context, q ListRobotsQuery) (interface{}, error) {
	return h.repo.List(ctx, q.Limit)
}

// ===== Repositories =====

// WriteRepository — хранилище для записи
type WriteRepository interface {
	Save(ctx context.Context, robot Robot) error
	Get(ctx context.Context, id string) (Robot, error)
}

// ReadRepository — хранилище для чтения
type ReadRepository interface {
	Get(ctx context.Context, id string) (Robot, error)
	List(ctx context.Context, limit int) ([]Robot, error)
}

// InMemoryWriteRepository — in-memory реализация для записи
type InMemoryWriteRepository struct {
	mu     sync.RWMutex
	robots map[string]Robot
}

func NewInMemoryWriteRepository() *InMemoryWriteRepository {
	return &InMemoryWriteRepository{
		robots: make(map[string]Robot),
	}
}

func (r *InMemoryWriteRepository) Save(ctx context.Context, robot Robot) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.robots[robot.ID] = robot
	return nil
}

func (r *InMemoryWriteRepository) Get(ctx context.Context, id string) (Robot, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	robot, ok := r.robots[id]
	if !ok {
		return Robot{}, errors.New("robot not found")
	}
	return robot, nil
}

// InMemoryReadRepository — in-memory реализация для чтения
type InMemoryReadRepository struct {
	mu     sync.RWMutex
	robots map[string]Robot
}

func NewInMemoryReadRepository() *InMemoryReadRepository {
	return &InMemoryReadRepository{
		robots: make(map[string]Robot),
	}
}

func (r *InMemoryReadRepository) Get(ctx context.Context, id string) (Robot, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	robot, ok := r.robots[id]
	if !ok {
		return Robot{}, errors.New("robot not found")
	}
	return robot, nil
}

func (r *InMemoryReadRepository) List(ctx context.Context, limit int) ([]Robot, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]Robot, 0, len(r.robots))
	for _, robot := range r.robots {
		if len(result) >= limit {
			break
		}
		result = append(result, robot)
	}
	return result, nil
}
```

---

## 🧪 Пример использования

```go
func main() {
	ctx := context.Background()

	// Инициализация репозиториев
	writeRepo := NewInMemoryWriteRepository()
	readRepo := NewInMemoryReadRepository()

	// Инициализация обработчиков
	cmdHandler := NewRobotCommandHandler(writeRepo)
	queryHandler := NewRobotQueryHandler(readRepo)

	// 1. Создаём робота (команда)
	err := cmdHandler.Handle(ctx, CreateRobotCommand{
		ID:   "robot-1",
		Name: "R2D2",
	})
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("✅ Робот создан")

	// 2. Синхронизация: обновляем read model (в реальном проекте — через события)
	// В данном примере синхронизация вручную
	robot, _ := writeRepo.Get(ctx, "robot-1")
	readRepo.Save(ctx, robot) // добавим метод в интерфейс для примера

	// 3. Запрос: получаем робота
	result, err := queryHandler.Handle(ctx, GetRobotQuery{ID: "robot-1"})
	if err != nil {
		fmt.Println(err)
	}
	robotData := result.(Robot)
	fmt.Printf("📖 Робот: %+v\n", robotData)

	// 4. Обновляем статус (команда)
	err = cmdHandler.Handle(ctx, UpdateRobotStatusCommand{
		ID:     "robot-1",
		Status: "busy",
	})
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("✅ Статус обновлён")

	// 5. Снова синхронизируем read model
	robot, _ = writeRepo.Get(ctx, "robot-1")
	readRepo.Save(ctx, robot)

	// 6. Запрос: список роботов
	result, err = queryHandler.Handle(ctx, ListRobotsQuery{Limit: 10})
	if err != nil {
		fmt.Println(err)
	}
	robots := result.([]Robot)
	fmt.Printf("📖 Список роботов: %+v\n", robots)
}
```

**Ожидаемый вывод:**
```
✅ Робот создан
📖 Робот: {ID:robot-1 Name:R2D2 Status:idle}
✅ Статус обновлён
📖 Список роботов: [{ID:robot-1 Name:R2D2 Status:busy}]
```

---

## 🧠 Когда я выбираю CQRS

Я выбираю CQRS, когда:

1. **Нагрузка на чтение и запись сильно различается** (читают много, пишут редко).
2. **Требуется разные модели для чтения и записи** (например, чтение — через Elasticsearch, запись — через PostgreSQL).
3. **Сложная бизнес-логика** требует отделения команд от запросов для упрощения.
4. **Event Sourcing** уже используется — CQRS идеально сочетается с ним.

Если нагрузка невелика и система простая, CQRS — это лишняя сложность.

---

## ⚖️ Плюсы и минусы

| Плюсы | Минусы |
|-------|--------|
| ✅ Оптимизация каждой стороны (чтение/запись) | ❌ Увеличение сложности |
| ✅ Упрощение моделей | ❌ Синхронизация между моделями (eventual consistency) |
| ✅ Независимое масштабирование | ❌ Дополнительная инфраструктура |
| ✅ Естественная интеграция с Event Sourcing | ❌ Сложность понимания для команды |

---

## 🚀 Как использовать в реальном проекте

1. **Определи команды и запросы** — что изменяет состояние, что только читает.
2. **Раздели модели данных** для записи и чтения.
3. **Внедри синхронизацию** между моделями (через события или асинхронные обновления).
4. **Оптимизируй read model** под конкретные запросы (индексы, кэши, денормализация).
5. **Следи за консистентностью** — eventual consistency допустима, но нужно объяснить бизнесу.

---

## 📎 Связанные документы

- [Event Sourcing](../event-sourcing/README.md) — часто используется вместе с CQRS
- [Event-Driven Architecture](../../architecture-patterns/event-driven/README.md)
- [ADR: Выбор CQRS для системы](../../docs/architecture/adr/017-cqrs-choice.md)

---

*CQRS — это не «сложность ради сложности», а **инструмент для работы с разными нагрузками**.*