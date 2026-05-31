package agentregistry

import (
	"context"
)

// SubagentRunner defines the interface for running a subagent task.
type SubagentRunner interface {
	RunTask(ctx context.Context, task string, parentContext string) (string, error)
}

// GlobalSubagentRunner is the registered runner for delegation tasks.
var GlobalSubagentRunner SubagentRunner

// Scheduler defines the interface for scheduled tasks.
type Scheduler interface {
	CreateTask(id, schedule, command string) error
	ListTasks() []string
	RemoveTask(id string) error
}

// GlobalScheduler is the shared scheduler instance.
var GlobalScheduler Scheduler
