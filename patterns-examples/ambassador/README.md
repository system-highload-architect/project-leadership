# Ambassador Pattern (Паттерн «Посол»)

**Ambassador Pattern** — это паттерн, при котором **вспомогательный сервис (ambassador) действует как посредник между основным приложением и внешним миром**. Он берёт на себя сквозные задачи: логирование, метрики, аутентификацию, кэширование, повторные попытки, Circuit Breaker и т.д.

В отличие от Sidecar (который работает на том же хосте и сопровождает основной сервис), Ambassador часто выступает как **прокси для внешних сервисов**, обеспечивая единую точку входа и стандартизированное поведение для всех исходящих вызовов.

---

## 🧠 Основная идея

Ambassador — это **прокси-сервис**, который:
- **Принимает запросы** от основного приложения.
- **Выполняет сквозные задачи** (логирование, аутентификация, ретраи, кэширование).
- **Перенаправляет запросы** к внешним API или сервисам.
- **Возвращает ответ** приложению.

Таким образом, основное приложение не знает о сложности внешних интеграций — оно просто отправляет запрос на localhost или в локальную сеть и получает ответ.

---

## 🧩 Когда я использую Ambassador

Я выбираю Ambassador Pattern, когда:

- **Много внешних интеграций** — нужно унифицировать поведение.
- **Сквозные задачи** (логирование, метрики, авторизация) должны быть стандартизированы.
- **Не хочу засорять основной код** — вся «инфраструктурная» логика выносится в прокси.
- **Нужно кэшировать ответы** от внешних сервисов для снижения нагрузки.
- **Требуется тестирование внешних API** — можно мокать Ambassador, а не реальный сервис.

---

## 🧩 Как это работает

```
[ Основное приложение ]
         │
         ▼
[ Ambassador Proxy ]  ← логирование, метрики, аутентификация, ретраи, кэширование
         │
         ▼
[ Внешний сервис / API ]
```

Все запросы из приложения проходят через Ambassador, который применяет сквозную логику и перенаправляет их дальше.

---

## 🧩 Пример реализации на Go

```go
package ambassador

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ===== Конфигурация =====

// Config — настройки Ambassador
type Config struct {
	TargetURL      string        // URL внешнего сервиса
	Timeout        time.Duration // таймаут запроса
	RetryCount     int           // количество повторных попыток
	RetryDelay     time.Duration // задержка между попытками
	EnableLogging  bool          // логировать запросы/ответы
	EnableMetrics  bool          // собирать метрики
	EnableCaching  bool          // кэшировать ответы
	CacheTTL       time.Duration // время жизни кэша
	AuthToken      string        // токен для аутентификации (если нужен)
}

// ===== Ambassador =====

// Ambassador — прокси для внешних сервисов
type Ambassador struct {
	cfg      Config
	client   *http.Client
	cache    map[string]cacheEntry // in-memory кэш
	metrics  MetricsCollector      // сборщик метрик
}

// cacheEntry — запись в кэше
type cacheEntry struct {
	response   []byte
	expiresAt  time.Time
}

// MetricsCollector — интерфейс для сбора метрик
type MetricsCollector interface {
	IncrementRequest(target string)
	IncrementError(target string)
	ObserveLatency(target string, duration time.Duration)
}

// NewAmbassador создаёт новый экземпляр Ambassador
func NewAmbassador(cfg Config, metrics MetricsCollector) *Ambassador {
	return &Ambassador{
		cfg: cfg,
		client: &http.Client{
			Timeout: cfg.Timeout,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: false},
				MaxIdleConns:    100,
			},
		},
		cache:   make(map[string]cacheEntry),
		metrics: metrics,
	}
}

// ===== Основной метод =====

// Request — выполняет запрос к внешнему сервису через Ambassador
func (a *Ambassador) Request(ctx context.Context, method, path string, body interface{}) ([]byte, error) {
	start := time.Now()

	// 1. Проверка кэша (для GET-запросов)
	if a.cfg.EnableCaching && method == http.MethodGet {
		cacheKey := fmt.Sprintf("%s:%s", path, a.hashBody(body))
		if entry, ok := a.cache[cacheKey]; ok && time.Now().Before(entry.expiresAt) {
			if a.cfg.EnableMetrics && a.metrics != nil {
				a.metrics.IncrementRequest(a.cfg.TargetURL)
			}
			return entry.response, nil
		}
	}

	// 2. Формирование запроса
	fullURL := a.cfg.TargetURL + path
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal body: %w", err)
		}
		reqBody = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Добавляем заголовки
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "ambassador-proxy/1.0")
	if a.cfg.AuthToken != "" {
		req.Header.Set("Authorization", "Bearer "+a.cfg.AuthToken)
	}

	// 3. Логирование (если включено)
	if a.cfg.EnableLogging {
		a.logRequest(method, fullURL, body)
	}

	// 4. Выполнение запроса с повторными попытками
	var lastErr error
	var resp *http.Response

	for attempt := 0; attempt <= a.cfg.RetryCount; attempt++ {
		if attempt > 0 {
			// Задержка между попытками
			select {
			case <-time.After(a.cfg.RetryDelay * time.Duration(attempt)):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}

		resp, err = a.client.Do(req)
		if err == nil {
			break
		}
		lastErr = err
	}

	if err != nil {
		if a.cfg.EnableMetrics && a.metrics != nil {
			a.metrics.IncrementError(a.cfg.TargetURL)
		}
		return nil, fmt.Errorf("request failed after %d attempts: %w", a.cfg.RetryCount+1, lastErr)
	}
	defer resp.Body.Close()

	// 5. Чтение ответа
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// 6. Проверка статуса
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if a.cfg.EnableMetrics && a.metrics != nil {
			a.metrics.IncrementError(a.cfg.TargetURL)
		}
		return nil, fmt.Errorf("external service returned %d: %s", resp.StatusCode, string(respBody))
	}

	// 7. Метрики
	if a.cfg.EnableMetrics && a.metrics != nil {
		a.metrics.IncrementRequest(a.cfg.TargetURL)
		a.metrics.ObserveLatency(a.cfg.TargetURL, time.Since(start))
	}

	// 8. Логирование ответа
	if a.cfg.EnableLogging {
		a.logResponse(resp.StatusCode, respBody)
	}

	// 9. Кэширование (для GET-запросов)
	if a.cfg.EnableCaching && method == http.MethodGet {
		cacheKey := fmt.Sprintf("%s:%s", path, a.hashBody(body))
		a.cache[cacheKey] = cacheEntry{
			response:  respBody,
			expiresAt: time.Now().Add(a.cfg.CacheTTL),
		}
	}

	return respBody, nil
}

// ===== Вспомогательные методы =====

// hashBody — вычисляет хэш тела запроса для ключа кэша
func (a *Ambassador) hashBody(body interface{}) string {
	if body == nil {
		return "nil"
	}
	b, _ := json.Marshal(body)
	return fmt.Sprintf("%x", b)
}

// logRequest — логирует исходящий запрос
func (a *Ambassador) logRequest(method, url string, body interface{}) {
	fmt.Printf("[Ambassador] → %s %s", method, url)
	if body != nil {
		b, _ := json.Marshal(body)
		fmt.Printf(" | body: %s", string(b))
	}
	fmt.Println()
}

// logResponse — логирует входящий ответ
func (a *Ambassador) logResponse(status int, body []byte) {
	fmt.Printf("[Ambassador] ← %d | body: %s\n", status, string(body))
}
```

