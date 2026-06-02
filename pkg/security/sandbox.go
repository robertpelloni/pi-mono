package security

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type SandboxConfig struct {
	AllowedRoot string
}

func GetSandboxConfig() SandboxConfig {
	return SandboxConfig{
		AllowedRoot: os.Getenv("PI_ALLOWED_ROOT"),
	}
}

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

func (c SandboxConfig) IsCommandSafe(command string) error {
	if c.AllowedRoot == "" {
		return nil
	}
	forbidden := []string{"../", "/etc/", "/var/", "/usr/bin/", "~/"}
	for _, f := range forbidden {
		if strings.Contains(command, f) {
			return fmt.Errorf("security violation: command contains potentially unsafe pattern '%s'", f)
		}
	}
	return nil
}
