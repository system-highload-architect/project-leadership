# Onion Architecture (Луковая архитектура)

**Onion Architecture** — это подход, который, как и Clean Architecture, ставит **ядро бизнес-логики в центр системы**, а все внешние зависимости располагаются на периферии. Однако в Onion акцент делается на **кольцевую структуру зависимостей**: каждый внешний слой может зависеть только от следующего внутреннего, но не наоборот.

Название «луковая» отражает идею множества слоёв, где **ядро — это доменные сущности**, а каждый следующий слой — это более «внешняя» абстракция (Use Cases, API, инфраструктура).

---

## 🧠 Основные принципы

1. **Ядро — независимо** — сущности и бизнес-правила не знают ни о каких внешних системах.
2. **Зависимости направлены внутрь** — внешние слои зависят от внутренних, но не наоборот.
3. **Интерфейсы определены внутри** — внешние слои реализуют интерфейсы, объявленные во внутренних слоях.
4. **Инверсия зависимостей** — инфраструктура зависит от абстракций Use Cases, а не наоборот.

---

## 🧩 Слои Onion Architecture

| Слой | Назначение | Что здесь живёт |
|------|------------|-----------------|
| **Domain Model** (ядро) | Бизнес-сущности, правила, Value Objects | `Robot`, `Task`, `Status` |
| **Domain Services** | Бизнес-логика, которая не помещается в одну сущность | `AssignTaskService`, `PricingEngine` |
| **Application (Use Cases)** | Оркестрация бизнес-логики, команды, запросы | `AssignTaskUseCase`, `GetRobotUseCase` |
| **Ports (Interfaces)** | Интерфейсы, определяющие взаимодействие с внешним миром | `RobotRepository`, `Notifier`, `Logger` |
| **Adapters (Infrastructure)** | Реализация интерфейсов (БД, HTTP, очереди) | `PostgresRobotRepository`, `KafkaNotifier` |
| **User Interface (API)** | Входные точки (HTTP, gRPC, CLI) | `HTTPHandler`, `GRPCServer` |

---

## 📁 Структура проекта (шаблон)

```
onion-example/
├── cmd/
│   └── server/
│       └── main.go                      # Сборка слоёв
├── internal/
│   ├── domain/                          # Ядро
│   │   ├── model/
│   │   │   ├── robot.go
│   │   │   └── task.go
│   │   └── services/
│   │       ├── assign_task.go           # Domain Service (использует сущности)
│   │       └── pricing_engine.go
│   ├── application/                     # Use Cases
│   │   ├── robot/
│   │   │   ├── assign_task.go
│   │   │   └── get_robot.go
│   │   └── ports/                       # Интерфейсы (Driven Ports)
│   │       ├── robot_repository.go
│   │       └── notifier.go
│   ├── infrastructure/                  # Адаптеры (реализация портов)
│   │   ├── postgres/
│   │   │   └── robot_repo.go
│   │   ├── redis/
│   │   │   └── cache_repo.go
│   │   └── kafka/
│   │       └── notifier.go
│   └── delivery/                        # Driving Ports (API)
│       ├── http/
│       │   └── robot_handler.go
│       └── grpc/
│           └── robot_server.go
└── pkg/
    ├── logger/
    └── config/
```

---

## 📝 Пример на Go

### Ядро (Domain Model + Service)

```go
// internal/domain/model/robot.go
package model

type Robot struct {
    ID     string
    Name   string
    Status string
    Tasks  []Task
}

// internal/domain/services/assign_task.go
package services

type TaskAssigner interface {
    AssignTask(robot *Robot, task Task) error
}

type DefaultAssigner struct{}

func (a *DefaultAssigner) AssignTask(robot *Robot, task Task) error {
    // бизнес-логика: проверка статуса, лимитов и т.д.
    robot.Tasks = append(robot.Tasks, task)
    return nil
}
```

### Use Case (Application Layer)

