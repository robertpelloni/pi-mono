package ai

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// validatePath ensures that the requested path is within the current working directory.
func validatePath(path string) (string, error) {
	absCwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// Reject Unix-style absolute paths on Windows (e.g., /etc/passwd)
	// On Windows, filepath.IsAbs returns false for paths starting with '/'
	if len(path) > 0 && path[0] == '/' && len(absCwd) > 1 && absCwd[1] == ':' {
		return "", fmt.Errorf("security violation: path %s is outside project root %s", path, absCwd)
	}

	// Resolve the absolute path of the target
	var absPath string
	if filepath.IsAbs(path) {
		absPath = filepath.Clean(path)
	} else {
		absPath = filepath.Join(absCwd, path)
	}

	absPath, err = filepath.Abs(absPath)
	if err != nil {
		return "", err
	}

	// Allow temporary system paths for specific tools
	tempDir := os.TempDir()
	relTemp, err := filepath.Rel(tempDir, absPath)
	if err == nil && !strings.HasPrefix(relTemp, "..") {
		return absPath, nil
	}

	rel, err := filepath.Rel(absCwd, absPath)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("security violation: path %s is outside project root %s", path, absCwd)
	}

	return absPath, nil
}
