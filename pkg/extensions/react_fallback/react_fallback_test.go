package react_fallback

import (
	"context"
	"strings"
	"testing"

	"github.com/badlogic/pi-mono/pkg/agent"
	"github.com/badlogic/pi-mono/pkg/ai"
)

func TestReActFallbackPlugin(t *testing.T) {
	// Test disabled
	pluginDisabled := NewReActFallbackPlugin()
	pluginDisabled.Enabled = false

	ctx := context.Background()
	callCtxError := agent.AfterToolCallContext{IsError: true}
	resDisabled, err := pluginDisabled.InterceptAfterToolCall(ctx, callCtxError)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if resDisabled != nil {
		t.Errorf("expected nil result when disabled")
	}

	// Test enabled - successful tool call
	pluginEnabled := NewReActFallbackPlugin()
	callCtxSuccess := agent.AfterToolCallContext{IsError: false, Result: agent.AgentToolResult{Content: []ai.Content{ai.TextContent{Text: "Success"}}}}
	resSuccess, err := pluginEnabled.InterceptAfterToolCall(ctx, callCtxSuccess)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if resSuccess != nil {
		t.Errorf("expected nil result for successful tool call")
	}

	// Test enabled - failed tool call (IsError = true)
	resError, err := pluginEnabled.InterceptAfterToolCall(ctx, callCtxError)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if resError == nil {
		t.Fatalf("expected non-nil result for failed tool call")
	}

	textData, ok := resError.Content[0].(ai.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", resError.Content[0])
	}
	contentStr := textData.Text
	if !strings.Contains(contentStr, "switch to ReAct reasoning") {
		t.Errorf("expected ReAct fallback prompt, got: %s", contentStr)
	}

	// Test enabled - failed tool call via string match
	callCtxToolNotFound := agent.AfterToolCallContext{IsError: false, Result: agent.AgentToolResult{Content: []ai.Content{ai.TextContent{Text: "tool not found: xyz"}}}}
	resNotFound, err := pluginEnabled.InterceptAfterToolCall(ctx, callCtxToolNotFound)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if resNotFound == nil {
		t.Fatalf("expected non-nil result for 'tool not found' result")
	}

	textDataNotFound, ok := resNotFound.Content[0].(ai.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", resNotFound.Content[0])
	}
	contentStrNotFound := textDataNotFound.Text
	if !strings.Contains(contentStrNotFound, "switch to ReAct reasoning") {
		t.Errorf("expected ReAct fallback prompt, got: %s", contentStrNotFound)
	}
}
