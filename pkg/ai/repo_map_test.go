package ai

import (
	"testing"
	"strings"
)

func TestHandleAiderRepoMap(t *testing.T) {
	res := handleRepomapGenerate(map[string]interface{}{})
	if !strings.Contains(res, "<repo_map>") {
		t.Errorf("Expected repo map tag, got %s", res)
	}
	if !strings.Contains(res, "clean_room_handlers.go") {
		t.Errorf("Expected clean_room_handlers.go in repo map, got %s", res)
	}
	// Check for a definition
	if !strings.Contains(res, "HandleUnifiedRead") {
		t.Errorf("Expected HandleUnifiedRead definition in repo map, got %s", res)
	}
}
