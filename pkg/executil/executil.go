package executil

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"time"
)

// ExecOptions configures command execution.
type ExecOptions struct {
	// Context to cancel the command
	Ctx context.Context
	// Timeout in milliseconds
	Timeout int
	// Working directory
	Cwd string
	// Environment variables
	Env []string
}

// ExecResult holds the output of a command execution.
type ExecResult struct {
	Stdout string `json:"stdout"`
	Stderr string `json:"stderr"`
	Code   int    `json:"code"`
	Killed bool   `json:"killed"`
}

// ExecCommand executes a shell command and returns its output.
// Supports timeout and context cancellation.
func ExecCommand(command string, args []string, cwd string, options *ExecOptions) (*ExecResult, error) {
	var ctx context.Context
	var cancel context.CancelFunc = func() {}

	if options != nil && options.Ctx != nil {
		ctx = options.Ctx
	} else {
		ctx = context.Background()
	}

	if options != nil && options.Timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, time.Duration(options.Timeout)*time.Millisecond)
	} else if ctx == context.Background() {
		ctx, cancel = context.WithCancel(ctx)
	}
	defer cancel()

	cmd := exec.CommandContext(ctx, command, args...)
	if options != nil && options.Cwd != "" {
		cmd.Dir = options.Cwd
	} else if cwd != "" {
		cmd.Dir = cwd
	}

	if options != nil && len(options.Env) > 0 {
		cmd.Env = options.Env
	} else {
		cmd.Env = os.Environ()
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Start()
	if err != nil {
		return &ExecResult{
			Stdout: stdout.String(),
			Stderr: stderr.String(),
			Code:   1,
		}, fmt.Errorf("failed to start command: %w", err)
	}

	err = cmd.Wait()

	result := &ExecResult{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
		Killed: ctx.Err() == context.DeadlineExceeded || ctx.Err() == context.Canceled,
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.Code = exitErr.ExitCode()
		} else {
			result.Code = 1
		}
	}

	return result, nil
}

// KillProcessTree kills a process and all its children (cross-platform).
func KillProcessTree(pid int) error {
	if runtime.GOOS == "windows" {
		return exec.Command("taskkill", "/F", "/T", "/PID", fmt.Sprintf("%d", pid)).Run()
	}
	// Unix: use pkill with process group
	cmd := exec.Command("kill", "-9", fmt.Sprintf("-%d", pid))
	if err := cmd.Run(); err != nil {
		// Fallback to killing just the child
		cmd := exec.Command("kill", "-9", fmt.Sprintf("%d", pid))
		return cmd.Run()
	}
	return nil
}

// GetShellConfig returns the shell and arguments for the current platform.
func GetShellConfig() (shell string, args []string) {
	if runtime.GOOS == "windows" {
		// Try Git Bash first
		gitBashPaths := []string{
			os.Getenv("ProgramFiles") + "\\Git\\bin\\bash.exe",
			os.Getenv("ProgramFiles(x86)") + "\\Git\\bin\\bash.exe",
		}
		for _, p := range gitBashPaths {
			if p != "" {
				if _, err := os.Stat(p); err == nil {
					return p, []string{"-c"}
				}
			}
		}
		// Try bash on PATH
		if bash, err := exec.LookPath("bash.exe"); err == nil {
			return bash, []string{"-c"}
		}
		// Fallback to cmd
		return "cmd", []string{"/C"}
	}

	// Unix: try /bin/bash, then sh
	if _, err := os.Stat("/bin/bash"); err == nil {
		return "/bin/bash", []string{"-c"}
	}
	shell = os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}
	return shell, []string{"-c"}
}

// SanitizeBinaryOutput removes characters that cause display issues.
func SanitizeBinaryOutput(s string) string {
	var buf bytes.Buffer
	for _, r := range s {
		switch {
		case r == 0x09, r == 0x0a, r == 0x0d: // tab, newline, carriage return
			buf.WriteRune(r)
		case r <= 0x1f: // control chars
			// skip
		case r >= 0xfff9 && r <= 0xfffb: // Unicode format characters
			// skip
		default:
			buf.WriteRune(r)
		}
	}
	return buf.String()
}
