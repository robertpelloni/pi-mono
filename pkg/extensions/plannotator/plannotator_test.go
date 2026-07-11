package plannotator

import (
	"context"
	"strings"
	"testing"

	"github.com/badlogic/pi-mono/pkg/agent"
	"github.com/badlogic/pi-mono/pkg/ai"
)

func TestPlannotatorPlugin(t *testing.T) {
	// Test Disabled state
	pluginDisabled := NewPlannotatorPlugin()
	toolsDisabled := pluginDisabled.AddTools([]agent.AgentTool{})
	if len(toolsDisabled) != 0 {
		t.Errorf("expected 0 tools when disabled, got %d", len(toolsDisabled))
	}

	// Test Enabled state
	pluginEnabled := NewPlannotatorPlugin()
	pluginEnabled.Enabled = true
	toolsEnabled := pluginEnabled.AddTools([]agent.AgentTool{})

	if len(toolsEnabled) != 1 {
		t.Fatalf("expected 1 tool when enabled, got %d", len(toolsEnabled))
	}

	tool := toolsEnabled[0]
	if tool.Name != "request_plan_review" {
		t.Errorf("expected tool name 'request_plan_review', got %q", tool.Name)
	}

	// Test Execute
	params := map[string]any{"plan": "Test plan 123"}
	result, err := tool.Execute(context.Background(), "call_123", params, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if !result.ApprovalRequired {
		t.Errorf("expected ApprovalRequired=true")
	}

	if result.ApprovalID != "call_123" {
		t.Errorf("expected ApprovalID='call_123', got %q", result.ApprovalID)
	}

	textData, ok := result.Content[0].(ai.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", result.Content[0])
	}
	contentStr := textData.Text
	if !strings.Contains(contentStr, "Test plan 123") {
		t.Errorf("expected plan in output, got: %s", contentStr)
	}
}
