package usecases

import (
	"context"
	"errors"

	"hexagonal-architecture/internal/core/domain"
	"hexagonal-architecture/internal/core/ports"
)

// GetRobotInput — запрос
type GetRobotInput struct {
	ID string
}

// RobotOutput — ответ
type RobotOutput struct {
	ID     string             `json:"id"`
	Name   string             `json:"name"`
	Status domain.RobotStatus `json:"status"`
}

// GetRobotUseCase — Use Case
type GetRobotUseCase struct {
	repo ports.RobotRepository // зависит от порта, а не от реализации
}

func NewGetRobotUseCase(repo ports.RobotRepository) *GetRobotUseCase {
	return &GetRobotUseCase{repo: repo}
}

func (uc *GetRobotUseCase) Execute(ctx context.Context, input GetRobotInput) (*RobotOutput, error) {
	if input.ID == "" {
		return nil, errors.New("robot id is required")
	}
	robot, err := uc.repo.Get(ctx, input.ID)
	if err != nil {
		return nil, err
	}
	return &RobotOutput{
		ID:     robot.ID,
		Name:   robot.Name,
		Status: robot.Status,
	}, nil
}
