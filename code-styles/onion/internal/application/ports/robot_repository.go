package ports

import (
	"context"

	"onion-architecture/internal/domain/model"
)

// RobotRepository — Driven Port (интерфейс для репозитория)
type RobotRepository interface {
	Get(ctx context.Context, id string) (*model.Robot, error)
	Save(ctx context.Context, robot *model.Robot) error
}
