package main

import (
	"fmt"
	"time"

	"broker"
)

func main() {
	b := broker.GetBroker()
	for i := 1; i <= 5; i++ {
		msg := fmt.Sprintf("Message #%d", i)
		b.Publish("orders", msg)
		fmt.Printf("[Producer] Sent: %s\n", msg)
		time.Sleep(500 * time.Millisecond)
	}
}