---

## 🧪 Пример использования

```go
package main

import (
	"context"
	"fmt"
	"time"
)

// SimpleMetrics — простая реализация сбора метрик
type SimpleMetrics struct{}

func (m *SimpleMetrics) IncrementRequest(target string) {
	fmt.Printf("[Metrics] Request to %s\n", target)
}

func (m *SimpleMetrics) IncrementError(target string) {
	fmt.Printf("[Metrics] Error from %s\n", target)
}

func (m *SimpleMetrics) ObserveLatency(target string, duration time.Duration) {
	fmt.Printf("[Metrics] Latency to %s: %v\n", target, duration)
}

func main() {
	// Конфигурация Ambassador
	cfg := Config{
		TargetURL:     "https://api.example.com",
		Timeout:       5 * time.Second,
		RetryCount:    2,
		RetryDelay:    500 * time.Millisecond,
		EnableLogging: true,
		EnableMetrics: true,
		EnableCaching: true,
		CacheTTL:      30 * time.Second,
		AuthToken:     "your-token-here",
	}

	// Создаём Ambassador
	amb := NewAmbassador(cfg, &SimpleMetrics{})

	// Выполняем запрос
	ctx := context.Background()

	// GET-запрос
	resp, err := amb.Request(ctx, http.MethodGet, "/users/123", nil)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Printf("Response: %s\n", string(resp))

	// POST-запрос с телом
	body := map[string]interface{}{
		"name":  "John Doe",
		"email": "john@example.com",
	}
	resp, err = amb.Request(ctx, http.MethodPost, "/users", body)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Printf("Response: %s\n", string(resp))
}
```

---

## ⚖️ Плюсы и минусы

| Плюсы | Минусы |
|-------|--------|
| ✅ Вынос сквозной логики из приложения | ❌ Дополнительный сетевой hop (задержка) |
| ✅ Стандартизация интеграций | ❌ Единая точка отказа (если не кластеризован) |
| ✅ Упрощение тестирования (можно мокать Ambassador) | ❌ Требует управления конфигурацией |
| ✅ Кэширование и ретраи без изменения кода | ❌ Может стать узким местом при высокой нагрузке |
| ✅ Единое место для логирования и метрик | ❌ Сложность отладки (дополнительный слой) |

---

## 🧠 Ambassador vs Sidecar

| Критерий | Ambassador | Sidecar |
|----------|------------|---------|
| **Назначение** | Прокси для внешних сервисов | Сопровождение основного сервиса |
| **Расположение** | Отдельный сервис (может быть общим) | На том же хосте, что и основной сервис |
| **Масштабирование** | Может быть общим для многих сервисов | Один на каждый экземпляр сервиса |
| **Пример** | API Gateway, внешний прокси | Логгер, метрики, Service Mesh (Envoy) |

---

## 🚀 Как использовать в реальном проекте

1. **Определи внешние интеграции** — какие API и сервисы будут вызываться.
2. **Настрой Ambassador** — таймауты, ретраи, кэширование, логирование.
3. **Встрой Ambassador в приложение** — все вызовы внешних сервисов идут через него.
4. **Мониторь метрики** — latency, ошибки, количество запросов.
5. **При необходимости кластеризуй** — для отказоустойчивости.

---

## 📎 Связанные документы

- [Sidecar Pattern](../sidecar/README.md) — альтернативный подход для сопровождения сервисов
- [Circuit Breaker](../circuit-breaker/README.md) — часто используется внутри Ambassador
- [Retry with Backoff](../retry-backoff/README.md) — реализация повторных попыток
- [API Gateway](../../architecture-patterns/microservices/api-gateway/README.md) — более масштабный вариант Ambassador

---

*Ambassador — это не просто «прокси», а **стандартизированный шлюз для внешнего мира**.*