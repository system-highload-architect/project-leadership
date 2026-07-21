# Sidecar Pattern (Паттерн «Боковой прицеп»)

**Sidecar Pattern** — это паттерн, при котором **вспомогательный процесс (sidecar) запускается рядом с основным приложением** (обычно в том же контейнере или на том же хосте) и предоставляет сквозные функции: логирование, метрики, сбор трассировки, аутентификацию, кэширование, управление конфигурацией и т.д.

Sidecar — это **независимый компонент**, который работает в одном жизненном цикле с основным сервисом, но не является его частью. Это позволяет выносить инфраструктурные задачи из основного кода, упрощая разработку и обеспечивая единообразие.

---

## 🧠 Основная идея

Sidecar — это **компаньон основного приложения**, который:

- **Запускается вместе с основным процессом** (например, как отдельный контейнер в Pod'е Kubernetes).
- **Перехватывает или дополняет** сетевые вызовы, логи, метрики.
- **Предоставляет стандартизированный интерфейс** для сквозной функциональности.
- **Не влияет на бизнес-логику** — основное приложение может даже не знать о его существовании.

Sidecar часто используется в **Service Mesh** (например, Envoy, Linkerd) как data plane, но может применяться и для более простых задач: логирование в Elasticsearch, сбор метрик в Prometheus, управление секретами и т.д.

---

## 🧩 Когда я использую Sidecar

Я выбираю Sidecar Pattern, когда:

- **Нужно стандартизировать инфраструктурные задачи** для множества сервисов.
- **Не хочу загрязнять основной код** логированием, метриками, ретраями.
- **Сервисы написаны на разных языках** — sidecar может быть реализован один раз и использоваться всеми.
- **Требуется централизованный контроль** (например, обновить логирование во всех сервисах, обновив один sidecar).
- **Service Mesh** уже используется или планируется.

---

## 🧩 Как это работает

```
┌──────────────────────────────────────┐
│  Pod / Хост                          │
│  ┌─────────────┐   ┌─────────────┐  │
│  │  Основное   │   │   Sidecar   │  │
│  │  приложение │◄─►│  (логи,     │  │
│  │  (Go, Java, │   │   метрики,  │  │
│  │   Python)   │   │   прокси)   │  │
│  └─────────────┘   └──────┬──────┘  │
│                            │         │
└────────────────────────────┼─────────┘
                             │
                      ┌──────▼──────┐
                      │  Централь-  │
                      │  ные сис-   │
                      │  темы (ES,  │
                      │  Prometheus)│
                      └─────────────┘
```

Основное приложение может:
- **Напрямую вызывать sidecar** (например, для логирования через HTTP или gRPC).
- **Перенаправлять трафик через sidecar** (как в Service Mesh, где sidecar является прокси).
- **Отправлять метрики** в sidecar, который уже передаёт их в центральную систему.

---

## 🧩 Пример реализации на Go

В этом примере мы реализуем **Sidecar**, который выполняет три функции:

1. **Сбор метрик** — принимает метрики через HTTP и агрегирует их.
2. **Логирование** — принимает логи и отправляет их в stdout (в реальном проекте — в Elasticsearch).
3. **Прокси-запросов** — перехватывает HTTP-запросы, логирует их и передаёт дальше.

```go
package sidecar

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// ===== Конфигурация =====

// Config — настройки Sidecar
type Config struct {
	MetricsPort    int           // порт для сбора метрик
	LoggingPort    int           // порт для сбора логов
	ProxyPort      int           // порт для прокси
	ForwardURL     string        // URL, куда проксировать запросы
	BufferSize     int           // размер буфера для метрик/логов
	FlushInterval  time.Duration // интервал сброса в центральную систему
}

// ===== Sidecar =====

// Sidecar — основной компонент
type Sidecar struct {
	cfg    Config
	server *http.Server
	wg     sync.WaitGroup

	// Хранилище метрик и логов
	metrics []Metric
	logs    []LogEntry
	mu      sync.Mutex
}

// Metric — структура метрики
type Metric struct {
	Name      string                 `json:"name"`
	Value     float64                `json:"value"`
	Labels    map[string]string      `json:"labels,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// LogEntry — структура лога
type LogEntry struct {
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// NewSidecar создаёт новый экземпляр Sidecar
func NewSidecar(cfg Config) *Sidecar {
	return &Sidecar{
		cfg:     cfg,
		metrics: make([]Metric, 0, cfg.BufferSize),
		logs:    make([]LogEntry, 0, cfg.BufferSize),
	}
}

// ===== Запуск и остановка =====

// Start запускает все серверы Sidecar
func (s *Sidecar) Start(ctx context.Context) error {
	// Запускаем периодический сброс данных
	s.wg.Add(1)
	go s.flushWorker(ctx)

	// HTTP-сервер для метрик
	metricsMux := http.NewServeMux()
	metricsMux.HandleFunc("POST /metrics", s.handleMetrics)
	metricsServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", s.cfg.MetricsPort),
		Handler: metricsMux,
	}
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("[Sidecar] Metrics server error: %v\n", err)
		}
	}()

	// HTTP-сервер для логов
	logsMux := http.NewServeMux()
	logsMux.HandleFunc("POST /logs", s.handleLogs)
	logsServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", s.cfg.LoggingPort),
		Handler: logsMux,
	}
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		if err := logsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("[Sidecar] Logs server error: %v\n", err)
		}
	}()

	// HTTP-сервер для прокси (если настроен)
	if s.cfg.ForwardURL != "" {
		proxyMux := http.NewServeMux()
		proxyMux.HandleFunc("/", s.handleProxy)
		proxyServer := &http.Server{
			Addr:    fmt.Sprintf(":%d", s.cfg.ProxyPort),
			Handler: proxyMux,
		}
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			if err := proxyServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				fmt.Printf("[Sidecar] Proxy server error: %v\n", err)
			}
		}()
	}

	// Ждём сигнала остановки
	<-ctx.Done()

	// Останавливаем серверы
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := metricsServer.Shutdown(shutdownCtx); err != nil {
		fmt.Printf("[Sidecar] Metrics server shutdown error: %v\n", err)
	}
	if err := logsServer.Shutdown(shutdownCtx); err != nil {
		fmt.Printf("[Sidecar] Logs server shutdown error: %v\n", err)
	}
	// proxy server аналогично, но для простоты пропустим

	s.wg.Wait()
	return nil
}

