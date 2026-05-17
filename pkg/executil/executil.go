package executil

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
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

// ExecCommandAsync executes a command with streaming output support.
// onData is called for each chunk of stdout/stderr output.
func ExecCommandAsync(command string, args []string, cwd string, options *ExecOptions, onData func(data []byte)) (*ExecResult, error) {
	var ctx context.Context
	var cancel context.CancelFunc = func() {}

	if options != nil && options.Ctx != nil {
		ctx = options.Ctx
	} else {
		ctx = context.Background()
	}

	if options != nil && options.Timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, time.Duration(options.Timeout)*time.Millisecond)
	}
	defer cancel()

	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Dir = cwd
	cmd.Env = os.Environ()

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start command: %w", err)
	}

	// Read stdout and stderr
	var stdoutBuf, stderrBuf bytes.Buffer
	done := make(chan struct{})

	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := stdoutPipe.Read(buf)
			if n > 0 {
				chunk := buf[:n]
				stdoutBuf.Write(chunk)
				if onData != nil {
					onData(chunk)
				}
			}
			if err != nil {
				break
			}
		}
		done <- struct{}{}
	}()

	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := stderrPipe.Read(buf)
			if n > 0 {
				chunk := buf[:n]
				stderrBuf.Write(chunk)
				if onData != nil {
					onData(chunk)
				}
			}
			if err != nil {
				break
			}
		}
		done <- struct{}{}
	}()

	<-done
	<-done
	cmd.Wait()

	result := &ExecResult{
		Stdout: stdoutBuf.String(),
		Stderr: stderrBuf.String(),
		Killed: ctx.Err() != nil,
	}

	if cmd.ProcessState != nil {
		result.Code = cmd.ProcessState.ExitCode()
	} else {
		result.Code = 1
	}

	return result, nil
}

// KillProcessTree kills a process and all its children (cross-platform).
func KillProcessTree(pid int) error {
	if runtime.GOOS == "windows" {
		return exec.Command("taskkill", "/F", "/T", "/PID", fmt.Sprintf("%d", pid)).Run()
	}
	// Unix: kill process group
	cmd := exec.Command("kill", "-9", fmt.Sprintf("-%d", pid))
	if err := cmd.Run(); err != nil {
		// Fallback to killing just the child
		return exec.Command("kill", "-9", fmt.Sprintf("%d", pid)).Run()
	}
	return nil
}

// GetShellConfig returns the shell and arguments for the current platform.
func GetShellConfig() (shell string, args []string) {
	if runtime.GOOS == "windows" {
		// Try Git Bash first
		gitBashPaths := []string{}
		if pf := os.Getenv("ProgramFiles"); pf != "" {
			gitBashPaths = append(gitBashPaths, filepath.Join(pf, "Git", "bin", "bash.exe"))
		}
		if pf := os.Getenv("ProgramFiles(x86)"); pf != "" {
			gitBashPaths = append(gitBashPaths, filepath.Join(pf, "Git", "bin", "bash.exe"))
		}
		for _, p := range gitBashPaths {
			if _, err := os.Stat(p); err == nil {
				return p, []string{"-c"}
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

// GetShellEnv returns environment variables with the bin directory added to PATH.
func GetShellEnv() []string {
	binDir := filepath.Join(getAgentDir(), "bin")
	env := os.Environ()

	pathKey := "PATH"
	for i, e := range env {
		if strings.HasPrefix(strings.ToUpper(e), "PATH=") {
			pathKey = e[:5]
			currentPath := e[5:]
			if !strings.Contains(currentPath, binDir) {
				env[i] = pathKey + binDir + string(os.PathListSeparator) + currentPath
			}
			break
		}
	}

	return env
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

// IsOfflineModeEnabled checks if offline mode is enabled.
func IsOfflineModeEnabled() bool {
	v := os.Getenv("PI_OFFLINE")
	return v == "1" || strings.EqualFold(v, "true") || strings.EqualFold(v, "yes")
}

// getAgentDir returns the pi agent directory.
func getAgentDir() string {
	if dir := os.Getenv("PI_AGENT_DIR"); dir != "" {
		return dir
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ".pi"
	}
	return filepath.Join(home, ".pi")
}

// FindBashOnPath searches for bash on PATH.
func FindBashOnPath() string {
	if runtime.GOOS == "windows" {
		if bash, err := exec.LookPath("bash.exe"); err == nil {
			return bash
		}
		return ""
	}
	if bash, err := exec.LookPath("bash"); err == nil {
		return bash
	}
	return ""
}
