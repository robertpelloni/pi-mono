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

	// If it's a /tmp path, allow it for certain tools
	if strings.HasPrefix(absPath, "/tmp/") {
		return absPath, nil
	}

	if !strings.HasPrefix(absPath, absCwd) {
		return "", fmt.Errorf("security violation: path %s is outside project root %s", path, absCwd)
	}

	return absPath, nil
}
