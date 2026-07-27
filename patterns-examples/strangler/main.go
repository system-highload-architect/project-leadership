package main

import (
	"fmt"
	"sync"
	"time"
)

// ===== Старый сервис =====

// OldService — устаревший, но пока ещё работающий сервис
type OldService struct{}

func (s *OldService) Process(data string) (string, error) {
	fmt.Printf("[OLD] Processing: %s\n", data)
	// Имитация работы
	time.Sleep(10 * time.Millisecond)
	return "old result", nil
}

// ===== Новый сервис =====

// NewService — современный, постепенно вытесняющий старый
type NewService struct{}

func (s *NewService) Process(data string) (string, error) {
	fmt.Printf("[NEW] Processing: %s\n", data)
	// Имитация работы
	time.Sleep(10 * time.Millisecond)
	return "new result", nil
}

// ===== Router =====

// Router — решает, куда направить запрос
type Router interface {
	Route(data string) bool // true — новый сервис, false — старый
}

// ===== Стратегии маршрутизации =====

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

// UserIDRouter — маршрутизация по идентификатору пользователя
type UserIDRouter struct {
	users map[string]bool // true — новый сервис, false — старый
}

func NewUserIDRouter() *UserIDRouter {
	return &UserIDRouter{
		users: make(map[string]bool),
	}
}

func (r *UserIDRouter) AddUser(userID string, useNew bool) {
	r.users[userID] = useNew
}

func (r *UserIDRouter) Route(data string) bool {
	// data — это userID (или строка, содержащая его)
	useNew, ok := r.users[data]
	if !ok {
		return false // по умолчанию — старый сервис
	}
	return useNew
}

// FeatureFlagRouter — маршрутизация по названию фичи
type FeatureFlagRouter struct {
	flags map[string]bool // true — новый сервис, false — старый
}

func NewFeatureFlagRouter() *FeatureFlagRouter {
	return &FeatureFlagRouter{
		flags: make(map[string]bool),
	}
}

func (r *FeatureFlagRouter) SetFeature(feature string, useNew bool) {
	r.flags[feature] = useNew
}

func (r *FeatureFlagRouter) Route(data string) bool {
	useNew, ok := r.flags[data]
	if !ok {
		return false
	}
	return useNew
}

// ===== Proxy =====

// StranglerProxy — прокси, который перенаправляет запросы
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

// ===== Пример использования =====

func main() {
	oldSvc := &OldService{}
	newSvc := &NewService{}

	fmt.Println("=== Strangler Pattern Demo ===")
	fmt.Println()

	// --- Стратегия 1: По проценту (Canary Release) ---
	fmt.Println("--- Strategy 1: Percentage Router (Canary Release) ---")
	router := NewPercentageRouter(0) // 0% — всё идёт на старый сервис
	proxy := NewStranglerProxy(oldSvc, newSvc, router)

	fmt.Println("Шаг 1: 0% → всё на старый сервис")
	result, _ := proxy.Process("test1")
	fmt.Printf("Result: %s\n\n", result)

	fmt.Println("Шаг 2: 50% → половина на новый сервис")
	router.SetThreshold(50)
	newCount := 0
	oldCount := 0
	for i := 0; i < 10; i++ {
		res, _ := proxy.Process("test2")
		if res == "new result" {
			newCount++
		} else {
			oldCount++
		}
	}
	fmt.Printf("Результат: новый = %d, старый = %d\n\n", newCount, oldCount)

	fmt.Println("Шаг 3: 100% → всё на новый сервис")
	router.SetThreshold(100)
	result, _ = proxy.Process("test3")
	fmt.Printf("Result: %s\n\n", result)

	// --- Стратегия 2: По пользователю (Beta-тестеры) ---
	fmt.Println("--- Strategy 2: User ID Router (Beta Testers) ---")
	userRouter := NewUserIDRouter()
	userRouter.AddUser("user-123", true)  // бета-тестер
	userRouter.AddUser("user-456", false) // обычный пользователь
	proxy2 := NewStranglerProxy(oldSvc, newSvc, userRouter)

	for _, userID := range []string{"user-123", "user-456", "user-789"} {
		result, _ := proxy2.Process(userID)
		fmt.Printf("User %s → %s\n", userID, result)
	}
	fmt.Println()

	// --- Стратегия 3: По фиче (Feature Flags) ---
	fmt.Println("--- Strategy 3: Feature Flag Router ---")
	featureRouter := NewFeatureFlagRouter()
	featureRouter.SetFeature("search", true) // поиск — на новый сервис
	featureRouter.SetFeature("cart", false)  // корзина — на старый
	proxy3 := NewStranglerProxy(oldSvc, newSvc, featureRouter)

	for _, feature := range []string{"search", "cart", "payment"} {
		result, _ := proxy3.Process(feature)
		fmt.Printf("Feature %s → %s\n", feature, result)
	}
}
