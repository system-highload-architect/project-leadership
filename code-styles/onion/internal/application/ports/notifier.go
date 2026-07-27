package ports

import "context"

// Notifier — Driven Port для уведомлений
type Notifier interface {
	Send(ctx context.Context, event string, payload interface{}) error
}
