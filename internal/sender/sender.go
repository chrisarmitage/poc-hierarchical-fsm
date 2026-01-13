package sender

import (
	"fmt"
	"time"

	"github.com/chrisarmitage/poc-hierarchical-fsm/internal/events"
)

// CommandSender code
type DeviceCommandSender interface {
	Send(cmd events.DeviceCommand) error
}

type FakeSender struct {
	eventsChan chan events.Event
}

func NewFakeSender(eventsChan chan events.Event) *FakeSender {
	return &FakeSender{eventsChan: eventsChan}
}

func (s *FakeSender) Send(cmd events.DeviceCommand) error {
	fmt.Printf("FakeSender: sending command %T\n", cmd)
	time.Sleep(1 * time.Second)
	// Simulate immediate ack for demo purposes
	switch cmd.(type) {
	case events.StartConfigCommand:
		fmt.Printf("FakeSender: triggering mock response: DeviceAck\n")
		go func() {
			s.eventsChan <- events.DeviceAck{}
		}()
	case events.EndConfigCommand:
		go func() {
			s.eventsChan <- events.EndConfigAck{}
		}()
	case events.SetSleepPeriodCommand:
		go func() {
			s.eventsChan <- events.Timeout{}
		}()

		time.AfterFunc(3*time.Second, func() {
			s.eventsChan <- events.DeviceAck{
				AckCode: "OTHER_COMMAND",
			}
		})

		time.AfterFunc(6*time.Second, func() {
			s.eventsChan <- events.DeviceAck{
				AckCode: "SLEEP_PERIOD",
			}
		})
	case events.ValueUnlockCommand:
		go func() {
			s.eventsChan <- events.DeviceAck{
				AckCode: "VALUE_UNLOCK",
			}

			// send earlier ack again to test idempotency
			time.Sleep(2 * time.Second)
			s.eventsChan <- events.DeviceAck{
				AckCode: "SLEEP_PERIOD",
			}
		}()
	case events.SetProtectedValueCommand:
		go func() {
			s.eventsChan <- events.DeviceAck{
				AckCode: "SET_PROTECTED_VALUE",
			}

			// send earlier ack again to test idempotency
			time.Sleep(2 * time.Second)
			s.eventsChan <- events.DeviceAck{
				AckCode: "SLEEP_PERIOD",
			}
		}()
	case events.ValueLockCommand:
		go func() {
			s.eventsChan <- events.DeviceAck{
				AckCode: "VALUE_LOCK",
			}
		}()
	}
	return nil
}
