# Retry with Backoff (Повторные попытки с задержкой)

**Retry with Backoff** — это паттерн, который используется для повторного выполнения операции, если она завершилась с ошибкой (например, временный сбой сети, перегрузка сервера). Задержка между попытками увеличивается с каждой последующей попыткой, что позволяет системе восстановиться и не создавать избыточную нагрузку на целевой сервис.

---

## 🧠 Как это работает

1. **Начальная попытка** выполняется сразу.
2. **Если ошибка** — система ждёт определённое время, затем повторяет.
3. **Задержка** увеличивается (экспоненциально или линейно) до достижения максимального значения.
4. **Джиттер** (случайное отклонение) добавляется, чтобы избежать «эффекта стада» (syn flood).
5. После исчерпания всех попыток возвращается последняя ошибка.

---

## 🧩 Пример реализации на Go

```go
package retry

import (
	"context"
	"errors"
	"math/rand"
	"time"
)

// BackoffStrategy определяет, как рассчитывается задержка между попытками
type BackoffStrategy func(attempt int) time.Duration

// ExponentialBackoff возвращает стратегию с экспоненциальным ростом и джиттером
func ExponentialBackoff(initial, max time.Duration, factor float64, jitter bool) BackoffStrategy {
	return func(attempt int) time.Duration {
		if attempt == 0 {
			return 0
		}
		// Экспоненциальный рост: initial * factor^(attempt-1)
		delay := float64(initial) * pow(factor, float64(attempt-1))
		if delay > float64(max) {
			delay = float64(max)
		}
		duration := time.Duration(delay)
		if jitter {
			// Добавляем случайное отклонение (0–30%)
			jitterFactor := 1 + 0.3*rand.Float64()
			duration = time.Duration(float64(duration) * jitterFactor)
		}
		return duration
	}
}

// Retry выполняет функцию fn с повторными попытками, используя заданную стратегию.
// Максимальное число попыток — maxAttempts.
// Контекст ctx позволяет прервать выполнение.
func Retry(ctx context.Context, maxAttempts int, strategy BackoffStrategy, fn func() error) error {
	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		// Проверяем, не отменён ли контекст
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Выполняем функцию
		err := fn()
		if err == nil {
			return nil
		}
		lastErr = err

		// Если это последняя попытка — выходим
		if attempt == maxAttempts-1 {
			break
		}

		// Рассчитываем задержку
		delay := strategy(attempt + 1) // attempt+1, потому что первая попытка без задержки
		if delay > 0 {
			timer := time.NewTimer(delay)
			select {
			case <-timer.C:
				// продолжаем
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			}
		}
	}
	return lastErr
}

// pow — вспомогательная функция для возведения в степень (без math.Pow для целых чисел)
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
```

---

## 🧪 Тесты

```go
package retry

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRetry_Success(t *testing.T) {
	ctx := context.Background()
	strategy := ExponentialBackoff(10*time.Millisecond, 1*time.Second, 2.0, false)
	attempts := 0
	err := Retry(ctx, 3, strategy, func() error {
		attempts++
		if attempts < 2 {
			return errors.New("temporary error")
		}
		return nil
	})
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
	if attempts != 2 {
		t.Errorf("expected 2 attempts, got %d", attempts)
	}
}

func TestRetry_MaxAttempts(t *testing.T) {
	ctx := context.Background()
	strategy := ExponentialBackoff(10*time.Millisecond, 1*time.Second, 2.0, false)
	err := Retry(ctx, 3, strategy, func() error {
		return errors.New("persistent error")
	})
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestRetry_ContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	strategy := ExponentialBackoff(100*time.Millisecond, 1*time.Second, 2.0, false)

	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	err := Retry(ctx, 5, strategy, func() error {
		time.Sleep(20 * time.Millisecond) // имитация работы
		return errors.New("error")
	})
	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}
```

---

## 🚀 Как использовать в реальном проекте

1. **Оберни вызов внешнего API, БД или файловой системы** в `Retry()`.
2. **Настрой стратегию** под свой контекст:
   - `initial` — начальная задержка (50–200 мс).
   - `max` — максимальная задержка (5–30 с).
   - `factor` — множитель для экспоненциального роста (1.5–2.0).
   - `jitter` — включи для распределённых систем.
3. **Определи максимальное число попыток** (обычно 3–5).
4. **Передавай контекст** для поддержки отмены и таймаутов.

---

## 📌 Важные нюансы

- **Идемпотентность** — операция должна быть безопасна при повторных вызовах.
- **Ошибки, не подлежащие ретраю** (например, 400 Bad Request) — лучше не повторять.
- **Мониторинг** — логируй попытки, чтобы отслеживать сбои и время восстановления.

---

## 📎 Связанные документы

- [ADR: Выбор стратегии повторов](../../docs/architecture/adr/014-retry-strategy.md)
- [Circuit Breaker](../circuit-breaker/README.md)
- [Bulkhead](../bulkhead/README.md)

---

*Retry with Backoff — это не «панацея», а **инструмент для повышения надёжности**.*