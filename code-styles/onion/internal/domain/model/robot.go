package model

import (
	"errors"
	"time"
)

// RobotStatus — статус робота
type RobotStatus string

const (
	StatusIdle        RobotStatus = "idle"
	StatusBusy        RobotStatus = "busy"
	StatusMaintenance RobotStatus = "maintenance"
	StatusOffline     RobotStatus = "offline"
)

// Robot — доменная сущность (ядро)
type Robot struct {
	ID        string
	Name      string
	Status    RobotStatus
	Tasks     []Task
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Task — задача (вложенная сущность)
type Task struct {
	ID          string
	Description string
	Status      string // pending, in_progress, completed
}

// NewRobot создаёт нового робота с валидацией
func NewRobot(id, name string, status RobotStatus) (*Robot, error) {
	if id == "" {
		return nil, errors.New("robot id cannot be empty")
	}
	if name == "" {
		return nil, errors.New("robot name cannot be empty")
	}
	if !IsValidStatus(status) {
		return nil, errors.New("invalid robot status")
	}
	return &Robot{
		ID:        id,
		Name:      name,
		Status:    status,
		Tasks:     []Task{},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}, nil
}

// ChangeStatus изменяет статус с проверкой правил
func (r *Robot) ChangeStatus(newStatus RobotStatus) error {
	if !IsValidStatus(newStatus) {
		return errors.New("invalid robot status")
	}
	if r.Status == StatusOffline && newStatus != StatusOffline {
		return errors.New("cannot change status from offline to other")
	}
	r.Status = newStatus
	r.UpdatedAt = time.Now()
	return nil
}

// AddTask добавляет задачу к роботу
func (r *Robot) AddTask(task Task) {
	r.Tasks = append(r.Tasks, task)
	r.UpdatedAt = time.Now()
}

// IsValidStatus проверяет допустимость статуса
func IsValidStatus(status RobotStatus) bool {
	switch status {
	case StatusIdle, StatusBusy, StatusMaintenance, StatusOffline:
		return true
	default:
		return false
	}
}
