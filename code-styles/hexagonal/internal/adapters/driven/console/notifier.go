package console

import (
	"context"
	"fmt"
)

// Notifier — Driven Adapter (реализация порта)
type Notifier struct{}

func NewNotifier() *Notifier {
	return &Notifier{}
}

func (n *Notifier) Notify(ctx context.Context, message string) error {
	fmt.Printf("[NOTIFIER] %s\n", message)
	return nil
}
