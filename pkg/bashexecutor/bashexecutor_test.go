package bashexecutor

import (
	"strings"
	"testing"
)

func TestExecuteBash_Echo(t *testing.T) {
	result, err := ExecuteBash("echo hello", ".", nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", result.ExitCode)
	}
	if !strings.Contains(result.Output, "hello") {
		t.Errorf("Expected output to contain 'hello', got %q", result.Output)
	}
}

func TestExecuteBash_FailingCommand(t *testing.T) {
	result, err := ExecuteBash("false", ".", nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.ExitCode == 0 {
		t.Error("Expected non-zero exit code")
	}
}

func TestExecuteBash_WithCwd(t *testing.T) {
	result, err := ExecuteBash("pwd", "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	output := strings.TrimSpace(result.Output)
	if output == "" {
		t.Error("Expected some output from pwd")
	}
}

func TestBashResult_Fields(t *testing.T) {
	r := &BashResult{ExitCode: 1, Output: "out", Cancelled: true, Truncated: false}
	if r.ExitCode != 1 || r.Output != "out" || !r.Cancelled {
		t.Error("BashResult field mismatch")
	}
}

func TestBashExecutorOptions_Fields(t *testing.T) {
	opts := &BashExecutorOptions{
		Timeout: 30000,
	}
	if opts.Timeout != 30000 {
		t.Error("Timeout mismatch")
	}
}

func TestKillProcess_InvalidPid(t *testing.T) {
	// Should not panic
	_ = KillProcess(-1)
}
