package inmemory

import (
	"context"
	"errors"
	"sync"

	"clean-architecture/internal/domain"
)

type RobotRepository struct {
	mu     sync.RWMutex
	robots map[string]*domain.Robot
}

func NewRobotRepository() *RobotRepository {
	return &RobotRepository{
		robots: make(map[string]*domain.Robot),
	}
}

func (r *RobotRepository) Get(ctx context.Context, id string) (*domain.Robot, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	robot, ok := r.robots[id]
	if !ok {
		return nil, errors.New("robot not found")
	}
	return robot, nil
}

func (r *RobotRepository) Save(ctx context.Context, robot *domain.Robot) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.robots[robot.ID] = robot
	return nil
}
