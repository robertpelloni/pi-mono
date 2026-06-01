package react_fallback

import (
	"context"
	"fmt"
	"strings"

	"github.com/badlogic/pi-mono/pkg/agent"
	"github.com/badlogic/pi-mono/pkg/ai"
)

// ReActFallbackPlugin provides a fallback reasoning loop when direct tool calling fails.
type ReActFallbackPlugin struct {
	Enabled bool
	MaxSteps int
}

func NewReActFallbackPlugin() *ReActFallbackPlugin {
	return &ReActFallbackPlugin{
		Enabled: true,
		MaxSteps: 3,
	}
}

func (p *ReActFallbackPlugin) InterceptAfterToolCall(ctx context.Context, callCtx agent.AfterToolCallContext) (*agent.AfterToolCallResult, error) {
	if !p.Enabled {
		return nil, nil
	}

	// Detect if the tool failed or was unrecognized
	if callCtx.IsError || strings.Contains(fmt.Sprintf("%v", callCtx.Result), "tool not found") {
		// Inject a steering message to the agent to use ReAct format for the next turn
		directive := "I encountered a tool execution failure. I must now switch to ReAct reasoning mode to complete the task.\n" +
			"Format: \nThought: [my reasoning]\nAction: [tool_name](args)\nObservation: [result]\n..."

		return &agent.AfterToolCallResult{
			Content: []ai.Content{
				ai.TextContent{Text: directive},
			},
		}, nil
	}

	return nil, nil
}
