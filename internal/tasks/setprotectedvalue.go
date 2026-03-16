package tasks

import (
	"fmt"
	"time"

	"github.com/chrisarmitage/poc-hierarchical-fsm/internal/events"
	"github.com/chrisarmitage/poc-hierarchical-fsm/internal/sender"
	"github.com/chrisarmitage/poc-hierarchical-fsm/internal/webserver"
)

// Multi-step task example
type SetProtectedValueTask struct {
	state           State
	sender          sender.DeviceCommandSender
	timeoutDuration time.Duration
	broadcastChan   chan<- webserver.StateUpdate
}

func NewSetProtectedValueTask(sender sender.DeviceCommandSender, broadcastChan chan<- webserver.StateUpdate) *SetProtectedValueTask {
	return &SetProtectedValueTask{
		sender:          sender,
		timeoutDuration: 10 * time.Second,
		broadcastChan:   broadcastChan,
	}
}

func (t *SetProtectedValueTask) Name() string {
	return "SetProtectedValue"
}

func (t *SetProtectedValueTask) GetTimeoutDuration() time.Duration {
	return t.timeoutDuration
}

func (t *SetProtectedValueTask) Start() error {
	t.state = "PendingValueUnlock"
	t.broadcastChan <- webserver.StateUpdate{
		Type:  "task",
		System: "SetProtectedValue",
		State: string(t.state),
	}
	fmt.Printf("SetProtectedValueTask: sending value unlock command\n")
	// send command to device
	t.sender.Send(events.ValueUnlockCommand{})
	return nil
}

func (t *SetProtectedValueTask) HandleEvent(event events.Event) TaskResult {
	switch t.state {
	case "PendingValueUnlock":
		switch e := event.(type) {
		case events.DeviceAck:
			if e.AckCode != "VALUE_UNLOCK" {
				fmt.Printf("SetProtectedValueTask: received ack for %s, ignoring\n", e.AckCode)
				// This device ack if not for us
				return TaskRunning
			}
			t.state = "PendingSetValue"
			t.broadcastChan <- webserver.StateUpdate{
				Type:  "task",
				System: "SetProtectedValue",
				State: string(t.state),
			}
			fmt.Printf("SetProtectedValueTask: value unlock acknowledged, sending set value command\n")
			// send set value command to device
			t.sender.Send(events.SetProtectedValueCommand{})
			return TaskRunning
		case events.DeviceReject, events.Timeout:
			return TaskFailedPermanent
		}
	case "PendingSetValue":
		switch e := event.(type) {
		case events.DeviceAck:
			if e.AckCode != "SET_PROTECTED_VALUE" {
				fmt.Printf("SetProtectedValueTask: received ack for %s, ignoring\n", e.AckCode)
				// This device ack if not for us
				return TaskRunning
			}
			t.state = "PendingValueLock"
			t.broadcastChan <- webserver.StateUpdate{
				Type:  "task",
				System: "SetProtectedValue",
				State: string(t.state),
			}
			fmt.Printf("SetProtectedValueTask: set value acknowledged, sending value lock command\n")
			// send value lock command to device
			t.sender.Send(events.ValueLockCommand{})
			return TaskRunning
		case events.DeviceReject, events.Timeout:
			return TaskFailedPermanent
		}
	case "PendingValueLock":
		switch e := event.(type) {
		case events.DeviceAck:
			if e.AckCode != "VALUE_LOCK" {
				fmt.Printf("SetProtectedValueTask: received ack for %s, ignoring\n", e.AckCode)
				// This device ack if not for us
				return TaskRunning
			}
			t.state = "Done"
			t.broadcastChan <- webserver.StateUpdate{
				Type:  "task",
				System: "SetProtectedValue",
				State: string(t.state),
			}
			fmt.Printf("SetProtectedValueTask: value lock acknowledged, task complete\n")
			fmt.Printf("SetProtectedValueTask: ** completed successfully\n")
			return TaskSucceeded
		case events.DeviceReject, events.Timeout:
			return TaskFailedPermanent
		}
	}
	return TaskRunning
}
