package command

import (
	"errors"
	"time"

	"order-service/internal/domain"
	readrepo "order-service/internal/repository/read/inmemory"
	writerepo "order-service/internal/repository/write/inmemory"
)

type UpdateStatusCommand struct {
	writeRepo *writerepo.WriteRepository
	readRepo  *readrepo.ReadRepository
}

func NewUpdateStatusCommand(writeRepo *writerepo.WriteRepository, readRepo *readrepo.ReadRepository) *UpdateStatusCommand {
	return &UpdateStatusCommand{
		writeRepo: writeRepo,
		readRepo:  readRepo,
	}
}

type UpdateStatusRequest struct {
	ID     string
	Status domain.OrderStatus
}

func (c *UpdateStatusCommand) Execute(req UpdateStatusRequest) error {
	if req.ID == "" {
		return errors.New("order id is required")
	}
	order, err := c.writeRepo.FindByID(req.ID)
	if err != nil {
		return err
	}
	// Простая валидация: можно менять статус только если он не равен текущему
	if order.Status == req.Status {
		return nil
	}
	// В реальном проекте здесь может быть сложная логика валидации переходов
	order.Status = req.Status
	order.UpdatedAt = time.Now()
	if err := c.writeRepo.Save(order); err != nil {
		return err
	}
	// Обновление read-модели
	c.updateReadModel(order)
	return nil
}

func (c *UpdateStatusCommand) updateReadModel(order domain.Order) {
	// Получаем существующую read-модель или создаём новую
	existing, _ := c.readRepo.FindByID(order.ID)
	readModel := domain.OrderReadModel{
		ID:         order.ID,
		CustomerID: order.CustomerID,
		Status:     string(order.Status),
		CreatedAt:  order.CreatedAt,
	}
	// Если есть существующие предметы, используем их
	if len(existing.Items) > 0 {
		readModel.Items = existing.Items
		readModel.Total = existing.Total
	}
	_ = c.readRepo.Save(readModel)
}
