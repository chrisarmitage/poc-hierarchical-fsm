package tasks

import "github.com/chrisarmitage/poc-hierarchical-fsm/internal/events"

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
	HandleEvent(event events.Event) TaskResult
}


// placeholder for enums
type State string
