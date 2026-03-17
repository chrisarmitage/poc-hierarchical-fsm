package main

import (
	"fmt"
	"log"
	"time"

	"github.com/chrisarmitage/poc-hierarchical-fsm/internal/devicefsm"
	"github.com/chrisarmitage/poc-hierarchical-fsm/internal/events"
	"github.com/chrisarmitage/poc-hierarchical-fsm/internal/sender"
	"github.com/chrisarmitage/poc-hierarchical-fsm/internal/timeoutmanager"
	"github.com/chrisarmitage/poc-hierarchical-fsm/internal/webserver"
)

func main() {
	eventsChan := make(chan events.Event)

	// Create web server
	webServer := webserver.NewServer(eventsChan)

	// Start web server in background
	go func() {
		log.Println("Starting web server on http://localhost:8080")
		if err := webServer.Start(":8080"); err != nil {
			log.Fatalf("Failed to start web server: %v", err)
		}
	}()

	// s := sender.NewFakeSender(eventsChan)
	s := sender.NewWebSender(eventsChan, webServer.GetBroadcastChannel())

	// Create timeout manager
	timeoutMgr := timeoutmanager.NewTimeoutManager(eventsChan)
	defer timeoutMgr.CancelAll() // Ensure cleanup on exit

	// init DeviceFSM
	deviceFSM := devicefsm.NewDeviceFSM(s, timeoutMgr, webServer.GetBroadcastChannel())

	// send intiial event
	go func() {
		time.Sleep(5 * time.Second)
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
	time.Sleep(500 * time.Millisecond) // Give web server time to send final update
}
