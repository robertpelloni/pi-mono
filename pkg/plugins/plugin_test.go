package plugins

import (
	"context"
	"strings"
	"testing"
)

func TestPluginManager_DateTimePlugin(t *testing.T) {
	manager := NewManager()
	ctx := context.Background()

	// Verify the mocked tool was registered
	toolName := "get_current_time"
	args := map[string]interface{}{}

	result, err := manager.HandleToolCall(ctx, toolName, args)
	if err != nil {
		t.Fatalf("expected successful execution of %s, got error: %v", toolName, err)
	}

	if !strings.Contains(result, "The current system date and time is") {
		t.Errorf("expected result to contain date string, got: %s", result)
	}
}

func TestPluginManager_UnknownTool(t *testing.T) {
	manager := NewManager()
	ctx := context.Background()

	_, err := manager.HandleToolCall(ctx, "non_existent_tool", map[string]interface{}{})
	if err == nil {
		t.Fatal("expected error when calling unknown tool, got nil")
	}

	expectedPrefix := "plugin manager: tool non_existent_tool not found"
	if !strings.HasPrefix(err.Error(), expectedPrefix) {
		t.Errorf("expected error to start with %q, got %q", expectedPrefix, err.Error())
	}
}
