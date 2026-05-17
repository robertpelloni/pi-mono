package pathsutil

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// GetAgentDir returns the path to the pi agent configuration directory.
func GetAgentDir() string {
	// Check env var first
	if dir := os.Getenv("PI_AGENT_DIR"); dir != "" {
		return dir
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ".pi"
	}
	return filepath.Join(home, ".pi")
}

// GetSessionsDir returns the default session storage directory.
func GetSessionsDir() string {
	// Check env var first
	if dir := os.Getenv("PI_SESSION_DIR"); dir != "" {
		return dir
	}
	return filepath.Join(GetAgentDir(), "sessions")
}

// GetDefaultSessionDir returns the default session dir for a given cwd and agentDir.
func GetDefaultSessionDir(cwd, agentDir string) string {
	return filepath.Join(agentDir, "sessions")
}

// GetDocsPath returns the path to documentation files.
func GetDocsPath() string {
	return "https://pi.dev/docs"
}

// GetBinDir returns the path to the bin directory for downloaded tools.
func GetBinDir() string {
	return filepath.Join(GetAgentDir(), "bin")
}

// GetSettingsPath returns the path to the settings file.
func GetSettingsPath() string {
	return filepath.Join(GetAgentDir(), "settings.json")
}

// IsLocalPath checks if a path looks like a local file path (not a URL).
func IsLocalPath(path string) bool {
	return !strings.HasPrefix(path, "http://") && !strings.HasPrefix(path, "https://")
}

// ExpandPath expands ~ to the home directory.
func ExpandPath(path string) string {
	if path == "~" {
		home, _ := os.UserHomeDir()
		return home
	}
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[2:])
	}
	return path
}

// FindExecutable searches for an executable on PATH.
func FindExecutable(name string) string {
	path, err := exec.LookPath(name)
	if err != nil {
		return ""
	}
	return path
}

// HasBash checks if bash is available on the system.
func HasBash() bool {
	if runtime.GOOS == "windows" {
		// Check Git Bash locations
		paths := []string{
			os.Getenv("ProgramFiles") + "\\Git\\bin\\bash.exe",
			os.Getenv("ProgramFiles(x86)") + "\\Git\\bin\\bash.exe",
		}
		for _, p := range paths {
			if p != "" {
				if _, err := os.Stat(p); err == nil {
					return true
				}
			}
		}
		// Check PATH
		if _, err := exec.LookPath("bash.exe"); err == nil {
			return true
		}
		return false
	}
	// Unix: check /bin/bash or PATH
	if _, err := os.Stat("/bin/bash"); err == nil {
		return true
	}
	_, err := exec.LookPath("bash")
	return err == nil
}
