package taskrunner

import (
	"fmt"

	"github.com/chrisarmitage/poc-hierarchical-fsm/internal/events"
	"github.com/chrisarmitage/poc-hierarchical-fsm/internal/sender"
	"github.com/chrisarmitage/poc-hierarchical-fsm/internal/tasks"
	"github.com/chrisarmitage/poc-hierarchical-fsm/internal/timeoutmanager"
)

// Task runner. Contains multiple tasks to be executed in sequence.
type TaskRunner struct {
	tasks          []tasks.Task
	index          int
	timeoutManager *timeoutmanager.TimeoutManager
}

func NewTaskRunner(tasks []tasks.Task, timeoutManager *timeoutmanager.TimeoutManager) *TaskRunner {
	return &TaskRunner{
		tasks:          tasks,
		timeoutManager: timeoutManager,
	}
}

func (tr *TaskRunner) Start() error {
	if len(tr.tasks) == 0 {
		return nil
	}
	fmt.Println("TaskRunner: starting")
	err := tr.tasks[tr.index].Start()
	if err != nil {
		return err
	}

	// Arm timeout for the first task
	tr.armTimeout()
	return nil
}

func (tr *TaskRunner) HandleEvent(event events.Event) (done bool, err error) {
	if tr.index >= len(tr.tasks) {
		return true, nil
	}

	fmt.Printf("TaskRunner: handling event %T for activetask %s\n", event, tr.tasks[tr.index].Name())

	task := tr.tasks[tr.index]
	result := task.HandleEvent(event)

	fmt.Printf("TaskRunner: task %s returned result %d\n", task.Name(), result)

	switch result {
	case tasks.TaskRunning:
		return false, nil

	case tasks.TaskSucceeded:
		// Cancel timeout for completed task
		tr.cancelTimeout()

		// Move to next task
		tr.index++
		if tr.index >= len(tr.tasks) {
			return true, nil
		}

		// Start next task and arm its timeout
		err := tr.tasks[tr.index].Start()
		if err != nil {
			return false, err
		}
		tr.armTimeout()
		return false, nil

	case tasks.TaskFailedRetryable:
		// optional: retry entire task
		return false, task.Start()

	case tasks.TaskFailedPermanent:
		return false, fmt.Errorf("task %s failed", task.Name())
	}

	return false, nil
}

func (tr *TaskRunner) armTimeout() {
	if tr.index >= len(tr.tasks) {
		return
	}

	task := tr.tasks[tr.index]
	duration := task.GetTimeoutDuration()
	tr.timeoutManager.Arm(task.Name(), duration)
}

// cancelTimeout cancels the timeout timer for the current task
func (tr *TaskRunner) cancelTimeout() {
	if tr.index >= len(tr.tasks) {
		return
	}

	task := tr.tasks[tr.index]
	tr.timeoutManager.Cancel(task.Name())
}

func BuildTasks(s sender.DeviceCommandSender) []tasks.Task {
	return []tasks.Task{
		tasks.NewSetSleepPeriodTask(s),
		tasks.NewSetProtectedValueTask(s),
		// Add more tasks here
	}
}
