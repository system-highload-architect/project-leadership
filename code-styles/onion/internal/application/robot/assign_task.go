package robot

import (
	"context"
	"errors"
	"fmt"

	"onion-architecture/internal/application/ports"
	"onion-architecture/internal/domain/model"
	"onion-architecture/internal/domain/services"
)

// AssignTaskInput — запрос
type AssignTaskInput struct {
	RobotID     string
	TaskID      string
	Description string
}

// AssignTaskUseCase — Use Case
type AssignTaskUseCase struct {
	repo     ports.RobotRepository
	assigner services.TaskAssigner
	notifier ports.Notifier
}

func NewAssignTaskUseCase(
	repo ports.RobotRepository,
	assigner services.TaskAssigner,
	notifier ports.Notifier,
) *AssignTaskUseCase {
	return &AssignTaskUseCase{
		repo:     repo,
		assigner: assigner,
		notifier: notifier,
	}
}

func (uc *AssignTaskUseCase) Execute(ctx context.Context, input AssignTaskInput) error {
	if input.RobotID == "" || input.TaskID == "" {
		return errors.New("robot id and task id are required")
	}
	robot, err := uc.repo.Get(ctx, input.RobotID)
	if err != nil {
		return err
	}
	task := model.Task{
		ID:          input.TaskID,
		Description: input.Description,
	}
	if err := uc.assigner.AssignTask(robot, task); err != nil {
		return err
	}
	if err := uc.repo.Save(ctx, robot); err != nil {
		return err
	}
	// Уведомление через порт
	_ = uc.notifier.Send(ctx, "task_assigned", map[string]interface{}{
		"robot_id": robot.ID,
		"task_id":  task.ID,
	})
	fmt.Printf("[UseCase] Task %s assigned to robot %s\n", task.ID, robot.ID)
	return nil
}
