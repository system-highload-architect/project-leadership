package main

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"
)

// ===== Retry с Backoff =====

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
			if attempt > 0 {
				fmt.Printf("[Retry] успешно на попытке %d\n", attempt+1)
			}
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
			fmt.Printf("[Retry] попытка %d/%d не удалась: %v. Ждём %v...\n", attempt+1, maxAttempts, err, delay)
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
	fmt.Printf("[Retry] все попытки (%d) не удались: %v\n", maxAttempts, lastErr)
	return lastErr
}

// pow — вспомогательная функция для возведения в степень
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

// ===== Тестовая функция =====

var callCounter int

// unstableOperation имитирует нестабильную операцию, которая успешно выполняется только на 3-й попытке
func unstableOperation() error {
	callCounter++
	if callCounter < 3 {
		return errors.New("temporary failure")
	}
	return nil
}

// ===== Главная функция =====

func main() {
	ctx := context.Background()

	// Стратегия: начальная задержка 100 мс, максимум 2 сек, множитель 2.0, джиттер включён
	strategy := ExponentialBackoff(100*time.Millisecond, 2*time.Second, 2.0, true)

	// Сбрасываем счётчик
	callCounter = 0

	err := Retry(ctx, 5, strategy, unstableOperation)
	if err != nil {
		fmt.Println("Ошибка:", err)
	} else {
		fmt.Println("Успешно!")
	}

	// Демонстрация с контекстом, который отменяется
	ctxCancel, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(300 * time.Millisecond)
		cancel()
	}()
	callCounter = 0
	err = Retry(ctxCancel, 10, strategy, func() error {
		callCounter++
		return errors.New("always fail")
	})
	if err != nil {
		fmt.Println("Контекст отменён:", err)
	}
}
