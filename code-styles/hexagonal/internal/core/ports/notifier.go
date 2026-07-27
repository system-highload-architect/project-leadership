package ports

import "context"

// Notifier — Driven Port (требуемый порт) для уведомлений
type Notifier interface {
	Notify(ctx context.Context, message string) error
}
