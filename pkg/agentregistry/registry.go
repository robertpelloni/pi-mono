package agentregistry

import (
	"context"
)

type SubagentRunner interface {
	RunTask(ctx context.Context, task string, parentContext string) (string, error)
}

var GlobalSubagentRunner SubagentRunner

type Scheduler interface {
	CreateTask(id, schedule, command string) error
	ListTasks() []string
	RemoveTask(id string) error
}

var GlobalScheduler Scheduler
