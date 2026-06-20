package ai

import (
	"os"
	"path/filepath"
	"testing"
)

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
