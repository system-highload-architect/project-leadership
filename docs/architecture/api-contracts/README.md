# 🔌 API-контракты

**Эта директория содержит все спецификации API, которые система предоставляет внешним потребителям.**  
Здесь описаны контракты для синхронного (REST/gRPC) и асинхронного (события) взаимодействия.

API-контракты — это **договор между системой и её клиентами**. Они определяют, как взаимодействовать с системой, какие данные отправлять и какие ответы ожидать.

---

## 🧭 Навигация по разделам

| Файл / Директория | Описание | Формат |
|-------------------|----------|--------|
| [openapi.yaml](openapi.yaml) | Спецификация REST API (OpenAPI 3.0) | YAML |
| [proto/](proto/) | gRPC-спецификации (Protocol Buffers) | `.proto` |
| [proto/robot.proto](proto/robot.proto) | gRPC-сервис для управления роботами | Protobuf |

---

## 🧠 Принципы проектирования API

1. **Контракт — первичен.** API проектируется сначала как контракт, затем реализуется в коде (Contract-First подход).
2. **Обратная совместимость.** Изменения API не должны ломать существующих клиентов. Для breaking changes — новая версия API.
3. **Чёткая версионность.** Версия API указывается в URL (например, `/v1/robots`) или в заголовках.
4. **Документированность.** Каждый эндпоинт, поле и статус ответа должны быть документированы.
5. **Единый стиль.** Все API следуют единым правилам именования, форматирования и обработки ошибок.

---

## 📁 Структура директории

```
api-contracts/
├── README.md                    # Этот файл
├── openapi.yaml                 # REST API (OpenAPI 3.0)
└── proto/                       # gRPC (Protocol Buffers)
    ├── robot.proto              # Сервис управления роботами
    ├── order.proto              # Сервис управления заказами
    └── task.proto               # Сервис управления задачами
```

---

## 🔧 REST API (OpenAPI)

Файл `openapi.yaml` содержит спецификацию REST API в формате OpenAPI 3.0.

**Основные эндпоинты:**

| Метод | Путь | Описание |
|-------|------|----------|
| `GET` | `/v1/robots` | Получить список всех роботов |
| `GET` | `/v1/robots/{id}` | Получить информацию о роботе |
| `POST` | `/v1/robots` | Создать нового робота |
| `PUT` | `/v1/robots/{id}/status` | Изменить статус робота |
| `DELETE` | `/v1/robots/{id}` | Удалить робота |
| `POST` | `/v1/orders` | Создать заказ |
| `GET` | `/v1/orders/{id}` | Получить информацию о заказе |
| ... | ... | ... |

**Пример фрагмента `openapi.yaml`:**

```yaml
openapi: 3.0.0
info:
  title: Robot Management System API
  version: 1.0.0
  description: API для управления роботами, задачами и заказами

paths:
  /v1/robots:
    get:
      summary: Получить список роботов
      parameters:
        - name: status
          in: query
          schema:
            type: string
            enum: [idle, busy, maintenance, offline]
          description: Фильтр по статусу
      responses:
        '200':
          description: Список роботов
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: '#/components/schemas/Robot'

components:
  schemas:
    Robot:
      type: object
      properties:
        id:
          type: string
          format: uuid
        name:
          type: string
        status:
          type: string
          enum: [idle, busy, maintenance, offline]
        location:
          type: string
```

---

## 🔧 gRPC API (Protocol Buffers)

Файлы `.proto` содержат спецификации gRPC-сервисов.

**Основные сервисы:**

| Сервис | Метод | Описание |
|--------|-------|----------|
| `RobotService` | `GetRobot` | Получить информацию о роботе |
| `RobotService` | `ListRobots` | Получить список роботов |
| `RobotService` | `UpdateStatus` | Изменить статус робота |
| `OrderService` | `CreateOrder` | Создать заказ |
| `OrderService` | `GetOrder` | Получить информацию о заказе |
| `TaskService` | `AssignTask` | Назначить задачу роботу |
| `TaskService` | `CompleteTask` | Завершить задачу |

**Пример фрагмента `robot.proto`:**

```protobuf
syntax = "proto3";

package robot.v1;

service RobotService {
  rpc GetRobot(GetRobotRequest) returns (Robot);
  rpc ListRobots(ListRobotsRequest) returns (ListRobotsResponse);
  rpc UpdateStatus(UpdateStatusRequest) returns (Robot);
}

message Robot {
  string id = 1;
  string name = 2;
  string status = 3;
  string location_id = 4;
}

message GetRobotRequest {
  string id = 1;
}

message ListRobotsRequest {
  optional string status = 1;
  int32 limit = 2;
  int32 offset = 3;
}

message ListRobotsResponse {
  repeated Robot robots = 1;
  int32 total = 2;
}
```

---

## 🚀 Как использовать API-контракты

1. **Для разработки бэкенда:** Контракты определяют интерфейсы, которые нужно реализовать.
2. **Для разработки фронтенда:** Контракты определяют, как взаимодействовать с бэкендом (можно генерировать клиентский код).
3. **Для тестирования:** Контракты используются для написания интеграционных тестов и моков.
4. **Для документации:** OpenAPI можно визуализировать через Swagger UI, а Protobuf — через `protoc` + `protoc-gen-doc`.

**Генерация документации:**

```bash
# Для OpenAPI — Swagger UI
npx swagger-ui-watcher openapi.yaml

# Для Protobuf — документация через protoc-gen-doc
protoc --doc_out=. --doc_opt=markdown,api.md proto/*.proto
```

---

## 📌 Важно

- **Любое изменение API** должно быть отражено в контракте **до** изменения кода.
- **Breaking changes** требуют создания новой версии API (например, `/v2/robots`).
- **Контракты должны быть валидными** — проверяй их валидаторами (например, `swagger-cli validate` для OpenAPI).
- **Все API должны быть задокументированы** — каждый эндпоинт, поле и статус ответа.

---

## 📎 Связанные документы

- [Архитектурные документы](../README.md) — общий раздел архитектуры.
- [Модель данных](../data-model/README.md) — структура данных, которые передаются через API.
- [ADR: Выбор gRPC vs REST](../../adr/008-grpc-vs-rest.md) — обоснование использования gRPC.

---

*API-контракты — это не просто файлы, а **договор между командами и системами**.*

---