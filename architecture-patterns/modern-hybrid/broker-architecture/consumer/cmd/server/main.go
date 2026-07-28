package main

import (
	"fmt"

	"broker"
)

func main() {
	b := broker.GetBroker()
	ch := b.Subscribe("orders")

	fmt.Println("[Consumer] Listening for messages...")
	for msg := range ch {
		fmt.Printf("[Consumer] Received: %v\n", msg.Payload)
	}
}
