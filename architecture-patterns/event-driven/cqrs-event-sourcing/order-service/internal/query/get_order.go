package query

import (
	"errors"

	"order-service/internal/domain"
	"order-service/internal/repository/read/inmemory"
)

type GetOrderQuery struct {
	readRepo *inmemory.ReadRepository
}

func NewGetOrderQuery(readRepo *inmemory.ReadRepository) *GetOrderQuery {
	return &GetOrderQuery{readRepo: readRepo}
}

type GetOrderRequest struct {
	ID string
}

func (q *GetOrderQuery) Execute(req GetOrderRequest) (domain.OrderReadModel, error) {
	if req.ID == "" {
		return domain.OrderReadModel{}, errors.New("order id is required")
	}
	return q.readRepo.FindByID(req.ID)
}
