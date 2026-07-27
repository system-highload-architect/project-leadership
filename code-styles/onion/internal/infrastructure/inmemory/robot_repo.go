package inmemory

import (
	"context"
	"errors"
	"sync"

	"onion-architecture/internal/domain/model"
)

type RobotRepository struct {
	mu     sync.RWMutex
	robots map[string]*model.Robot
}

func NewRobotRepository() *RobotRepository {
	return &RobotRepository{
		robots: make(map[string]*model.Robot),
	}
}

func (r *RobotRepository) Get(ctx context.Context, id string) (*model.Robot, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	robot, ok := r.robots[id]
	if !ok {
		return nil, errors.New("robot not found")
	}
	return robot, nil
}

func (r *RobotRepository) Save(ctx context.Context, robot *model.Robot) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.robots[robot.ID] = robot
	return nil
}
