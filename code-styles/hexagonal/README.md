# Hexagonal Architecture (Ports & Adapters)

**Hexagonal Architecture**, также известная как **Ports & Adapters** — это стиль организации кода, при котором **ядро системы полностью изолировано от внешнего мира**. Все взаимодействия с внешними системами (БД, HTTP, очереди) происходят через **порты (интерфейсы)** и **адаптеры (реализации)**.

Этот подход — эволюция Clean Architecture, где акцент смещается с «слоёв» на «границы»: ядро не знает, кто и как его использует, а внешние адаптеры подключаются через чёткие контракты.

---

## 🧠 Основная идея

Hexagonal Architecture строится на метафоре **шестиугольника**:

- **Внутри** — ядро: бизнес-логика, доменные сущности, правила.
- **Снаружи** — адаптеры: HTTP-серверы, gRPC-клиенты, репозитории, брокеры.
- **Между ними** — порты: интерфейсы, через которые ядро общается с внешним миром.

Ядро **не знает**, какой адаптер его вызывает и какие адаптеры оно вызывает. Оно работает с портами — и только с ними.

---

## 🧩 Основные понятия

| Понятие | Описание | Пример |
|---------|----------|--------|
| **Ядро (Core)** | Бизнес-логика, доменные сущности, Use Cases | `Robot`, `Task`, `AssignTaskUseCase` |
| **Порт (Port)** | Интерфейс, который ядро ожидает или предоставляет | `RobotRepository`, `Logger`, `Notifier` |
| **Адаптер (Adapter)** | Реализация порта для конкретной технологии | `PostgresRobotRepository`, `KafkaNotifier`, `HTTPHandler` |

---

## 🧩 Связь между портами и адаптерами

| Тип порта | Направление | Пример |
|-----------|-------------|--------|
| **Driving Port** (Предоставляемый) | Внешний мир → Ядро | HTTP/gRPC API, CLI, события |
| **Driven Port** (Требуемый) | Ядро → Внешний мир | Репозиторий, отправка email, логирование |

---

## 📁 Структура проекта (шаблон)

```
hexagonal-example/
├── cmd/
│   └── server/
│       └── main.go                      # Точка входа: сборка адаптеров
├── internal/
│   ├── core/                            # Ядро — не зависит ни от чего
│   │   ├── domain/
│   │   │   ├── robot.go
│   │   │   └── task.go
│   │   ├── ports/
│   │   │   ├── robot_repository.go      # Интерфейс (Driven Port)
│   │   │   └── notifier.go              # Интерфейс (Driven Port)
│   │   └── usecases/
│   │       ├── assign_task.go
│   │       └── get_robot.go
│   └── adapters/                         # Реализация адаптеров
│       ├── driving/                      # Driving Ports (API)
│       │   ├── http/
│       │   │   └── robot_handler.go
│       │   └── grpc/
│       │       └── robot_server.go
│       └── driven/                       # Driven Ports (инфраструктура)
│           ├── postgres/
│           │   └── robot_repo.go
│           ├── redis/
│           │   └── cache_repo.go
│           └── kafka/
│               └── notifier.go
└── pkg/                                  # Переиспользуемые утилиты
    ├── logger/
    └── config/
```

---

## 📝 Пример на Go

### Ядро (порт и Use Case)

```go
// internal/core/ports/robot_repository.go
package ports

import "context"

type RobotRepository interface {
    Get(ctx context.Context, id string) (*domain.Robot, error)
    Save(ctx context.Context, robot *domain.Robot) error
}
```

```go
// internal/core/usecases/assign_task.go
package usecases

type AssignTaskInput struct {
    RobotID string
    TaskID  string
}

type AssignTaskUseCase struct {
    repo   RobotRepository
    notify Notifier
}

func (uc *AssignTaskUseCase) Execute(ctx context.Context, input AssignTaskInput) error {
    robot, err := uc.repo.Get(ctx, input.RobotID)
    if err != nil {
        return err
    }
    // бизнес-логика...
    return uc.repo.Save(ctx, robot)
}
```

### Адаптер (Driving: HTTP)

```go
// internal/adapters/driving/http/robot_handler.go
package http

type RobotHandler struct {
    assignTaskUC *usecases.AssignTaskUseCase
}

func (h *RobotHandler) AssignTask(w http.ResponseWriter, r *http.Request) {
    // парсим запрос → вызываем Use Case → возвращаем ответ
}
```

### Адаптер (Driven: PostgreSQL)

```go
// internal/adapters/driven/postgres/robot_repo.go
package postgres

type RobotRepository struct {
    db *sql.DB
}

func (r *RobotRepository) Get(ctx context.Context, id string) (*domain.Robot, error) {
    // SQL-запрос → маппинг → возврат доменной сущности
}
```

---

## ⚖️ Hexagonal vs Clean Architecture

| Критерий | Clean Architecture | Hexagonal Architecture |
|----------|--------------------|-------------------------|
| **Фокус** | Слои (Domain, UseCase, Delivery, Infrastructure) | Порты и адаптеры (границы) |
| **Ядро** | Знает о Use Cases и Domain | Знает только о Domain и портах |
| **Направление зависимостей** | Внутрь к ядру | Ядро не знает о внешнем мире (через порты) |
| **Тестируемость** | Высокая | Очень высокая |
| **Сложность** | Средняя | Выше (больше интерфейсов) |

---

## 🧠 Когда я выбираю Hexagonal

Я выбираю Hexagonal Architecture, когда:

- Система **должна быть максимально изолирована** от внешних систем.
- Есть **много интеграций** (разные БД, брокеры, API) и они могут меняться.
- Команда готова к **дополнительной сложности** ради гибкости.
- Важна **тестируемость на уровне ядра** без моков внешних систем.

---

## 📌 Ключевые преимущества

- **Максимальная изоляция** — ядро не знает о внешнем мире.
- **Лёгкая замена адаптеров** — можно переключиться с PostgreSQL на MongoDB, изменив только один файл.
- **Тестируемость** — Use Cases тестируются с мок-портами.

---

## 🚀 Как использовать этот раздел

1. **Определи порты** — какие интерфейсы нужны ядру и что оно предоставляет.
2. **Напиши ядро** — доменные сущности, Use Cases.
3. **Реализуй адаптеры** — для каждого порта.
4. **Собери всё в `main.go`** — внедри зависимости.

---

## 📎 Связанные документы

- [ADR: Выбор Hexagonal Architecture](../../docs/architecture/adr/009-hexagonal-choice.md)
- [Пример реализации Hexagonal в Go](../../patterns-examples/hexagonal-example/README.md)
- [Clean Architecture vs Hexagonal](../../docs/strategic/clean-vs-hexagonal.md)

---

*Hexagonal Architecture — это не про «шестиугольник». Это про **чёткие границы между ядром и миром**.*