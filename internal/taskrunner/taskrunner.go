package taskrunner

import (
	"fmt"

	"github.com/chrisarmitage/poc-hierarchical-fsm/internal/events"
	"github.com/chrisarmitage/poc-hierarchical-fsm/internal/sender"
	"github.com/chrisarmitage/poc-hierarchical-fsm/internal/tasks"
)

// Task runner. Contains multiple tasks to be executed in sequence.
type TaskRunner struct {
	tasks []tasks.Task
	index int
}

func NewTaskRunner(tasks []tasks.Task) *TaskRunner {
	return &TaskRunner{tasks: tasks}
}

func (tr *TaskRunner) Start() error {
	if len(tr.tasks) == 0 {
		return nil
	}
	fmt.Println("TaskRunner: starting")
	return tr.tasks[tr.index].Start()
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
		tr.index++
		if tr.index >= len(tr.tasks) {
			return true, nil
		}
		return false, tr.tasks[tr.index].Start()

	case tasks.TaskFailedRetryable:
		// optional: retry entire task
		return false, task.Start()

	case tasks.TaskFailedPermanent:
		return false, fmt.Errorf("task %s failed", task.Name())
	}

	return false, nil
}

func BuildTasks(s sender.DeviceCommandSender) []tasks.Task {
	return []tasks.Task{
		tasks.NewSetSleepPeriodTask(s),
		tasks.NewSetProtectedValueTask(s),
		// Add more tasks here
	}
}
