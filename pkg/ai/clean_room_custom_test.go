package ai

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCleanRoomCustom_ClaudeCodeListFiles(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "claude-code-test-*")
	defer os.RemoveAll(tmpDir)

	os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("data"), 0644)
	os.Mkdir(filepath.Join(tmpDir, "sub"), 0755)

	args := map[string]interface{}{"path": tmpDir}
	resp := handleClaudeCodeListFiles(args)
	if !strings.Contains(resp, "test.txt") || !strings.Contains(resp, "sub/") {
		t.Errorf("expected list of files, got: %s", resp)
	}
}

func TestCleanRoomCustom_ClaudeCodeEdit(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "claude-code-test-*")
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(testFile, []byte("line1\nline2\nline3"), 0644)

	args := map[string]interface{}{
		"file_path":  testFile,
		"old_string": "line2",
		"new_string": "line2 modified",
	}
	resp := handleClaudeCodeEdit(args)
	if !strings.Contains(resp, "successfully") {
		t.Errorf("expected success message, got: %s", resp)
	}

	content, _ := os.ReadFile(testFile)
	if !strings.Contains(string(content), "line2 modified") {
		t.Errorf("replacement failed, got content: %s", string(content))
	}
}

func TestCleanRoomCustom_ClaudeCodeRead(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "claude-code-test-read-*")
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(testFile, []byte("data"), 0644)

	args := map[string]interface{}{"file_path": testFile}
	resp := handleClaudeCodeRead(args)
	if !strings.Contains(resp, "data") {
		t.Errorf("expected read data, got: %s", resp)
	}
}

func TestCleanRoomCustom_ClaudeCodeBash(t *testing.T) {
	args := map[string]interface{}{"command": "echo test_bash"}
	resp := handleClaudeCodeBash(args)
	if !strings.Contains(resp, "test_bash") {
		t.Errorf("expected test_bash in output, got: %s", resp)
	}
}

func TestCleanRoomCustom_ClaudeCodeWriteFile(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "claude-code-test-write-*")
	defer os.RemoveAll(tmpDir)

	testFile := filepath.Join(tmpDir, "test.txt")

	args := map[string]interface{}{
		"file_path": testFile,
		"content":   "write_test",
	}
	resp := handleClaudeCodeWriteFile(args)
	if !strings.Contains(resp, "successfully") {
		t.Errorf("expected success message, got: %s", resp)
	}

	content, _ := os.ReadFile(testFile)
	if string(content) != "write_test" {
		t.Errorf("expected write_test, got: %s", string(content))
	}
}

func TestCleanRoomCustom_HarnessRouting(t *testing.T) {
	// Not needed as harness routing has changed. We just test handlers.
}

func TestCleanRoomCustom_AmpHandlers(t *testing.T) {
	t.Run("AmpReadFile", func(t *testing.T) {
		tmpDir, _ := os.MkdirTemp("", "amp-test-*")
		defer os.RemoveAll(tmpDir)

		testFile := filepath.Join(tmpDir, "test.txt")
		os.WriteFile(testFile, []byte("amp_data"), 0644)

		args := map[string]interface{}{"path": testFile}
		resp := handleAmpReadFile(args)
		if !strings.Contains(resp, "amp_data") {
			t.Errorf("expected read data, got: %s", resp)
		}
	})

	t.Run("AmpWriteFile", func(t *testing.T) {
		tmpDir, _ := os.MkdirTemp("", "amp-test-*")
		defer os.RemoveAll(tmpDir)

		testFile := filepath.Join(tmpDir, "test.txt")

		args := map[string]interface{}{
			"path": testFile,
			"content":  "amp_write",
		}
		resp := handleAmpWriteFile(args)
		if !strings.Contains(resp, "successfully") {
			t.Errorf("expected success message, got: %s", resp)
		}

		content, _ := os.ReadFile(testFile)
		if string(content) != "amp_write" {
			t.Errorf("expected amp_write, got: %s", string(content))
		}
	})

	t.Run("AmpRunCommand", func(t *testing.T) {
		args := map[string]interface{}{"cmd": "echo amp_bash"}
		resp := handleAmpRunCommand(args)
		if !strings.Contains(resp, "amp_bash") {
			t.Errorf("expected amp_bash in output, got: %s", resp)
		}
	})
}