```go
// internal/application/robot/assign_task.go
package robot

type AssignTaskUseCase struct {
    repo       ports.RobotRepository
    assigner   domain.TaskAssigner
    notifier   ports.Notifier
}

func (uc *AssignTaskUseCase) Execute(ctx context.Context, input AssignTaskInput) error {
    robot, err := uc.repo.Get(ctx, input.RobotID)
    if err != nil {
        return err
    }
    task := domain.Task{ID: input.TaskID, Status: "pending"}
    if err := uc.assigner.AssignTask(robot, task); err != nil {
        return err
    }
    if err := uc.repo.Save(ctx, robot); err != nil {
        return err
    }
    return uc.notifier.Send(ctx, "task_assigned", robot.ID)
}
```

### Порты (интерфейсы)

```go
// internal/application/ports/robot_repository.go
package ports

type RobotRepository interface {
    Get(ctx context.Context, id string) (*domain.Robot, error)
    Save(ctx context.Context, robot *domain.Robot) error
}

// internal/application/ports/notifier.go
type Notifier interface {
    Send(ctx context.Context, event string, payload interface{}) error
}
```

### Адаптер (Infrastructure)

```go
// internal/infrastructure/postgres/robot_repo.go
package postgres

type RobotRepository struct {
    db *sql.DB
}

func (r *RobotRepository) Get(ctx context.Context, id string) (*domain.Robot, error) {
    // SQL-запрос → маппинг → возврат доменной сущности
}
```

### API (Delivery)

```go
// internal/delivery/http/robot_handler.go
package http

type RobotHandler struct {
    assignTaskUC *robot.AssignTaskUseCase
}

func (h *RobotHandler) AssignTask(w http.ResponseWriter, r *http.Request) {
    // парсинг запроса → вызов Use Case → ответ
}
```

---

## ⚖️ Onion vs Clean vs Hexagonal

| Критерий | Onion Architecture | Clean Architecture | Hexagonal Architecture |
|----------|--------------------|--------------------|-------------------------|
| **Центр** | Domain Model + Domain Services | Domain + Use Cases | Core (Domain + Use Cases) |
| **Интерфейсы** | Определены во внутренних слоях | Определены в Use Cases или Domain | Определены в ядре (Ports) |
| **Инфраструктура** | Реализует интерфейсы из Application слоя | Реализует интерфейсы из Use Cases | Реализует Ports из ядра |
| **Тестируемость** | Высокая (благодаря инверсии) | Высокая | Очень высокая |
| **Сложность** | Высокая | Средняя | Высокая |

---

## 🧠 Когда я выбираю Onion

Я выбираю Onion Architecture, когда:

- Нужна **максимальная изоляция бизнес-логики** от инфраструктуры.
- Система **очень сложная** и требует чёткого разделения ответственности.
- Команда готова к **дополнительной сложности** ради гибкости.
- Важно иметь **возможность заменять внешние сервисы** без изменений в ядре.

---

## 📌 Ключевые преимущества

- **Чёткое разделение слоёв** — бизнес-логика полностью изолирована.
- **Инверсия зависимостей** — внешние слои зависят от внутренних интерфейсов.
- **Высокая тестируемость** — можно тестировать Use Cases без поднятия БД или HTTP.
- **Эволюционность** — легко заменять инфраструктуру (например, БД или брокеры) без изменения ядра.

---

## 🚀 Как использовать этот раздел

1. **Начни с Domain Model** — определи сущности и бизнес-правила.
2. **Добавь Domain Services** — для логики, которая не помещается в одну сущность.
3. **Спроектируй Application слой** — Use Cases и интерфейсы портов.
4. **Реализуй Infrastructure** — адаптеры для БД, брокеров, внешних API.
5. **Добавь Delivery** — HTTP/gRPC для взаимодействия с пользователями.

---

## 📎 Связанные документы

- [ADR: Выбор Onion Architecture vs Clean](../../docs/architecture/adr/010-onion-choice.md)
- [Пример реализации Onion в Go](../../patterns-examples/onion-example/README.md)
- [Сравнение архитектурных стилей](../../docs/strategic/architecture-styles-comparison.md)

---

*Onion Architecture — это не про «лук», а про **чёткие границы и направление зависимостей к центру**.*