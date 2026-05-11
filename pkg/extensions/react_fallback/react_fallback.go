package react_fallback

import (
	"context"
	"fmt"
	"strings"

	"github.com/badlogic/pi-mono/pkg/agent"
	"github.com/badlogic/pi-mono/pkg/ai"
)

// ReActFallbackPlugin provides a fallback reasoning loop when direct tool calling fails or models hallucinate tool names.
type ReActFallbackPlugin struct {
	Enabled bool
}

// NewReActFallbackPlugin initializes the plugin
func NewReActFallbackPlugin() *ReActFallbackPlugin {
	return &ReActFallbackPlugin{Enabled: false}
}

// InterceptAfterToolCall evaluates if a tool call failed due to hallucination and prompts a ReAct reasoning step.
func (p *ReActFallbackPlugin) InterceptAfterToolCall(ctx context.Context, callCtx agent.AfterToolCallContext) (*agent.AfterToolCallResult, error) {
	if !p.Enabled {
		return nil, nil
	}

	// Simple heuristic: If tool failed or was unrecognized, inject a specific ReAct thought directive.
	if callCtx.IsError || strings.Contains(fmt.Sprintf("%v", callCtx.Result), "Not implemented") {
		overrideText := "I hallucinated a tool or encountered an error. I must now think step-by-step using ReAct format (Thought: ... Action: ... Observation: ...) to resolve this."
		return &agent.AfterToolCallResult{
			Content: []ai.Content{
				ai.TextContent{Text: overrideText},
			},
		}, nil
	}

	return nil, nil
}
