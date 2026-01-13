package devicefsm

import (
	"fmt"

	"github.com/chrisarmitage/poc-hierarchical-fsm/internal/events"
	"github.com/chrisarmitage/poc-hierarchical-fsm/internal/sender"
	"github.com/chrisarmitage/poc-hierarchical-fsm/internal/taskrunner"
	"github.com/chrisarmitage/poc-hierarchical-fsm/internal/tasks"
)

// device-level FSM
type DeviceFSM struct {
	state      tasks.State
	taskRunner *taskrunner.TaskRunner
	sender     sender.DeviceCommandSender
}

func NewDeviceFSM(sender sender.DeviceCommandSender) *DeviceFSM {
	return &DeviceFSM{state: "Ready", sender: sender}
}

func (d *DeviceFSM) HandleEvent(event events.Event) error {
	fmt.Printf("DeviceFSM: handling event %T in state %s\n", event, d.state)
	switch d.state {
	case "Ready":
		if _, ok := event.(events.StartConfig); ok {
			// Enter config mode
			d.state = "PendingConfiguring"
			// send StartConfig command
			fmt.Printf("DeviceFSM: entered PendingConfiguring state\n")
			return d.sender.Send(events.StartConfigCommand{})
		}
	case "PendingConfiguring":
		if _, ok := event.(events.DeviceAck); ok {
			d.state = "Configuring"
			fmt.Printf("DeviceFSM: entering Configuring state, starting tasks\n")
			d.taskRunner = taskrunner.NewTaskRunner(taskrunner.BuildTasks(d.sender))
			return d.taskRunner.Start()
		}
	case "Configuring":
		done, err := d.taskRunner.HandleEvent(event)
		fmt.Printf("DeviceFSM: task runner returned done=%v, err=%v for event %T\n", done, err, event)
		if err != nil {
			// abort policy decision here
			d.state = "EndingConfiguring"
			fmt.Printf("DeviceFSM: task runner error, entering EndingConfiguring state\n")
			// send EndConfig
			return err
		}
		if done {
			d.state = "EndingConfiguring"
			fmt.Printf("DeviceFSM: tasks completed, entering EndingConfiguring state\n")
			// send EndConfig
			return d.sender.Send(events.EndConfigCommand{})
		}
	case "EndingConfiguring":
		if _, ok := event.(events.DeviceAck); ok {
			d.state = "Ready"
			fmt.Printf("DeviceFSM: configuration ended, entering Ready state\n")
			fmt.Printf("DeviceFSM: ** all tasks completed successfully **\n")
		}
	}
	return nil
}
