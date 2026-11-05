package ai

import (
	"context"
	"os"
	"path/filepath"
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

	t.Run("Hyper Theme Sync", func(t *testing.T) {
		args := map[string]interface{}{
			"config": `{"config": {"colors": {"black": "#000"}}}`,
		}
		resp, err := h.ExecuteTool(context.Background(), "hyper_theme_sync", args)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !strings.Contains(resp, "initialized") {
			t.Errorf("expected success message, got: %s", resp)
		}
	})

	t.Run("Community Plugin Routing", func(t *testing.T) {
		args := map[string]interface{}{}
		resp, err := h.ExecuteTool(context.Background(), "get_current_time", args)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !strings.Contains(resp, "The current system date and time is") {
			t.Errorf("expected date/time output, got: %s", resp)
		}
	})
}

func TestHarness_MixedOSC133EdgeCases(t *testing.T) {
	reg := NewRegistry()
	h := NewHarness(reg)
	ctx := context.Background()

	originalWd, _ := os.Getwd()
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)
	defer os.Chdir(originalWd)

	t.Run("Warp Action - Malformed OSC 133 Traversal", func(t *testing.T) {
		// Simulates a Warp ReadFiles payload where the LLM generated a path
		// containing both an escape sequence and a traversal attempt.
		malformedPath := "\x1b]133;A\x07../../../etc/passwd\x1b]133;B\x07"
		args := map[string]interface{}{
			"type": "ReadFiles",
			"params": map[string]interface{}{
				"path": malformedPath,
			},
		}
		resp, err := h.ExecuteTool(ctx, "warp_action", args)
		if err != nil {
			t.Errorf("unexpected execute tool error (should be handled in response): %v", err)
		}
		// The safePath validator should throw a security violation because it computes the path out-of-bounds,
		// and the response from HandleWarpAction packages this error into its JSON response.
		if !strings.Contains(resp, "security violation") {
			t.Errorf("expected security violation for malformed traversal, got: %s", resp)
		}
	})

	t.Run("Wave Action - OSC 133 Writefile Traversal", func(t *testing.T) {
		malformedPath := "../\x1b]133;C\x07secret.txt"
		args := map[string]interface{}{
			"type": "writefile",
			"params": map[string]interface{}{
				"path":    malformedPath,
				"content": "hacked",
			},
		}
		resp, err := h.ExecuteTool(ctx, "wave_action", args)
		if err != nil {
			t.Errorf("unexpected execute tool error (should be handled in response): %v", err)
		}
		if !strings.Contains(resp, "security violation") {
			t.Errorf("expected security violation for malformed traversal write, got: %s", resp)
		}
	})
}

func TestHarness_ParityRegistryRouting(t *testing.T) {
	reg := NewRegistry()
	h := NewHarness(reg)
	ctx := context.Background()

	t.Run("Claude Code Action Routing", func(t *testing.T) {
		args := map[string]interface{}{
			"action": "test_claude_action",
		}
		resp, err := h.ExecuteTool(ctx, "claude_code", args)
		if err != nil {
			t.Errorf("unexpected error routing claude_code: %v", err)
		}
		if !strings.Contains(resp, "test_claude_action") {
			t.Errorf("expected response to contain action payload, got: %s", resp)
		}
	})

	t.Run("Gemini Assist Action Routing", func(t *testing.T) {
		args := map[string]interface{}{
			"prompt": "test_gemini_prompt",
		}
		resp, err := h.ExecuteTool(ctx, "gemini_assist", args)
		if err != nil {
			t.Errorf("unexpected error routing gemini_assist: %v", err)
		}
		if !strings.Contains(resp, "test_gemini_prompt") {
			t.Errorf("expected response to contain prompt payload, got: %s", resp)
		}
	})

	t.Run("Codex CLI Action Routing", func(t *testing.T) {
		args := map[string]interface{}{
			"command": "test_codex_command",
		}
		resp, err := h.ExecuteTool(ctx, "codex_cli", args)
		if err != nil {
			t.Errorf("unexpected error routing codex_cli: %v", err)
		}
		if !strings.Contains(resp, "test_codex_command") {
			t.Errorf("expected response to contain command payload, got: %s", resp)
		}
	})

	t.Run("Claude Desktop Action Routing", func(t *testing.T) {
		args := map[string]interface{}{
			"event": "test_desktop_event",
		}
		resp, err := h.ExecuteTool(ctx, "claude_desktop", args)
		if err != nil {
			t.Errorf("unexpected error routing claude_desktop: %v", err)
		}
		if !strings.Contains(resp, "test_desktop_event") {
			t.Errorf("expected response to contain event payload, got: %s", resp)
		}
	})
}

