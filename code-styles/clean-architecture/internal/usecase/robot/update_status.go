package robot

import (
	"context"
	"errors"

	"clean-architecture/internal/domain"
)

// UpdateStatusInput — запрос на изменение статуса
type UpdateStatusInput struct {
	ID     string
	Status domain.RobotStatus
}

// UpdateStatusUseCase — Use Case для обновления статуса
type UpdateStatusUseCase struct {
	repo RobotRepository
}

func NewUpdateStatusUseCase(repo RobotRepository) *UpdateStatusUseCase {
	return &UpdateStatusUseCase{repo: repo}
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
	return uc.repo.Save(ctx, robot)
}
