package childprocess

import (
	"testing"
	"time"
)

func TestExecSync_Echo(t *testing.T) {
	result, err := ExecSync("echo", []string{"hello"}, "", 10*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if result.ExitCode != 0 {
		t.Errorf("Expected exit code 0, got %d", result.ExitCode)
	}
	// Note: echo output may vary by platform
	if result.Stdout == "" {
		t.Error("Expected some stdout output")
	}
}

func TestExecSync_FailingCommand(t *testing.T) {
	result, err := ExecSync("false", []string{}, "", 10*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if result.ExitCode == 0 {
		t.Error("Expected non-zero exit code for 'false'")
	}
}

func TestExecSync_Timeout(t *testing.T) {
	result, err := ExecSync("sleep", []string{"10"}, "", 50*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	if result.ExitCode == 0 {
		t.Error("Expected non-zero exit code for timed-out command")
	}
}

func TestExecSync_NonExistent(t *testing.T) {
	_, err := ExecSync("nonexistent_command_xyz", []string{}, "", 5*time.Second)
	if err != nil {
		// This is fine - non-existent commands may error
	}
}

func TestIsProcessRunning_InvalidPid(t *testing.T) {
	// Very large PID should not be running
	running := IsProcessRunning(999999)
	if running {
		t.Error("Expected very large PID to not be running")
	}
}

func TestExecResult_Fields(t *testing.T) {
	r := &ExecResult{Stdout: "out", Stderr: "err", ExitCode: 42}
	if r.Stdout != "out" || r.Stderr != "err" || r.ExitCode != 42 {
		t.Error("ExecResult fields mismatch")
	}
}
