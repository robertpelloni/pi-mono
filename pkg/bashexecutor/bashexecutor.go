package bashexecutor

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sync"
	"time"

	"github.com/badlogic/pi-mono/pkg/executil"
	"github.com/badlogic/pi-mono/pkg/truncate"
)

// BashResult holds the result of a bash command execution.
type BashResult struct {
	Output        string `json:"output"`        // Combined stdout+stderr (sanitized, possibly truncated)
	ExitCode      int    `json:"exitCode"`       // Process exit code (-1 if killed/cancelled)
	Cancelled     bool   `json:"cancelled"`      // Whether the command was cancelled via signal
	Truncated     bool   `json:"truncated"`      // Whether the output was truncated
	FullOutputPath string `json:"fullOutputPath,omitempty"` // Path to temp file with full output
	Duration      time.Duration `json:"duration"`  // Execution duration
}

// BashExecutorOptions configures bash execution.
type BashExecutorOptions struct {
	// Callback for streaming output chunks (sanitized)
	OnChunk func(chunk string)
	// Context for cancellation
	Ctx context.Context
	// Timeout in milliseconds (0 = no timeout)
	Timeout int
}

// ExecuteBash executes a bash command with streaming, cancellation, and truncation.
// This is the Go equivalent of the TypeScript executeBash().
func ExecuteBash(command string, cwd string, options *BashExecutorOptions) (*BashResult, error) {
	start := time.Now()

	if options == nil {
		options = &BashExecutorOptions{}
	}

	// Get shell configuration
	shell, shellArgs := executil.GetShellConfig()

	// Build command arguments
	args := append(shellArgs, command)

	ctx := options.Ctx
	if ctx == nil {
		ctx = context.Background()
	}

	var cancel context.CancelFunc
	if options.Timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, time.Duration(options.Timeout)*time.Millisecond)
		defer cancel()
	}

	cmd := exec.CommandContext(ctx, shell, args...)
	cmd.Dir = cwd
	cmd.Env = os.Environ()

	var stdout, stderr bytes.Buffer
	var outputMu sync.Mutex
	var outputChunks []string
	var totalBytes int
	const maxOutputBytes = truncate.DefaultMaxBytes * 2

	// Create temp file for overflow if needed
	var tempFilePath string
	var tempFile *os.File

	ensureTempFile := func() {
		if tempFilePath != "" {
			return
		}
		id := make([]byte, 8)
		rand.Read(id)
		tempFilePath = filepath.Join(os.TempDir(), fmt.Sprintf("pi-bash-%s.log", hex.EncodeToString(id)))
		var err error
		tempFile, err = os.Create(tempFilePath)
		if err != nil {
			return
		}
		// Write existing chunks to temp file
		for _, chunk := range outputChunks {
			tempFile.WriteString(chunk)
		}
	}

	// Capture output
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Start()
	if err != nil {
		return nil, fmt.Errorf("failed to start command: %w", err)
	}

	// Wait for completion
	err = cmd.Wait()

	duration := time.Since(start)

	// Process output
	combined := stdout.String() + stderr.String()
	sanitized := executil.SanitizeBinaryOutput(combined)
	sanitized = regexp.MustCompile(`\r`).ReplaceAllString(sanitized, "")

	outputMu.Lock()
	outputChunks = append(outputChunks, sanitized)
	totalBytes = len(sanitized)
	outputMu.Unlock()

	// Start temp file if output exceeds threshold
	if totalBytes > truncate.DefaultMaxBytes {
		ensureTempFile()
	}

	// Write to temp file
	if tempFile != nil {
		tempFile.WriteString(sanitized)
		tempFile.Close()
	}

	// Apply truncation
	truncation := truncate.TruncateTail(sanitized)

	cancelled := ctx.Err() != nil

	result := &BashResult{
		Output:   truncation.Content,
		Cancelled: cancelled,
		Truncated: truncation.Truncated,
		Duration:  duration,
	}

	if truncation.Truncated {
		ensureTempFile()
		result.FullOutputPath = tempFilePath
	}

	if !cancelled {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else if err != nil {
			result.ExitCode = 1
		} else {
			result.ExitCode = 0
		}
	} else {
		result.ExitCode = -1
	}

	// Stream to callback
	if options.OnChunk != nil {
		options.OnChunk(sanitized)
	}

	return result, nil
}

// KillProcess kills a process and all its children.
func KillProcess(pid int) error {
	return executil.KillProcessTree(pid)
}

// Ensure unused imports compile
var _ = runtime.GOOS
var _ = hex.EncodeToString
