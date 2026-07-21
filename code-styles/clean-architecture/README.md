# Clean Architecture

**Clean Architecture** — это подход к организации кода, который я использую как основной. Он позволяет **отделить бизнес-логику от внешних зависимостей** (БД, HTTP, gRPC, очереди), делая систему **тестируемой**, **гибкой** и **эволюционирующей**.

Этот подход не привязан к монолиту или микросервисам — он работает внутри любого модуля или сервиса.

---

## 🧠 Основные принципы

Clean Architecture строится на трёх ключевых идеях:

1. **Зависимости направлены внутрь** — внешние слои зависят от внутренних, но не наоборот.
2. **Бизнес-логика не зависит от инфраструктуры** — вы можете заменить БД или транспортный слой без изменения Use Cases.
3. **Ядро системы не знает о внешнем мире** — оно работает с интерфейсами, а не с конкретными реализациями.

---

## 🧩 Слои Clean Architecture

| Слой | Назначение | Что здесь живёт |
|------|------------|-----------------|
| **Domain (Entity)** | Бизнес-правила и сущности | Структуры, методы, валидация, Value Objects |
| **Usecase (Application)** | Оркестрация бизнес-логики | Use Cases, команды, запросы, интерфейсы для репозиториев |
| **Controller (Delivery)** | Транспортный слой | HTTP-хендлеры, gRPC-серверы, CLI-команды |
| **Infrastructure (Repository)** | Реализация внешних зависимостей | Репозитории (Postgres, Redis), клиенты, брокеры |

---

## 📁 Структура проекта (шаблон)

```
clean-architecture-example/
├── cmd/
│   └── server/
│       └── main.go                 # Точка входа: DI, запуск сервера
├── internal/
│   ├── domain/                      # Ядро
│   │   ├── robot.go
│   │   └── task.go
│   ├── usecase/                     # Бизнес-логика
│   │   ├── robot/
│   │   │   ├── get_robot.go
│   │   │   └── update_status.go
│   │   └── task/
│   │       └── assign_task.go
│   ├── delivery/                    # Транспорт
│   │   ├── http/
│   │   │   ├── robot_handler.go
│   │   │   └── middlewares.go
│   │   └── grpc/
│   │       └── robot_server.go
│   └── repository/                  # Инфраструктура
│       ├── robot_repo.go            # Интерфейс
│       └── postgres/
│           └── robot_repo.go        # Реализация
└── pkg/                             # Переиспользуемые утилиты
    ├── logger/
    ├── config/
    └── validator/
```

---

## 🧠 Правила зависимостей

- **Domain** не зависит ни от чего.
- **Usecase** зависит от Domain и от интерфейсов репозиториев (но не от их реализации).
- **Controller (Delivery)** зависит от Usecase.
- **Infrastructure (Repository)** зависит от интерфейсов Usecase и от Domain.

Направление зависимостей — **внутрь**:

```
Controller → Usecase → Domain
Repository → Usecase → Domain
```

---

## 📌 Ключевые правила

- **Тестирование бизнес-логики** не требует поднятия БД или HTTP-сервера.
- **Интерфейсы должны быть узкими** — репозиторий знает только о методах, которые ему нужны.
- **Инфраструктура — это плагин**, который можно заменить без изменения Use Cases.
- **Каждый Use Case** — это отдельный метод или объект (Command/Query).

---

## 📝 Пример Use Case на Go

```go
// internal/usecase/robot/get_robot.go
package robot

import "context"

// GetRobotInput — запрос на получение робота
type GetRobotInput struct {
    ID string
}

// RobotOutput — ответ
type RobotOutput struct {
    ID     string
    Status string
    Name   string
}

// RobotGetter — интерфейс репозитория, нужный этому Use Case
type RobotGetter interface {
    Get(ctx context.Context, id string) (*domain.Robot, error)
}

// GetRobotUseCase — Use Case
type GetRobotUseCase struct {
    repo RobotGetter
}

func NewGetRobotUseCase(repo RobotGetter) *GetRobotUseCase {
    return &GetRobotUseCase{repo: repo}
}

func (uc *GetRobotUseCase) Execute(ctx context.Context, input GetRobotInput) (*RobotOutput, error) {
    robot, err := uc.repo.Get(ctx, input.ID)
    if err != nil {
        return nil, err
    }
    return &RobotOutput{
        ID:     robot.ID,
        Status: string(robot.Status),
        Name:   robot.Name,
    }, nil
}
```

---

## 🧠 Когда я использую Clean Architecture

Я использую Clean Architecture:

- В **модульном монолите** — для каждого модуля.
- В **микросервисах** — внутри каждого сервиса.
- В **сложных системах**, где бизнес-логика может меняться независимо от инфраструктуры.
- Всегда, когда важна **тестируемость** и **поддерживаемость**.

---

## 🚀 Как использовать этот раздел

1. **Начни с Domain** — определи сущности и бизнес-правила.
2. **Спроектируй Use Cases** — опиши, что система должна делать.
3. **Создай интерфейсы** для внешних зависимостей (репозитории, клиенты).
4. **Реализуй инфраструктуру** — репозитории, HTTP/gRPC.
5. **Свяжи всё через Dependency Injection** в `main.go`.

---

## 📎 Связанные документы

- [ADR: Выбор Clean Architecture](../../docs/architecture/adr/007-clean-architecture-choice.md)
- [Пример реализации Clean Architecture в Go](../../patterns-examples/clean-architecture-example/README.md)
- [Принципы DDD в Clean Architecture](../../docs/strategic/ddd-in-clean-architecture.md)

---

*Clean Architecture — это не про «слои ради слоёв». Это про **контроль над сложностью**.*