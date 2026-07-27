package domain

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
	CreatedAt time.Time
	UpdatedAt time.Time
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
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}, nil
}

// ChangeStatus изменяет статус робота с проверкой правил
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

// IsValidStatus проверяет допустимость статуса
func IsValidStatus(status RobotStatus) bool {
	switch status {
	case StatusIdle, StatusBusy, StatusMaintenance, StatusOffline:
		return true
	default:
		return false
	}
}
