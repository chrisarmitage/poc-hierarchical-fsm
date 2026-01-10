package main

import (
	"fmt"
	"time"
)

func main() {
	// Example usage
	// tasks := []Task{
	// 	// &SetSleepPeriodTask{},
	// 	&SetProtectedValueTask{},
	// 	// Add more tasks here
	// }

	// Create a task runner and start it
	// taskRunner := NewTaskRunner(tasks)
	// err := taskRunner.Start()
	// if err != nil {
	// 	fmt.Println("Error starting tasks:", err)
	// 	return
	// }

	// init DeviceFSM
	deviceFSM := &DeviceFSM{state: "Ready"}

	// Simulate incoming events
	events := []Event{
		StartConfig{},
		DeviceAck{}, // StartConfig ack

		Timeout{}, // SetSleepPeriod timeout
		Timeout{},
		Timeout{},
		DeviceAck{}, // SetSleepPeriod ack
		
		DeviceAck{}, // SetProtectedValue acks
		DeviceAck{},
		DeviceAck{},

		DeviceAck{}, // EndConfig ack
		// Add more events here
	}

	for _, event := range events {
		err := deviceFSM.HandleEvent(event)
		if err != nil {
			fmt.Println("Error handling event:", err)
			return
		}
	}

	fmt.Println("All tasks completed successfully")
}

func buildTasks() []Task {
	return []Task{
		&SetSleepPeriodTask{},
		&SetProtectedValueTask{},
		// Add more tasks here
	}
}

// placeholder for enums
type State string

// placeholder for event data
type Event any

// sample events
type DeviceAck struct{}
type DeviceReject struct{}
type Timeout struct{}

// Task interface. Can be single or multi-step.
type TaskResult int

const (
	TaskRunning TaskResult = iota
	TaskSucceeded
	TaskFailedRetryable
	TaskFailedPermanent
)

type Task interface {
	Name() string
	Start() error
	HandleEvent(event Event) TaskResult
}

// Task runner. Contains multiple tasks to be executed in sequence.
type TaskRunner struct {
	tasks []Task
	index int
}

func NewTaskRunner(tasks []Task) *TaskRunner {
	return &TaskRunner{tasks: tasks}
}

func (tr *TaskRunner) Start() error {
	if len(tr.tasks) == 0 {
		return nil
	}
	fmt.Println("TaskRunner: starting")
	return tr.tasks[tr.index].Start()
}

func (tr *TaskRunner) HandleEvent(event Event) (done bool, err error) {
	if tr.index >= len(tr.tasks) {
		return true, nil
	}

	fmt.Printf("TaskRunner: handling event %T for activetask %s\n", event, tr.tasks[tr.index].Name())

	task := tr.tasks[tr.index]
	result := task.HandleEvent(event)

	fmt.Printf("TaskRunner: task %s returned result %d\n", task.Name(), result)

	switch result {
	case TaskRunning:
		return false, nil

	case TaskSucceeded:
		tr.index++
		if tr.index >= len(tr.tasks) {
			return true, nil
		}
		return false, tr.tasks[tr.index].Start()
	
	case TaskFailedRetryable:
		// optional: retry entire task
		return false, task.Start()

	case TaskFailedPermanent:
		return false, fmt.Errorf("task %s failed", task.Name())
	}

	return false, nil
}

// Single-step task example
type SetSleepPeriodTask struct {
	state State
	retries int
	max     int
	backoff Backoff
}

func (t *SetSleepPeriodTask) Name() string {
	return "SetSleepPeriod"
}

func (t *SetSleepPeriodTask) Start() error {
	t.state = "Pending"
	t.retries = 0
	t.max = 5
	if t.backoff == nil {
		t.backoff = NewExponentialBackoff(1*time.Second, 16*time.Second)
	}
	t.backoff.Reset()
	// send command to device
	return nil
}

