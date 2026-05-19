package executil

import (
	"strings"
	"testing"
)

func TestGetShellConfig(t *testing.T) {
	shell, args := GetShellConfig()
	if shell == "" {
		t.Error("Expected non-empty shell path")
	}
	_ = args
}

func TestGetShellEnv(t *testing.T) {
	env := GetShellEnv()
	if len(env) == 0 {
		t.Error("Expected non-empty environment")
	}
}

func TestSanitizeBinaryOutput(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello world", "hello world"},
		{"hello\x00world", "helloworld"},
		{"\x01\x02\x03test", "test"},
	}

	for _, tt := range tests {
		result := SanitizeBinaryOutput(tt.input)
		if result != tt.expected {
			t.Errorf("SanitizeBinaryOutput(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestIsOfflineModeEnabled(t *testing.T) {
	// Just test it doesn't panic
	_ = IsOfflineModeEnabled()
}

func TestExecCommand_Echo(t *testing.T) {
	result, err := ExecCommand("echo", []string{"hello"}, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.Stdout, "hello") {
		t.Errorf("Expected stdout to contain 'hello', got %q", result.Stdout)
	}
}

func TestExecCommand_WithCwd(t *testing.T) {
	result, err := ExecCommand("pwd", []string{}, "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	stdout := strings.TrimSpace(result.Stdout)
	if stdout != "/" && stdout != "C:\\" && !strings.HasPrefix(stdout, "/") {
		t.Errorf("Expected root directory, got %q", stdout)
	}
}

func TestExecCommand_Failing(t *testing.T) {
	result, err := ExecCommand("false", []string{}, "", nil)
	// false returns exit code 1
	if err == nil && result.Code == 0 {
		t.Error("Expected non-zero exit code for 'false'")
	}
}

func TestKillProcessTree_InvalidPid(t *testing.T) {
	// Should not panic with an invalid PID
	_ = KillProcessTree(-1)
}
