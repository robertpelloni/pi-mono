package ai

import (
	"context"
	"os"
	"strings"
	"testing"
)

// TestParityMatrix_SafePathEnforcement validates that Claude Code, Gemini, and other assimilated
// handlers properly route requests and universally enforce directory traversal boundaries.
func TestParityMatrix_SafePathEnforcement(t *testing.T) {
	// Setup an isolated temporary directory to act as the test workspace root.
	originalWd, _ := os.Getwd()
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)
	defer os.Chdir(originalWd)

	// Create a safe dummy file for overlapping edit tests.
	testFile := "target.txt"
	os.WriteFile(testFile, []byte("line1\nline2\nline3"), 0644)

	reg := NewRegistry()
	h := NewHarness(reg)
	ctx := context.Background()

	matrix := []struct {
		name           string
		tool           string
		args           map[string]interface{}
		expectErrorStr string
		expectSuccess  bool
	}{
		// Claude Code handlers
		{
			name: "Claude Code - Read valid file",
			tool: "read_file",
			args: map[string]interface{}{"file_path": testFile},
			expectSuccess: true,
		},
		{
			name: "Claude Code - Read directory traversal attempt",
			tool: "read_file",
			args: map[string]interface{}{"file_path": "../../../etc/passwd"},
			expectErrorStr: "security violation",
		},
		{
			name: "Claude Code - Write file traversal attempt",
			tool: "write_file",
			args: map[string]interface{}{
				"file_path": "../malicious.sh",
				"content":   "echo hacked",
			},
			expectErrorStr: "security violation",
		},

		// Amp Code handlers
		{
			name: "Amp - Write file valid",
			tool: "Amp_WriteFile",
			args: map[string]interface{}{
				"path":    "amp_output.txt",
				"content": "amp content",
			},
			expectSuccess: true,
		},
		{
			name: "Amp - Write file traversal attempt",
			tool: "Amp_WriteFile",
			args: map[string]interface{}{
				"path":    "../amp_malicious.sh",
				"content": "echo hacked",
			},
			expectErrorStr: "security violation",
		},

		// Aider / General multi-file handlers
		{
			name: "Aider - Replace Lines traversal attempt",
			tool: "replace_lines",
			args: map[string]interface{}{
				"file_path":   "../outside.go",
				"start_line":  float64(1),
				"end_line":    float64(2),
				"replacement": "bad code",
			},
			expectErrorStr: "security violation",
		},
		{
			name: "Aider - Replace Lines valid (overlap check)",
			tool: "replace_lines",
			args: map[string]interface{}{
				"file_path":   testFile,
				"start_line":  float64(1),
				"end_line":    float64(2),
				"replacement": "replaced1\nreplaced2",
			},
			expectSuccess: true,
		},

		// Hermes handlers
		{
			name: "Hermes - Write to File traversal attempt",
			tool: "write_to_file",
			args: map[string]interface{}{
				"path":    "../../system_file.txt",
				"content": "corrupt",
			},
			expectErrorStr: "security violation",
		},
	}

	for _, tc := range matrix {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := h.ExecuteTool(ctx, tc.tool, tc.args)

			if tc.expectErrorStr != "" {
				// Harness tools sometimes return the error in the string response, sometimes in `err`.
				hasErrStr := (err != nil && strings.Contains(err.Error(), tc.expectErrorStr)) ||
				             strings.Contains(resp, tc.expectErrorStr)

				if !hasErrStr {
					t.Errorf("Expected error containing '%s', got resp: '%s', err: '%v'", tc.expectErrorStr, resp, err)
				}
			} else if tc.expectSuccess {
				if err != nil {
					t.Errorf("Expected success, got err: %v", err)
				}
				if strings.Contains(resp, "Error") || strings.Contains(resp, "security violation") {
					t.Errorf("Expected success, got error string in response: %s", resp)
				}
			}
		})
	}
}
