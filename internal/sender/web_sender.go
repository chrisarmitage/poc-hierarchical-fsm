package sender

import (
	"fmt"
	"time"

	"github.com/chrisarmitage/poc-hierarchical-fsm/internal/events"
	"github.com/chrisarmitage/poc-hierarchical-fsm/internal/webserver"
)

type WebSender struct {
	eventsChan chan events.Event
	broadcastChan chan<- webserver.StateUpdate
}

func NewWebSender(
	eventsChan chan events.Event,
	broadcastChan chan<- webserver.StateUpdate,
) *WebSender {
	return &WebSender{
		eventsChan: eventsChan,
		broadcastChan: broadcastChan,
	}
}

func (s *WebSender) Send(cmd events.DeviceCommand) error {
	fmt.Printf("WebSender: sending command %T\n", cmd)

	// Broadcast the command to the web server
	s.broadcastChan <- webserver.StateUpdate{
		Type: "payload",
		System: "WebSender",
		Timestamp: time.Now().Format(time.RFC3339),
		Payload: fmt.Sprintf("%T", cmd),
	}

	return nil
}
