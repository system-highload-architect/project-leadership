package console

import (
	"context"
	"fmt"
)

type Notifier struct{}

func NewNotifier() *Notifier {
	return &Notifier{}
}

func (n *Notifier) Send(ctx context.Context, event string, payload interface{}) error {
	fmt.Printf("[NOTIFIER] event=%s payload=%v\n", event, payload)
	return nil
}
