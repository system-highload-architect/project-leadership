# Руководство по развертыванию: E‑Commerce Platform

**Версия:** 1.0  
**Дата:** 2026-07-26  
**Статус:** Черновик

---

## 1. Введение

Данный документ описывает процесс развертывания E‑Commerce платформы в различных окружениях: **development**, **staging** и **production**. Он предназначен для DevOps-инженеров, разработчиков и администраторов, участвующих в настройке и поддержке инфраструктуры.

Цель — обеспечить **воспроизводимый**, **автоматизированный** и **безопасный** процесс развертывания с учётом архитектурных решений (модульный монолит, EDA, Kafka, PostgreSQL, CQRS, Event Sourcing, Saga).

---

## 2. Архитектура развертывания

### 2.1 Компоненты

| Компонент | Описание | Технология |
|-----------|----------|------------|
| **Backend API** | REST + gRPC сервисы (модульный монолит) | Go, Gin, gRPC-Go |
| **Frontend UI** | Web-интерфейс для покупателей, менеджеров, администраторов | React, TypeScript, Vite |
| **База данных (Write)** | Хранение данных (нормализованная схема) | PostgreSQL 15+ |
| **База данных (Read)** | Денормализованные проекции для CQRS | PostgreSQL (или ClickHouse для аналитики) |
| **Event Store** | Хранение событий для Event Sourcing (Order Service) | PostgreSQL (таблица `order_events`) |
| **Кэш** | Сессии, кэширование, корзина | Redis |
| **Брокер событий** | Асинхронная коммуникация | Kafka 3.x |
| **Мониторинг** | Сбор метрик и алертинг | Prometheus + Grafana |
| **Логирование** | Сбор и анализ логов | Loki + Grafana (или ELK) |
| **Трассировка** | Распределённая трассировка | Jaeger / OpenTelemetry |
| **Оркестрация** | Управление контейнерами | Kubernetes (k3s на старте, позже managed) |
| **CI/CD** | Автоматизация сборки и деплоя | GitHub Actions |

### 2.2 Схема развертывания

```
┌─────────────────────────────────────────────────────────┐
│                  Kubernetes Cluster                     │
│  ┌───────────────┐  ┌───────────────┐  ┌─────────────┐ │
│  │   Frontend    │  │    Backend    │  │   Backend   │ │
│  │   (React)     │  │  (REST API)   │  │   (gRPC)    │ │
│  └───────┬───────┘  └───────┬───────┘  └──────┬──────┘ │
│          │                  │                  │        │
│          ▼                  ▼                  ▼        │
│  ┌──────────────────────────────────────────────────┐  │
│  │          Service Mesh (опционально)             │  │
│  └──────────────────────────────────────────────────┘  │
│                         │                              │
│         ┌───────────────┼───────────────┐              │
│         ▼               ▼               ▼              │
│  ┌────────────┐  ┌────────────┐  ┌────────────┐      │
│  │ PostgreSQL │  │   Redis    │  │   Kafka    │      │
│  │ (Write DB) │  │            │  │            │      │
│  └────────────┘  └────────────┘  └────────────┘      │
│  ┌────────────┐                                      │
│  │ PostgreSQL │  (Read DB / Event Store)              │
│  └────────────┘                                      │
└─────────────────────────────────────────────────────────┘
```

---

## 3. Требования к окружению

### 3.1 Аппаратные и программные требования

| Компонент | Требование |
|-----------|------------|
| **Kubernetes** | Версия 1.24+ |
| **Docker** | Версия 20.10+ |
| **Helm** (опционально) | Версия 3.0+ |
| **kubectl** | Настроенный доступ к кластеру |
| **PostgreSQL** | Версия 15+ (managed или self-hosted) |
| **Kafka** | Версия 3.x (managed или self-hosted) |
| **Redis** | Версия 6.2+ |
| **Go** | 1.23+ (для сборки) |
| **Node.js** | 18+ (для фронтенда) |

### 3.2 Переменные окружения (пример)

Создайте файл `.env` для каждого окружения:

```bash
# Общие
APP_ENV=production
LOG_LEVEL=info

# База данных (Write)
DB_WRITE_HOST=postgres-write-service
DB_WRITE_PORT=5432
DB_WRITE_USER=ecommerce_user
DB_WRITE_PASSWORD=<secure>
DB_WRITE_NAME=ecommerce_db

# База данных (Read) — опционально
DB_READ_HOST=postgres-read-service
DB_READ_PORT=5432
DB_READ_USER=ecommerce_user
DB_READ_PASSWORD=<secure>
DB_READ_NAME=ecommerce_read_db

# Event Store (может быть отдельной БД)
EVENT_STORE_HOST=postgres-event-store
EVENT_STORE_PORT=5432
EVENT_STORE_USER=ecommerce_user
EVENT_STORE_PASSWORD=<secure>
EVENT_STORE_NAME=ecommerce_events

# Redis
REDIS_HOST=redis-service
REDIS_PORT=6379

# Kafka
KAFKA_BROKERS=kafka-broker:9092

# API
API_PORT=8080
GRPC_PORT=50051

# JWT
JWT_SECRET=<secure>
JWT_EXPIRE=24h

# Платёжный шлюз
PAYMENT_GATEWAY_URL=https://payment.example.com
PAYMENT_GATEWAY_API_KEY=<secure>

# Мониторинг
PROMETHEUS_ENABLED=true
```

