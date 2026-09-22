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

	// Prevent generic traversal out of the active project root
	rel, err := filepath.Rel(absCwd, absPath)

	// Exception: If the user specifically passed a path inside the system temporary directory,
	// ensure it's not being used to escape a workspace that also happens to be in TempDir.
	// Only allow absolute temporary paths if they don't break relative isolation rules.
	if err != nil || strings.HasPrefix(rel, "..") {
		// If it's outside the workspace, strictly check if it's a direct absolute temp path mapping
		tempDir := os.TempDir()
		relTemp, errTemp := filepath.Rel(tempDir, absPath)
		if errTemp == nil && !strings.HasPrefix(relTemp, "..") {
			// Only allow if the workspace itself isn't housed in TempDir. If it is, the traversal was malicious.
			relRootToTemp, _ := filepath.Rel(tempDir, absCwd)
			if !strings.HasPrefix(relRootToTemp, "..") {
				return "", fmt.Errorf("security violation: path %s is outside project root %s", path, absCwd)
			}
			return absPath, nil
		}
		return "", fmt.Errorf("security violation: path %s is outside project root %s", path, absCwd)
	}

	return absPath, nil
}
