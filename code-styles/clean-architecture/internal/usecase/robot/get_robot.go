package robot

import (
	"context"
	"errors"

	"clean-architecture/internal/domain"
)

// GetRobotInput — запрос на получение робота
type GetRobotInput struct {
	ID string
}

// RobotOutput — ответ
type RobotOutput struct {
	ID     string             `json:"id"`
	Name   string             `json:"name"`
	Status domain.RobotStatus `json:"status"`
}

// GetRobotUseCase — Use Case для получения робота
type GetRobotUseCase struct {
	repo RobotRepository
}

func NewGetRobotUseCase(repo RobotRepository) *GetRobotUseCase {
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
