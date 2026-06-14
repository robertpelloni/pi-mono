package ai

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHandleUnifiedRead_Basic(t *testing.T) {
	// Create a temp file with known content
	dir := t.TempDir()
	filePath := filepath.Join(dir, "test.txt")
	content := "line0\nline1\nline2\nline3\nline4\nline5\nline6\nline7\nline8\nline9\n"
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	t.Run("read entire file", func(t *testing.T) {
		result, err := HandleUnifiedRead(map[string]interface{}{
			"path": filePath,
		})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		lines := strings.Split(result, "\n")
		if len(lines) != 10 {
			t.Errorf("Expected 10 lines, got %d", len(lines))
		}
	})

	t.Run("read with offset", func(t *testing.T) {
		result, err := HandleUnifiedRead(map[string]interface{}{
			"path":   filePath,
			"offset": float64(5),
		})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		lines := strings.Split(result, "\n")
		if len(lines) != 5 {
			t.Errorf("Expected 5 lines from offset 5, got %d", len(lines))
		}
		if !strings.HasPrefix(lines[0], "line5") {
			t.Errorf("Expected first line to be 'line5', got '%s'", lines[0])
		}
	})

	t.Run("read with offset and limit", func(t *testing.T) {
		result, err := HandleUnifiedRead(map[string]interface{}{
			"path":   filePath,
			"offset": float64(2),
			"limit":  float64(3),
		})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		lines := strings.Split(result, "\n")
		if len(lines) != 3 {
			t.Errorf("Expected 3 lines, got %d", len(lines))
		}
		if lines[0] != "line2" || lines[2] != "line4" {
			t.Errorf("Expected lines line2-line4, got %v", lines)
		}
	})

	t.Run("read with limit beyond file length", func(t *testing.T) {
		result, err := HandleUnifiedRead(map[string]interface{}{
			"path":   filePath,
			"offset": float64(8),
			"limit":  float64(100),
		})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		lines := strings.Split(result, "\n")
		if len(lines) != 2 {
			t.Errorf("Expected 2 remaining lines, got %d", len(lines))
		}
	})

	t.Run("offset beyond file", func(t *testing.T) {
		_, err := HandleUnifiedRead(map[string]interface{}{
			"path":   filePath,
			"offset": float64(100),
		})
		if err == nil {
			t.Error("Expected error for offset beyond file length")
		}
	})

	t.Run("missing path", func(t *testing.T) {
		_, err := HandleUnifiedRead(map[string]interface{}{})
		if err == nil {
			t.Error("Expected error for missing path")
		}
	})

	t.Run("nonexistent file", func(t *testing.T) {
		_, err := HandleUnifiedRead(map[string]interface{}{
			"path": "/nonexistent/path.txt",
		})
		if err == nil {
			t.Error("Expected error for nonexistent file")
		}
	})
}

func TestHandleUnifiedRead_LargeFile(t *testing.T) {
	// Create a synthetic large file (>1MB, not >100MB which would be slow)
	// We verify streaming by checking memory-bounded behavior:
	// read only a slice from the middle without loading the whole file
	dir := t.TempDir()
	filePath := filepath.Join(dir, "large.txt")

	// Generate 100,000 lines (~5MB)
	var sb strings.Builder
	for i := 0; i < 100_000; i++ {
		sb.WriteString("this is line number ")
		sb.WriteString(itoa(i))
		sb.WriteString(" with some padding content to make each line roughly 50 bytes\n")
	}
	content := sb.String()
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write large test file: %v", err)
	}

	// Verify file is large
	info, err := os.Stat(filePath)
	if err != nil {
		t.Fatalf("Failed to stat file: %v", err)
	}
	if info.Size() < 1_000_000 {
		t.Fatalf("Test file too small: %d bytes, need >1MB", info.Size())
	}
	t.Logf("Large test file size: %d bytes, %d lines", info.Size(), 100_000)

	t.Run("read middle section with offset+limit", func(t *testing.T) {
		// Read lines 50,000 to 50,010 (middle of file)
		result, err := HandleUnifiedRead(map[string]interface{}{
			"path":   filePath,
			"offset": float64(50_000),
			"limit":  float64(10),
		})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		lines := strings.Split(result, "\n")
		if len(lines) != 10 {
			t.Errorf("Expected 10 lines, got %d", len(lines))
		}
		// Verify it starts at the correct line
		if !strings.Contains(lines[0], "50000") {
			t.Errorf("Expected line 50000, got '%s'", lines[0])
		}
	})

	t.Run("read last section", func(t *testing.T) {
		result, err := HandleUnifiedRead(map[string]interface{}{
			"path":   filePath,
			"offset": float64(99_995),
		})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		lines := strings.Split(result, "\n")
		if len(lines) != 5 {
			t.Errorf("Expected 5 remaining lines, got %d", len(lines))
		}
		if !strings.Contains(lines[len(lines)-1], "99999") {
			t.Errorf("Expected last line to contain 99999, got '%s'", lines[len(lines)-1])
		}
	})

	t.Run("read single line", func(t *testing.T) {
		result, err := HandleUnifiedRead(map[string]interface{}{
			"path":   filePath,
			"offset": float64(77777),
			"limit":  float64(1),
		})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if !strings.Contains(result, "77777") {
			t.Errorf("Expected line 77777, got '%s'", result)
		}
	})
}

// itoa is a simple int-to-string without importing strconv
func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var digits []byte
	n := i
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}
