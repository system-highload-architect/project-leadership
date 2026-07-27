package ports

import (
	"context"

	"hexagonal-architecture/internal/core/domain"
)

// RobotRepository — Driven Port (требуемый порт)
// Ядро определяет интерфейс, а адаптер реализует его.
type RobotRepository interface {
	Get(ctx context.Context, id string) (*domain.Robot, error)
	Save(ctx context.Context, robot *domain.Robot) error
}
