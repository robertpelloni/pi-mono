package pathsutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetAgentDir(t *testing.T) {
	dir := GetAgentDir()
	if dir == "" {
		t.Error("Expected non-empty agent dir")
	}
}

func TestGetAgentDir_EnvOverride(t *testing.T) {
	os.Setenv("PI_AGENT_DIR", "/custom/dir")
	defer os.Unsetenv("PI_AGENT_DIR")
	dir := GetAgentDir()
	if dir != "/custom/dir" {
		t.Errorf("Expected /custom/dir, got %q", dir)
	}
}

func TestGetSessionsDir(t *testing.T) {
	dir := GetSessionsDir()
	if dir == "" {
		t.Error("Expected non-empty sessions dir")
	}
}

func TestGetDefaultSessionDir(t *testing.T) {
	dir := GetDefaultSessionDir("/cwd", "/agent")
	expected := filepath.Join("/agent", "sessions")
	if dir != expected {
		t.Errorf("Expected %q, got %q", expected, dir)
	}
}

func TestGetDocsPath(t *testing.T) {
	path := GetDocsPath()
	if path == "" {
		t.Error("Expected non-empty docs path")
	}
}

func TestGetBinDir(t *testing.T) {
	dir := GetBinDir()
	if dir == "" {
		t.Error("Expected non-empty bin dir")
	}
}

func TestGetSettingsPath(t *testing.T) {
	path := GetSettingsPath()
	if path == "" {
		t.Error("Expected non-empty settings path")
	}
}

func TestIsLocalPath(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"https://example.com", false},
		{"http://example.com", false},
		{"/home/user/file.txt", true},
		{"./relative/path", true},
		{"file.txt", true},
	}
	for _, tt := range tests {
		result := IsLocalPath(tt.path)
		if result != tt.expected {
			t.Errorf("IsLocalPath(%q) = %v, want %v", tt.path, result, tt.expected)
		}
	}
}

func TestExpandPath_Home(t *testing.T) {
	result := ExpandPath("~")
	home, _ := os.UserHomeDir()
	if result != home {
		t.Errorf("Expected %q, got %q", home, result)
	}
}

func TestExpandPath_HomeSubdir(t *testing.T) {
	result := ExpandPath("~/Documents")
	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, "Documents")
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestExpandPath_Absolute(t *testing.T) {
	result := ExpandPath("/absolute/path")
	if result != "/absolute/path" {
		t.Errorf("Expected /absolute/path, got %q", result)
	}
}

func TestFindExecutable_Existing(t *testing.T) {
	path := FindExecutable("echo")
	if path == "" {
		t.Error("Expected to find echo on PATH")
	}
}

func TestFindExecutable_NonExistent(t *testing.T) {
	path := FindExecutable("nonexistent_binary_xyz123")
	if path != "" {
		t.Errorf("Expected empty path for non-existent binary, got %q", path)
	}
}
