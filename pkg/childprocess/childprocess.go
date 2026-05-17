package childprocess

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"
)

// WaitForChildProcess waits for a child process to complete, even if
// inherited stdio handles are held open by detached descendants.
// Returns the exit code or an error.
func WaitForChildProcess(cmd *exec.Cmd) (int, error) {
	if cmd.Process == nil {
		return -1, fmt.Errorf("process not started")
	}

	err := cmd.Wait()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode(), nil
		}
		return -1, err
	}

	return 0, nil
}

// SpawnDetached spawns a command as a detached process.
// Returns the process ID or an error.
func SpawnDetached(command string, args []string, cwd string, env []string) (int, error) {
	cmd := exec.Command(command, args...)
	cmd.Dir = cwd
	if env != nil {
		cmd.Env = env
	} else {
		cmd.Env = os.Environ()
	}
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil

	// Set process group on Unix

	err := cmd.Start()
	if err != nil {
		return 0, fmt.Errorf("failed to spawn detached process: %w", err)
	}

	return cmd.Process.Pid, nil
}

// IsProcessRunning checks if a process with the given PID is running.
func IsProcessRunning(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// On Unix, signal 0 checks if the process exists
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

// ExecResult holds the result of a child process execution.
type ExecResult struct {
	Stdout  string
	Stderr  string
	ExitCode int
}

// ExecSync executes a command synchronously with a timeout.
func ExecSync(command string, args []string, cwd string, timeout time.Duration) (*ExecResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Dir = cwd
	cmd.Env = os.Environ()

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	result := &ExecResult{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: 0,
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.ExitCode = 1
		}
	}

	return result, nil
}

// Ensure fmt is used
var _ = fmt.Sprintf
