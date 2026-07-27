package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"sync"
	"time"
)

// ===== Конфигурация =====

// Config — настройки Ambassador
type Config struct {
	TargetURL     string        // URL внешнего сервиса
	Timeout       time.Duration // таймаут запроса
	RetryCount    int           // количество повторных попыток
	RetryDelay    time.Duration // начальная задержка между попытками
	RetryFactor   float64       // множитель для экспоненциального роста
	EnableLogging bool          // логировать запросы/ответы
	EnableMetrics bool          // собирать метрики
	EnableCaching bool          // кэшировать ответы
	CacheTTL      time.Duration // время жизни кэша
	AuthToken     string        // токен для аутентификации (если нужен)
}

// ===== Ambassador =====

// Ambassador — прокси для внешних сервисов
type Ambassador struct {
	cfg     Config
	client  *http.Client
	cache   map[string]cacheEntry // in-memory кэш
	mu      sync.RWMutex
	metrics MetricsCollector // сборщик метрик
}

// cacheEntry — запись в кэше
type cacheEntry struct {
	response  []byte
	expiresAt time.Time
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

// Request выполняет запрос к внешнему сервису через Ambassador
func (a *Ambassador) Request(ctx context.Context, method, path string, body interface{}) ([]byte, error) {
	start := time.Now()
	cacheKey := a.generateCacheKey(method, path, body)

	// 1. Проверка кэша (для GET-запросов)
	if a.cfg.EnableCaching && method == http.MethodGet {
		a.mu.RLock()
		entry, ok := a.cache[cacheKey]
		a.mu.RUnlock()
		if ok && time.Now().Before(entry.expiresAt) {
			if a.cfg.EnableMetrics && a.metrics != nil {
				a.metrics.IncrementRequest(a.cfg.TargetURL)
			}
			if a.cfg.EnableLogging {
				fmt.Printf("[Ambassador] CACHED %s %s\n", method, path)
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

	// 4. Выполнение запроса с повторными попытками (экспоненциальный бэкофф)
	var lastErr error
	var resp *http.Response
	var respBody []byte

	for attempt := 0; attempt <= a.cfg.RetryCount; attempt++ {
		if attempt > 0 {
			// Экспоненциальная задержка с джиттером
			delay := float64(a.cfg.RetryDelay) * pow(a.cfg.RetryFactor, float64(attempt-1))
			jitter := 1 + 0.3*rand.Float64()
			delay = delay * jitter
			select {
			case <-time.After(time.Duration(delay)):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
			if a.cfg.EnableLogging {
				fmt.Printf("[Ambassador] Retry %d/%d after %v\n", attempt, a.cfg.RetryCount, time.Duration(delay))
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
	respBody, err = io.ReadAll(resp.Body)
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
		a.mu.Lock()
		a.cache[cacheKey] = cacheEntry{
			response:  respBody,
			expiresAt: time.Now().Add(a.cfg.CacheTTL),
		}
		a.mu.Unlock()
	}

	return respBody, nil
}

// ===== Вспомогательные методы =====

// generateCacheKey — вычисляет ключ для кэша
func (a *Ambassador) generateCacheKey(method, path string, body interface{}) string {
	key := method + ":" + path
	if body != nil {
		b, _ := json.Marshal(body)
		key += ":" + string(b)
	}
	return key
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

// pow — возведение в степень для float64
func pow(base float64, exp float64) float64 {
	if exp == 0 {
		return 1
	}
	if exp == 1 {
		return base
	}
	result := 1.0
	for i := 0; i < int(exp); i++ {
		result *= base
	}
	return result
}

// ===== SimpleMetrics — простая реализация сбора метрик =====

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

// ===== Главная функция =====

func main() {
	// Конфигурация Ambassador
	cfg := Config{
		TargetURL:     "https://jsonplaceholder.typicode.com", // Пример REST API
		Timeout:       5 * time.Second,
		RetryCount:    2,
		RetryDelay:    500 * time.Millisecond,
		RetryFactor:   2.0,
		EnableLogging: true,
		EnableMetrics: true,
		EnableCaching: true,
		CacheTTL:      30 * time.Second,
		AuthToken:     "", // опционально
	}

	// Создаём Ambassador
	amb := NewAmbassador(cfg, &SimpleMetrics{})

	// Выполняем запросы
	ctx := context.Background()

	// GET-запрос
	fmt.Println("=== GET Request ===")
	resp, err := amb.Request(ctx, "GET", "/posts/1", nil)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Printf("Response: %s\n", string(resp))

	// POST-запрос с телом
	fmt.Println("\n=== POST Request ===")
	body := map[string]interface{}{
		"title":  "foo",
		"body":   "bar",
		"userId": 1,
	}
	resp, err = amb.Request(ctx, "POST", "/posts", body)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Printf("Response: %s\n", string(resp))

	// Повторный GET (должен взять из кэша)
	fmt.Println("\n=== Cached GET Request ===")
	resp, err = amb.Request(ctx, "GET", "/posts/1", nil)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Printf("Response: %s\n", string(resp))
}
