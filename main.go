package main

import (
	"fmt"
)

func main() {
	// Example usage
	tasks := []Task{
		&SetSleepPeriodTask{},
		// Add more tasks here
	}

	// Create a task runner and start it
	taskRunner := NewTaskRunner(tasks)
	err := taskRunner.Start()
	if err != nil {
		fmt.Println("Error starting tasks:", err)
		return
	}

	// Simulate handling events
	events := []Event{
		DeviceAck{},
		// Add more events here
	}

	
	for _, event := range events {
		done, err := taskRunner.HandleEvent(event)
		if err != nil {
			fmt.Println("Error handling event:", err)
			return
		}
		if done {
			fmt.Println("All tasks completed successfully")
			break
		}
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
	TaskFailed
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

	case TaskFailed:
		return false, fmt.Errorf("task %s failed", task.Name())
	}

	return false, nil
}

// Single-step task example
type SetSleepPeriodTask struct {
	state State
}

func (t *SetSleepPeriodTask) Name() string {
	return "SetSleepPeriod"
}

func (t *SetSleepPeriodTask) Start() error {
	t.state = "Pending"
	// send command to device
	return nil
}

func (t *SetSleepPeriodTask) HandleEvent(event Event) TaskResult {
	switch t.state {
	case "Pending":
		switch event.(type) {
		case DeviceAck:
			t.state = "Done"
			return TaskSucceeded
		case DeviceReject, Timeout:
			return TaskFailed
		}
	}
	return TaskRunning
}
