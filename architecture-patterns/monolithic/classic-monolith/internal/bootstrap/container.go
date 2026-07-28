package bootstrap

import (
	"classic-monolith/internal/delivery/http"
	"classic-monolith/internal/repository"
	"classic-monolith/internal/service"
)

type Container struct {
	OrderHandler   *http.OrderHandler
	ProductHandler *http.ProductHandler
}

func NewContainer() *Container {
	// Инициализация репозиториев
	orderRepo := repository.NewOrderRepository()
	productRepo := repository.NewProductRepository()

	// Инициализация сервисов
	orderSvc := service.NewOrderService(orderRepo, productRepo)

	// Инициализация хендлеров
	orderHandler := http.NewOrderHandler(orderSvc)
	productHandler := http.NewProductHandler(productRepo)

	return &Container{
		OrderHandler:   orderHandler,
		ProductHandler: productHandler,
	}
}
