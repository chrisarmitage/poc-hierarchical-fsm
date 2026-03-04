package timeoutmanager

import (
	"fmt"
	"sync"
	"time"

	"github.com/chrisarmitage/poc-hierarchical-fsm/internal/events"
)

type TimeoutManager struct {
	timers     map[string]*time.Timer
	mu         sync.Mutex
	eventsChan chan events.Event
}

func NewTimeoutManager(eventsChan chan events.Event) *TimeoutManager {
	return &TimeoutManager{
		timers:     make(map[string]*time.Timer),
		eventsChan: eventsChan,
	}
}

func (tm *TimeoutManager) Arm(taskID string, duration time.Duration) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Cancel existing timer if present
	if existingTimer, exists := tm.timers[taskID]; exists {
		existingTimer.Stop()
		delete(tm.timers, taskID)
	}

	// Create new timer
	timer := time.AfterFunc(duration, func() {
		fmt.Printf("TimeoutManager: timeout fired for task %s\n", taskID)
		tm.eventsChan <- events.Timeout{TaskID: taskID}

		tm.mu.Lock()
		delete(tm.timers, taskID)
		tm.mu.Unlock()
	})

	tm.timers[taskID] = timer
	fmt.Printf("TimeoutManager: armed timer for task %s with duration %v\n", taskID, duration)
}

func (tm *TimeoutManager) Cancel(taskID string) bool {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	timer, exists := tm.timers[taskID]
	if !exists {
		return false
	}

	timer.Stop()
	delete(tm.timers, taskID)
	fmt.Printf("TimeoutManager: cancelled timer for task %s\n", taskID)
	return true
}
func (tm *TimeoutManager) CancelAll() {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	for taskID, timer := range tm.timers {
		timer.Stop()
		fmt.Printf("TimeoutManager: cancelled timer for task %s during cleanup\n", taskID)
	}

	// Clear the map
	tm.timers = make(map[string]*time.Timer)
}
