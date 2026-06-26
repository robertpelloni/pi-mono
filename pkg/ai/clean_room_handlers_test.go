package ai

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCleanRoomHandlers_Todo(t *testing.T) {
	// 1. Add
	args := map[string]interface{}{"action": "add", "task": "test task"}
	resp := handleHermesTodo(args)
	if !strings.Contains(resp, "Added") {
		t.Errorf("expected added message, got: %s", resp)
	}

	// 2. List
	args = map[string]interface{}{"action": "list"}
	resp = handleHermesTodo(args)
	if !strings.Contains(resp, "test task") {
		t.Errorf("expected task in list, got: %s", resp)
	}

	// 3. Clear
	args = map[string]interface{}{"action": "clear"}
	resp = handleHermesTodo(args)
	if !strings.Contains(resp, "cleared") {
		t.Errorf("expected cleared message, got: %s", resp)
	}
}

func TestCleanRoomHandlers_Memory(t *testing.T) {
	defer os.RemoveAll(".pi_memory")
	args := map[string]interface{}{"key": "test_key", "value": "test_value"}
	resp := handleHermesMemory(args)
	if !strings.Contains(resp, "success") {
		t.Errorf("expected success message, got: %s", resp)
	}
}

func TestCleanRoomHandlers_SkillManage(t *testing.T) {
	defer os.RemoveAll(".pi")
	args := map[string]interface{}{"action": "create", "name": "test_skill", "content": "test content"}
	resp := handleHermesSkillManage(args)
	if !strings.Contains(resp, "successfully") {
		t.Errorf("expected success message, got: %s", resp)
	}
}

func TestCleanRoomHandlers_AiderReplaceLines(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "aider-test-*")
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(testFile, []byte("line1\nline2\nline3"), 0644)

	args := map[string]interface{}{
		"file_path":   testFile,
		"start_line":  float64(2),
		"end_line":    float64(2),
		"replacement": "line2 modified",
	}
	resp := handleAiderReplaceLines(args)
	if !strings.Contains(resp, "successfully") {
		t.Errorf("expected success message, got: %s", resp)
	}

	content, _ := os.ReadFile(testFile)
	if !strings.Contains(string(content), "line2 modified") {
		t.Errorf("replacement failed, got content: %s", string(content))
	}
}

func TestCleanRoomHandlers_MapHermesParams(t *testing.T) {
	raw := []byte(`{"file_path": "test.go", "content": "package main"}`)
	unified, err := MapHermesCleanRoomParams("write_to_file", raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if unified["path"] != "test.go" {
		t.Errorf("expected path to be mapped, got: %v", unified["path"])
	}
}

func TestCleanRoomHandlers_AntigravityAutoClick(t *testing.T) {
	args := map[string]interface{}{"selectors": []interface{}{"button"}}
	resp := handleAntigravityAutoClick(args)
	if !strings.Contains(resp, "Antigravity") {
		t.Errorf("expected Antigravity prefix, got: %s", resp)
	}
}