// ===== Обработчики HTTP =====

// handleMetrics принимает метрики от приложения
func (s *Sidecar) handleMetrics(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var m Metric
	if err := json.Unmarshal(body, &m); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if m.Timestamp.IsZero() {
		m.Timestamp = time.Now()
	}

	s.mu.Lock()
	s.metrics = append(s.metrics, m)
	// Если буфер переполнен — немедленно отправляем
	if len(s.metrics) >= s.cfg.BufferSize {
		s.flushMetrics()
	}
	s.mu.Unlock()

	w.WriteHeader(http.StatusOK)
}

// handleLogs принимает логи от приложения
func (s *Sidecar) handleLogs(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var entry LogEntry
	if err := json.Unmarshal(body, &entry); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}

	s.mu.Lock()
	s.logs = append(s.logs, entry)
	if len(s.logs) >= s.cfg.BufferSize {
		s.flushLogs()
	}
	s.mu.Unlock()

	w.WriteHeader(http.StatusOK)
}

// handleProxy — проксирует запросы к внешнему сервису (пример)
func (s *Sidecar) handleProxy(w http.ResponseWriter, r *http.Request) {
	// Логируем входящий запрос
	s.mu.Lock()
	s.logs = append(s.logs, LogEntry{
		Level:   "info",
		Message: fmt.Sprintf("Proxying request: %s %s", r.Method, r.URL.Path),
		Fields: map[string]interface{}{
			"method": r.Method,
			"path":   r.URL.Path,
			"remote": r.RemoteAddr,
		},
		Timestamp: time.Now(),
	})
	s.mu.Unlock()

	// Если прокси не настроен — возвращаем ошибку
	if s.cfg.ForwardURL == "" {
		http.Error(w, "proxy not configured", http.StatusServiceUnavailable)
		return
	}

	// Формируем запрос к целевому сервису
	targetURL := s.cfg.ForwardURL + r.URL.Path
	req, err := http.NewRequestWithContext(r.Context(), r.Method, targetURL, r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// Копируем заголовки
	for key, values := range r.Header {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Копируем ответ
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)

	// Логируем ответ
	s.mu.Lock()
	s.logs = append(s.logs, LogEntry{
		Level:   "info",
		Message: fmt.Sprintf("Proxy response: %d", resp.StatusCode),
		Fields: map[string]interface{}{
			"status": resp.StatusCode,
			"path":   r.URL.Path,
		},
		Timestamp: time.Now(),
	})
	s.mu.Unlock()
}

// ===== Фоновый сброс данных =====

// flushWorker периодически сбрасывает буферизированные данные
func (s *Sidecar) flushWorker(ctx context.Context) {
	defer s.wg.Done()
	ticker := time.NewTicker(s.cfg.FlushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// Финальный сброс
			s.mu.Lock()
			s.flushMetrics()
			s.flushLogs()
			s.mu.Unlock()
			return
		case <-ticker.C:
			s.mu.Lock()
			s.flushMetrics()
			s.flushLogs()
			s.mu.Unlock()
		}
	}
}

// flushMetrics отправляет метрики в центральную систему (в примере — просто печатает)
func (s *Sidecar) flushMetrics() {
	if len(s.metrics) == 0 {
		return
	}
	// В реальном проекте — отправка в Prometheus, InfluxDB и т.д.
	for _, m := range s.metrics {
		fmt.Printf("[Sidecar] Metric: %s = %f (labels: %v)\n", m.Name, m.Value, m.Labels)
	}
	s.metrics = s.metrics[:0]
}

