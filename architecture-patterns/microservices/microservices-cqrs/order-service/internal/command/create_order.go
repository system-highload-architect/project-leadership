package command

import (
	"errors"
	"time"

	"order-service/internal/domain"
	readrepo "order-service/internal/repository/read/inmemory"
	writerepo "order-service/internal/repository/write/inmemory"
)

type CreateOrderCommand struct {
	writeRepo *writerepo.WriteRepository
	readRepo  *readrepo.ReadRepository
}

func NewCreateOrderCommand(writeRepo *writerepo.WriteRepository, readRepo *readrepo.ReadRepository) *CreateOrderCommand {
	return &CreateOrderCommand{
		writeRepo: writeRepo,
		readRepo:  readRepo,
	}
}

type CreateOrderRequest struct {
	CustomerID string             `json:"customer_id"`
	Items      []domain.OrderItem `json:"items"`
}

func (c *CreateOrderCommand) Execute(req CreateOrderRequest) (domain.Order, error) {
	if req.CustomerID == "" {
		return domain.Order{}, errors.New("customer id is required")
	}
	if len(req.Items) == 0 {
		return domain.Order{}, errors.New("order must have at least one item")
	}

	order := domain.Order{
		ID:         "ord-" + time.Now().Format("20060102150405"),
		CustomerID: req.CustomerID,
		Items:      req.Items,
		Status:     domain.OrderStatusNew,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	if err := c.writeRepo.Save(order); err != nil {
		return domain.Order{}, err
	}

	// Обновление read-модели (синхронно)
	c.updateReadModel(order)

	return order, nil
}

func (c *CreateOrderCommand) updateReadModel(order domain.Order) {
	readModel := domain.OrderReadModel{
		ID:         order.ID,
		CustomerID: order.CustomerID,
		Status:     string(order.Status),
		CreatedAt:  order.CreatedAt,
	}
	var total float64
	for _, item := range order.Items {
		readModel.Items = append(readModel.Items, domain.OrderItemReadModel{
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
			Price:     item.Price,
			Total:     item.Price * float64(item.Quantity),
		})
		total += item.Price * float64(item.Quantity)
	}
	readModel.Total = total
	_ = c.readRepo.Save(readModel)
}
