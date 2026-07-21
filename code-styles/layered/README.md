# Layered Architecture (n-tier)

**Layered Architecture** (также известная как **n-tier**) — это классический стиль организации кода, при котором система делится на **горизонтальные слои**, каждый из которых выполняет строго определённую функцию. Это самый простой и интуитивно понятный подход, который часто используется как «входная точка» для начинающих архитекторов.

Несмотря на свою простоту, Layered Architecture остаётся актуальной для многих проектов, особенно на начальных этапах, когда сложность системы ещё не требует более изощрённых подходов.

---

## 🧠 Основные принципы

1. **Каждый слой выполняет свою функцию** — слои не пересекаются по ответственности.
2. **Коммуникация сверху вниз** — верхний слой вызывает нижний, но не наоборот.
3. **Чёткие границы** — изменения в одном слое не должны затрагивать другие (при соблюдении контрактов).
4. **Простота** — лёгкость понимания и быстрый старт.

---

## 🧩 Слои классической n-tier

| Слой | Назначение | Пример |
|------|------------|--------|
| **Presentation** | Взаимодействие с пользователем (UI, API, CLI) | HTTP-хендлеры, gRPC-серверы, CLI-команды |
| **Business Logic** (BLL) | Бизнес-правила, валидация, оркестрация | Use Cases, Service Layer, Domain Models |
| **Data Access** (DAL) | Работа с БД, внешними сервисами, кэшем | Репозитории, ORM, клиенты для внешних API |
| **Cross-cutting** (опционально) | Логирование, мониторинг, безопасность | Middleware, Interceptors, Filters |

---

## 📁 Структура проекта (шаблон)

```
layered-example/
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── delivery/                         # Presentation Layer
│   │   ├── http/
│   │   │   └── robot_handler.go
│   │   └── grpc/
│   │       └── robot_server.go
│   ├── service/                          # Business Logic Layer
│   │   ├── robot_service.go              # Использует репозитории
│   │   └── task_service.go
│   ├── repository/                       # Data Access Layer
│   │   ├── robot_repo.go                 # Интерфейс
│   │   └── postgres/
│   │       └── robot_repo.go             # Реализация
│   └── domain/                           # Модели (data transfer)
│       ├── robot.go
│       └── task.go
└── pkg/
    ├── logger/
    └── config/
```

---

## 📝 Пример на Go

### Presentation Layer (HTTP)

```go
// internal/delivery/http/robot_handler.go
package http

type RobotHandler struct {
    robotSvc *service.RobotService
}

func (h *RobotHandler) GetRobot(w http.ResponseWriter, r *http.Request) {
    // 1. Получить данные из запроса
    // 2. Вызвать сервисный слой
    // 3. Вернуть ответ
}
```

### Business Logic Layer (Service)

```go
// internal/service/robot_service.go
package service

type RobotService struct {
    repo repository.RobotRepository
}

func (s *RobotService) GetRobot(id string) (*domain.Robot, error) {
    return s.repo.Get(id)
}
```

### Data Access Layer (Repository)

```go
// internal/repository/postgres/robot_repo.go
package postgres

type RobotRepository struct {
    db *sql.DB
}

func (r *RobotRepository) Get(id string) (*domain.Robot, error) {
    // SQL-запрос → маппинг → возврат
}
```

---

## ⚖️ Layered vs Clean/Hexagonal

| Критерий | Layered Architecture | Clean/Hexagonal |
|----------|-----------------------|-----------------|
| **Сложность** | Низкая | Средняя/Высокая |
| **Скорость разработки** | Высокая (на старте) | Ниже (требует больше проектирования) |
| **Тестируемость** | Средняя (зависит от БД) | Высокая (изоляция бизнес-логики) |
| **Гибкость** | Низкая (замена слоёв сложна) | Высокая |
| **Поддержка эволюции** | Сложная (слои могут «распухнуть») | Хорошая |

---

## 🧠 Когда я выбираю Layered Architecture

Я выбираю Layered Architecture, когда:

- **Проект простой** — мало бизнес-логики, нет сложных интеграций.
- **Команда небольшая** (до 5 разработчиков) и нужен быстрый старт.
- **Проект имеет короткий жизненный цикл** (прототип, MVP, PoC).
- **Нет жёстких требований к гибкости** и замене технологий.

---

## ⚠️ Ограничения

- **«Распухание» слоёв** — бизнес-логика может начать жить и в Presentation, и в Data Access.
- **Сложность тестирования** — бизнес-логика часто привязана к БД.
- **Ограниченная эволюционность** — замена слоя (например, БД) может быть болезненной.

---

## 🚀 Как использовать этот раздел

1. **Начни с Layered Architecture** для быстрого старта.
2. **Переходи к Clean Architecture**, когда сложность вырастет.
3. **Используй Layered как «первый шаг»** к более гибким подходам.
4. **Не бойся комбинировать** — например, Layered + DDD внутри одного слоя.

---

## 📎 Связанные документы

- [ADR: Выбор Layered Architecture для MVP](../../docs/architecture/adr/011-layered-choice.md)
- [Пример реализации Layered в Go](../../patterns-examples/layered-example/README.md)
- [Миграция от Layered к Clean Architecture](../../docs/strategic/migration-layered-to-clean.md)

---

*Layered Architecture — это не «устаревший» подход. Это **прагматичный выбор для простых задач**, когда важна скорость и простота.*