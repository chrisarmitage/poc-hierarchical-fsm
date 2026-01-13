package tasks

import (
	"fmt"

	"github.com/chrisarmitage/poc-hierarchical-fsm/internal/events"
	"github.com/chrisarmitage/poc-hierarchical-fsm/internal/sender"
)

// Multi-step task example
type SetProtectedValueTask struct {
	state  State
	sender sender.DeviceCommandSender
}

func NewSetProtectedValueTask(sender sender.DeviceCommandSender) *SetProtectedValueTask {
	return &SetProtectedValueTask{sender: sender}
}

func (t *SetProtectedValueTask) Name() string {
	return "SetProtectedValue"
}

func (t *SetProtectedValueTask) Start() error {
	t.state = "PendingValueUnlock"
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
			fmt.Printf("SetProtectedValueTask: value lock acknowledged, task complete\n")
			fmt.Printf("SetProtectedValueTask: ** completed successfully\n")
			return TaskSucceeded
		case events.DeviceReject, events.Timeout:
			return TaskFailedPermanent
		}
	}
	return TaskRunning
}
