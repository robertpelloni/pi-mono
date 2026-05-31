package agent

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// ScheduledTask represents a background task.
type ScheduledTask struct {
	ID        string
	Schedule  string // e.g. "1m", "1h"
	Command   string
	Context   context.Context
	Cancel    context.CancelFunc
}

// TaskScheduler manages persistent background tasks.
type TaskScheduler struct {
	mu    sync.Mutex
	tasks map[string]*ScheduledTask
	agent *Agent
}

// GlobalScheduler is the shared scheduler instance.

// NewTaskScheduler creates a new scheduler.
func NewTaskScheduler(ag *Agent) *TaskScheduler {
	return &TaskScheduler{
		tasks: make(map[string]*ScheduledTask),
		agent: ag,
	}
}

// CreateTask adds a new periodic task.
func (s *TaskScheduler) CreateTask(id, schedule, command string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.tasks[id]; exists {
		return fmt.Errorf("task %s already exists", id)
	}

	duration, err := time.ParseDuration(schedule)
	if err != nil {
		return fmt.Errorf("invalid schedule format: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	task := &ScheduledTask{
		ID:       id,
		Schedule: schedule,
		Command:  command,
		Context:  ctx,
		Cancel:   cancel,
	}
	s.tasks[id] = task

	go s.runTaskLoop(task, duration)
	return nil
}

func (s *TaskScheduler) runTaskLoop(task *ScheduledTask, duration time.Duration) {
	ticker := time.NewTicker(duration)
	defer ticker.Stop()

	for {
		select {
		case <-task.Context.Done():
			return
		case <-ticker.C:
			// Execute command via agent's bash tool
			s.agent.ExecuteToolCall(task.Context, "bash", map[string]interface{}{
				"command": task.Command,
			})
		}
	}
}

// ListTasks returns all active tasks.
func (s *TaskScheduler) ListTasks() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	var list []string
	for id, t := range s.tasks {
		list = append(list, fmt.Sprintf("%s (%s): %s", id, t.Schedule, t.Command))
	}
	return list
}

// RemoveTask stops and removes a task.
func (s *TaskScheduler) RemoveTask(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	task, exists := s.tasks[id]
	if !exists {
		return fmt.Errorf("task %s not found", id)
	}
	task.Cancel()
	delete(s.tasks, id)
	return nil
}
