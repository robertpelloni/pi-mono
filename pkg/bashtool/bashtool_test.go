package bashtool

import (
	"testing"
	"time"
)

func TestCreateBashTool(t *testing.T) {
	tool := CreateBashTool(".")
	if tool.Name != "bash" {
		t.Errorf("Expected tool name 'bash', got %q", tool.Name)
	}
}

func TestBashResult_Fields(t *testing.T) {
	result := BashResult{
		ExitCode: 0,
		Stdout:   "hello",
		Stderr:   "",
		TimedOut: false,
		Duration: "1.5s",
	}
	if result.ExitCode != 0 {
		t.Error("ExitCode mismatch")
	}
	if result.Stdout != "hello" {
		t.Error("Stdout mismatch")
	}
	if result.Duration != "1.5s" {
		t.Error("Duration mismatch")
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		d        time.Duration
		expected string
	}{
		{100 * time.Millisecond, "0.1s"},
		{1500 * time.Millisecond, "1.5s"},
		{5 * time.Second, "5.0s"},
	}
	for _, tt := range tests {
		result := formatDuration(tt.d)
		if result != tt.expected {
			t.Errorf("formatDuration(%v) = %q, want %q", tt.d, result, tt.expected)
		}
	}
}

func TestGetShellConfig(t *testing.T) {
	shell, args := getShellConfig()
	if shell == "" {
		t.Error("Expected non-empty shell")
	}
	_ = args
}

func TestGetTempFilePath(t *testing.T) {
	path := getTempFilePath()
	if path == "" {
		t.Error("Expected non-empty temp file path")
	}
}
