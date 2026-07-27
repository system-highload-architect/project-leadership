package main

import (
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// ===== Circuit Breaker =====

// CircuitBreaker — структура с состояниями и логикой переключения
type CircuitBreaker struct {
	mu              sync.Mutex
	state           string // "Closed", "Open", "Half-Open"
	failures        int
	lastFailureTime time.Time

	maxFailures      int           // порог ошибок для перехода в Open
	timeout          time.Duration // время в Open перед переходом в Half-Open
	halfOpenMaxCalls int           // сколько пробных вызовов разрешено в Half-Open
	halfOpenCalls    int           // текущее количество пробных вызовов
}

// NewCircuitBreaker создаёт новый экземпляр предохранителя
func NewCircuitBreaker(maxFailures int, timeout time.Duration, halfOpenMaxCalls int) *CircuitBreaker {
	return &CircuitBreaker{
		state:            "Closed",
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
	switch cb.state {
	case "Open":
		if time.Since(cb.lastFailureTime) > cb.timeout {
			// Переход в Half-Open
			cb.state = "Half-Open"
			cb.halfOpenCalls = 0
			fmt.Println("[CB] State: Open → Half-Open (timeout)")
		} else {
			return errors.New("circuit breaker is open")
		}
	case "Half-Open":
		if cb.halfOpenCalls >= cb.halfOpenMaxCalls {
			return errors.New("circuit breaker is open (too many half-open calls)")
		}
		cb.halfOpenCalls++
	}

	// Выполняем функцию
	err := fn()

	if err != nil {
		// Ошибка — учитываем
		cb.failures++
		cb.lastFailureTime = time.Now()

		if cb.state == "Closed" && cb.failures >= cb.maxFailures {
			// Closed → Open
			cb.state = "Open"
			fmt.Printf("[CB] State: Closed → Open (failures: %d)\n", cb.failures)
		}
		if cb.state == "Half-Open" {
			// Half-Open с ошибкой → снова Open
			cb.state = "Open"
			cb.failures = 0
			fmt.Println("[CB] State: Half-Open → Open (test call failed)")
		}
		return err
	}

	// Успех
	if cb.state == "Half-Open" {
		// Half-Open с успехом → Closed
		cb.state = "Closed"
		cb.failures = 0
		fmt.Println("[CB] State: Half-Open → Closed (all calls successful)")
	} else {
		// Closed — сбрасываем счётчик ошибок при успехе
		cb.failures = 0
	}

	return nil
}

// ===== Внешний сервис (имитация) =====

// externalService имитирует вызов внешнего API, который иногда падает
func externalService() error {
	// 30% ошибок
	if rand.Intn(100) < 30 {
		return errors.New("external service error")
	}
	return nil
}

// ===== Главная функция =====

func main() {
	// Создаём Circuit Breaker
	cb := NewCircuitBreaker(5, 3*time.Second, 2)

	// Выполняем 20 запросов с задержкой
	for i := 1; i <= 20; i++ {
		err := cb.Call(externalService)
		if err != nil {
			fmt.Printf("[CB] Request #%d: blocked or error: %v\n", i, err)
		} else {
			fmt.Printf("[CB] Request #%d: success\n", i)
		}
		time.Sleep(500 * time.Millisecond)
	}
}
