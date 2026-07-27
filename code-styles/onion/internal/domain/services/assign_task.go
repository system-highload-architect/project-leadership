package services

import (
	"errors"

	"onion-architecture/internal/domain/model"
)

// TaskAssigner — доменный сервис для назначения задач
type TaskAssigner interface {
	AssignTask(robot *model.Robot, task model.Task) error
}

// DefaultAssigner — стандартная реализация
type DefaultAssigner struct{}

func (a *DefaultAssigner) AssignTask(robot *model.Robot, task model.Task) error {
	if robot == nil {
		return errors.New("robot is nil")
	}
	if task.ID == "" {
		return errors.New("task id cannot be empty")
	}
	// Бизнес-правило: робот должен быть в статусе idle или busy, чтобы получить задачу
	if robot.Status != model.StatusIdle && robot.Status != model.StatusBusy {
		return errors.New("robot cannot accept tasks in current status")
	}
	// Проверяем, нет ли уже такой задачи
	for _, t := range robot.Tasks {
		if t.ID == task.ID {
			return errors.New("task already assigned")
		}
	}
	// Назначаем задачу
	task.Status = "pending"
	robot.AddTask(task)
	return nil
}
