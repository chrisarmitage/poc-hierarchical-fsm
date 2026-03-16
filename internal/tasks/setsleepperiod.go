package tasks

import (
	"fmt"
	"time"

	"github.com/chrisarmitage/poc-hierarchical-fsm/internal/backoff"
	"github.com/chrisarmitage/poc-hierarchical-fsm/internal/events"
	"github.com/chrisarmitage/poc-hierarchical-fsm/internal/sender"
	"github.com/chrisarmitage/poc-hierarchical-fsm/internal/webserver"
)

// Single-step task example
type SetSleepPeriodTask struct {
	state           State
	retries         int
	max             int
	backoff         backoff.Backoff
	sender          sender.DeviceCommandSender
	timeoutDuration time.Duration
	broadcastChan   chan<- webserver.StateUpdate
}

func NewSetSleepPeriodTask(sender sender.DeviceCommandSender, broadcastChan chan<- webserver.StateUpdate) *SetSleepPeriodTask {
	return &SetSleepPeriodTask{
		sender:          sender,
		timeoutDuration: 10 * time.Second,
		broadcastChan:   broadcastChan,
	}
}

func (t *SetSleepPeriodTask) Name() string {
	return "SetSleepPeriod"
}

func (t *SetSleepPeriodTask) GetTimeoutDuration() time.Duration {
	return t.timeoutDuration
}

func (t *SetSleepPeriodTask) Start() error {
	t.state = "Pending"
	t.broadcastChan <- webserver.StateUpdate{
		Type:  "task",
		System: "SetSleepPeriod",
		State: string(t.state),
	}
	t.retries = 0
	t.max = 5
	if t.backoff == nil {
		t.backoff = backoff.NewExponentialBackoff(1*time.Second, 16*time.Second)
	}
	t.backoff.Reset()
	// send command to device
	t.sender.Send(events.SetSleepPeriodCommand{})
	return nil
}

func (t *SetSleepPeriodTask) HandleEvent(event events.Event) TaskResult {
	fmt.Printf("SetSleepPeriodTask: handling event %T in state %s\n", event, t.state)
	switch t.state {
	case "Pending":
		switch e := event.(type) {
		case events.DeviceAck:
			if e.AckCode != "SLEEP_PERIOD" {
				fmt.Printf("SetSleepPeriodTask: received ack for %s, ignoring\n", e.AckCode)
				// This device ack if not for us
				return TaskRunning
			}
			t.state = "Done"
			t.broadcastChan <- webserver.StateUpdate{
				Type:  "task",
				System: "SetSleepPeriod",
				State: string(t.state),
			}
			fmt.Printf("SetSleepPeriodTask: acknowledged, task complete\n")
			fmt.Printf("SetSleepPeriodTask: ** completed successfully\n")
			return TaskSucceeded
		case events.Timeout:

			t.broadcastChan <- webserver.StateUpdate{
				Type:  "task",
				System: "SetSleepPeriod",
				State: "timeout",
			}
			t.retries++
			if t.retries > t.max {
				return TaskFailedPermanent
			}
			backoffDuration := t.backoff.Next()
			fmt.Printf("SetSleepPeriodTask: timeout, retrying in %v (attempt %d/%d)\n", backoffDuration, t.retries, t.max)

			// Update timeout duration to include backoff
			t.timeoutDuration = backoffDuration + 10*time.Second

			// Signal retry - TaskRunner will cancel old timeout, call Start(), and arm new timeout
			return TaskFailedRetryable
		case events.DeviceReject:
			return TaskFailedPermanent
		}
	}
	return TaskRunning
}
