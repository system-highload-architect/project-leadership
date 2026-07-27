package service

import (
	"context"
	"errors"
	"fmt"

	"layered-architecture/internal/domain"
	"layered-architecture/internal/repository"
)

// RobotService — бизнес-логика (сервисный слой)
type RobotService struct {
	repo repository.RobotRepository
}

func NewRobotService(repo repository.RobotRepository) *RobotService {
	return &RobotService{repo: repo}
}

// GetRobot возвращает робота по ID
func (s *RobotService) GetRobot(ctx context.Context, id string) (*domain.Robot, error) {
	if id == "" {
		return nil, errors.New("robot id is required")
	}
	return s.repo.Get(ctx, id)
}

// UpdateStatus изменяет статус робота
func (s *RobotService) UpdateStatus(ctx context.Context, id string, status domain.RobotStatus) error {
	if id == "" {
		return errors.New("robot id is required")
	}
	robot, err := s.repo.Get(ctx, id)
	if err != nil {
		return err
	}
	if err := robot.ChangeStatus(status); err != nil {
		return err
	}
	if err := s.repo.Save(ctx, robot); err != nil {
		return err
	}
	// здесь можно добавить логирование или отправку события
	fmt.Printf("[service] Robot %s status updated to %s\n", robot.ID, robot.Status)
	return nil
}
