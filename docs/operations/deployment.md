# Руководство по развертыванию: RMS (Robot Management System)

**Версия:** 1.0  
**Дата:** 2026-07-22  
**Статус:** Черновик

---

## 1. Введение

Данный документ описывает процесс развертывания RMS в различных окружениях: **development**, **staging** и **production**. Он предназначен для DevOps-инженеров, разработчиков и администраторов, участвующих в настройке и поддержке инфраструктуры.

Цель — обеспечить **воспроизводимый**, **автоматизированный** и **безопасный** процесс развертывания.

---

## 2. Архитектура развертывания

### 2.1 Компоненты

| Компонент | Описание | Технология |
|-----------|----------|------------|
| **Backend API** | REST + gRPC сервисы | Go, Gin, gRPC-Go |
| **Frontend UI** | Web-интерфейс для операторов | React, TypeScript |
| **База данных** | Хранение данных | PostgreSQL |
| **Кэш** | Сессии, кэширование | Redis |
| **Брокер событий** | Асинхронная коммуникация | Kafka |
| **Мониторинг** | Сбор метрик и алертинг | Prometheus + Grafana |
| **Логирование** | Сбор и анализ логов | ELK / Loki |
| **Оркестрация** | Управление контейнерами | Kubernetes |

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
│  │          Service Mesh (Istio / Linkerd)         │  │
│  └──────────────────────────────────────────────────┘  │
│                         │                              │
│         ┌───────────────┼───────────────┐              │
│         ▼               ▼               ▼              │
│  ┌────────────┐  ┌────────────┐  ┌────────────┐      │
│  │ PostgreSQL │  │   Redis    │  │   Kafka    │      │
│  └────────────┘  └────────────┘  └────────────┘      │
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

### 3.2 Переменные окружения (пример)

Создайте файл `.env` для каждого окружения:

```bash
# Общие
APP_ENV=production
LOG_LEVEL=info

# База данных
DB_HOST=postgres-service
DB_PORT=5432
DB_USER=rms_user
DB_PASSWORD=<secure>
DB_NAME=rms_db

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
   git clone https://github.com/your-org/rms.git
   cd rms
   ```

2. **Запустить локальное окружение через Docker Compose:**
   ```bash
   make dev-up
   ```
   Это поднимает:
   - PostgreSQL (порт 5432)
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
   curl http://staging.rms.local/health
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
   - Smoke-тесты (ключевые сценарии).
   - Мониторинг метрик и ошибок.

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
  name: rms-config
data:
  APP_ENV: production
  LOG_LEVEL: info
  DB_HOST: postgres-service
  REDIS_HOST: redis-service
  KAFKA_BROKERS: kafka-broker:9092
  API_PORT: "8080"
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

- **Метрики:** доступны в Grafana (дашборд `RMS Production`).
- **Логи:** доступны в ELK / Loki через Kibana / Grafana Explore.
- **Алерты:** настроены в Alertmanager (уведомления в Slack).

---

## 8. Тестирование развертывания

- **Health-проверка:** проверить все эндпоинты `/health`.
- **Smoke-тесты:** выполнить ключевые сценарии (создание робота, задачи, заказа).
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