package cleanroomtools

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHandleHermesWriteFile(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "cleanroom_test")
	defer os.RemoveAll(tmpDir)

	filePath := filepath.Join(tmpDir, "test.txt")
	result := HandleHermesWriteFile(map[string]interface{}{
		"file_path": filePath,
		"content":   "Hello world",
	})
	if result != "File written successfully" {
		t.Errorf("Expected success, got %s", result)
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("File should exist: %v", err)
	}
	if string(data) != "Hello world" {
		t.Errorf("Expected 'Hello world', got %s", string(data))
	}
}

func TestHandleHermesPatch(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "cleanroom_test")
	defer os.RemoveAll(tmpDir)

	filePath := filepath.Join(tmpDir, "patch.txt")
	os.WriteFile(filePath, []byte("Hello old world"), 0644)

	result := HandleHermesPatch(map[string]interface{}{
		"file_path": filePath,
		"find":      "old",
		"replace":   "new",
	})
	if result != "Patch applied successfully" {
		t.Errorf("Expected success, got %s", result)
	}

	data, _ := os.ReadFile(filePath)
	if string(data) != "Hello new world" {
		t.Errorf("Expected 'Hello new world', got %s", string(data))
	}
}

func TestHandleHermesPatch_NotFound(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "cleanroom_test")
	defer os.RemoveAll(tmpDir)

	filePath := filepath.Join(tmpDir, "patch2.txt")
	os.WriteFile(filePath, []byte("Hello world"), 0644)

	result := HandleHermesPatch(map[string]interface{}{
		"file_path": filePath,
		"find":      "nonexistent",
		"replace":   "replacement",
	})
	if result != "Error: Target string not found in file." {
		t.Errorf("Expected not found error, got %s", result)
	}
}

func TestHandleHermesMemory(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "cleanroom_test")
	defer os.RemoveAll(tmpDir)

	// Change to tmp dir so .pi_memory is created there
	oldDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldDir)

	result := HandleHermesMemory(map[string]interface{}{
		"key":   "test_key",
		"value": "test_value",
	})
	if result != "Memory saved successfully for key: test_key" {
		t.Errorf("Expected success, got %s", result)
	}

	data, err := os.ReadFile(filepath.Join(tmpDir, ".pi_memory", "test_key.txt"))
	if err != nil {
		t.Fatalf("Memory file should exist: %v", err)
	}
	if string(data) != "test_value" {
		t.Errorf("Expected 'test_value', got %s", string(data))
	}
}

func TestHandleClineAskFollowup(t *testing.T) {
	result := HandleClineAskFollowup(map[string]interface{}{
		"question": "Do you want to continue?",
	})
	if result != "[Follow-up Question Sent to User]: Do you want to continue?" {
		t.Errorf("Unexpected result: %s", result)
	}
}

func TestHandleClineBrowserAction(t *testing.T) {
	tests := []struct {
		action   string
		expected string
	}{
		{"launch", "Browser launched"},
		{"click", "Clicked at coordinate"},
		{"type", "Typed text"},
		{"close", "Browser closed"},
		{"scroll_down", "Scrolled browser"},
		{"unknown", "Unknown browser action"},
	}

	for _, tt := range tests {
		result := HandleClineBrowserAction(map[string]interface{}{
			"action": tt.action,
		})
		if !contains(result, tt.expected) {
			t.Errorf("Action %s: expected to contain %q, got %q", tt.action, tt.expected, result)
		}
	}
}

func TestHandleOpenInterpreterComputerUse(t *testing.T) {
	result := HandleOpenInterpreterComputerUse(map[string]interface{}{
		"action": "screenshot",
	})
	if !contains(result, "screenshot") {
		t.Errorf("Expected action name in result, got %q", result)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstr(s, substr))
}

func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
