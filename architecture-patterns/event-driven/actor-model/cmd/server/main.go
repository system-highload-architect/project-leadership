package main

import (
	"fmt"
	"time"

	"actor-model/internal/actor"
)

func main() {
	system := actor.NewSystem()

	printer := system.Spawn("printer", func(msg interface{}) {
		fmt.Printf("[Printer] %v\n", msg)
	})

	var counter *actor.Actor
	counter = system.Spawn("counter", func(msg interface{}) {
		count, ok := msg.(int)
		if !ok {
			return
		}
		fmt.Printf("[Counter] Received: %d\n", count)
		if count < 5 {
			time.Sleep(500 * time.Millisecond)
			counter.Send(count + 1)
		} else {
			printer.Send("Counter finished!")
		}
	})

	counter.Send(1)

	time.Sleep(5 * time.Second)
}