// flushLogs отправляет логи в центральную систему (в примере — просто печатает)
func (s *Sidecar) flushLogs() {
	if len(s.logs) == 0 {
		return
	}
	// В реальном проекте — отправка в Elasticsearch, Loki и т.д.
	for _, entry := range s.logs {
		fmt.Printf("[Sidecar] Log [%s]: %s (fields: %v)\n", entry.Level, entry.Message, entry.Fields)
	}
	s.logs = s.logs[:0]
}
```

---

## 🧪 Пример использования

### Основное приложение (клиент Sidecar)

```go
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

func main() {
	sidecarMetricsURL := "http://localhost:8081/metrics"
	sidecarLogsURL := "http://localhost:8082/logs"

	// Отправляем метрику
	metric := map[string]interface{}{
		"name":      "orders_processed",
		"value":     42,
		"labels":    map[string]string{"service": "order-api"},
		"timestamp": time.Now(),
	}
	sendJSON(sidecarMetricsURL, metric)

	// Отправляем лог
	logEntry := map[string]interface{}{
		"level":   "info",
		"message": "User 123 placed an order",
		"fields":  map[string]interface{}{"user_id": 123, "order_id": "ord-456"},
	}
	sendJSON(sidecarLogsURL, logEntry)

	fmt.Println("Data sent to sidecar")
}

func sendJSON(url string, data interface{}) {
	b, _ := json.Marshal(data)
	resp, err := http.Post(url, "application/json", bytes.NewReader(b))
	if err != nil {
		fmt.Println("Error sending:", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		fmt.Printf("Sidecar responded with status %d\n", resp.StatusCode)
	}
}
```

### Запуск Sidecar

```go
package main

import (
	"context"
	"os"
	"os/signal"
	"time"
)

func main() {
	cfg := sidecar.Config{
		MetricsPort:   8081,
		LoggingPort:   8082,
		ProxyPort:     8083,
		ForwardURL:    "https://api.example.com",
		BufferSize:    10,
		FlushInterval: 5 * time.Second,
	}

	sidecar := sidecar.NewSidecar(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := sidecar.Start(ctx); err != nil {
			fmt.Printf("Sidecar error: %v\n", err)
		}
	}()

	fmt.Println("Sidecar started on ports 8081 (metrics), 8082 (logs), 8083 (proxy)")
	// Ждём сигнал завершения
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	<-sigCh
	fmt.Println("Shutting down sidecar...")
	cancel()
	time.Sleep(2 * time.Second)
}
```

---

## ⚖️ Плюсы и минусы

| Плюсы | Минусы |
|-------|--------|
| ✅ Вынос инфраструктурного кода из приложения | ❌ Дополнительный процесс (ресурсы) |
| ✅ Единообразие для всех сервисов (независимо от языка) | ❌ Усложнение развёртывания (оркестрация) |
| ✅ Легко обновлять логику без пересборки сервисов | ❌ Задержка (дополнительный hop) |
| ✅ Поддержка Service Mesh и продвинутых сценариев | ❌ Может стать единой точкой отказа (если не кластеризован) |
| ✅ Упрощает тестирование — можно мокать sidecar | ❌ Требуется управление версиями sidecar и приложения |

---

## 🧠 Sidecar vs Ambassador

| Критерий | Sidecar | Ambassador |
|----------|---------|------------|
| **Расположение** | На том же хосте/контейнере, что и основной сервис | Отдельный сервис (может быть общим) |
| **Назначение** | Сопровождение основного приложения (логи, метрики, прокси) | Прокси для внешних вызовов |
| **Масштабирование** | Один sidecar на экземпляр сервиса | Один Ambassador может обслуживать несколько сервисов |
| **Пример** | Envoy, Logstash sidecar | API Gateway, внешний прокси |

---

## 🚀 Как использовать в реальном проекте

1. **Определи сквозные задачи** — логи, метрики, трассировка, безопасность.
2. **Реализуй sidecar** на языке, который удобно использовать в инфраструктуре (Go, Python, Java).
3. **Настрой коммуникацию** между приложением и sidecar (обычно через localhost или Unix socket).
4. **Разверни sidecar вместе с каждым экземпляром приложения** (в Kubernetes — как sidecar-контейнер в Pod'е).
5. **Мониторь сам sidecar** — чтобы он не стал узким местом.
6. **Обновляй sidecar независимо** — новая версия логирования не требует пересборки приложения.

---

## 📎 Связанные документы

- [Ambassador Pattern](../ambassador/README.md) — альтернативный подход для внешних интеграций
- [Circuit Breaker](../circuit-breaker/README.md) — часто используется внутри sidecar
- [Service Mesh](../../architecture-patterns/modern-hybrid/mesh/README.md) — где sidecar является основным элементом
- [ADR: Выбор Sidecar для логирования и метрик](../../docs/architecture/adr/020-sidecar-choice.md)

---

*Sidecar — это не «лишний процесс», а **стандартизированный компаньон для инфраструктурных задач**.*