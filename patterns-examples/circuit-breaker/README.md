# Circuit Breaker (Предохранитель)

**Circuit Breaker** — это паттерн, который защищает систему от каскадных отказов при вызове внешних сервисов. Он предотвращает бесконечные повторные попытки, когда целевой сервис явно не работает или перегружен.

Аналогия из реального мира: электрический предохранитель — если ток слишком высок, он «размыкает» цепь и защищает оборудование. То же самое в программировании: если вызовы внешнего сервиса начинают падать с ошибками, Circuit Breaker временно блокирует все дальнейшие вызовы, давая сервису время восстановиться.

---

## 🧠 Как это работает

Circuit Breaker имеет три состояния:

1. **Closed (Замкнут)** — обычный режим. Все вызовы проходят. Если количество ошибок превышает порог, переключается в Open.
2. **Open (Разомкнут)** — все вызовы немедленно возвращают ошибку (без реального запроса к целевому сервису). Через некоторое время (timeout) переходит в Half-Open.
3. **Half-Open (Полуоткрыт)** — разрешает ограниченное количество пробных вызовов. Если они успешны, переключается обратно в Closed. Если нет — снова в Open.

---

## 🧩 Пример реализации на Go

```go
package circuitbreaker

import (
	"errors"
	"sync"
	"time"
)

// ErrCircuitOpen возвращается, когда вызов заблокирован
var ErrCircuitOpen = errors.New("circuit breaker is open")

// CircuitBreaker — структура с состояниями и логикой переключения
type CircuitBreaker struct {
	mu              sync.Mutex
	state           int // 0: Closed, 1: Open, 2: Half-Open
	failures        int
	lastFailureTime time.Time

	maxFailures      int           // порог ошибок для перехода в Open
	timeout          time.Duration // время в Open перед переходом в Half-Open
	halfOpenMaxCalls int           // сколько пробных вызовов разрешено в Half-Open
	halfOpenCalls    int
}

// NewCircuitBreaker создаёт новый экземпляр предохранителя
func NewCircuitBreaker(maxFailures int, timeout time.Duration, halfOpenMaxCalls int) *CircuitBreaker {
	return &CircuitBreaker{
		state:            0, // Closed
		maxFailures:      maxFailures,
		timeout:          timeout,
		halfOpenMaxCalls: halfOpenMaxCalls,
	}
}

// Call выполняет функцию fn и применяет логику Circuit Breaker
func (cb *CircuitBreaker) Call(fn func() error) error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	// Проверка состояния
	if cb.state == 1 { // Open
		if time.Since(cb.lastFailureTime) > cb.timeout {
			// Переход в Half-Open
			cb.state = 2
			cb.halfOpenCalls = 0
		} else {
			return ErrCircuitOpen
		}
	}

	if cb.state == 2 { // Half-Open
		if cb.halfOpenCalls >= cb.halfOpenMaxCalls {
			// Больше пробных вызовов не разрешено — считаем, что сервис ещё не готов
			return ErrCircuitOpen
		}
		cb.halfOpenCalls++
	}

	// Выполняем функцию
	err := fn()

	if err != nil {
		// Ошибка — учитываем
		cb.failures++
		cb.lastFailureTime = time.Now()

		if cb.state == 0 && cb.failures >= cb.maxFailures {
			// Closed → Open
			cb.state = 1
		}
		if cb.state == 2 {
			// Half-Open с ошибкой → снова Open
			cb.state = 1
			cb.failures = 0 // можно сбросить счётчик, но лучше оставить для статистики
		}
		return err
	}

	// Успех
	if cb.state == 2 {
		// Half-Open с успехом → Closed
		cb.state = 0
		cb.failures = 0
	} else {
		// Closed — сбрасываем счётчик ошибок при успехе
		cb.failures = 0
	}

	return nil
}
```

---

## 🧪 Тесты

```go
package circuitbreaker

import (
	"errors"
	"testing"
	"time"
)

func TestCircuitBreaker_ClosedToOpen(t *testing.T) {
	cb := NewCircuitBreaker(2, 5*time.Second, 1)

	for i := 0; i < 2; i++ {
		err := cb.Call(func() error {
			return errors.New("fail")
		})
		if err == nil {
			t.Errorf("expected error on attempt %d", i)
		}
	}

	// Третий вызов должен вернуть ErrCircuitOpen, а не реальную ошибку
	err := cb.Call(func() error {
		return nil
	})
	if err != ErrCircuitOpen {
		t.Errorf("expected ErrCircuitOpen, got %v", err)
	}
}

func TestCircuitBreaker_OpenToHalfOpenToClosed(t *testing.T) {
	cb := NewCircuitBreaker(1, 100*time.Millisecond, 2)

	// Открываем
	_ = cb.Call(func() error {
		return errors.New("fail")
	})

	// Ждём перехода в Half-Open
	time.Sleep(150 * time.Millisecond)

	// Пробные вызовы — успешные
	for i := 0; i < 2; i++ {
		err := cb.Call(func() error {
			return nil
		})
		if err != nil {
			t.Errorf("expected nil, got %v", err)
		}
	}

	// Теперь должны быть в Closed — все вызовы проходят
	err := cb.Call(func() error {
		return nil
	})
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}
```

---

## 🚀 Как использовать в реальном проекте

1. **Оберни вызов внешнего API или БД** в `CircuitBreaker.Call()`.
2. **Настрой параметры** под свой контекст:
   - `maxFailures` — количество ошибок для перехода в Open.
   - `timeout` — время ожидания перед попыткой восстановления.
   - `halfOpenMaxCalls` — сколько пробных вызовов разрешить.
3. **Логируй переходы состояний** для мониторинга и отладки.

---

## 📌 Важные нюансы

- **Idempotency** — в Half-Open пробные вызовы должны быть идемпотентными (чтобы не повредить данные).
- **Мониторинг** — важно отслеживать состояние Circuit Breaker (метрики, логи).
- **Распределённые системы** — в микросервисах часто используется общий Circuit Breaker через Redis или service mesh.

---

## 📎 Связанные документы

- [ADR: Выбор Circuit Breaker для интеграций](../../docs/architecture/adr/013-circuit-breaker-choice.md)
- [Retry with Backoff](../retry-backoff/README.md)
- [Bulkhead](../bulkhead/README.md)

---

*Circuit Breaker — это не «магия», а **страховка от каскадного отказа**.*