package agent

import (
	"context"

	"github.com/badlogic/pi-mono/pkg/ai"
)

// AgentToolResult represents the final or partial result produced by a tool.
type AgentToolResult struct {
	Content []ai.Content `json:"content"` // Contains ai.TextContent or ai.ImageContent
	Details any          `json:"details,omitempty"`
}

// AgentToolUpdateCallback is used by tools to stream partial execution updates.
type AgentToolUpdateCallback func(partialResult AgentToolResult)

// AgentTool represents a tool definition used by the agent runtime.
type AgentTool struct {
	Name        string
	Description string
	Label       string
	Parameters  any // Usually a JSON schema map

	// PrepareArguments is an optional compatibility shim for raw tool-call arguments before schema validation.
	PrepareArguments func(args map[string]any) (map[string]any, error)

	// Execute runs the tool call.
	// It accepts a context.Context for cancellation instead of an AbortSignal.
	Execute func(ctx context.Context, toolCallId string, params map[string]any, onUpdate AgentToolUpdateCallback) (AgentToolResult, error)
}

// ToAITool converts the AgentTool to the underlying ai.Tool expected by the provider streams.
func (t AgentTool) ToAITool() ai.Tool {
	return ai.Tool{
		Name:        t.Name,
		Description: t.Description,
		Parameters:  t.Parameters,
	}
}
