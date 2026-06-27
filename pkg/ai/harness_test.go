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

	t.Run("Auggie Search RepoMap Optimization Context", func(t *testing.T) {
		// Mock the clean room tools registration since it happens in init() normally and may not be active in this test runner instance
		CleanRoomTools["auggie_search"] = func(args map[string]interface{}) string {
			return "mocked"
		}

		args := map[string]interface{}{
			"query": "find auth loop",
		}
		_, err := h.ExecuteTool(ctx, "auggie_search", args)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		// Confirm the _optimized_repomap_context injection took place internally
		if opt, ok := args["_optimized_repomap_context"].(bool); !ok || !opt {
			t.Errorf("expected _optimized_repomap_context to be true, got %v", args["_optimized_repomap_context"])
		}
	})
}
