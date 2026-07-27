package usecases

import (
	"context"
	"errors"
	"fmt"

	"hexagonal-architecture/internal/core/domain"
	"hexagonal-architecture/internal/core/ports"
)

// UpdateStatusInput — запрос
type UpdateStatusInput struct {
	ID     string
	Status domain.RobotStatus
}

// UpdateStatusUseCase — Use Case
type UpdateStatusUseCase struct {
	repo     ports.RobotRepository
	notifier ports.Notifier
}

func NewUpdateStatusUseCase(repo ports.RobotRepository, notifier ports.Notifier) *UpdateStatusUseCase {
	return &UpdateStatusUseCase{
		repo:     repo,
		notifier: notifier,
	}
}

func (uc *UpdateStatusUseCase) Execute(ctx context.Context, input UpdateStatusInput) error {
	if input.ID == "" {
		return errors.New("robot id is required")
	}
	robot, err := uc.repo.Get(ctx, input.ID)
	if err != nil {
		return err
	}
	if err := robot.ChangeStatus(input.Status); err != nil {
		return err
	}
	if err := uc.repo.Save(ctx, robot); err != nil {
		return err
	}
	// Уведомление через порт (ядро не знает, кто и как уведомляет)
	_ = uc.notifier.Notify(ctx, fmt.Sprintf("Robot %s status changed to %s", robot.ID, robot.Status))
	return nil
}
