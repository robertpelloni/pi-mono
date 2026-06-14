package cron

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Task represents a scheduled task.
type Task struct {
	ID       string
	Interval time.Duration
	Command  func(context.Context) error
	LastRun  time.Time
	NextRun  time.Time
	Label    string
	enabled  bool
}

// Scheduler manages periodic task execution.
type Scheduler struct {
	mu      sync.Mutex
	tasks   map[string]*Task
	cancel  context.CancelFunc
	ctx     context.Context
	running bool
	wg      sync.WaitGroup
}

// NewScheduler creates a new scheduler.
func NewScheduler() *Scheduler {
	return &Scheduler{
		tasks: make(map[string]*Task),
	}
}

// AddTask registers a new periodic task.
func (s *Scheduler) AddTask(t *Task) error {
	if t.Interval <= 0 {
		return fmt.Errorf("interval must be positive")
	}
	if t.Command == nil {
		return fmt.Errorf("command must not be nil")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.tasks[t.ID]; exists {
		return fmt.Errorf("task %s already exists", t.ID)
	}

	t.NextRun = time.Now().Add(t.Interval)
	t.enabled = true
	s.tasks[t.ID] = t
	return nil
}

// RemoveTask unregisters a task by ID.
func (s *Scheduler) RemoveTask(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.tasks[id]; !exists {
		return fmt.Errorf("task %s not found", id)
	}
	delete(s.tasks, id)
	return nil
}

// Start begins executing scheduled tasks.
func (s *Scheduler) Start() {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.ctx = ctx
	s.cancel = cancel
	s.running = true
	s.mu.Unlock()

	go s.runLoop()
}

// Stop cancels all scheduled tasks and waits for them to finish.
func (s *Scheduler) Stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	s.cancel()
	s.running = false
	s.mu.Unlock()
	s.wg.Wait()
}

// Tasks returns a copy of the current task metadata.
func (s *Scheduler) Tasks() []map[string]interface{} {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]map[string]interface{}, 0, len(s.tasks))
	for _, t := range s.tasks {
		result = append(result, map[string]interface{}{
			"id":          t.ID,
			"label":       t.Label,
			"interval":    t.Interval.String(),
			"lastRun":     t.LastRun,
			"nextRun":     t.NextRun,
			"enabled":     t.enabled,
		})
	}
	return result
}

func (s *Scheduler) runLoop() {
	for {
		s.mu.Lock()
		now := time.Now()
		var nextTick time.Duration = time.Hour // sleep at most 1 hour if no tasks

		for _, t := range s.tasks {
			if !t.enabled {
				continue
			}
			if now.After(t.NextRun) || now.Equal(t.NextRun) {
				// Launch task in background
				taskCtx := s.ctx
				task := t
				s.wg.Add(1)
				go func() {
					defer s.wg.Done()
					task.Command(taskCtx)
					s.mu.Lock()
					task.LastRun = time.Now()
					task.NextRun = time.Now().Add(task.Interval)
					s.mu.Unlock()
				}()
			}
			// Find minimum time until next tick
			remaining := time.Until(t.NextRun)
			if remaining < nextTick {
				nextTick = remaining
			}
		}
		s.mu.Unlock()

		// Clamp to positive duration
		if nextTick < 0 {
			nextTick = time.Millisecond
		}

		// Wait for next tick or cancellation
		select {
		case <-s.ctx.Done():
			return
		case <-time.After(nextTick):
		}
	}
}
