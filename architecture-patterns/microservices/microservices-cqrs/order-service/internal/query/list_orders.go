package query

import (
	"order-service/internal/domain"
	"order-service/internal/repository/read/inmemory"
)

type ListOrdersQuery struct {
	readRepo *inmemory.ReadRepository
}

func NewListOrdersQuery(readRepo *inmemory.ReadRepository) *ListOrdersQuery {
	return &ListOrdersQuery{readRepo: readRepo}
}

func (q *ListOrdersQuery) Execute() ([]domain.OrderReadModel, error) {
	return q.readRepo.ListAll()
}
