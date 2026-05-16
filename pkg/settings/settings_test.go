package settings

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestCreateEmpty(t *testing.T) {
	cwd := t.TempDir()
	agentDir := t.TempDir()

	sm := Create(cwd, agentDir)
	if sm == nil {
		t.Fatal("settings manager should not be nil")
	}
}

func TestDefaultProvider(t *testing.T) {
	cwd := t.TempDir()
	agentDir := t.TempDir()

	// Write global settings
	settings := Settings{DefaultProvider: "anthropic"}
	data, _ := json.Marshal(settings)
	os.WriteFile(filepath.Join(agentDir, "settings.json"), data, 0644)

	sm := Create(cwd, agentDir)
	if sm.GetDefaultProvider() != "anthropic" {
		t.Errorf("expected 'anthropic', got '%s'", sm.GetDefaultProvider())
	}
}

func TestDefaultModel(t *testing.T) {
	cwd := t.TempDir()
	agentDir := t.TempDir()

	settings := Settings{DefaultModel: "claude-sonnet-4-20250514"}
	data, _ := json.Marshal(settings)
	os.WriteFile(filepath.Join(agentDir, "settings.json"), data, 0644)

	sm := Create(cwd, agentDir)
	if sm.GetDefaultModel() != "claude-sonnet-4-20250514" {
		t.Errorf("expected 'claude-sonnet-4-20250514', got '%s'", sm.GetDefaultModel())
	}
}

func TestProjectOverridesGlobal(t *testing.T) {
	cwd := t.TempDir()
	agentDir := t.TempDir()

	// Global: provider=openai
	globalSettings := Settings{DefaultProvider: "openai"}
	data, _ := json.Marshal(globalSettings)
	os.WriteFile(filepath.Join(agentDir, "settings.json"), data, 0644)

	// Project: provider=anthropic
	os.MkdirAll(filepath.Join(cwd, ".pi"), 0755)
	projectSettings := Settings{DefaultProvider: "anthropic"}
	data, _ = json.Marshal(projectSettings)
	os.WriteFile(filepath.Join(cwd, ".pi", "settings.json"), data, 0644)

	sm := Create(cwd, agentDir)
	if sm.GetDefaultProvider() != "anthropic" {
		t.Errorf("project should override global: expected 'anthropic', got '%s'", sm.GetDefaultProvider())
	}
}

func TestSetDefaultProvider(t *testing.T) {
	agentDir := t.TempDir()
	cwd := t.TempDir()

	sm := Create(cwd, agentDir)
	sm.SetDefaultProvider("google")

	if sm.GetDefaultProvider() != "google" {
		t.Errorf("expected 'google', got '%s'", sm.GetDefaultProvider())
	}

	// Verify persistence
	sm2 := Create(cwd, agentDir)
	if sm2.GetDefaultProvider() != "google" {
		t.Errorf("persisted setting should be 'google', got '%s'", sm2.GetDefaultProvider())
	}
}

func TestGetTheme(t *testing.T) {
	cwd := t.TempDir()
	agentDir := t.TempDir()

	sm := Create(cwd, agentDir)
	if sm.GetTheme() != "default" {
		t.Errorf("expected default theme, got '%s'", sm.GetTheme())
	}

	sm.SetTheme("monokai")
	if sm.GetTheme() != "monokai" {
		t.Errorf("expected 'monokai', got '%s'", sm.GetTheme())
	}
}

func TestGetQuietStartup(t *testing.T) {
	cwd := t.TempDir()
	agentDir := t.TempDir()

	sm := Create(cwd, agentDir)
	if sm.GetQuietStartup() {
		t.Error("quiet startup should default to false")
	}
}

func TestGetImageAutoResize(t *testing.T) {
	cwd := t.TempDir()
	agentDir := t.TempDir()

	sm := Create(cwd, agentDir)
	if !sm.GetImageAutoResize() {
		t.Error("image auto resize should default to true")
	}
}

func TestCustomSettings(t *testing.T) {
	cwd := t.TempDir()
	agentDir := t.TempDir()

	sm := Create(cwd, agentDir)
	sm.SetCustom("foo", "bar")

	val, ok := sm.GetCustom("foo")
	if !ok {
		t.Error("custom setting should exist")
	}
	if val != "bar" {
		t.Errorf("expected 'bar', got '%v'", val)
	}
}

func TestDrainErrors(t *testing.T) {
	cwd := t.TempDir()
	agentDir := t.TempDir()

	// Write invalid JSON
	os.WriteFile(filepath.Join(agentDir, "settings.json"), []byte("invalid json"), 0644)

	sm := Create(cwd, agentDir)
	errs := sm.DrainErrors()

	if len(errs) == 0 {
		t.Error("invalid JSON should produce errors")
	}

	// Second drain should be empty
	errs2 := sm.DrainErrors()
	if len(errs2) != 0 {
		t.Error("drained errors should be empty on second call")
	}
}

func TestInitAgentDir(t *testing.T) {
	// Can't easily test this since it uses home dir, but we can test it doesn't error
	dir, err := InitAgentDir()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if dir == "" {
		t.Error("agent dir should not be empty")
	}
}
