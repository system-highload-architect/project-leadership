package robot

import (
	"context"

	"clean-architecture/internal/domain"
)

// RobotRepository — интерфейс репозитория (порт)
type RobotRepository interface {
	Get(ctx context.Context, id string) (*domain.Robot, error)
	Save(ctx context.Context, robot *domain.Robot) error
}