func (t *SetSleepPeriodTask) HandleEvent(event Event) TaskResult {
	fmt.Printf("SetSleepPeriodTask: handling event %T in state %s\n", event, t.state)
	switch t.state {
	case "Pending":
		switch event.(type) {
		case DeviceAck:
			t.state = "Done"
			fmt.Printf("SetSleepPeriodTask: acknowledged, task complete\n")
			fmt.Printf("SetSleepPeriodTask: ** completed successfully\n")
			return TaskSucceeded
		case Timeout:
			t.retries++
			if t.retries > t.max {
				return TaskFailedPermanent
			}
			backoffDuration := t.backoff.Next()
			fmt.Printf("SetSleepPeriodTask: timeout, retrying in %v (attempt %d/%d)\n", backoffDuration, t.retries, t.max)
			time.Sleep(backoffDuration)
			// resend command to device
			return TaskRunning
		case DeviceReject:
			return TaskFailedPermanent
		}
	}
	return TaskRunning
}

// Multi-step task example
type SetProtectedValueTask struct {
	state State
}

func (t *SetProtectedValueTask) Name() string {
	return "SetProtectedValue"
}


func (t *SetProtectedValueTask) Start() error {
	t.state = "PendingValueUnlock"
	fmt.Printf("SetProtectedValueTask: sending value unlock command\n")
	// send command to device
	return nil
}

func (t *SetProtectedValueTask) HandleEvent(event Event) TaskResult {
	switch t.state {
	case "PendingValueUnlock":
		switch event.(type) {
		case DeviceAck:
			t.state = "PendingSetValue"
			fmt.Printf("SetProtectedValueTask: value unlock acknowledged, sending set value command\n")
			// send set value command to device
			return TaskRunning
		case DeviceReject, Timeout:
			return TaskFailedPermanent
		}
	case "PendingSetValue":
		switch event.(type) {
		case DeviceAck:
			t.state = "PendingValueLock"
			fmt.Printf("SetProtectedValueTask: set value acknowledged, sending value lock command\n")
			// send value lock command to device
			return TaskRunning
		case DeviceReject, Timeout:
			return TaskFailedPermanent
		}
	case "PendingValueLock":
		switch event.(type) {
		case DeviceAck:
			t.state = "Done"
			fmt.Printf("SetProtectedValueTask: value lock acknowledged, task complete\n")
			fmt.Printf("SetProtectedValueTask: ** completed successfully\n")
			return TaskSucceeded
		case DeviceReject, Timeout:
			return TaskFailedPermanent
		}
	}
	return TaskRunning
}

// device-level FSM
type DeviceFSM struct {
	state State
	taskRunner *TaskRunner
}

func (d *DeviceFSM) HandleEvent(event Event) error {
	switch d.state {
	case "Ready":
		if _, ok := event.(StartConfig); ok {
			// Enter config mode
			d.state = "PendingConfiguring"
			// send StartConfig command
			fmt.Printf("DeviceFSM: entering PendingConfiguring state\n")
		}
	case "PendingConfiguring":
		if _, ok := event.(DeviceAck); ok {
			d.state = "Configuring"
			fmt.Printf("DeviceFSM: entering Configuring state, starting tasks\n")
			d.taskRunner = NewTaskRunner(buildTasks())
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
		}
	case "EndingConfiguring":
		if _, ok := event.(DeviceAck); ok {
			d.state = "Ready"
			fmt.Printf("DeviceFSM: configuration ended, entering Ready state\n")
			fmt.Printf("DeviceFSM: ** all tasks completed successfully **\n")
		}
	}
	return nil
}

type StartConfig struct{}

// backoff utility class

// backoff
type Backoff interface {
	Next() time.Duration
	Reset()
}

type ExponentialBackoff struct {
	base time.Duration
	max  time.Duration
	curr time.Duration
}

func NewExponentialBackoff(base, max time.Duration) *ExponentialBackoff {
	return &ExponentialBackoff{base: base, max: max}
}

func (b *ExponentialBackoff) Next() time.Duration {
	if b.curr == 0 {
		b.curr = b.base
	} else {
		b.curr *= 2
		if b.curr > b.max {
			b.curr = b.max
		}
	}
	return b.curr
}

func (b *ExponentialBackoff) Reset() {
	b.curr = 0
}
