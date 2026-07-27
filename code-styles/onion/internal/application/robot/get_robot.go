package robot

import (
	"context"
	"errors"

	"onion-architecture/internal/application/ports"
	"onion-architecture/internal/domain/model"
)

// GetRobotInput — запрос
type GetRobotInput struct {
	ID string
}

// GetRobotUseCase — Use Case
type GetRobotUseCase struct {
	repo ports.RobotRepository
}

func NewGetRobotUseCase(repo ports.RobotRepository) *GetRobotUseCase {
	return &GetRobotUseCase{repo: repo}
}

func (uc *GetRobotUseCase) Execute(ctx context.Context, input GetRobotInput) (*model.Robot, error) {
	if input.ID == "" {
		return nil, errors.New("robot id is required")
	}
	return uc.repo.Get(ctx, input.ID)
}
