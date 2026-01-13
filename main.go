package main

import (
	"fmt"
	"time"

	"github.com/chrisarmitage/poc-hierarchical-fsm/internal/devicefsm"
	"github.com/chrisarmitage/poc-hierarchical-fsm/internal/events"
	"github.com/chrisarmitage/poc-hierarchical-fsm/internal/sender"
)

func main() {
	eventsChan := make(chan events.Event)

	s := sender.NewFakeSender(eventsChan)

	// init DeviceFSM
	deviceFSM := devicefsm.NewDeviceFSM(s)

	// send intiial event
	go func() {
		time.Sleep(1 * time.Second)
		eventsChan <- events.StartConfig{}
	}()

	for event := range eventsChan {
		fmt.Printf("\nMain: received event %T\n", event)
		err := deviceFSM.HandleEvent(event)
		if err != nil {
			fmt.Println("Error handling event:", err)
			return
		}

		if eventAck, ok := event.(events.EndConfigAck); ok {
			fmt.Printf("Main: received EndConfigAck event %T, closing events channel\n", eventAck)
			close(eventsChan)
		}
	}

	fmt.Println("All tasks completed successfully")
}
