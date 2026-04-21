package ai

import (
	"testing"
)

func TestMapCleanRoomParams(t *testing.T) {
	// Test aliases for read paths
	jsonRead := []byte(`{"uri": "src/main.ts"}`)
	mapped, err := MapCleanRoomParams("read_file", jsonRead)
	if err != nil {
		t.Fatalf("Failed to map params: %v", err)
	}

	if path, ok := mapped["path"].(string); !ok || path != "src/main.ts" {
		t.Errorf("Expected path 'src/main.ts', got '%v'", path)
	}

	// Test command aliases
	jsonCmd := []byte(`{"script": "echo hello"}`)
	mappedCmd, err := MapCleanRoomParams("shell", jsonCmd)
	if err != nil {
		t.Fatalf("Failed to map params: %v", err)
	}

	if cmd, ok := mappedCmd["command"].(string); !ok || cmd != "echo hello" {
		t.Errorf("Expected command 'echo hello', got '%v'", cmd)
	}
}

func TestMapHermesCleanRoomParams(t *testing.T) {
	// Test aliases for read paths
	jsonRead := []byte(`{"file_path": "src/main.ts", "find": "foo", "replace": "bar"}`)
	mapped, err := MapHermesCleanRoomParams("patch", jsonRead)
	if err != nil {
		t.Fatalf("Failed to map params: %v", err)
	}

	if path, ok := mapped["path"].(string); !ok || path != "src/main.ts" {
		t.Errorf("Expected path 'src/main.ts', got '%v'", path)
	}
}
