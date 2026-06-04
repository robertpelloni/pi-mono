package ai

import (
	"context"
	"strings"
	"testing"
)

func TestHarness_ExecuteTool(t *testing.T) {
	reg := &Registry{}
	h := NewHarness(reg)

	t.Run("Tabby Completion - No Models", func(t *testing.T) {
		args := map[string]interface{}{
			"segments": map[string]interface{}{
				"prefix": "func main() {",
				"suffix": "}",
			},
		}
		// Since we have no models registered in the global modelsRegistry (which GetDefaultModel uses),
		// this should return a result string containing the error.
		ClearModelsRegistry()
		resp, err := h.ExecuteTool(context.Background(), "tabby_completion", args)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !strings.Contains(resp, "Error") {
			t.Errorf("expected error message in response, got: %s", resp)
		}
	})

	t.Run("Warp Action", func(t *testing.T) {
		args := map[string]interface{}{
			"type": "RequestCommandOutput",
			"params": map[string]interface{}{
				"command": "echo 'hello'",
			},
		}
		resp, err := h.ExecuteTool(context.Background(), "warp_action", args)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if resp == "" {
			t.Errorf("expected non-empty response")
		}
	})

	t.Run("Unknown Tool", func(t *testing.T) {
		_, err := h.ExecuteTool(context.Background(), "unknown_tool", nil)
		if err == nil {
			t.Errorf("expected error for unknown tool, got nil")
		}
	})
}
