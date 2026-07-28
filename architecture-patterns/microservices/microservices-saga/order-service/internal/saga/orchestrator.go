package saga

import (
	"context"
	"fmt"
)

// Orchestrator — управляет последовательностью шагов
type Orchestrator struct {
	steps []Step
}

func NewOrchestrator() *Orchestrator {
	// Создаём шаги (можно менять порядок и добавлять новые)
	steps := []Step{
		&CreateOrderStep{},
		NewProcessPaymentStep(false), // true — для демонстрации ошибки
		&ScheduleDeliveryStep{},
	}
	return &Orchestrator{steps: steps}
}

// Execute запускает сагу
func (o *Orchestrator) Execute(ctx context.Context, data map[string]interface{}) error {
	history := []int{} // хранит индексы успешно выполненных шагов

	for i, step := range o.steps {
		if err := step.Execute(ctx, data); err != nil {
			// Ошибка → откат
			fmt.Printf("[Saga] Failed at step %d: %v. Starting compensation...\n", i, err)
			o.compensate(ctx, data, history)
			return fmt.Errorf("saga failed at step %d: %w", i, err)
		}
		history = append(history, i)
	}
	fmt.Println("[Saga] Completed successfully")
	return nil
}

func (o *Orchestrator) compensate(ctx context.Context, data map[string]interface{}, history []int) {
	for i := len(history) - 1; i >= 0; i-- {
		idx := history[i]
		if err := o.steps[idx].Compensate(ctx, data); err != nil {
			fmt.Printf("[Saga] Compensation error at step %d: %v\n", idx, err)
		}
	}
}
