package bashtool

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/badlogic/pi-mono/pkg/agent"
	"github.com/badlogic/pi-mono/pkg/ai"
	"github.com/badlogic/pi-mono/pkg/truncate"
)

// CreateBashTool creates the bash tool with streaming output, timeout, and truncation.
func CreateBashTool(cwd string) agent.AgentTool {
	return agent.AgentTool{
		Name:        "bash",
		Label:       "bash",
		Description: fmt.Sprintf("Execute a bash command in the current working directory. Returns stdout and stderr. Output is truncated to last %d lines or %dKB (whichever is hit first). If truncated, full output is saved to a temp file. Optionally provide a timeout in seconds.", truncate.DefaultMaxLines, truncate.DefaultMaxBytes/1024),
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"command": map[string]interface{}{
					"type":        "string",
					"description": "Bash command to execute",
				},
				"timeout": map[string]interface{}{
					"type":        "number",
					"description": "Timeout in seconds (optional, no default timeout)",
				},
			},
			"required": []string{"command"},
		},
		PromptSnippet: "Execute bash commands (ls, grep, find, etc.)",
		Execute: func(ctx context.Context, toolCallId string, params map[string]any, onUpdate agent.AgentToolUpdateCallback) (agent.AgentToolResult, error) {
			command, _ := params["command"].(string)
			if command == "" {
				return agent.AgentToolResult{}, fmt.Errorf("missing command parameter")
			}

			var timeoutSec float64
			if t, ok := params["timeout"].(float64); ok && t > 0 {
				timeoutSec = t
			}

			return executeBash(ctx, command, cwd, timeoutSec, onUpdate)
		},
	}
}

// BashResult contains the output of a bash command execution.
type BashResult struct {
	ExitCode    int    `json:"exitCode"`
	Stdout      string `json:"stdout"`
	Stderr      string `json:"stderr"`
	TimedOut    bool   `json:"timedOut"`
	Duration    string `json:"duration"`
	Truncation  *truncate.TruncationResult `json:"truncation,omitempty"`
	FullOutputPath string   `json:"fullOutputPath,omitempty"`
}

// executeBash runs a bash command with streaming output and timeout support.
func executeBash(ctx context.Context, command, cwd string, timeoutSec float64, onUpdate agent.AgentToolUpdateCallback) (agent.AgentToolResult, error) {
	// Check if cwd exists
	if _, err := os.Stat(cwd); err != nil {
		return agent.AgentToolResult{}, fmt.Errorf("working directory does not exist: %s", cwd)
	}

	startTime := time.Now()

	// Set up the command
	shell, args := getShellConfig()
	allArgs := append(args, command)

	cmd := exec.CommandContext(ctx, shell, allArgs...)
	cmd.Dir = cwd
	cmd.Env = os.Environ()

	// Set up pipes for stdout and stderr
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return agent.AgentToolResult{}, fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return agent.AgentToolResult{}, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return agent.AgentToolResult{}, fmt.Errorf("failed to start command: %w", err)
	}

	// Set up timeout
	var timeoutTimer *time.Timer
	timeoutCtx, timeoutCancel := context.WithCancel(ctx)
	defer timeoutCancel()

	if timeoutSec > 0 {
		timeoutTimer = time.AfterFunc(time.Duration(timeoutSec*float64(time.Second)), func() {
			timeoutCancel()
			killProcessTree(cmd.Process)
		})
		defer timeoutTimer.Stop()
	}

	// Stream output
	var outputBuf bytes.Buffer
	var mu sync.Mutex

	// Merge stdout and stderr
	mergeReader := io.MultiReader(stdoutPipe, stderrPipe)

	// Read in a goroutine
	done := make(chan struct{})
	go func() {
		defer close(done)
		buf := make([]byte, 4096)
		for {
			n, err := mergeReader.Read(buf)
			if n > 0 {
				chunk := buf[:n]
				mu.Lock()
				outputBuf.Write(chunk)
				mu.Unlock()
			}
			if err != nil {
				break
			}
		}
	}()

	// Wait for the process to complete
	err = cmd.Wait()
	<-done // Wait for output reading to finish

	duration := time.Since(startTime)

	mu.Lock()
	output := outputBuf.String()
	mu.Unlock()

	// Determine exit code and status
	exitCode := 0
	timedOut := false
	if err != nil {
		if timeoutCtx.Err() == context.DeadlineExceeded || timeoutCtx.Err() == context.Canceled {
			timedOut = true
		}
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else if !timedOut {
			return agent.AgentToolResult{}, fmt.Errorf("command failed: %w", err)
		}
	}

	// Build result text
	var resultText string
	if timedOut {
		resultText = fmt.Sprintf("[Timeout after %s]\n%s", formatDuration(duration), output)
	} else if exitCode != 0 {
		resultText = fmt.Sprintf("[Exit code: %d]\n%s", exitCode, output)
	} else {
		resultText = output
	}

	// Truncate the output (tail truncation for bash)
	truncResult := truncate.TruncateTail(resultText)

	details := map[string]any{
		"exitCode": exitCode,
		"timedOut": timedOut,
		"duration": formatDuration(duration),
	}

	if truncResult.Truncated {
		details["truncation"] = truncResult
		// Save full output to temp file
		tmpFile := getTempFilePath()
		if err := os.WriteFile(tmpFile, []byte(resultText), 0644); err == nil {
			details["fullOutputPath"] = tmpFile
		}

		// Add truncation notice
		var warnings []string
		if path, ok := details["fullOutputPath"].(string); ok {
			warnings = append(warnings, fmt.Sprintf("Full output: %s", path))
		}
		if truncResult.TruncatedBy == "lines" {
			warnings = append(warnings, fmt.Sprintf("Truncated: showing %d of %d lines", truncResult.OutputLines, truncResult.TotalLines))
		} else {
			warnings = append(warnings, fmt.Sprintf("Truncated: %d lines shown (%s limit)", truncResult.OutputLines, truncate.FormatSize(truncResult.MaxBytes)))
		}

		resultText = truncResult.Content + "\n[" + strings.Join(warnings, ". ") + "]"
	}

	// Add duration footer
	resultText += fmt.Sprintf("\n\nTook %s", formatDuration(duration))

	return agent.AgentToolResult{
		Content: []ai.Content{
			ai.TextContent{Text: resultText},
		},
		Details: details,
		IsError: exitCode != 0 || timedOut,
	}, nil
}

// getShellConfig returns the shell and args for the current platform.
func getShellConfig() (string, []string) {
	if runtime.GOOS == "windows" {
		// Try powershell first, fall back to cmd
		if psh, _ := exec.LookPath("powershell"); psh != "" {
			return psh, []string{"-Command"}
		}
		return "cmd", []string{"/C"}
	}
	return "/bin/bash", []string{"-c"}
}

// killProcessTree kills a process and all its children.
func killProcessTree(proc *os.Process) {
	if proc == nil {
		return
	}
	// On Unix, send SIGTERM to the process group
	if runtime.GOOS != "windows" {
		// Kill the process group (negative PID)
		exec.Command("kill", fmt.Sprintf("-%d", proc.Pid)).Run()
	}
	proc.Kill()
}

// getTempFilePath generates a unique temp file path for bash output.
func getTempFilePath() string {
	return filepath.Join(os.TempDir(), fmt.Sprintf("pi-bash-%d.log", time.Now().UnixNano()))
}

// formatDuration formats a duration as a human-readable string.
func formatDuration(d time.Duration) string {
	return fmt.Sprintf("%.1fs", d.Seconds())
}
