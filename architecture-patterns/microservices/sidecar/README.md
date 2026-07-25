# Микросервисы с Sidecar

**Sidecar** — это паттерн, при котором вспомогательный процесс (sidecar) запускается рядом с основным приложением (обычно в том же Pod'е Kubernetes или на том же хосте) и предоставляет сквозные функции: логирование, метрики, трассировку, аутентификацию, управление конфигурацией и т.д.

Sidecar — это независимый компонент, который работает в одном жизненном цикле с основным сервисом, но не является его частью. Это позволяет выносить инфраструктурные задачи из основного кода, упрощая разработку и обеспечивая единообразие.

---

## 🧠 Основные принципы

1. **Сопровождение основного приложения.**  
   Sidecar работает рядом с сервисом и предоставляет сквозные функции.

2. **Независимость от языка.**  
   Sidecar может быть написан на одном языке (например, Go), а сервис — на любом другом.

3. **Единообразие.**  
   Все сервисы используют один и тот же sidecar, что обеспечивает единые логи, метрики, трассировку.

4. **Обновляемость.**  
   Обновление sidecar не требует пересборки сервиса.

5. **Изоляция.**  
   Sidecar изолирует инфраструктурный код от бизнес-логики.

---

## 🧩 Когда использовать Sidecar

- Нужно стандартизировать инфраструктурные задачи для множества сервисов.
- Сервисы написаны на разных языках — sidecar может быть реализован один раз и использоваться всеми.
- Требуется централизованный контроль (например, обновить логирование во всех сервисах, обновив один sidecar).
- Service Mesh уже используется или планируется (sidecar — это основа Service Mesh).

---

## 🏗️ Архитектура

```
[ Pod ]
  ├── [ Main Service ] ← бизнес-логика
  └── [ Sidecar ]      ← логи, метрики, трассировка, health, конфигурация
        │
        ▼
   [ Центральные системы (Prometheus, Elasticsearch, Jaeger) ]
```

---

## 🔧 Реализация на Go

**Простой sidecar (HTTP-сервер для метрик и логов):**

```go
package sidecar

import (
    "encoding/json"
    "net/http"
    "sync"
    "time"
)

type Sidecar struct {
    mu      sync.Mutex
    metrics []Metric
    logs    []LogEntry
}

type Metric struct {
    Name      string                 `json:"name"`
    Value     float64                `json:"value"`
    Labels    map[string]string      `json:"labels,omitempty"`
    Timestamp time.Time              `json:"timestamp"`
}

type LogEntry struct {
    Level     string                 `json:"level"`
    Message   string                 `json:"message"`
    Fields    map[string]interface{} `json:"fields,omitempty"`
    Timestamp time.Time              `json:"timestamp"`
}

func (s *Sidecar) HandleMetrics(w http.ResponseWriter, r *http.Request) {
    var m Metric
    json.NewDecoder(r.Body).Decode(&m)
    s.mu.Lock()
    s.metrics = append(s.metrics, m)
    s.mu.Unlock()
    w.WriteHeader(http.StatusOK)
}

func (s *Sidecar) HandleLogs(w http.ResponseWriter, r *http.Request) {
    var l LogEntry
    json.NewDecoder(r.Body).Decode(&l)
    s.mu.Lock()
    s.logs = append(s.logs, l)
    s.mu.Unlock()
    w.WriteHeader(http.StatusOK)
}
```

---

## ⚖️ Плюсы и минусы

| Плюсы | Минусы |
|-------|--------|
| ✅ Вынос инфраструктурного кода из приложения | ❌ Дополнительный процесс (ресурсы) |
| ✅ Единообразие для всех сервисов (независимо от языка) | ❌ Усложнение развёртывания (оркестрация) |
| ✅ Легко обновлять логику без пересборки сервисов | ❌ Задержка (дополнительный hop) |
| ✅ Поддержка Service Mesh и продвинутых сценариев | ❌ Может стать единой точкой отказа |
| ✅ Упрощает тестирование — можно мокать sidecar | ❌ Требуется управление версиями sidecar и приложения |

---

## 🚀 Использование в реальном проекте

1. **Определи сквозные задачи** — логи, метрики, трассировка, безопасность.
2. **Реализуй sidecar** на языке, который удобно использовать в инфраструктуре (Go, Python, Java).
3. **Настрой коммуникацию** между приложением и sidecar (обычно через localhost или Unix socket).
4. **Разверни sidecar вместе с каждым экземпляром приложения** (в Kubernetes — как sidecar-контейнер в Pod'е).
5. **Мониторь сам sidecar** — чтобы он не стал узким местом.
6. **Обновляй sidecar независимо** — новая версия логирования не требует пересборки приложения.

---

## 📎 Связанные документы

- [ADR: Sidecar Choice](../../../docs/architecture/adr/020-sidecar-choice.md)
- [Ambassador Pattern (альтернативный подход)](../../../patterns-examples/ambassador/README.md)
- [Пример реализации Sidecar в Go](../../../patterns-examples/sidecar/README.md)