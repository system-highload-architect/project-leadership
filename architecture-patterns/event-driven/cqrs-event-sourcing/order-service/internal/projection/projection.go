package projection

import (
	"order-service/internal/domain"
	"order-service/internal/event"
	"order-service/internal/repository/read/inmemory"
)

type Projection struct {
	readRepo *inmemory.ReadRepository
}

func NewProjection(readRepo *inmemory.ReadRepository) *Projection {
	return &Projection{readRepo: readRepo}
}

// Apply обновляет read-модель на основе события
func (p *Projection) Apply(e event.Event) {
	switch ev := e.(type) {
	case *event.OrderCreated:
		// Создаём read-модель
		readModel := domain.OrderReadModel{
			ID:         ev.AggregateID,
			CustomerID: ev.CustomerID,
			Status:     string(domain.OrderStatusNew),
			CreatedAt:  ev.Timestamp,
		}
		var total float64
		for _, item := range ev.Items {
			readModel.Items = append(readModel.Items, domain.OrderItemReadModel{
				ProductID: item.ProductID,
				Quantity:  item.Quantity,
				Price:     item.Price,
				Total:     float64(item.Quantity) * item.Price,
			})
			total += float64(item.Quantity) * item.Price
		}
		readModel.Total = total
		_ = p.readRepo.Save(readModel)

	case *event.OrderPaid:
		// Обновляем статус в read-модели
		existing, err := p.readRepo.FindByID(ev.AggregateID)
		if err != nil {
			return
		}
		existing.Status = string(domain.OrderStatusPaid)
		_ = p.readRepo.Save(existing)
	}
}
