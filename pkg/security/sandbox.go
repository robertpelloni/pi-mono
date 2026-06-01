package security

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SandboxConfig holds security restrictions for tool execution.
type SandboxConfig struct {
	AllowedRoot string // If set, all file operations and commands must stay within this root.
}

// GetSandboxConfig returns a config initialized from environment variables.
func GetSandboxConfig() SandboxConfig {
	return SandboxConfig{
		AllowedRoot: os.Getenv("PI_ALLOWED_ROOT"),
	}
}

// ValidatePath checks if a path is within the allowed root directory.
func (c SandboxConfig) ValidatePath(path string) error {
	if c.AllowedRoot == "" {
		return nil
	}

	absRoot, err := filepath.Abs(c.AllowedRoot)
	if err != nil {
		return fmt.Errorf("invalid allowed root: %v", err)
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("invalid path: %v", err)
	}

	if !strings.HasPrefix(absPath, absRoot) {
		return fmt.Errorf("security violation: path %s is outside of allowed root %s", path, absRoot)
	}

	return nil
}

// IsCommandSafe performs basic heuristic checks on commands (very limited protection).
func (c SandboxConfig) IsCommandSafe(command string) error {
	if c.AllowedRoot == "" {
		return nil
	}

	// Check for obvious attempts to escape the root in the command string itself
	// (Note: This is easily bypassed by clever shell syntax, but provides a basic sanity check)
	forbidden := []string{"../", "/etc/", "/var/", "/usr/bin/", "~/"}
	for _, f := range forbidden {
		if strings.Contains(command, f) {
			return fmt.Errorf("security violation: command contains potentially unsafe pattern '%s'", f)
		}
	}

	return nil
}
