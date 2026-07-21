# Strangler Pattern (Паттерн «Душитель»)

**Strangler Pattern** — это стратегия постепенной замены устаревших частей системы на новые, без остановки всей системы целиком. Название происходит от биологической аналогии: фиговое дерево-душитель обвивает старое дерево и постепенно заменяет его, пока старое не отмирает.

В архитектуре это означает: мы **постепенно вытесняем старый функционал**, перенаправляя трафик на новые сервисы, пока старый код не остаётся без использования и может быть удалён.

Это безопасный способ модернизации legacy-систем без «большого взрыва» и длительных простоев.

---

## 🧠 Как это работает

1. **Сначала** — существующая система обрабатывает все запросы.
2. **Затем** — мы внедряем «прокси» или API Gateway, который перенаправляет часть запросов (по маршрутам, пользователям, фичам) на новый сервис.
3. **Постепенно** — доля нового функционала увеличивается.
4. **В конце** — старый код полностью заменён и может быть удалён.

Важно: на всех этапах система остаётся работоспособной — пользователи не замечают изменений.

---

## 🧩 Пример реализации на Go (прокси + фабрика сервисов)

```go
package strangler

import (
	"errors"
	"fmt"
	"sync"
)

// ===== Старый сервис =====

// OldService — устаревший, но пока ещё работающий сервис
type OldService struct{}

func (s *OldService) Process(data string) (string, error) {
	fmt.Printf("[OLD] Обработка: %s\n", data)
	return "old result", nil
}

// ===== Новый сервис =====

// NewService — современный, постепенно вытесняющий старый
type NewService struct{}

func (s *NewService) Process(data string) (string, error) {
	fmt.Printf("[NEW] Обработка: %s\n", data)
	return "new result", nil
}

// ===== Proxy =====

// Router — решает, куда направить запрос
type Router interface {
	Route(data string) bool // true — новый сервис, false — старый
}

// PercentageRouter — маршрутизация по проценту запросов
type PercentageRouter struct {
	mu        sync.RWMutex
	threshold int // 0–100, сколько процентов запросов идёт на новый сервис
}

func NewPercentageRouter(initialPercent int) *PercentageRouter {
	if initialPercent < 0 {
		initialPercent = 0
	}
	if initialPercent > 100 {
		initialPercent = 100
	}
	return &PercentageRouter{threshold: initialPercent}
}

func (r *PercentageRouter) Route(data string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	// Простейшая эмуляция: хэш строки → 0..100
	hash := 0
	for _, ch := range data {
		hash += int(ch)
	}
	percent := hash % 101
	return percent < r.threshold
}

func (r *PercentageRouter) SetThreshold(newPercent int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if newPercent < 0 {
		newPercent = 0
	}
	if newPercent > 100 {
		newPercent = 100
	}
	r.threshold = newPercent
}

// ===== Strangler Proxy =====

type StranglerProxy struct {
	oldSvc *OldService
	newSvc *NewService
	router Router
}

func NewStranglerProxy(old *OldService, new *NewService, router Router) *StranglerProxy {
	return &StranglerProxy{
		oldSvc: old,
		newSvc: new,
		router: router,
	}
}

func (p *StranglerProxy) Process(data string) (string, error) {
	if p.router.Route(data) {
		return p.newSvc.Process(data)
	}
	return p.oldSvc.Process(data)
}
```

---

## 🧪 Тестирование

```go
package strangler

import (
	"testing"
)

func TestStranglerProxy(t *testing.T) {
	oldSvc := &OldService{}
	newSvc := &NewService{}
	router := NewPercentageRouter(0) // 0% — всё идёт на старый сервис

	proxy := NewStranglerProxy(oldSvc, newSvc, router)

	// Шаг 1: 0% → всё старое
	result, _ := proxy.Process("test1")
	if result != "old result" {
		t.Errorf("expected 'old result', got '%s'", result)
	}

	// Шаг 2: 50% → половина на новый
	router.SetThreshold(50)
	newCount := 0
	oldCount := 0
	for i := 0; i < 100; i++ {
		res, _ := proxy.Process("test2")
		if res == "new result" {
			newCount++
		} else {
			oldCount++
		}
	}
	if newCount == 0 || oldCount == 0 {
		t.Errorf("expected both old and new, got new=%d, old=%d", newCount, oldCount)
	}
}
```

---

## 🧠 Когда я использую Strangler Pattern

- **Миграция монолита в микросервисы**.
- **Замена старого стека (язык, БД, фреймворк)** на новый.
- **Переход от устаревших интеграций к современным**.
- **Обновление архитектуры без остановки бизнес-процессов**.

---

## ⚖️ Плюсы и минусы

| Плюсы | Минусы |
|-------|--------|
| ✅ Безопасная миграция без простоя | ❌ Требует временного дублирования логики |
| ✅ Возможность отката на любой стадии | ❌ Усложнение архитектуры (прокси, роутинг) |
| ✅ Позволяет учиться на реальном трафике | ❌ Может затянуться на годы, если не контролировать |
| ✅ Уменьшает риски больших релизов | ❌ Требует тестирования двух систем параллельно |

---

## 🚀 Как использовать в реальном проекте

1. **Внедри API Gateway или прокси** перед старым и новым сервисами.
2. **Настрой роутинг** (по пользователю, фиче, проценту).
3. **Постепенно увеличивай долю нового сервиса**.
4. **Мониторь ошибки и производительность** на каждом шаге.
5. **Когда новый сервис полностью заменит старый, удали старый код.**

---

## 📎 Связанные документы

- [ADR: Стратегия миграции на микросервисы](../../docs/architecture/adr/018-strangler-approach.md)
- [Модульный монолит](../../architecture-patterns/monolithic/README.md)
- [API Gateway](../../architecture-patterns/microservices/api-gateway/README.md)

---

*Strangler Pattern — это не «тянуть до последнего», а **контролируемая эволюция системы**.*