---

## 4. Процесс развертывания

### 4.1 Development (локальное окружение)

**Цель:** Быстрая разработка и отладка.

**Шаги:**

1. **Клонировать репозиторий:**
   ```bash
   git clone https://github.com/your-org/ecommerce.git
   cd ecommerce
   ```

2. **Запустить локальное окружение через Docker Compose:**
   ```bash
   make dev-up
   ```
   Это поднимает:
   - PostgreSQL (Write DB, Read DB, Event Store) — порты 5432, 5433, 5434
   - Redis (6379)
   - Kafka (9092)
   - Backend API (8080)
   - Frontend UI (3000)

3. **Применить миграции БД:**
   ```bash
   make migrate-up
   ```

4. **Проверить работу:**
   - API: `http://localhost:8080/health`
   - gRPC: `localhost:50051`
   - UI: `http://localhost:3000`

---

### 4.2 Staging (промежуточное окружение)

**Цель:** Тестирование перед релизом, интеграционные тесты, демо.

**Шаги:**

1. **Сборка Docker-образов:**
   ```bash
   make build-images
   ```

2. **Публикация образов в реестр:**
   ```bash
   make push-images
   ```

3. **Развертывание в Kubernetes:**
   ```bash
   kubectl apply -f k8s/staging/
   ```

4. **Применить миграции:**
   ```bash
   kubectl exec -it pod/backend -- ./migrate up
   ```

5. **Проверка:**
   ```bash
   kubectl get pods
   curl http://staging.ecommerce.local/health
   ```

---

### 4.3 Production (боевое окружение)

**Цель:** Стабильная работа в продакшене.

**Шаги:**

1. **Подготовка релизного тега:**
   ```bash
   git tag v1.0.0
   git push origin v1.0.0
   ```

2. **CI/CD автоматически:**
   - Сборка образов с тегом `v1.0.0`.
   - Публикация в реестре.
   - Развертывание в production через **GitHub Actions** (с ручным подтверждением).

3. **Стратегия развертывания:** **Rolling Update** (постепенное обновление pod'ов без простоя).
   ```yaml
   spec:
     strategy:
       type: RollingUpdate
       rollingUpdate:
         maxSurge: 1
         maxUnavailable: 0
   ```

4. **Проверка после деплоя:**
   - Health-проверка (`/health`).
   - Smoke-тесты (ключевые сценарии: создание заказа, платёж).
   - Мониторинг метрик и ошибок (Grafana).

---

## 5. Управление конфигурацией

- Используйте **ConfigMap** и **Secrets** в Kubernetes.
- Секреты хранятся в **HashiCorp Vault** или **Kubernetes Secrets**.
- Для разных окружений используйте разные файлы конфигурации (например, `config-dev.yaml`, `config-prod.yaml`).

**Пример ConfigMap:**
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: ecommerce-config
data:
  APP_ENV: production
  LOG_LEVEL: info
  DB_WRITE_HOST: postgres-write-service
  DB_READ_HOST: postgres-read-service
  REDIS_HOST: redis-service
  KAFKA_BROKERS: kafka-broker:9092
  API_PORT: "8080"
  GRPC_PORT: "50051"
```

---

## 6. Откат (Rollback)

В случае проблем выполните откат к предыдущей версии:

```bash
# Получить историю релизов
kubectl rollout history deployment/backend

# Откат к предыдущей версии
kubectl rollout undo deployment/backend

# Откат к конкретной версии (например, к ревизии 3)
kubectl rollout undo deployment/backend --to-revision=3
```

**Время отката:** не более 5 минут.

---

## 7. Мониторинг и логи

- **Метрики:** доступны в Grafana (дашборд `E‑Commerce Production`).
- **Логи:** доступны в Loki / Elasticsearch через Grafana Explore.
- **Алерты:** настроены в Alertmanager (уведомления в Slack).

---

## 8. Тестирование развертывания

- **Health-проверка:** проверить все эндпоинты `/health`.
- **Smoke-тесты:** выполнить ключевые сценарии (создание заказа, платёж, просмотр каталога).
- **Нагрузочное тестирование:** K6 / Locust перед релизом (опционально).

---

## 9. Безопасность

- **TLS:** все соединения зашифрованы (Ingress с Let's Encrypt).
- **Сетевые политики:** ограничен доступ между Pod'ами.
- **Аутентификация:** JWT для API, RBAC для Kubernetes.
- **Сканирование образов:** перед деплоем проверяются на уязвимости (Trivy / Snyk).

---

## 10. Связь с другими документами

- [engineering-practices.md](../project-management/engineering-practices.md) — CI/CD пайплайн.
- [runbook.md](runbook.md) — инструкции при сбоях.
- [disaster-recovery.md](disaster-recovery.md) — план восстановления.
- [slo-sli.md](slo-sli.md) — метрики доступности и производительности.

---

*Руководство обновляется при изменении инфраструктуры или процесса развертывания.*

---