func TestHarness_AmpAction(t *testing.T) {
	reg := NewRegistry()
	h := NewHarness(reg)
	ctx := context.Background()

	t.Run("Amp Action - Diff", func(t *testing.T) {
		args := map[string]interface{}{
			"file_path": "test.txt",
		}
		resp, err := h.ExecuteTool(ctx, "amp_diff", args)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !strings.Contains(resp, "Reviewed and staged changes for test.txt") {
			t.Errorf("unexpected response: %s", resp)
		}
	})

	t.Run("Amp Action - Review", func(t *testing.T) {
		args := map[string]interface{}{
			"diff_id": "123",
		}
		resp, err := h.ExecuteTool(ctx, "amp_review", args)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !strings.Contains(resp, "checked its own work for diff 123") {
			t.Errorf("unexpected response: %s", resp)
		}
	})

	t.Run("Factory Action - Review", func(t *testing.T) {
		args := map[string]interface{}{
			"review_type": "base_branch",
			"target":      "main",
		}
		resp, err := h.ExecuteTool(ctx, "factory_review", args)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !strings.Contains(resp, "Performed base_branch review against target: main") {
			t.Errorf("unexpected response: %s", resp)
		}
	})

	t.Run("Factory Action - Readiness Report", func(t *testing.T) {
		args := map[string]interface{}{
			"directory": "./src",
		}
		resp, err := h.ExecuteTool(ctx, "factory_readiness_report", args)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !strings.Contains(resp, "directory: ./src") {
			t.Errorf("unexpected response: %s", resp)
		}
	})
}

func TestHarness_TabbyNextEdit(t *testing.T) {
	reg := NewRegistry()
	h := NewHarness(reg)
	ctx := context.Background()

	t.Run("Tabby Next Edit - Invalid", func(t *testing.T) {
		args := map[string]interface{}{
			"filepath": "main.go",
		}
		resp, err := h.ExecuteTool(ctx, "tabby_completion", args)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !strings.Contains(resp, "Error") {
			t.Errorf("expected error message in response, got: %s", resp)
		}
	})
}

func TestHarness_WaveAction(t *testing.T) {
	reg := NewRegistry()
	h := NewHarness(reg)
	ctx := context.Background()

	tmpDir, err := os.MkdirTemp("", "wave-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(testFile, []byte("hello wave"), 0644)

	t.Run("Wave Action - ReadFile", func(t *testing.T) {
		args := map[string]interface{}{
			"type": "readfile",
			"params": map[string]interface{}{
				"path": testFile,
			},
		}
		resp, err := h.ExecuteTool(ctx, "wave_action", args)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !strings.Contains(resp, "hello wave") {
			t.Errorf("unexpected response: %s", resp)
		}
	})
}

func TestHarness_HandleUnifiedRequest(t *testing.T) {
	reg := NewRegistry()
	h := NewHarness(reg)
	ctx := context.Background()

	t.Run("Valid JSON Args", func(t *testing.T) {
		rawArgs := []byte(`{"config": "{\"colors\": {\"black\": \"#000\"}}"}`)
		resp, err := h.HandleUnifiedRequest(ctx, "hyper_theme_sync", rawArgs)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(resp, "initialized") {
			t.Errorf("expected success message, got: %s", resp)
		}
	})

	t.Run("Invalid JSON Args", func(t *testing.T) {
		rawArgs := []byte(`{"config": invalid json}`)
		_, err := h.HandleUnifiedRequest(ctx, "hyper_theme_sync", rawArgs)
		if err == nil {
			t.Fatal("expected error for invalid json, got nil")
		}
		if !strings.Contains(err.Error(), "failed to unmarshal") {
			t.Errorf("expected unmarshal error, got: %v", err)
		}
	})

	t.Run("Unknown Tool", func(t *testing.T) {
		_, err := h.ExecuteTool(ctx, "unknown_tool", map[string]interface{}{})
		if err == nil {
			t.Fatal("expected error for unknown tool, got nil")
		}
		if !strings.Contains(err.Error(), "unknown") {
			t.Errorf("expected unknown tool error, got: %v", err)
		}
	})
}
