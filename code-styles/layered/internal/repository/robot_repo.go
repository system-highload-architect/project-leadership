package repository

import (
	"context"

	"layered-architecture/internal/domain"
)

// RobotRepository — интерфейс для работы с хранилищем роботов
type RobotRepository interface {
	Get(ctx context.Context, id string) (*domain.Robot, error)
	Save(ctx context.Context, robot *domain.Robot) error
}
