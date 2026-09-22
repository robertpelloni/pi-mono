package ai

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "update golden files")

func TestValidatePath(t *testing.T) {
	cwd, _ := os.Getwd()
	tempDir := os.TempDir()

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{"Relative path inside root", "go.mod", false},
		{"Absolute path inside root", filepath.Join(cwd, "go.mod"), false},
		{"Relative escape attempt", "../go.mod", true},
		{"Absolute escape attempt", "/etc/passwd", true},
		{"Prefix overlap attempt", cwd + "-secret/file", true},
		{"Temp dir allowed", filepath.Join(tempDir, "test.txt"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := validatePath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("validatePath() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestValidateURL_Golden validates the URL edge-case router security using deterministic golden files.
func TestValidateURL_Golden(t *testing.T) {
	matrix := []string{
		"",
		"example.com",
		"//example.com/path",
		"file:///etc/passwd",
		"ftp://example.com/file",
		"http://example.com",
		"https://example.com/path?q=123",
		"javascript:alert(1)",
		"data:text/html,<h1>test</h1>",
	}

	var output bytes.Buffer
	for _, rawURL := range matrix {
		safeURL, err := validateURL(rawURL)
		if err != nil {
			output.WriteString(fmt.Sprintf("URL: %q -> ERROR: %s\n", rawURL, err.Error()))
		} else {
			output.WriteString(fmt.Sprintf("URL: %q -> OK: %s\n", rawURL, safeURL))
		}
	}

	goldenFile := filepath.Join("testdata", "url_validation.golden")

	if *update {
		if err := os.WriteFile(goldenFile, output.Bytes(), 0644); err != nil {
			t.Fatalf("Failed to update golden file: %v", err)
		}
	}

	expected, err := os.ReadFile(goldenFile)
	if err != nil {
		t.Fatalf("Failed reading golden file (run with -update to generate): %v", err)
	}

	if !bytes.Equal(output.Bytes(), expected) {
		t.Errorf("URL validation mismatch.\nExpected:\n%s\nGot:\n%s", string(expected), output.String())
	}
}

// FuzzValidateURL ensures that the URL validator never panics on malformed inputs.
func FuzzValidateURL(f *testing.F) {
	// Add seed corpus for fuzzing
	seeds := []string{
		"http://example.com",
		"https://google.com/search?q=test",
		"file:///etc/passwd",
		"javascript:alert(1)",
		"ftp://malicious.com",
		"http://127.0.0.1:8080",
		"//relative.com",
		"",
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, rawURL string) {
		// We only care that it doesn't panic.
		_, _ = validateURL(rawURL)
	})
}

// FuzzValidatePath ensures the boundary enforcement logic never panics on malformed paths.
func FuzzValidatePath(f *testing.F) {
	// Add seed corpus for fuzzing
	seeds := []string{
		".",
		"..",
		"../..",
		"/etc/passwd",
		"C:\\Windows\\System32",
		"../../tmp/test",
		"./relative/path/file.txt",
		"",
		"\x00\x00malicious",
		strings.Repeat("../", 100),
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, path string) {
		// We only care that it doesn't panic.
		_, _ = validatePath(path)
	})
}
